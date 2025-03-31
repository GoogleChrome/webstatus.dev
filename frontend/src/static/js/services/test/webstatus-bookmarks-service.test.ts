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
import {DEFAULT_BOOKMARKS} from '../../utils/constants.js';
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
    return html`<div>${this.appBookmarkInfo?.globalBookmarks?.length}</div>`;
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
      html`<webstatus-bookmarks-service> </webstatus-bookmarks-service>`,
    );
    expect(component).to.exist;
    expect(component!.appBookmarkInfo.globalBookmarks).to.deep.equal(
      DEFAULT_BOOKMARKS,
    );
    expect(component!.appBookmarkInfo.currentGlobalBookmark).to.deep.equal(
      undefined,
    );
  });
  it('provides appBookmarkInfo to consuming components', async () => {
    getLocationStub.returns({search: '', href: '', pathname: ''});
    const el = await fixture<WebstatusBookmarksService>(html`
      <webstatus-bookmarks-service .getLocation=${getLocationStub}>
        <test-bookmark-consumer></test-bookmark-consumer>
      </webstatus-bookmarks-service>
    `);
    const consumer = el.querySelector<TestBookmarkConsumer>(
      'test-bookmark-consumer',
    );
    expect(el).to.exist;
    expect(consumer).to.exist;
    expect(consumer!.appBookmarkInfo!.globalBookmarks).to.deep.equal(
      DEFAULT_BOOKMARKS,
    );
    expect(consumer!.appBookmarkInfo!.currentGlobalBookmark).to.deep.equal(
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
      <webstatus-bookmarks-service .getLocation=${getLocationStub}>
        <test-bookmark-consumer></test-bookmark-consumer>
      </webstatus-bookmarks-service>
    `);
    const consumer = el.querySelector<TestBookmarkConsumer>(
      'test-bookmark-consumer',
    );
    el._globalBookmarks = [
      {
        name: 'Test Bookmark 1',
        query: 'test_query_1',
      },
    ];
    el.appBookmarkInfo = {
      globalBookmarks: [
        {
          name: 'Test Bookmark 1',
          query: 'test_query_1',
        },
      ],
      currentGlobalBookmark: undefined,
    };
    await el.updateComplete;
    await consumer!.updateComplete;

    // Initial state
    expect(consumer!.appBookmarkInfo).to.deep.equal({
      globalBookmarks: [
        {
          name: 'Test Bookmark 1',
          query: 'test_query_1',
        },
      ],
      currentGlobalBookmark: undefined,
    });

    // Simulate popstate event with a query
    const popStateEvent = new PopStateEvent('popstate', {
      state: {},
    });
    window.dispatchEvent(popStateEvent);
    await el.updateComplete;
    await consumer!.updateComplete;

    // Updated state
    expect(consumer!.appBookmarkInfo!.globalBookmarks).to.deep.equal([
      {
        name: 'Test Bookmark 1',
        query: 'test_query_1',
      },
    ]);
    expect(consumer!.appBookmarkInfo!.currentGlobalBookmark).to.deep.equal({
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
          service.appBookmarkInfo.userSavedSearchBookmarkTask?.status !==
          TaskStatus.PENDING,
      );
      expect(
        service.appBookmarkInfo.userSavedSearchBookmarkTask?.error,
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
          service.appBookmarkInfo.userSavedSearchBookmarkTask?.status !==
          TaskStatus.PENDING,
      );
      expect(
        service.appBookmarkInfo.userSavedSearchBookmarkTask?.error,
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
          service.appBookmarkInfo.userSavedSearchBookmarkTask?.status !==
          TaskStatus.PENDING,
      );
      expect(
        service.appBookmarkInfo.userSavedSearchBookmarkTask?.error,
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
          service.appBookmarkInfo.userSavedSearchBookmarkTask?.status !==
          TaskStatus.PENDING,
      );
      expect(
        service.appBookmarkInfo.userSavedSearchBookmarkTask?.status,
      ).to.equal(TaskStatus.COMPLETE);
      expect(service.appBookmarkInfo.userSavedSearchBookmarkTask?.data).to.be
        .undefined;
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
          service.appBookmarkInfo.userSavedSearchBookmarkTask?.status !==
          TaskStatus.PENDING,
      );
      expect(
        service.appBookmarkInfo.userSavedSearchBookmarkTask?.status,
      ).to.equal(TaskStatus.COMPLETE);
      expect(
        service.appBookmarkInfo.userSavedSearchBookmarkTask?.data,
      ).to.deep.equal(mockBookmark);
    });

    it('should complete successfully with bookmark data and update page url when "q" is the same as the bookmark\'s query', async () => {
      const mockBookmark = {
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
      apiClientStub.getSavedSearchByID.resolves(mockBookmark);
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
        ></webstatus-bookmarks-service>`,
      );
      await waitUntil(
        () =>
          service.appBookmarkInfo.userSavedSearchBookmarkTask?.status !==
          TaskStatus.PENDING,
        '',
        {timeout: 5000},
      );
      expect(apiClientStub.getSavedSearchByID).to.have.been.calledOnce;
      expect(
        service.appBookmarkInfo.userSavedSearchBookmarkTask?.status,
      ).to.equal(TaskStatus.COMPLETE);
      expect(
        service.appBookmarkInfo.userSavedSearchBookmarkTask?.data,
      ).to.deep.equal(mockBookmark);
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
        ></webstatus-bookmarks-service>`,
      );

      // Assert that getSavedSearchByID was only called once
      expect(apiClientStub.getSavedSearchByID.calledOnce).to.be.true;
      expect(service.appBookmarkInfo.userSavedSearchBookmarkTask?.status).to.eq(
        TaskStatus.COMPLETE,
      );
      expect(
        service.appBookmarkInfo.userSavedSearchBookmarkTask?.data,
      ).to.deep.eq({
        created_at: '',
        id: 'test',
        query: 'test',
        name: 'test',
        updated_at: '',
      });

      // Manually trigger a re-run
      service.loadingUserSavedBookmarkByIDTask.run();
      await service.updateComplete;

      // Assert that getSavedSearchByID was only called once still
      expect(apiClientStub.getSavedSearchByID.calledOnce).to.be.true;
      expect(service.appBookmarkInfo.userSavedSearchBookmarkTask?.status).to.eq(
        TaskStatus.COMPLETE,
      );
    });
  });

  describe('loadingUserSavedBookmarksTask', () => {
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
      expect(
        service.appBookmarkInfo.userSavedSearchBookmarksTask?.status,
      ).to.eq(TaskStatus.ERROR);
      expect(
        service.appBookmarkInfo.userSavedSearchBookmarksTask?.error,
      ).to.be.instanceOf(UserSavedSearchesInternalError);
      expect(toastStub.toast).to.have.been.calledOnceWithExactly(
        'Internal error fetching list of bookmarked saved searches for user: Something went wrong',
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
      expect(
        service.appBookmarkInfo.userSavedSearchBookmarksTask?.status,
      ).to.eq(TaskStatus.ERROR);
      expect(
        service.appBookmarkInfo.userSavedSearchBookmarksTask?.error,
      ).to.be.instanceOf(UserSavedSearchesUnknownError);
      expect(toastStub.toast).to.have.been.calledOnceWithExactly(
        'Unknown error fetching list of bookmarked saved searches for user. Check console for details.',
        'danger',
        'exclamation-triangle',
      );
    });

    it('should complete successfully with bookmark data', async () => {
      const mockBookmarks = [
        {
          id: '123',
          query: 'test',
          name: 'Test Bookmark',
          created_at: '2023-08-13',
          updated_at: '2023-08-13',
        },
      ];
      apiClientStub.getAllUserSavedSearches.resolves(mockBookmarks);
      const service = await fixture<WebstatusBookmarksService>(
        html`<webstatus-bookmarks-service
          .apiClient=${apiClientStub}
          .user=${{
            getIdToken: async () => 'test-token',
          } as User}
        ></webstatus-bookmarks-service>`,
      );
      expect(
        service.appBookmarkInfo.userSavedSearchBookmarksTask?.status,
      ).to.equal(TaskStatus.COMPLETE);
      expect(
        service.appBookmarkInfo.userSavedSearchBookmarksTask?.data,
      ).to.deep.equal(mockBookmarks);
    });

    it('should handle null user', async () => {
      const service = await fixture<WebstatusBookmarksService>(
        html`<webstatus-bookmarks-service
          .apiClient=${apiClientStub}
          .user=${null}
        ></webstatus-bookmarks-service>`,
      );
      expect(
        service.appBookmarkInfo.userSavedSearchBookmarksTask?.status,
      ).to.eq(TaskStatus.COMPLETE);
      expect(service.appBookmarkInfo.userSavedSearchBookmarksTask?.data).to.be
        .undefined;
    });
  });

  describe('findCurrentBookmarkByQuery', () => {
    it('should return undefined for empty bookmarks', () => {
      const service = new WebstatusBookmarksService();
      expect(service.findCurrentBookmarkByQuery()).to.be.undefined;
    });

    it('should find a matching bookmark', () => {
      const service = new WebstatusBookmarksService();
      service._currentLocation = {
        search: '?q=test',
        href: '?q=test',
        pathname: '',
      };
      const bookmarks = [{query: 'test', name: 'Test'}];
      expect(service.findCurrentBookmarkByQuery(bookmarks)).to.deep.equal({
        query: 'test',
        name: 'Test',
      });
    });

    it('should return undefined if no bookmark matches', () => {
      const service = new WebstatusBookmarksService();
      service._currentLocation = {
        search: '?q=test',
        href: '?q=test',
        pathname: '',
      };
      const bookmarks = [{query: 'other', name: 'Other'}];
      expect(service.findCurrentBookmarkByQuery(bookmarks)).to.be.undefined;
    });
  });

  it('should refresh appBookmarkInfo correctly', () => {
    const service = new WebstatusBookmarksService();
    service._globalBookmarks = [{query: 'global', name: 'Global'}];
    service._currentGlobalBookmark = {query: 'global', name: 'Global'};
    service._userSavedBookmarkByIDTaskTracker = {
      status: TaskStatus.COMPLETE,
      data: {id: 'uuid', query: 'saved', name: 'Saved'},
      error: undefined,
    };
    service._userSavedBookmarksTaskTracker = {
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
      globalBookmarks: [{query: 'global', name: 'Global'}],
      currentGlobalBookmark: {query: 'global', name: 'Global'},
      userSavedSearchBookmarkTask: {
        status: TaskStatus.COMPLETE,
        data: {id: 'uuid', query: 'saved', name: 'Saved'},
        error: undefined,
      },
      currentLocation: {search: '?q=test', href: '?q=test', pathname: ''},
      userSavedSearchBookmarksTask: {
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
});
