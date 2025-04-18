/**
 * Copyright 2025 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import {LitElement, html} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {consume} from '@lit/context';
import {
  AppBookmarkInfo,
  SavedSearchInternalError,
  SavedSearchNotFoundError,
  SavedSearchUnknownError,
  UserSavedSearchesInternalError,
  UserSavedSearchesUnknownError,
  appBookmarkInfoContext,
} from '../../contexts/app-bookmark-info-context.js';
import {WebstatusBookmarksService} from '../webstatus-bookmarks-service.js';
import {fixture, expect, waitUntil} from '@open-wc/testing';
import '../webstatus-bookmarks-service.js';
import {
  DEFAULT_GLOBAL_SAVED_SEARCHES,
  UserSavedSearch,
} from '../../utils/constants.js';
import {APIClient} from '../../api/client.js';
import {SinonStubbedInstance} from 'sinon';
import {TaskStatus} from '@lit/task';
import sinon from 'sinon';
import {NotFoundError, ApiError} from '../../api/errors.js';
import {Toast} from '../../utils/toast.js';
import {User} from '../../contexts/firebase-user-context.js';

@customElement('test-bookmark-consumer')
class TestBookmarkConsumer extends LitElement {
  @consume({context: appBookmarkInfoContext, subscribe: true})
  @state()
  appBookmarkInfo?: AppBookmarkInfo;

  render() {
    return html`<div>
      ${this.appBookmarkInfo?.globalSavedSearches?.length}
    </div>`;
  }
}

describe('webstatus-bookmarks-service', () => {
  let toastStub: SinonStubbedInstance<Toast>;
  let getSearchIDStub: sinon.SinonStub;
  let updatePageUrlStub: sinon.SinonStub;
  let getLocationStub: sinon.SinonStub;
  beforeEach(() => {
    toastStub = sinon.stub(Toast.prototype);
    getSearchIDStub = sinon.stub();
    updatePageUrlStub = sinon.stub();
    getLocationStub = sinon.stub();
  });
  afterEach(() => {
    sinon.restore();
  });
  it('can be added to the page with the defaults', async () => {
    const component = await fixture<WebstatusBookmarksService>(
      html`<webstatus-bookmarks-service .user=${null}>
      </webstatus-bookmarks-service>`,
    );
    expect(component).to.exist;
    expect(component!.appBookmarkInfo.globalSavedSearches).to.deep.equal(
      DEFAULT_GLOBAL_SAVED_SEARCHES,
    );
    expect(component!.appBookmarkInfo.currentGlobalSavedSearch).to.deep.equal(
      undefined,
    );
  });
  it('provides appBookmarkInfo to consuming components', async () => {
    getLocationStub.returns({search: '', href: '', pathname: ''});
    const el = await fixture<WebstatusBookmarksService>(html`
      <webstatus-bookmarks-service
        .getLocation=${getLocationStub}
        .user=${null}
      >
        <test-bookmark-consumer></test-bookmark-consumer>
      </webstatus-bookmarks-service>
    `);
    const consumer = el.querySelector<TestBookmarkConsumer>(
      'test-bookmark-consumer',
    );
    expect(el).to.exist;
    expect(consumer).to.exist;
    expect(consumer!.appBookmarkInfo!.globalSavedSearches).to.deep.equal(
      DEFAULT_GLOBAL_SAVED_SEARCHES,
    );
    expect(consumer!.appBookmarkInfo!.currentGlobalSavedSearch).to.deep.equal(
      undefined,
    );
  });

  it('updates appBookmarkInfo on popstate event', async () => {
    // Will be used during the popstate event
    getLocationStub.returns({
      search: '?q=test_query_1',
      href: '?q=test_query_1',
      pathname: '',
    });
    const el = await fixture<WebstatusBookmarksService>(html`
      <webstatus-bookmarks-service
        .getLocation=${getLocationStub}
        .user=${null}
      >
        <test-bookmark-consumer></test-bookmark-consumer>
      </webstatus-bookmarks-service>
    `);
    const consumer = el.querySelector<TestBookmarkConsumer>(
      'test-bookmark-consumer',
    );
    el._globalSavedSearches = [
      {
        name: 'Test Bookmark 1',
        query: 'test_query_1',
      },
    ];
    el.appBookmarkInfo = {
      globalSavedSearches: [
        {
          name: 'Test Bookmark 1',
          query: 'test_query_1',
        },
      ],
      currentGlobalSavedSearch: undefined,
    };
    await el.updateComplete;
    await consumer!.updateComplete;

    // Initial state
    expect(consumer!.appBookmarkInfo).to.deep.equal({
      globalSavedSearches: [
        {
          name: 'Test Bookmark 1',
          query: 'test_query_1',
        },
      ],
      currentGlobalSavedSearch: undefined,
    });

    // Simulate popstate event with a query
    const popStateEvent = new PopStateEvent('popstate', {
      state: {},
    });
    window.dispatchEvent(popStateEvent);
    await el.updateComplete;
    await consumer!.updateComplete;

    // Updated state
    expect(consumer!.appBookmarkInfo!.globalSavedSearches).to.deep.equal([
      {
        name: 'Test Bookmark 1',
        query: 'test_query_1',
      },
    ]);
    expect(consumer!.appBookmarkInfo!.currentGlobalSavedSearch).to.deep.equal({
      name: 'Test Bookmark 1',
      query: 'test_query_1',
    });
  });
  describe('loadingUserSavedBookmarkByIDTask', () => {
    let apiClientStub: SinonStubbedInstance<APIClient>;

    beforeEach(async () => {
      apiClientStub = sinon.stub(new APIClient(''));
    });

    it('should handle NotFoundError', async () => {
      apiClientStub = sinon.stub(new APIClient(''));
      apiClientStub.getSavedSearchByID.rejects(new NotFoundError(''));
      const service = await fixture<WebstatusBookmarksService>(
        html`<webstatus-bookmarks-service
          .apiClient=${apiClientStub}
          .getSearchID=${() => 'test'}
          .user=${null}
          .getLocation=${() => {
            return {
              search: '?search_id=test',
              href: '?search_id=test',
              pathname: '',
            };
          }}
        ></webstatus-bookmarks-service>`,
      );

      await waitUntil(
        () =>
          service.appBookmarkInfo.userSavedSearchTask?.status !==
          TaskStatus.PENDING,
      );
      expect(
        service.appBookmarkInfo.userSavedSearchTask?.error,
      ).to.be.instanceOf(SavedSearchNotFoundError);
      expect(toastStub.toast).to.have.been.calledOnceWithExactly(
        'Saved search with id test not found',
        'danger',
        'exclamation-triangle',
      );
    });

    it('should handle ApiError', async () => {
      apiClientStub.getSavedSearchByID.rejects(
        new ApiError('Something went wrong', 500),
      );
      const service = await fixture<WebstatusBookmarksService>(
        html`<webstatus-bookmarks-service
          .apiClient=${apiClientStub}
          .getSearchID=${() => 'test'}
          .user=${null}
          .getLocation=${() => {
            return {
              search: '?search_id=test',
              href: '?search_id=test',
              pathname: '',
            };
          }}
        ></webstatus-bookmarks-service>`,
      );
      await waitUntil(
        () =>
          service.appBookmarkInfo.userSavedSearchTask?.status !==
          TaskStatus.PENDING,
      );
      expect(
        service.appBookmarkInfo.userSavedSearchTask?.error,
      ).to.be.instanceOf(SavedSearchInternalError);
      expect(toastStub.toast).to.have.been.calledOnceWithExactly(
        'Error fetching saved search ID test: Something went wrong',
        'danger',
        'exclamation-triangle',
      );
    });

    it('should handle unknown errors', async () => {
      apiClientStub.getSavedSearchByID.rejects(
        new Error('Saved Search Unknown Test Error'),
      );
      const service = await fixture<WebstatusBookmarksService>(
        html`<webstatus-bookmarks-service
          .apiClient=${apiClientStub}
          .getSearchID=${() => 'test'}
          .user=${null}
          .getLocation=${() => {
            return {
              search: '?search_id=test',
              href: '?search_id=test',
              pathname: '',
            };
          }}
        ></webstatus-bookmarks-service>`,
      );
      await waitUntil(
        () =>
          service.appBookmarkInfo.userSavedSearchTask?.status !==
          TaskStatus.PENDING,
      );
      expect(
        service.appBookmarkInfo.userSavedSearchTask?.error,
      ).to.be.instanceOf(SavedSearchUnknownError);
      expect(toastStub.toast).to.have.been.calledOnceWithExactly(
        'Unknown error fetching saved search ID test. Check console for details.',
        'danger',
        'exclamation-triangle',
      );
    });

    it('should complete successfully if searchID is not found', async () => {
      apiClientStub.getSavedSearchByID.resolves(undefined);
      const service = await fixture<WebstatusBookmarksService>(
        html`<webstatus-bookmarks-service
          .apiClient=${apiClientStub}
          .getSearchID=${() => 'test'}
          .user=${null}
          .getLocation=${() => {
            return {
              search: '?search_id=test',
              href: '?search_id=test',
              pathname: '',
            };
          }}
        ></webstatus-bookmarks-service>`,
      );
      await waitUntil(
        () =>
          service.appBookmarkInfo.userSavedSearchTask?.status !==
          TaskStatus.PENDING,
      );
      expect(service.appBookmarkInfo.userSavedSearchTask?.status).to.equal(
        TaskStatus.COMPLETE,
      );
      expect(service.appBookmarkInfo.userSavedSearchTask?.data).to.be.undefined;
    });

    it('should complete successfully with bookmark data', async () => {
      const mockBookmark = {
        id: '123',
        query: 'test',
        name: 'Test Bookmark',
        description: 'Test Description',
        created_at: '2023-08-13',
        updated_at: '2023-08-13',
      };
      apiClientStub.getSavedSearchByID.resolves(mockBookmark);
      const service = await fixture<WebstatusBookmarksService>(
        html`<webstatus-bookmarks-service
          .apiClient=${apiClientStub}
          .getSearchID=${() => 'test'}
          .user=${null}
          .getLocation=${() => {
            return {
              search: '?search_id=test',
              href: '?search_id=test',
              pathname: '',
            };
          }}
        ></webstatus-bookmarks-service>`,
      );
      await waitUntil(
        () =>
          service.appBookmarkInfo.userSavedSearchTask?.status !==
          TaskStatus.PENDING,
      );
      expect(service.appBookmarkInfo.userSavedSearchTask?.status).to.equal(
        TaskStatus.COMPLETE,
      );
      expect(service.appBookmarkInfo.userSavedSearchTask?.data).to.deep.equal(
        mockBookmark,
      );
    });

    it('should complete successfully with saved searches data and update page url when "q" is the same as the saved search\'s query', async () => {
      const mockSavedSearch = {
        id: '123',
        query: 'foo',
        name: 'Test Bookmark',
        description: 'Test Description',
        created_at: '2023-08-13',
        updated_at: '2023-08-13',
      };
      // First call is for the current search ID, second call is for the previous search ID
      getSearchIDStub.onCall(0).returns('test');
      getSearchIDStub.onCall(1).returns('');
      apiClientStub.getSavedSearchByID.resolves(mockSavedSearch);
      const mockLocation = {
        search: '?search_id=test&q=foo',
        href: '?search_id=test&q=foo',
        pathname: '',
      };
      getLocationStub.returns(mockLocation);
      const service = await fixture<WebstatusBookmarksService>(
        html`<webstatus-bookmarks-service
          .apiClient=${apiClientStub}
          .getSearchID=${getSearchIDStub}
          .updatePageUrl=${updatePageUrlStub}
          .getLocation=${getLocationStub}
          .user=${null}
        ></webstatus-bookmarks-service>`,
      );
      await waitUntil(
        () =>
          service.appBookmarkInfo.userSavedSearchTask?.status !==
          TaskStatus.PENDING,
        '',
        {timeout: 5000},
      );
      expect(apiClientStub.getSavedSearchByID).to.have.been.calledOnce;
      expect(service.appBookmarkInfo.userSavedSearchTask?.status).to.equal(
        TaskStatus.COMPLETE,
      );
      expect(service.appBookmarkInfo.userSavedSearchTask?.data).to.deep.equal(
        mockSavedSearch,
      );
      expect(getLocationStub.callCount).to.eq(1);
      expect(updatePageUrlStub).to.have.been.calledOnce;
    });

    it('should use the cache for duplicate runs', async () => {
      apiClientStub.getSavedSearchByID.resolves({
        id: 'test',
        query: 'test',
        name: 'test',
        created_at: '',
        updated_at: '',
      });
      getLocationStub.returns({
        search: '?search_id=test',
        href: '?search_id=test',
        pathname: '',
      });
      getSearchIDStub.returns('test');

      const service = await fixture<WebstatusBookmarksService>(
        html`<webstatus-bookmarks-service
          .apiClient=${apiClientStub}
          .getSearchID=${getSearchIDStub}
          .getLocation=${getLocationStub}
          .user=${null}
        ></webstatus-bookmarks-service>`,
      );

      // Assert that getSavedSearchByID was only called once
      expect(apiClientStub.getSavedSearchByID.calledOnce).to.be.true;
      expect(service.appBookmarkInfo.userSavedSearchTask?.status).to.eq(
        TaskStatus.COMPLETE,
      );
      expect(service.appBookmarkInfo.userSavedSearchTask?.data).to.deep.eq({
        created_at: '',
        id: 'test',
        query: 'test',
        name: 'test',
        updated_at: '',
      });

      // Manually trigger a re-run
      service.loadingUserSavedSearchByIDTask.run();
      await service.updateComplete;

      // Assert that getSavedSearchByID was only called once still
      expect(apiClientStub.getSavedSearchByID.calledOnce).to.be.true;
      expect(service.appBookmarkInfo.userSavedSearchTask?.status).to.eq(
        TaskStatus.COMPLETE,
      );
    });
  });

  describe('loadingUserSavedSearchesTask', () => {
    let apiClientStub: SinonStubbedInstance<APIClient>;
    beforeEach(async () => {
      apiClientStub = sinon.stub(new APIClient(''));
    });

    it('should handle ApiError', async () => {
      apiClientStub.getAllUserSavedSearches.rejects(
        new ApiError('Something went wrong', 500),
      );
      const service = await fixture<WebstatusBookmarksService>(
        html`<webstatus-bookmarks-service
          .apiClient=${apiClientStub}
          .user=${{
            getIdToken: async () => 'test-token',
          } as User}
        ></webstatus-bookmarks-service>`,
      );
      expect(service.appBookmarkInfo.userSavedSearchesTask?.status).to.eq(
        TaskStatus.ERROR,
      );
      expect(
        service.appBookmarkInfo.userSavedSearchesTask?.error,
      ).to.be.instanceOf(UserSavedSearchesInternalError);
      expect(toastStub.toast).to.have.been.calledOnceWithExactly(
        'Internal error fetching list of saved searches for user: Something went wrong',
        'danger',
        'exclamation-triangle',
      );
    });

    it('should handle unknown errors', async () => {
      apiClientStub.getAllUserSavedSearches.rejects(
        new Error('User Saved Searches Unknown Test Error'),
      );
      const service = await fixture<WebstatusBookmarksService>(
        html`<webstatus-bookmarks-service
          .apiClient=${apiClientStub}
          .user=${{
            getIdToken: async () => 'test-token',
          } as User}
        ></webstatus-bookmarks-service>`,
      );
      expect(service.appBookmarkInfo.userSavedSearchesTask?.status).to.eq(
        TaskStatus.ERROR,
      );
      expect(
        service.appBookmarkInfo.userSavedSearchesTask?.error,
      ).to.be.instanceOf(UserSavedSearchesUnknownError);
      expect(toastStub.toast).to.have.been.calledOnceWithExactly(
        'Unknown error fetching list of saved searches for user. Check console for details.',
        'danger',
        'exclamation-triangle',
      );
    });

    it('should complete successfully with bookmark data', async () => {
      const mockSavedSearches = [
        {
          id: '123',
          query: 'test',
          name: 'Test Bookmark',
          created_at: '2023-08-13',
          updated_at: '2023-08-13',
        },
      ];
      apiClientStub.getAllUserSavedSearches.resolves(mockSavedSearches);
      const service = await fixture<WebstatusBookmarksService>(
        html`<webstatus-bookmarks-service
          .apiClient=${apiClientStub}
          .user=${{
            getIdToken: async () => 'test-token',
          } as User}
        ></webstatus-bookmarks-service>`,
      );
      expect(service.appBookmarkInfo.userSavedSearchesTask?.status).to.equal(
        TaskStatus.COMPLETE,
      );
      expect(service.appBookmarkInfo.userSavedSearchesTask?.data).to.deep.equal(
        mockSavedSearches,
      );
    });

    it('should handle null user', async () => {
      const service = await fixture<WebstatusBookmarksService>(
        html`<webstatus-bookmarks-service
          .apiClient=${apiClientStub}
          .user=${null}
        ></webstatus-bookmarks-service>`,
      );
      expect(service.appBookmarkInfo.userSavedSearchesTask?.status).to.eq(
        TaskStatus.COMPLETE,
      );
      expect(service.appBookmarkInfo.userSavedSearchesTask?.data).to.be
        .undefined;
    });
  });

  describe('findCurrentBookmarkByQuery', () => {
    it('should return undefined for empty saved searches', () => {
      const service = new WebstatusBookmarksService();
      expect(service.findCurrentSavedSearchByQuery()).to.be.undefined;
    });

    it('should find a matching saved search', () => {
      const service = new WebstatusBookmarksService();
      service._currentLocation = {
        search: '?q=test',
        href: '?q=test',
        pathname: '',
      };
      const savedSearches = [{query: 'test', name: 'Test'}];
      expect(
        service.findCurrentSavedSearchByQuery(savedSearches),
      ).to.deep.equal({
        query: 'test',
        name: 'Test',
      });
    });

    it('should return undefined if no saved search matches', () => {
      const service = new WebstatusBookmarksService();
      service._currentLocation = {
        search: '?q=test',
        href: '?q=test',
        pathname: '',
      };
      const savedSearches = [{query: 'other', name: 'Other'}];
      expect(service.findCurrentSavedSearchByQuery(savedSearches)).to.be
        .undefined;
    });
  });

  it('should refresh appBookmarkInfo correctly', () => {
    const service = new WebstatusBookmarksService();
    service._globalSavedSearches = [{query: 'global', name: 'Global'}];
    service._currentGlobalSavedSearch = {query: 'global', name: 'Global'};
    service._userSavedSearchByIDTaskTracker = {
      status: TaskStatus.COMPLETE,
      data: {id: 'uuid', query: 'saved', name: 'Saved'},
      error: undefined,
    };
    service._userSavedSearchesTaskTracker = {
      status: TaskStatus.COMPLETE,
      data: [
        {id: 'uuid1', query: 'saved', name: 'Saved'},
        {id: 'uuid2', query: 'saved', name: 'Saved'},
      ],
      error: undefined,
    };
    service._currentLocation = {
      search: '?q=test',
      href: '?q=test',
      pathname: '',
    };
    service.refreshAppBookmarkInfo();
    expect(service.appBookmarkInfo).to.deep.equal({
      globalSavedSearches: [{query: 'global', name: 'Global'}],
      currentGlobalSavedSearch: {query: 'global', name: 'Global'},
      userSavedSearchTask: {
        status: TaskStatus.COMPLETE,
        data: {id: 'uuid', query: 'saved', name: 'Saved'},
        error: undefined,
      },
      currentLocation: {search: '?q=test', href: '?q=test', pathname: ''},
      userSavedSearchesTask: {
        status: TaskStatus.COMPLETE,
        data: [
          {id: 'uuid1', query: 'saved', name: 'Saved'},
          {id: 'uuid2', query: 'saved', name: 'Saved'},
        ],
        error: undefined,
      },
    });
  });

  it('getLocation should return the current location', () => {
    const service = new WebstatusBookmarksService();
    const location = service.getLocation();
    expect(location).to.be.an('object');
    expect(location).to.have.property('search');
    expect(location).to.have.property('href');
    expect(location).to.have.property('pathname');
  });

  describe('Event Handlers', () => {
    let service: WebstatusBookmarksService;
    const mockLocation = {
      search: '?q=initial',
      href: '?q=initial',
      pathname: '/features',
    };
    const mockSavedSearch1: UserSavedSearch = {
      id: 'uuid1',
      query: 'query1',
      name: 'Saved 1',
      created_at: '2024-01-01',
      updated_at: '2024-01-01',
    };
    const mockSavedSearch2: UserSavedSearch = {
      id: 'uuid2',
      query: 'query2',
      name: 'Saved 2',
      created_at: '2024-01-02',
      updated_at: '2024-01-02',
    };
    const mockNewSavedSearch: UserSavedSearch = {
      id: 'uuid-new',
      query: 'query-new',
      name: 'New Saved',
      created_at: '2024-01-03',
      updated_at: '2024-01-03',
    };

    beforeEach(async () => {
      getLocationStub.returns(mockLocation);
      service = await fixture<WebstatusBookmarksService>(
        html`<webstatus-bookmarks-service
          .getLocation=${getLocationStub}
          .updatePageUrl=${updatePageUrlStub}
          .user=${null}
        ></webstatus-bookmarks-service>`,
      );
      // Ensure initial state is set for handlers
      service._currentLocation = mockLocation;
    });

    it('handleSavedSearchSaved should add new search and update URL', () => {
      // Initial state (empty)
      expect(service.appBookmarkInfo.userSavedSearchesTask?.data).to.be
        .undefined;
      expect(service.appBookmarkInfo.userSavedSearchTask?.data).to.be.undefined;

      const event = new CustomEvent<UserSavedSearch>('saved-search-saved', {
        detail: mockNewSavedSearch,
      });
      service.dispatchEvent(event);

      expect(service.appBookmarkInfo.userSavedSearchesTask?.data).to.deep.equal(
        [mockNewSavedSearch],
      );
      expect(service.appBookmarkInfo.userSavedSearchTask?.data).to.deep.equal(
        mockNewSavedSearch,
      );
      expect(updatePageUrlStub).to.have.been.calledOnceWithExactly(
        mockLocation.pathname,
        mockLocation,
        {search_id: mockNewSavedSearch.id},
      );
    });

    it('handleSavedSearchSaved should append to existing searches', () => {
      // Set initial state with one search
      service._userSavedSearchesTaskTracker = {
        status: TaskStatus.COMPLETE,
        data: [mockSavedSearch1],
        error: undefined,
      };

      const event = new CustomEvent<UserSavedSearch>('saved-search-saved', {
        detail: mockNewSavedSearch,
      });
      service.dispatchEvent(event);

      expect(service.appBookmarkInfo.userSavedSearchesTask?.data).to.deep.equal(
        [mockSavedSearch1, mockNewSavedSearch],
      );
      // It should also update the current single search tracker
      expect(service.appBookmarkInfo.userSavedSearchTask?.data).to.deep.equal(
        mockNewSavedSearch,
      );
      expect(updatePageUrlStub).to.have.been.calledOnceWithExactly(
        mockLocation.pathname,
        mockLocation,
        {search_id: mockNewSavedSearch.id},
      );
    });

    it('handleSavedSearchEdited should update existing search', () => {
      // Set initial state
      service._userSavedSearchesTaskTracker = {
        status: TaskStatus.COMPLETE,
        data: [mockSavedSearch1, mockSavedSearch2],
        error: undefined,
      };
      service._userSavedSearchByIDTaskTracker = {
        status: TaskStatus.COMPLETE,
        data: mockSavedSearch1, // Currently viewing mockSavedSearch1
        error: undefined,
      };

      const editedSearch: UserSavedSearch = {
        ...mockSavedSearch1,
        name: 'Edited Name',
        query: 'edited-query',
      };
      const event = new CustomEvent<UserSavedSearch>('saved-search-edited', {
        detail: editedSearch,
      });
      service.dispatchEvent(event);

      // Check list update
      expect(service.appBookmarkInfo.userSavedSearchesTask?.data).to.deep.equal(
        [editedSearch, mockSavedSearch2],
      );
      // Check current search update
      expect(service.appBookmarkInfo.userSavedSearchTask?.data).to.deep.equal(
        editedSearch,
      );
      // URL should not change on edit
      expect(updatePageUrlStub).not.to.have.been.called;
    });

    it('handleSavedSearchEdited should only update list if current search is different', () => {
      // Set initial state
      service._userSavedSearchesTaskTracker = {
        status: TaskStatus.COMPLETE,
        data: [mockSavedSearch1, mockSavedSearch2],
        error: undefined,
      };
      service._userSavedSearchByIDTaskTracker = {
        status: TaskStatus.COMPLETE,
        data: mockSavedSearch2, // Currently viewing mockSavedSearch2
        error: undefined,
      };

      const editedSearch: UserSavedSearch = {
        ...mockSavedSearch1, // Editing search 1
        name: 'Edited Name',
      };
      const event = new CustomEvent<UserSavedSearch>('saved-search-edited', {
        detail: editedSearch,
      });
      service.dispatchEvent(event);

      // Check list update
      expect(service.appBookmarkInfo.userSavedSearchesTask?.data).to.deep.equal(
        [editedSearch, mockSavedSearch2],
      );
      // Check current search (should NOT be updated)
      expect(service.appBookmarkInfo.userSavedSearchTask?.data).to.deep.equal(
        mockSavedSearch2,
      );
      expect(updatePageUrlStub).not.to.have.been.called;
    });

    it('handleSavedSearchDeleted should remove search and clear URL', () => {
      // Set initial state
      service._userSavedSearchesTaskTracker = {
        status: TaskStatus.COMPLETE,
        data: [mockSavedSearch1, mockSavedSearch2],
        error: undefined,
      };
      service._userSavedSearchByIDTaskTracker = {
        status: TaskStatus.COMPLETE,
        data: mockSavedSearch1, // Currently viewing the one to be deleted
        error: undefined,
      };

      const event = new CustomEvent<string>('saved-search-deleted', {
        detail: mockSavedSearch1.id, // Delete search 1
      });
      service.dispatchEvent(event);

      // Check list update
      expect(service.appBookmarkInfo.userSavedSearchesTask?.data).to.deep.equal(
        [mockSavedSearch2],
      );
      // Check current search update (should be cleared)
      expect(service.appBookmarkInfo.userSavedSearchTask?.data).to.be.undefined;
      // Check URL update
      expect(updatePageUrlStub).to.have.been.calledOnceWithExactly(
        mockLocation.pathname,
        mockLocation,
        {search_id: ''},
      );
    });

    it('handleSavedSearchDeleted should only remove from list if current search is different', () => {
      // Set initial state
      service._userSavedSearchesTaskTracker = {
        status: TaskStatus.COMPLETE,
        data: [mockSavedSearch1, mockSavedSearch2],
        error: undefined,
      };
      service._userSavedSearchByIDTaskTracker = {
        status: TaskStatus.COMPLETE,
        data: mockSavedSearch2, // Currently viewing search 2
        error: undefined,
      };

      const event = new CustomEvent<string>('saved-search-deleted', {
        detail: mockSavedSearch1.id, // Delete search 1
      });
      service.dispatchEvent(event);

      // Check list update
      expect(service.appBookmarkInfo.userSavedSearchesTask?.data).to.deep.equal(
        [mockSavedSearch2],
      );
      // Check current search (should NOT be cleared)
      expect(service.appBookmarkInfo.userSavedSearchTask?.data).to.deep.equal(
        mockSavedSearch2,
      );
      // Check URL update (still called to clear potential search_id)
      expect(updatePageUrlStub).to.have.been.calledOnceWithExactly(
        mockLocation.pathname,
        mockLocation,
        {search_id: ''},
      );
    });
    it('saved-search-bookmarked event should add search and update URL (like saved)', () => {
      // Initial state (empty)
      expect(service.appBookmarkInfo.userSavedSearchesTask?.data).to.be
        .undefined;
      expect(service.appBookmarkInfo.userSavedSearchTask?.data).to.be.undefined;

      const event = new CustomEvent<UserSavedSearch>(
        'saved-search-bookmarked',
        {
          detail: mockNewSavedSearch,
        },
      );
      service.dispatchEvent(event);

      // Verify it behaves like 'saved-search-saved'
      expect(service.appBookmarkInfo.userSavedSearchesTask?.data).to.deep.equal(
        [mockNewSavedSearch],
      );
      expect(service.appBookmarkInfo.userSavedSearchTask?.data).to.deep.equal(
        mockNewSavedSearch,
      );
      expect(updatePageUrlStub).to.have.been.calledOnceWithExactly(
        mockLocation.pathname,
        mockLocation,
        {search_id: mockNewSavedSearch.id},
      );
    });
    it('saved-search-unbookmarked event should not clear current search if unbookmarking a different search', () => {
      // Set initial state
      service._userSavedSearchesTaskTracker = {
        status: TaskStatus.COMPLETE,
        data: [mockSavedSearch1, mockSavedSearch2],
        error: undefined,
      };
      service._userSavedSearchByIDTaskTracker = {
        status: TaskStatus.COMPLETE,
        data: mockSavedSearch2, // Currently viewing search 2
        error: undefined,
      };

      const event = new CustomEvent<string>('saved-search-unbookmarked', {
        detail: mockSavedSearch1.id, // Unbookmark search 1
      });
      service.dispatchEvent(event);

      // Verify list update
      expect(service.appBookmarkInfo.userSavedSearchesTask?.data).to.deep.equal(
        [mockSavedSearch2], // Search 1 removed
      );
      // Verify current search is NOT cleared
      expect(service.appBookmarkInfo.userSavedSearchTask?.data).to.deep.equal(
        mockSavedSearch2,
      );
      // Verify URL is still cleared
      expect(updatePageUrlStub).to.have.been.calledOnceWithExactly(
        mockLocation.pathname,
        mockLocation,
        {search_id: ''},
      );
    });
  });
});
