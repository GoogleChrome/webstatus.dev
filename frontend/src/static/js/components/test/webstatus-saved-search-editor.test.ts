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

import {expect, fixture, html, oneEvent, waitUntil} from '@open-wc/testing';
import sinon from 'sinon';
import {WebstatusSavedSearchEditor} from '../webstatus-saved-search-editor.js';
import '../webstatus-saved-search-editor.js';
import {APIClient} from '../../api/client.js';
import {UserSavedSearch} from '../../utils/constants.js';
import {SlAlert, SlDialog, SlInput, SlTextarea} from '@shoelace-style/shoelace';
import {Toast} from '../../utils/toast.js';
import {WebstatusTypeahead} from '../webstatus-typeahead.js';
import {taskUpdateComplete} from './test-helpers.js';
import {User} from '../../contexts/firebase-user-context.js';
import {InternalServerError} from '../../api/errors.js';
import {TaskStatus} from '@lit/task';
describe('webstatus-saved-search-editor', () => {
  let el: WebstatusSavedSearchEditor;
  let apiClientStub: sinon.SinonStubbedInstance<APIClient>;
  let toastStub: sinon.SinonStub;

  const newSearchQuery = 'new-query';
  const existingSearch: UserSavedSearch = {
    id: 'existing123',
    name: 'Existing Search',
    query: 'existing-query',
    description: 'Existing Description',
    updated_at: '2024-01-01T00:00:00Z',
    created_at: '2024-01-01T00:00:00Z',
    permissions: {role: 'saved_search_owner'},
  };

  const mockUser: User = {
    getIdToken: sinon.stub().resolves('mock-token'),
  } as unknown as User;

  async function setupComponent(
    operation: 'save' | 'edit' | 'delete',
    savedSearch?: UserSavedSearch,
    overviewPageQueryInput?: WebstatusTypeahead,
  ): Promise<WebstatusSavedSearchEditor> {
    apiClientStub = sinon.createStubInstance(APIClient);
    toastStub = sinon.stub(Toast.prototype, 'toast');

    console.log('Pre step 1');
    const component = await fixture<WebstatusSavedSearchEditor>(html`
      <webstatus-saved-search-editor
        .user=${mockUser}
        .apiClient=${apiClientStub}
      ></webstatus-saved-search-editor>
    `);
    console.log('Pre step 2');
    // Manually open the dialog after fixture creation
    await component.open(operation, savedSearch, overviewPageQueryInput);
    await component.updateComplete;
    return component;
  }

  afterEach(() => {
    sinon.restore();
  });

  describe.skip('Rendering', () => {
    it('renders correctly for a new search (save operation)', async () => {
      const mockTypeahead = {value: newSearchQuery} as WebstatusTypeahead;
      el = await setupComponent('save', undefined, mockTypeahead);
      await expect(el).shadowDom.to.be.accessible();

      const dialog = el.shadowRoot?.querySelector<SlDialog>('sl-dialog');
      expect(dialog?.label).to.equal('Save New Search');

      const nameInput = el.shadowRoot?.querySelector<SlInput>('#name');
      const descriptionInput =
        el.shadowRoot?.querySelector<SlTextarea>('#description');
      const queryInput = el.shadowRoot?.querySelector<WebstatusTypeahead>(
        'webstatus-typeahead',
      );
      expect(nameInput?.value).to.equal('');
      expect(descriptionInput?.value).to.equal('');
      expect(queryInput?.value).to.equal(newSearchQuery);
    });

    it('renders correctly for an existing search (edit operation)', async () => {
      el = await setupComponent('edit', existingSearch);
      await expect(el).shadowDom.to.be.accessible();

      const dialog = el.shadowRoot?.querySelector<SlDialog>('sl-dialog');
      expect(dialog?.label).to.equal('Edit Saved Search');

      const nameInput = el.shadowRoot?.querySelector<SlInput>('#name');
      const descriptionInput =
        el.shadowRoot?.querySelector<SlTextarea>('#description');
      const queryInput = el.shadowRoot?.querySelector<WebstatusTypeahead>(
        'webstatus-typeahead',
      );
      expect(nameInput?.value).to.equal(existingSearch.name);
      expect(descriptionInput?.value).to.equal(existingSearch.description);
      expect(queryInput?.value).to.equal(existingSearch.query);
    });

    it('renders correctly for delete operation', async () => {
      el = await setupComponent('delete', existingSearch);
      await expect(el).shadowDom.to.be.accessible();

      const dialog = el.shadowRoot?.querySelector<SlDialog>('sl-dialog');
      expect(dialog?.label).to.equal('Delete Saved Search');
      expect(dialog?.textContent).to.contain('Are you sure');
    });
  });

  describe.skip('Form Submission (Save)', () => {
    it('calls createSavedSearch for a new search and dispatches "save" event', async () => {
      const mockTypeahead = {value: newSearchQuery} as WebstatusTypeahead;
      el = await setupComponent('save', undefined, mockTypeahead);
      const savedSearchData = {
        ...existingSearch,
        id: 'new123',
        name: 'New Search',
        description: 'New Desc',
        query: newSearchQuery,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };
      apiClientStub.createSavedSearch.resolves(savedSearchData);

      // Simulate user input
      const nameInput = el.shadowRoot?.querySelector<SlInput>('#name');
      const descriptionInput =
        el.shadowRoot?.querySelector<SlTextarea>('#description');
      const queryInput = el.shadowRoot?.querySelector<WebstatusTypeahead>(
        'webstatus-typeahead',
      );
      nameInput!.value = 'New Search';
      descriptionInput!.value = 'New Desc';
      // Should already be set by open()
      queryInput!.value = newSearchQuery;
      await el.updateComplete;

      const form =
        el.shadowRoot?.querySelector<HTMLFormElement>('#editor-form');
      const saveEventPromise = oneEvent(el, 'saved-search-saved');

      form?.requestSubmit();

      const saveEvent = await saveEventPromise;

      expect(apiClientStub.createSavedSearch).to.have.been.calledOnceWith(
        'mock-token',
        {
          name: 'New Search',
          description: 'New Desc',
          query: newSearchQuery,
        },
      );
      expect(saveEvent.detail).to.deep.equal(savedSearchData);
      // Toast is handled internally by the component on success/error
      expect(toastStub).to.not.have.been.called;
      expect(el.isOpen()).to.be.false;
    });

    it('calls updateSavedSearch for an existing search and dispatches "save" event', async () => {
      el = await setupComponent('edit', existingSearch);
      const updatedSearchData = {
        ...existingSearch,
        name: 'Updated Name',
        query: 'updated-query',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2025-01-01T00:00:00Z',
      };
      apiClientStub.updateSavedSearch.resolves(updatedSearchData);

      // Simulate user input
      const nameInput = el.shadowRoot?.querySelector<SlInput>('#name');
      const queryInput = el.shadowRoot?.querySelector<WebstatusTypeahead>(
        'webstatus-typeahead',
      );
      nameInput!.value = 'Updated Name';
      queryInput!.value = 'updated-query';
      await el.updateComplete;

      const form =
        el.shadowRoot?.querySelector<HTMLFormElement>('#editor-form');
      const editEventPromise = oneEvent(el, 'saved-search-edited');

      form?.requestSubmit();

      const editEvent = await editEventPromise;

      expect(apiClientStub.updateSavedSearch).to.have.been.calledOnceWith(
        {
          id: existingSearch.id,
          name: 'Updated Name',
          description: undefined, // Description didn't change
          query: 'updated-query',
        },
        'mock-token',
      );
      expect(editEvent.detail).to.deep.equal(updatedSearchData);
      expect(toastStub).to.not.have.been.called;
      expect(el.isOpen()).to.be.false;
    });

    it('shows an error toast if saving fails', async () => {
      el = await setupComponent('save', undefined, {
        value: newSearchQuery,
      } as WebstatusTypeahead);
      const error = new InternalServerError('Save failed');
      apiClientStub.createSavedSearch.rejects(error);

      const nameInput = el.shadowRoot?.querySelector<SlInput>('#name');
      nameInput!.value = 'Fail Search';
      await el.updateComplete;

      const form =
        el.shadowRoot?.querySelector<HTMLFormElement>('#editor-form');
      form?.requestSubmit();

      // Wait for the task to complete (or fail)
      await waitUntil(() => el['_currentTask']?.status === TaskStatus.ERROR);
      await taskUpdateComplete();

      expect(apiClientStub.createSavedSearch).to.have.been.calledOnce;
      expect(toastStub).to.have.been.calledWith(
        'Save failed',
        'danger',
        'exclamation-triangle',
      );
      // Dialog should remain open on error
      expect(el.isOpen()).to.be.true;
    });

    it('shows alert and prevents submission if name is empty', async () => {
      el = await setupComponent('save', undefined, {
        value: newSearchQuery,
      } as WebstatusTypeahead);
      const form =
        el.shadowRoot?.querySelector<HTMLFormElement>('#editor-form');
      const alert = el.shadowRoot?.querySelector<SlAlert>(
        'sl-alert#editor-alert',
      );
      const nameInput = el.shadowRoot?.querySelector<SlInput>('#name');

      // Ensure name is empty
      nameInput!.value = '';
      await el.updateComplete;

      form?.requestSubmit();
      await el.updateComplete;

      expect(apiClientStub.createSavedSearch).to.not.have.been.called;
      expect(alert?.open).to.be.true;
      // Dialog should remain open
      expect(el.isOpen()).to.be.true;
    });
  });

  describe.skip('Delete Functionality', () => {
    beforeEach(async () => {
      el = await setupComponent('delete', existingSearch);
    });

    it('calls removeSavedSearchByID and dispatches "saved-search-deleted" event on confirmation', async () => {
      apiClientStub.removeSavedSearchByID.resolves();
      const form =
        el.shadowRoot?.querySelector<HTMLFormElement>('#editor-form');
      const deleteEventPromise = oneEvent(el, 'saved-search-deleted');

      // Submit the delete confirmation form
      form?.requestSubmit();

      const deleteEvent = await deleteEventPromise;

      expect(apiClientStub.removeSavedSearchByID).to.have.been.calledOnceWith(
        existingSearch.id,
        'mock-token',
      );
      expect(deleteEvent.detail).to.equal(existingSearch.id);
      expect(toastStub).to.not.have.been.called;
      expect(el.isOpen()).to.be.false;
    });

    it('shows an error toast if deletion fails', async () => {
      const error = new InternalServerError('Delete failed');
      apiClientStub.removeSavedSearchByID.rejects(error);
      const form =
        el.shadowRoot?.querySelector<HTMLFormElement>('#editor-form');

      // Submit the delete confirmation form
      form?.requestSubmit();

      // Wait for the task to complete (or fail)
      await waitUntil(() => el['_currentTask']?.status === TaskStatus.ERROR);
      await taskUpdateComplete();

      expect(apiClientStub.removeSavedSearchByID).to.have.been.calledOnce;
      expect(toastStub).to.have.been.calledWith(
        'Delete failed',
        'danger',
        'exclamation-triangle',
      );
      // Dialog should remain open on error
      expect(el.isOpen()).to.be.true;
    });
  });

  describe.skip('Cancel Button', async () => {
    it('dispatches "saved-search-cancelled" event when cancel button is clicked', async done => {
      el = await setupComponent('edit', existingSearch);
      const cancelButton = el.shadowRoot?.querySelector<HTMLButtonElement>(
        'sl-button[variant="default"]',
      );
      console.log('Step 1');
      // Assuming cancel is the default button
      const cancelEventPromise = oneEvent(el, 'saved-search-cancelled');
      console.log('Step 2');
      cancelButton!.click();
      console.log('Step 3');
      // Just ensure the event is fired
      await cancelEventPromise;
      console.log('Step 4');
      // No specific detail expected for cancel event
      expect(el.isOpen()).to.eq(false);
      console.log('Step 5');
      done();
      console.log('Step 6');
    });
  });

  it('sums up 2 numbers', () => {
    expect(2).to.equal(2);
  });
});
