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
import {elementUpdated, expect, fixture, html} from '@open-wc/testing';

import {APIClient} from '../../api/client.js';

import {stub} from 'sinon'; // Make sure you have sinon installed
import {savedSearchHelpers} from '../../contexts/app-bookmark-info-context.js';
import sinon from 'sinon';
import {WebstatusSavedSearchEditor} from '../webstatus-saved-search-editor.js';
import {
  BookmarkOwnerRole,
  BookmarkStatusActive,
  UserSavedSearch,
} from '../../utils/constants.js';
import {UserContext} from '../../contexts/firebase-user-context.js';

describe('webstatus-overview-content', () => {
  let element: WebstatusOverviewContent;
  let apiClientMock: sinon.SinonStubbedInstance<APIClient>;
  let userMock: UserContext;
  let editor: WebstatusSavedSearchEditor;
  let editorIsOpenStub: sinon.SinonStub;
  let editorOpenSpy: sinon.SinonSpy;
  let getEditSavedSearchStub: sinon.SinonStub;
  let updatePageUrlStub: sinon.SinonStub;

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

  const mockLocation = {search: '?q=feature:css'};

  beforeEach(async () => {
    apiClientMock = sinon.createStubInstance(APIClient);
    userMock = {
      user: {
        getIdToken: sinon.stub().resolves('mock-token'),
      },
    } as unknown as UserContext;

    element = await fixture<WebstatusOverviewContent>(html`
      <webstatus-overview-content
        .apiClient=${apiClientMock}
        .userContext=${userMock}
        .location=${mockLocation}
      >
      </webstatus-overview-content>
    `);

    element._getOrigin = () => 'http://localhost:8080';

    getEditSavedSearchStub = sinon
      .stub(element, '_getEditSavedSearch')
      .returns(false);
    updatePageUrlStub = sinon.stub(element, '_updatePageUrl');
    // Get the mocked editor instance after the element is rendered
    editor = element.shadowRoot!.querySelector<WebstatusSavedSearchEditor>(
      'webstatus-saved-search-editor',
    )!;
    editorOpenSpy = sinon.spy(editor, 'open');
    editorIsOpenStub = sinon.stub(editor, 'isOpen');
  });

  afterEach(() => {
    sinon.restore();
  });

  it('should correctly update activeQuery based on getCurrentQuery return value', async () => {
    const apiClient = new APIClient('');
    const location = {search: ''};

    const getCurrentQueryStub = stub(savedSearchHelpers, 'getCurrentQuery');

    // Test case 1: Empty query
    getCurrentQueryStub.returns('');
    let component = await fixture<WebstatusOverviewContent>(
      html`<webstatus-overview-content
        .location=${location}
        .apiClient=${apiClient}
      ></webstatus-overview-content>`,
    );
    await elementUpdated(component);
    expect(component.activeQuery).to.eq('');

    // Test case 2: A specific query
    getCurrentQueryStub.returns('my-test-query');
    component = await fixture<WebstatusOverviewContent>(
      html`<webstatus-overview-content
        .location=${location}
        .apiClient=${apiClient}
      ></webstatus-overview-content>`,
    );
    await elementUpdated(component);
    expect(component.activeQuery).to.eq('my-test-query');

    // Test case 3: Another query
    getCurrentQueryStub.returns('another-test-query');
    component = await fixture<WebstatusOverviewContent>(
      html`<webstatus-overview-content
        .location=${location}
        .apiClient=${apiClient}
      ></webstatus-overview-content>`,
    );
    await elementUpdated(component);
    expect(component.activeQuery).to.eq('another-test-query');

    getCurrentQueryStub.restore();
  });

  describe('RenderBookmarkUI', () => {
    let container: HTMLElement;
    afterEach(() => {
      document.body.removeChild(container);
    });

    it('should display the bookmark title and description when query is matched', async () => {
      container = document.createElement('div');
      container.innerHTML = `
          <webstatus-overview-content>
          </webstatus-overview-content>
      `;
      const element: WebstatusOverviewContent = container.querySelector(
        'webstatus-overview-content',
      ) as WebstatusOverviewContent;
      // Set location to one of the globalSavedSearches.
      element.location = {search: '?q=test_query_1'};
      element.appBookmarkInfo = {
        globalSavedSearches: [
          {
            name: 'Test Bookmark 1',
            query: 'test_query_1',
            description: 'test description1',
          },
          {
            name: 'Test Bookmark 2',
            query: 'test_query_2',
            description: 'test description2',
          },
        ],
        currentGlobalSavedSearch: {
          name: 'Test Bookmark 1',
          query: 'test_query_1',
          description: 'test description1',
        },
      };
      document.body.appendChild(container);
      await element.updateComplete;

      const title = element?.shadowRoot?.querySelector('#overview-title');
      expect(title).to.exist;
      expect(title!.textContent!.trim()).to.equal('Test Bookmark 1');

      const description = element?.shadowRoot?.querySelector(
        '#overview-description',
      );
      expect(description).to.exist;
      expect(description!.textContent).to.contain('test description1');
    });
    it('should not display description UI when it is empty', async () => {
      container = document.createElement('div');
      container.innerHTML = `
          <webstatus-overview-content>
          </webstatus-overview-content>
      `;
      const element: WebstatusOverviewContent = container.querySelector(
        'webstatus-overview-content',
      ) as WebstatusOverviewContent;
      // Set location to one of the globalSavedSearches.
      element.location = {search: '?q=test_query_1'};
      element.appBookmarkInfo = {
        globalSavedSearches: [
          {
            name: 'Test Bookmark 1',
            query: 'test_query_1',
          },
        ],
        currentGlobalSavedSearch: {
          name: 'Test Bookmark 1',
          query: 'test_query_1',
        },
      };
      document.body.appendChild(container);
      await element.updateComplete;

      const title = element?.shadowRoot?.querySelector('#overview-title');
      expect(title).to.exist;
      expect(title!.textContent!.trim()).to.equal('Test Bookmark 1');

      const description = element?.shadowRoot?.querySelector(
        '#overview-description',
      );
      expect(description).to.not.exist;
    });
  });

  describe('updated lifecycle hook', () => {
    it('opens edit dialog and updates URL if edit_saved_search param is present', async () => {
      element.location = {search: 'test'};
      element.appBookmarkInfo = {
        globalSavedSearches: [
          {
            name: 'Test Bookmark 1',
            query: 'test_query_1',
            description: 'test description1',
          },
          {
            name: 'Test Bookmark 2',
            query: 'test_query_2',
            description: 'test description2',
          },
        ],
        currentGlobalSavedSearch: {
          name: 'Test Bookmark 1',
          query: 'test_query_1',
          description: 'test description1',
        },
      };
      element.savedSearch = {...mockSavedSearchOwner};
      getEditSavedSearchStub.returns(true); // Simulate finding the param
      editorIsOpenStub.returns(false); // Simulate editor not already open

      // Trigger the updated lifecycle hook manually for testing
      element.requestUpdate();
      await element.updateComplete;

      // It should call openSavedSearchDialog, which calls editor.open
      expect(editorOpenSpy).to.have.been.calledOnceWith(
        'edit',
        element.savedSearch,
        element.savedSearch.query,
      );
      // It should remove the URL parameter
      expect(updatePageUrlStub).to.have.been.calledOnceWith(
        '',
        element.location,
        {edit_saved_search: undefined},
      );
    });

    it('does not open edit dialog if editor is already open', async () => {
      element.savedSearch = {...mockSavedSearchOwner};
      getEditSavedSearchStub.returns(true);
      editorIsOpenStub.returns(true);

      element.requestUpdate();
      await element.updateComplete;

      expect(editorOpenSpy).to.not.have.been.called;
      // Should not update URL if dialog wasn't opened by this hook
      expect(updatePageUrlStub).to.not.have.been.called;
      expect(getEditSavedSearchStub).to.have.been.called;
    });

    it('does not open edit dialog if edit_saved_search param is not present', async () => {
      element.savedSearch = {...mockSavedSearchOwner};
      getEditSavedSearchStub.returns(false); // Param not present

      editorIsOpenStub.returns(false);

      element.requestUpdate();
      await element.updateComplete;

      expect(editorIsOpenStub).to.not.have.been.called;
      expect(updatePageUrlStub).to.not.have.been.called;
    });

    it('does not open edit dialog if savedSearch is not available', async () => {
      element.savedSearch = undefined; // No saved search loaded yet
      getEditSavedSearchStub.returns(true);
      editorIsOpenStub.returns(false);

      element.requestUpdate();
      await element.updateComplete;

      expect(editorOpenSpy).to.not.have.been.called;
      // updatePageUrl might still be called depending on exact logic,
      // but the primary action (opening dialog) shouldn't happen.
      // Let's assert it's not called for clarity, though the original code
      // might call it regardless. The important part is the dialog doesn't open.
      expect(updatePageUrlStub).to.not.have.been.called;
    });
  });
});
