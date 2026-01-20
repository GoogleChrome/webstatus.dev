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

import {fixture, expect, html, waitUntil, oneEvent} from '@open-wc/testing';
import sinon from 'sinon';
import {TaskStatus} from '@lit/task';

import '../webstatus-saved-search-controls.js';
import {WebstatusSavedSearchControls} from '../webstatus-saved-search-controls.js';
import {User} from '../../contexts/firebase-user-context.js';
import {
  BookmarkOwnerRole,
  BookmarkStatusActive,
  SavedSearchOperationType,
  UserSavedSearch,
} from '../../utils/constants.js';
import {APIClient} from '../../api/client.js';
import {ApiError} from '../../api/errors.js';
import * as toastUtils from '../../utils/toast.js';

// Mock child component
import {WebstatusTypeahead} from '../webstatus-typeahead.js';
import {SlIconButton} from '@shoelace-style/shoelace';

describe('WebstatusSavedSearchControls', () => {
  let element: WebstatusSavedSearchControls;
  let apiClientMock: sinon.SinonStubbedInstance<APIClient>;
  let userMock: User;
  let typeaheadMock: WebstatusTypeahead;
  let formatOverviewPageUrlStub: sinon.SinonStub;
  let openSavedSearch: sinon.SinonStub;
  let toastStub: sinon.SinonStub;

  const typeaheadQuery = 'mock query';

  const mockSavedSearchOwner: UserSavedSearch = {
    id: 'owner123',
    name: 'My Search',
    query: 'feature:css',
    description: 'A search I own',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    permissions: {role: BookmarkOwnerRole},
    bookmark_status: {status: BookmarkStatusActive}, // Owners always have it bookmarked implicitly
  };

  const mockSavedSearchViewerBookmarked: UserSavedSearch = {
    id: 'viewerBM456',
    name: 'Shared Search Bookmarked',
    query: 'feature:js',
    description: 'A search I view and bookmarked',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    permissions: undefined,
    bookmark_status: {status: BookmarkStatusActive},
  };

  const mockSavedSearchViewerNotBookmarked: UserSavedSearch = {
    id: 'viewerNB789',
    name: 'Shared Search Not Bookmarked',
    query: 'feature:html',
    description: 'A search I view but have not bookmarked',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    permissions: undefined,
    // No bookmark_status
  };

  const mockLocation = {search: '?q=feature:css'};

  beforeEach(async () => {
    apiClientMock = sinon.createStubInstance(APIClient);
    userMock = {
      user: {
        getIdToken: sinon.stub().resolves('mock-token'),
      },
      syncState: 'idle',
    } as unknown as User;

    toastStub = sinon.stub(toastUtils.Toast.prototype, 'toast');

    typeaheadMock = await fixture<WebstatusTypeahead>(
      html`<webstatus-typeahead
        .value=${typeaheadQuery}
      ></webstatus-typeahead>`,
    );

    element = await fixture<WebstatusSavedSearchControls>(html`
      <webstatus-saved-search-controls
        .apiClient=${apiClientMock}
        .user=${userMock}
        .location=${mockLocation}
        .overviewPageQueryInput=${typeaheadMock}
        .openSavedSearchDialog=${(
          _t: SavedSearchOperationType,
          _uss?: UserSavedSearch,
          _q?: string,
        ) => undefined}
      >
      </webstatus-saved-search-controls>
    `);

    element._getOrigin = () => 'http://localhost:8080';

    formatOverviewPageUrlStub = sinon
      .stub(element, '_formatOverviewPageUrl')
      .callsFake((location, params) => {
        const url = new URL('http://localhost:8080/features');
        url.search = location!.search;
        if (params?.search_id) {
          url.searchParams.set('search_id', params.search_id);
        }
        return url.pathname + url.search;
      });
    openSavedSearch = sinon.stub(element, 'openSavedSearch');
  });

  afterEach(() => {
    sinon.restore();
  });

  it('renders the save button initially', () => {
    const saveButton = element.shadowRoot!.querySelector(
      'sl-icon-button[name="floppy"]',
    );
    expect(saveButton).to.exist;
  });

  it('does not render active search controls when no savedSearch is provided', () => {
    const shareButton = element.shadowRoot!.querySelector('sl-copy-button');
    const bookmarkButton = element.shadowRoot!.querySelector(
      'sl-icon-button[name^="star"]',
    );
    const editButton = element.shadowRoot!.querySelector(
      'sl-icon-button[name="pencil"]',
    );
    const deleteButton = element.shadowRoot!.querySelector(
      'sl-icon-button[name="trash"]',
    );

    expect(shareButton).to.not.exist;
    expect(bookmarkButton).to.not.exist;
    expect(editButton).to.not.exist;
    expect(deleteButton).to.not.exist;
  });

  it('calls openSavedSearchDialog when save button is clicked', async () => {
    const saveButton = element.shadowRoot!.querySelector<SlIconButton>(
      'sl-icon-button[name="floppy"]',
    )!;
    saveButton.click();
    await element.updateComplete;

    expect(openSavedSearch).to.have.been.calledOnceWith(
      'save',
      undefined,
      typeaheadQuery,
    );
  });

  describe('with active saved search (Viewer, Not Bookmarked)', () => {
    beforeEach(async () => {
      element.savedSearch = {...mockSavedSearchViewerNotBookmarked};
      await element.updateComplete;
    });

    it('renders save, share, and bookmark (empty star) buttons', () => {
      const saveButton = element.shadowRoot!.querySelector(
        'sl-icon-button[name="floppy"]',
      );
      const shareButton = element.shadowRoot!.querySelector('sl-copy-button');
      const bookmarkButton = element.shadowRoot!.querySelector(
        'sl-icon-button[name="star"]',
      );
      const bookmarkFillButton = element.shadowRoot!.querySelector(
        'sl-icon-button[name="star-fill"]',
      );
      const editButton = element.shadowRoot!.querySelector(
        'sl-icon-button[name="pencil"]',
      );
      const deleteButton = element.shadowRoot!.querySelector(
        'sl-icon-button[name="trash"]',
      );

      expect(saveButton).to.exist;
      expect(shareButton).to.exist;
      expect(bookmarkButton).to.exist;
      expect(bookmarkButton).to.not.have.attribute('disabled');
      expect(bookmarkFillButton).to.not.exist;
      expect(editButton).to.not.exist;
      expect(deleteButton).to.not.exist;
    });

    it('configures share button correctly', () => {
      const copyButton = element.shadowRoot!.querySelector('sl-copy-button');
      const expectedUrl = `http://localhost:8080/features?q=feature%3Acss&search_id=${mockSavedSearchViewerNotBookmarked.id}`;
      expect(copyButton).to.have.attribute('value', expectedUrl);
      expect(formatOverviewPageUrlStub).to.have.been.calledWith(mockLocation, {
        search_id: mockSavedSearchViewerNotBookmarked.id,
      });
    });

    it('calls handleBookmarkSavedSearch to bookmark when bookmark button is clicked', async () => {
      apiClientMock.putUserSavedSearchBookmark.resolves();
      const bookmarkButton = element.shadowRoot!.querySelector<SlIconButton>(
        'sl-icon-button[name="star"]',
      )!;
      const eventPromise = oneEvent(element, 'saved-search-bookmarked');

      bookmarkButton.click();

      // Wait for the task to complete
      await waitUntil(
        () => element['_bookmarkTask']?.status === TaskStatus.COMPLETE,
      );
      await element.updateComplete; // Ensure UI updates

      const event = await eventPromise;

      expect((userMock.user.getIdToken as sinon.SinonStub).calledOnce).to.be
        .true;
      expect(
        apiClientMock.putUserSavedSearchBookmark,
      ).to.have.been.calledOnceWith(
        mockSavedSearchViewerNotBookmarked.id,
        'mock-token',
      );
      expect(apiClientMock.removeUserSavedSearchBookmark).to.not.have.been
        .called;

      // Check event detail
      expect(event.detail).to.deep.equal({
        ...mockSavedSearchViewerNotBookmarked,
        bookmark_status: {status: BookmarkStatusActive},
      });

      // Check UI update
      const bookmarkFillButton = element.shadowRoot!.querySelector(
        'sl-icon-button[name="star-fill"]',
      );
      expect(bookmarkFillButton).to.exist;
      expect(element.shadowRoot!.querySelector('sl-icon-button[name="star"]'))
        .to.not.exist;
    });

    it('shows spinner during bookmark operation', async () => {
      apiClientMock.putUserSavedSearchBookmark.resolves(); // Will resolve eventually
      const bookmarkButton = element.shadowRoot!.querySelector<SlIconButton>(
        'sl-icon-button[name="star"]',
      )!;

      bookmarkButton.click();
      await element.updateComplete; // Let the task start

      // Check immediately after click (task is PENDING)
      expect(element['_bookmarkTask']?.status).to.equal(TaskStatus.PENDING);
      let spinner = element.shadowRoot!.querySelector('#bookmark-task-spinner');
      expect(spinner).to.exist;
      expect(bookmarkButton).to.have.attribute('disabled');

      // Wait for completion
      await waitUntil(
        () => element['_bookmarkTask']?.status === TaskStatus.COMPLETE,
      );
      await element.updateComplete; // Let UI update after completion

      spinner = element.shadowRoot!.querySelector('#bookmark-task-spinner');
      expect(spinner).to.not.exist;
      const bookmarkFillButton = element.shadowRoot!.querySelector(
        'sl-icon-button[name="star-fill"]',
      );
      expect(bookmarkFillButton).to.exist;
      expect(bookmarkFillButton).to.not.have.attribute('disabled');
    });

    it('handles API error during bookmarking', async () => {
      const error = new ApiError('Bookmark failed', 500);
      apiClientMock.putUserSavedSearchBookmark.rejects(error);
      const bookmarkButton = element.shadowRoot!.querySelector<SlIconButton>(
        'sl-icon-button[name="star"]',
      )!;

      bookmarkButton.click();

      await waitUntil(
        () => element['_bookmarkTask']?.status === TaskStatus.ERROR,
      );
      await element.updateComplete;

      expect(apiClientMock.putUserSavedSearchBookmark).to.have.been.calledOnce;
      expect(toastStub).to.have.been.calledOnceWith(
        error.message,
        'danger',
        'exclamation-triangle',
      );

      // UI should revert/remain unchanged
      const bookmarkEmptyButton = element.shadowRoot!.querySelector(
        'sl-icon-button[name="star"]',
      );
      expect(bookmarkEmptyButton).to.exist;
      expect(bookmarkEmptyButton).to.not.have.attribute('disabled');
      expect(
        element.shadowRoot!.querySelector('sl-icon-button[name="star-fill"]'),
      ).to.not.exist;
      expect(element.shadowRoot!.querySelector('#bookmark-task-spinner')).to.not
        .exist;
    });
  });

  describe('with active saved search (Viewer, Bookmarked)', () => {
    beforeEach(async () => {
      element.savedSearch = {...mockSavedSearchViewerBookmarked};
      await element.updateComplete;
    });

    it('renders save, share, and bookmark (filled star) buttons', () => {
      const saveButton = element.shadowRoot!.querySelector(
        'sl-icon-button[name="floppy"]',
      );
      const shareButton = element.shadowRoot!.querySelector('sl-copy-button');
      const bookmarkButton = element.shadowRoot!.querySelector(
        'sl-icon-button[name="star"]',
      );
      const bookmarkFillButton = element.shadowRoot!.querySelector(
        'sl-icon-button[name="star-fill"]',
      );
      const editButton = element.shadowRoot!.querySelector(
        'sl-icon-button[name="pencil"]',
      );
      const deleteButton = element.shadowRoot!.querySelector(
        'sl-icon-button[name="trash"]',
      );

      expect(saveButton).to.exist;
      expect(shareButton).to.exist;
      expect(bookmarkButton).to.not.exist;
      expect(bookmarkFillButton).to.exist;
      expect(bookmarkFillButton).to.not.have.attribute('disabled');
      expect(editButton).to.not.exist;
      expect(deleteButton).to.not.exist;
    });

    it('calls handleBookmarkSavedSearch to unbookmark when bookmark button is clicked', async () => {
      apiClientMock.removeUserSavedSearchBookmark.resolves();
      const bookmarkButton = element.shadowRoot!.querySelector<SlIconButton>(
        'sl-icon-button[name="star-fill"]',
      )!;
      const eventPromise = oneEvent(element, 'saved-search-unbookmarked');

      bookmarkButton.click();

      await waitUntil(
        () => element['_bookmarkTask']?.status === TaskStatus.COMPLETE,
      );
      await element.updateComplete;

      const event = await eventPromise;

      expect((userMock.user.getIdToken as sinon.SinonStub).calledOnce).to.be
        .true;
      expect(
        apiClientMock.removeUserSavedSearchBookmark,
      ).to.have.been.calledOnceWith(
        mockSavedSearchViewerBookmarked.id,
        'mock-token',
      );
      expect(apiClientMock.putUserSavedSearchBookmark).to.not.have.been.called;

      // Check event detail
      expect(event.detail).to.deep.equal(mockSavedSearchViewerBookmarked.id);

      // Assume the parent updates the saved search object from the bookmark info store
      element.savedSearch = {
        ...mockSavedSearchViewerBookmarked,
        bookmark_status: undefined,
      };
      await element.updateComplete;
      // Check UI update
      const bookmarkEmptyButton =
        element.shadowRoot!.querySelector<SlIconButton>(
          'sl-icon-button[name="star"]',
        );
      expect(bookmarkEmptyButton).to.exist;
      expect(
        element.shadowRoot!.querySelector('sl-icon-button[name="star-fill"]'),
      ).to.not.exist;
    });

    it('handles API error during unbookmarking', async () => {
      const error = new ApiError('Unbookmark failed', 500);
      apiClientMock.removeUserSavedSearchBookmark.rejects(error);
      const bookmarkButton = element.shadowRoot!.querySelector<SlIconButton>(
        'sl-icon-button[name="star-fill"]',
      )!;

      bookmarkButton.click();

      await waitUntil(
        () => element['_bookmarkTask']?.status === TaskStatus.ERROR,
      );
      await element.updateComplete;

      expect(apiClientMock.removeUserSavedSearchBookmark).to.have.been
        .calledOnce;
      expect(toastStub).to.have.been.calledOnceWith(
        error.message,
        'danger',
        'exclamation-triangle',
      );

      // UI should revert/remain unchanged
      const bookmarkFillButton = element.shadowRoot!.querySelector(
        'sl-icon-button[name="star-fill"]',
      );
      expect(bookmarkFillButton).to.exist;
      expect(bookmarkFillButton).to.not.have.attribute('disabled');
      expect(element.shadowRoot!.querySelector('sl-icon-button[name="star"]'))
        .to.not.exist;
      expect(element.shadowRoot!.querySelector('#bookmark-task-spinner')).to.not
        .exist;
    });
  });

  describe('with active saved search (Owner)', () => {
    beforeEach(async () => {
      element.savedSearch = {...mockSavedSearchOwner};
      await element.updateComplete;
    });

    it('renders save, share, bookmark (filled, disabled), edit, and delete buttons', () => {
      const saveButton = element.shadowRoot!.querySelector(
        'sl-icon-button[name="floppy"]',
      );
      const shareButton = element.shadowRoot!.querySelector('sl-copy-button');
      const bookmarkButton = element.shadowRoot!.querySelector(
        'sl-icon-button[name="star"]',
      );
      const bookmarkFillButton = element.shadowRoot!.querySelector(
        'sl-icon-button[name="star-fill"]',
      );
      const editButton = element.shadowRoot!.querySelector(
        'sl-icon-button[name="pencil"]',
      );
      const deleteButton = element.shadowRoot!.querySelector(
        'sl-icon-button[name="trash"]',
      );

      expect(saveButton).to.exist;
      expect(shareButton).to.exist;
      expect(bookmarkButton).to.not.exist;
      expect(bookmarkFillButton).to.exist;
      expect(bookmarkFillButton).to.have.attribute('disabled'); // Key check for owner
      expect(editButton).to.exist;
      expect(deleteButton).to.exist;
    });

    it('configures share button correctly for owner', () => {
      const copyButton = element.shadowRoot!.querySelector('sl-copy-button');
      const expectedUrl = `http://localhost:8080/features?q=feature%3Acss&search_id=${mockSavedSearchOwner.id}`;
      expect(copyButton).to.have.attribute('value', expectedUrl);
      expect(formatOverviewPageUrlStub).to.have.been.calledWith(mockLocation, {
        search_id: mockSavedSearchOwner.id,
      });
    });

    it('calls openSavedSearchDialog when edit button is clicked', async () => {
      const editButton = element.shadowRoot!.querySelector<SlIconButton>(
        'sl-icon-button[name="pencil"]',
      )!;
      editButton.click();
      await element.updateComplete;

      expect(openSavedSearch).to.have.been.calledOnceWith(
        'edit',
        element.savedSearch,
        typeaheadQuery,
      );
    });

    it('calls openSavedSearchDialog when delete button is clicked', async () => {
      const deleteButton = element.shadowRoot!.querySelector<SlIconButton>(
        'sl-icon-button[name="trash"]',
      )!;
      deleteButton.click();
      await element.updateComplete;

      expect(openSavedSearch).to.have.been.calledOnceWith(
        'delete',
        element.savedSearch,
      );
    });

    it('does not call handleBookmarkSavedSearch when disabled bookmark button is clicked', async () => {
      const bookmarkButton = element.shadowRoot!.querySelector<SlIconButton>(
        'sl-icon-button[name="star-fill"]',
      )!;
      const handleBookmarkSpy = sinon.spy(element, 'handleBookmarkSavedSearch');

      bookmarkButton.click(); // Click the disabled button
      await element.updateComplete;

      expect(handleBookmarkSpy).to.not.have.been.called;
      expect(apiClientMock.putUserSavedSearchBookmark).to.not.have.been.called;
      expect(apiClientMock.removeUserSavedSearchBookmark).to.not.have.been
        .called;
    });
  });
});
