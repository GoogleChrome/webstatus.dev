/**
 * Copyright 2024 Google LLC
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

import {WebstatusOverviewContent} from '../webstatus-overview-content.js';
import '../webstatus-overview-content.js';
import {expect, fixture, html} from '@open-wc/testing';

import {
  savedSearchHelpers,
  SavedSearchScope,
} from '../../contexts/app-bookmark-info-context.js';
import sinon from 'sinon';
import {
  BookmarkOwnerRole,
  BookmarkStatusActive,
  UserSavedSearch,
} from '../../utils/constants.js';

import {TaskStatus} from '@lit/task';
import {WebstatusSavedSearchEditor} from '../webstatus-saved-search-editor.js';

describe('WebstatusOverviewContent', () => {
  let element: WebstatusOverviewContent;

  const mockUserSearch: UserSavedSearch = {
    id: 'user-123',
    name: 'My CSS',
    query: 'feature:css',
    description: 'test description',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    permissions: {role: BookmarkOwnerRole},
    bookmark_status: {status: BookmarkStatusActive},
  };

  beforeEach(async () => {
    element = await fixture<WebstatusOverviewContent>(html`
      <webstatus-overview-content></webstatus-overview-content>
    `);
    element._getOrigin = () => 'http://localhost';
    sinon.stub(element, '_getEditSavedSearch').returns(false);
    sinon.stub(element, '_updatePageUrl');
  });

  afterEach(() => {
    sinon.restore();
  });

  describe('Property Synchronization (willUpdate)', () => {
    it('syncs activeQuery and savedSearch from helpers when appBookmarkInfo changes', async () => {
      sinon.stub(savedSearchHelpers, 'getCurrentQuery').returns('test-q');
      sinon.stub(savedSearchHelpers, 'getCurrentSavedSearch').returns({
        scope: SavedSearchScope.UserSavedSearch,
        value: mockUserSearch,
      });

      // Trigger willUpdate
      element.appBookmarkInfo = {currentLocation: {search: ''}} as any;
      await element.updateComplete;

      expect(element.activeQuery).to.equal('test-q');
      expect(element.savedSearch).to.deep.equal(mockUserSearch);
    });

    it('resets savedSearch to undefined if no search is found', async () => {
      element.savedSearch = mockUserSearch;
      sinon
        .stub(savedSearchHelpers, 'getCurrentSavedSearch')
        .returns(undefined);

      element.appBookmarkInfo = {} as any;
      await element.updateComplete;

      expect(element.savedSearch).to.be.undefined;
    });
  });

  describe('Logic: subscribeButtonConfig', () => {
    it('returns config for GlobalSavedSearch scope', () => {
      sinon.stub(savedSearchHelpers, 'getCurrentSavedSearch').returns({
        scope: SavedSearchScope.GlobalSavedSearch,
        value: {id: 'global-456', name: 'Global Search', query: ''},
      });
      expect(element.subscribeButtonConfig?.id).to.equal('global-456');
    });

    it('returns config for UserSavedSearch scope', () => {
      sinon.stub(savedSearchHelpers, 'getCurrentSavedSearch').returns({
        scope: SavedSearchScope.UserSavedSearch,
        value: mockUserSearch,
      });
      expect(element.subscribeButtonConfig?.id).to.equal('user-123');
    });

    it('returns "all" config for the home page (no query, no search)', () => {
      element.activeQuery = '';
      sinon
        .stub(savedSearchHelpers, 'getCurrentSavedSearch')
        .returns(undefined);
      expect(element.subscribeButtonConfig?.id).to.equal('all');
    });

    it('returns null when a query is present but no search is matched', () => {
      element.activeQuery = 'some-unsaved-query';
      sinon
        .stub(savedSearchHelpers, 'getCurrentSavedSearch')
        .returns(undefined);
      expect(element.subscribeButtonConfig).to.be.null;
    });

    it('returns null if GlobalSavedSearch is missing an ID', () => {
      sinon.stub(savedSearchHelpers, 'getCurrentSavedSearch').returns({
        scope: SavedSearchScope.GlobalSavedSearch,
        value: {name: 'Broken Search', id: undefined} as any,
      });
      // Updated to expect null instead of an empty string
      expect(element.subscribeButtonConfig).to.be.null;
    });

    it('returns "all" config for whitespace-only queries', () => {
      element.activeQuery = '   ';
      sinon
        .stub(savedSearchHelpers, 'getCurrentSavedSearch')
        .returns(undefined);
      expect(element.subscribeButtonConfig?.id).to.equal('all');
    });
  });

  describe('Logic: pageDisplayData', () => {
    it('returns search name and description when search is active', () => {
      sinon.stub(savedSearchHelpers, 'getCurrentSavedSearch').returns({
        scope: SavedSearchScope.GlobalSavedSearch,
        value: {name: 'Global Title', description: 'Global Desc', query: ''},
      });
      expect(element.pageDisplayData.title).to.equal('Global Title');
      expect(element.pageDisplayData.description).to.equal('Global Desc');
    });

    it('returns default title when no search is active', () => {
      sinon
        .stub(savedSearchHelpers, 'getCurrentSavedSearch')
        .returns(undefined);
      expect(element.pageDisplayData.title).to.equal('Features overview');
    });
  });

  describe('UI Rendering: Feature Count', () => {
    it('renders error message when task fails', async () => {
      element.taskTracker = {
        status: TaskStatus.ERROR,
        error: {} as any,
        data: undefined,
      };
      element.requestUpdate();
      await element.updateComplete;
      expect(element.shadowRoot?.textContent).to.contain(
        'Failed to load features',
      );
    });
    it('renders 0 features if data is missing', async () => {
      element.taskTracker = {
        status: TaskStatus.COMPLETE,
        error: undefined,
        data: undefined,
      };
      element.requestUpdate();
      await element.updateComplete;
      expect(element.shadowRoot?.textContent).to.contain('0 features');
    });
  });

  describe('UI Rendering: Templates', () => {
    it('renders the subscribe button only when config is non-null', async () => {
      const configStub = sinon.stub(element, 'subscribeButtonConfig');

      // Branch: config exists
      configStub.get(() => ({id: '1', title: 'T'}));
      element.requestUpdate();
      await element.updateComplete;
      expect(element.shadowRoot?.querySelector('webstatus-subscribe-button')).to
        .exist;

      // Branch: config is null
      configStub.get(() => null);
      element.requestUpdate();
      await element.updateComplete;
      expect(element.shadowRoot?.querySelector('webstatus-subscribe-button')).to
        .not.exist;
    });

    it('renders the description block only when present', async () => {
      sinon
        .stub(element, 'pageDisplayData')
        .get(() => ({title: 'T', description: 'Show me'}));
      element.requestUpdate();
      await element.updateComplete;
      expect(element.shadowRoot?.querySelector('#overview-description')).to
        .exist;
    });

    it('renders the correct feature count states', async () => {
      // Pending state
      element.taskTracker = {
        status: TaskStatus.PENDING,
        error: undefined,
        data: undefined,
      };
      element.requestUpdate();
      await element.updateComplete;
      expect(element.shadowRoot?.textContent).to.contain('Loading features...');

      // Complete state
      element.taskTracker = {
        status: TaskStatus.COMPLETE,
        error: undefined,
        data: {metadata: {total: 42}, items: []},
      } as any;
      element.requestUpdate();
      await element.updateComplete;
      expect(
        element.shadowRoot?.querySelector('.stats-summary')?.textContent,
      ).to.contain('42 features');
    });
  });

  describe('Behavior: Lifecycle & URL Params', () => {
    it('automatically opens the editor when edit_saved_search param is true', async () => {
      const editor =
        element.shadowRoot?.querySelector<WebstatusSavedSearchEditor>(
          'webstatus-saved-search-editor',
        );
      if (!editor) throw new Error('Editor not found');
      const openSpy = sinon.spy(editor, 'open');

      (element._getEditSavedSearch as sinon.SinonStub).returns(true);
      element.savedSearch = mockUserSearch;
      sinon.stub(editor, 'isOpen').returns(false);

      element.requestUpdate();
      await element.updateComplete;

      expect(openSpy).to.have.been.calledOnce;
      expect(element._updatePageUrl).to.have.been.calledWith(
        '',
        element.location,
        {edit_saved_search: false},
      );
    });
    it('waits for savedSearch to be available before opening editor from URL', async () => {
      const editor =
        element.shadowRoot?.querySelector<WebstatusSavedSearchEditor>(
          'webstatus-saved-search-editor',
        );
      const openSpy = sinon.spy(editor!, 'open');

      (element._getEditSavedSearch as sinon.SinonStub).returns(true);
      element.savedSearch = undefined; // Data hasn't arrived yet

      element.requestUpdate();
      await element.updateComplete;
      expect(openSpy).to.not.have.been.called;

      // Now data arrives
      element.savedSearch = mockUserSearch;
      element.requestUpdate();
      await element.updateComplete;

      expect(openSpy).to.have.been.calledOnce;
    });
  });

  describe('Events & Interactions', () => {
    it('triggers openSavedSearchDialog when open-saved-search-editor event is received', async () => {
      const dialogSpy = sinon.spy(element, 'openSavedSearchDialog');
      const eventDetail = {
        type: 'edit',
        savedSearch: mockUserSearch,
        overviewPageQueryInput: 'q',
      };

      element.dispatchEvent(
        new CustomEvent('open-saved-search-editor', {
          detail: eventDetail,
          bubbles: true,
          composed: true,
        }),
      );

      expect(dialogSpy).to.have.been.calledWith('edit', mockUserSearch, 'q');
    });
  });
});
