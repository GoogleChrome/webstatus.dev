/**
 * Copyright 2026 Google LLC
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

import {expect, fixture, html, waitUntil} from '@open-wc/testing';
import sinon from 'sinon';
import {APIClient} from '../../api/client.js';
import {ForbiddenError} from '../../api/errors.js';
import {UserContext} from '../../contexts/firebase-user-context.js';
import '../webstatus-manage-subscriptions-dialog.js';
import {ManageSubscriptionsDialog} from '../webstatus-manage-subscriptions-dialog.js';
import type {components as backend} from 'webstatus.dev-backend';
import {
  SlAlert,
  SlButton,
  SlDialog,
  SlCheckbox,
} from '@shoelace-style/shoelace';

describe('webstatus-manage-subscriptions-dialog', () => {
  let sandbox: sinon.SinonSandbox;
  let apiClient: APIClient;
  let userContext: UserContext;
  let element: ManageSubscriptionsDialog;

  const mockSavedSearch: backend['schemas']['SavedSearchResponse'] = {
    id: 'test-search-id',
    name: 'Test Saved Search',
    query: 'is:test',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    permissions: {role: 'saved_search_owner'},
    bookmark_status: {status: 'bookmark_none'},
  };

  const mockNotificationChannels: backend['schemas']['NotificationChannelResponse'][] =
    [
      {
        id: 'test-channel-id',
        type: 'email',
        name: 'test@example.com',
        value: 'test@example.com',
        status: 'enabled',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
      {
        id: 'other-channel-id',
        type: 'email',
        name: 'other@example.com',
        value: 'other@example.com',
        status: 'enabled',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
    ];

  const mockInitialSubscription: backend['schemas']['SubscriptionResponse'] = {
    id: 'initial-sub-id',
    subscribable: {id: 'test-search-id', name: 'Test Search'},
    channel_id: 'test-channel-id',
    frequency: 'weekly',
    triggers: [{value: 'feature_baseline_to_newly'}],
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  };

  const mockOtherSubscription: backend['schemas']['SubscriptionResponse'] = {
    id: 'other-sub-id',
    subscribable: {id: 'test-search-id', name: 'Test Search'},
    channel_id: 'other-channel-id',
    frequency: 'monthly',
    triggers: [{value: 'feature_baseline_to_widely'}],
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  };

  beforeEach(async () => {
    sandbox = sinon.createSandbox();
    apiClient = {
      listNotificationChannels: sandbox
        .stub()
        .resolves(mockNotificationChannels),
      getSavedSearchByID: sandbox.stub().resolves(mockSavedSearch),
      listSubscriptions: sandbox
        .stub()
        .resolves([mockInitialSubscription, mockOtherSubscription]),
      createSubscription: sandbox.stub().resolves(mockInitialSubscription),
      updateSubscription: sandbox.stub().resolves(mockInitialSubscription),
      deleteSubscription: sandbox.stub().resolves(undefined),
      getSubscription: sandbox.stub().resolves(mockInitialSubscription),
    } as unknown as APIClient;
    userContext = {user: {getIdToken: async () => 'test-token'}} as UserContext;

    element = await fixture<ManageSubscriptionsDialog>(html`
      <webstatus-manage-subscriptions-dialog
        .apiClient=${apiClient as APIClient}
        .userContext=${userContext}
        .savedSearchId=${'test-search-id'}
      ></webstatus-manage-subscriptions-dialog>
    `);

    // Ensure the loading task completes and the component re-renders
    await element['_loadingTask'].taskComplete;
    await element.updateComplete;
  });

  afterEach(() => {
    sandbox.restore();
  });

  it('renders correctly initially', async () => {
    expect(element).to.be.instanceOf(ManageSubscriptionsDialog);
    // Dialog is initially closed, so its content should not be visible.
    expect(element.open).to.be.false;
    expect(element.shadowRoot?.querySelector('sl-dialog[open]')).to.be.null;
    // We will test content when the dialog is explicitly opened in another test.
    expect(element.isDirty).to.be.false;
    expect(element['_actionState']).to.deep.equal({phase: 'idle'});
  });

  it('shows content when opened', async () => {
    element.open = true;
    await element['_loadingTask'].run();
    await element.updateComplete;
    expect(element.shadowRoot?.querySelector('sl-spinner')).to.be.null;
    expect(element.shadowRoot?.textContent).to.include('Test Saved Search');
    expect(element.shadowRoot?.textContent).to.include('Notification channels');
  });

  it('shows an error if initial data loading fails', async () => {
    const expectedError = new Error('Failed to fetch channels');
    (apiClient.listNotificationChannels as sinon.SinonStub).rejects(
      expectedError,
    );

    // Create a fresh fixture to ensure a clean lifecycle for this test.
    const errorElement = await fixture<ManageSubscriptionsDialog>(html`
      <webstatus-manage-subscriptions-dialog
        .apiClient=${apiClient as APIClient}
        .userContext=${userContext}
        .savedSearchId=${'test-search-id'}
      ></webstatus-manage-subscriptions-dialog>
    `);

    errorElement.open = true;
    await errorElement.updateComplete;

    await waitUntil(() => {
      const text = errorElement.shadowRoot?.textContent;
      return text?.includes('Failed to fetch channels');
    }, 'Error message did not render in time');

    // Final assertion to be certain.
    const errorContent = errorElement.shadowRoot?.textContent;
    expect(errorContent).to.include('Error:');
    expect(errorContent).to.include('Failed to fetch channels');
  });

  it('renders a subscription indicator for subscribed channels', async () => {
    element.open = true;
    await element['_loadingTask'].run();
    await element.updateComplete;

    const channelItems =
      element.shadowRoot?.querySelectorAll('.channel-item') || [];
    const initialSubChannelItem = Array.from(channelItems).find(item =>
      item.textContent?.includes(mockNotificationChannels[0].name),
    );

    const indicator = initialSubChannelItem?.querySelector(
      'sl-icon[name="circle-fill"].subscription-indicator',
    );
    expect(indicator).to.exist;
  });

  it('pre-selects channel when subscriptionId is provided', async () => {
    element.subscriptionId = mockInitialSubscription.id;
    element.open = true;
    await element['_loadingTask'].run();
    await element.updateComplete;

    expect(element['_activeChannelId']).to.equal(
      mockInitialSubscription.channel_id,
    );
  });

  it('fetches data when opened for a saved search', async () => {
    element.open = true;
    await element['_loadingTask'].run(); // Explicitly re-run the task

    // The beforeEach already triggers the loading task
    // We just need to assert the calls and state.
    expect(apiClient.listNotificationChannels).to.have.been.calledWith(
      'test-token',
    );
    expect(apiClient.getSavedSearchByID).to.have.been.calledWith(
      'test-search-id',
      'test-token',
    );
    expect(apiClient.listSubscriptions).to.have.been.calledWith('test-token');
    // Also verify that the dialog's internal state is updated
    expect(element['_notificationChannels']).to.deep.equal(
      mockNotificationChannels,
    );
    expect(element['_savedSearch']).to.deep.equal(mockSavedSearch);
    expect(element['_subscriptionsForSavedSearch']).to.deep.equal([
      mockInitialSubscription,
      mockOtherSubscription,
    ]);
  });

  it('is dirty when frequency changes', async () => {
    element.open = true;
    await element['_loadingTask'].run();
    await element.updateComplete;

    element['_selectedFrequency'] = 'monthly'; // Change from initial 'weekly'
    expect(element.isDirty).to.be.true;
  });

  it('is dirty when triggers change', async () => {
    element.open = true;
    await element['_loadingTask'].run();
    await element.updateComplete;

    element['_selectedTriggers'] = ['feature_baseline_to_widely']; // Change from initial [ { value: 'feature_baseline_to_newly' } ]
    expect(element.isDirty).to.be.true;
  });

  it('is not dirty when changes are reverted', async () => {
    element.open = true;
    await element['_loadingTask'].run();
    await element.updateComplete;

    // Simulate selecting the initial channel to set the baseline state correctly.
    element['_handleChannelChange'](mockInitialSubscription.channel_id);
    await element.updateComplete;
    expect(element.isDirty, 'Should not be dirty after initialization').to.be
      .false;

    // Make a change
    element['_selectedFrequency'] = 'monthly';
    await element.updateComplete;
    expect(element.isDirty, 'Should be dirty after change').to.be.true;

    // Revert the change
    element['_selectedFrequency'] = mockInitialSubscription.frequency;
    await element.updateComplete;
    expect(element.isDirty, 'Should not be dirty after reverting change').to.be
      .false;
  });

  it('sets actionState correctly on successful create', async () => {
    const eventSpy = sandbox.spy();
    element.addEventListener('subscription-save-success', eventSpy);

    // Make it dirty for creation.
    element.savedSearchId = 'new-saved-search-id';
    element['_activeChannelId'] = mockNotificationChannels[0].id;
    element['_selectedFrequency'] = 'monthly';
    element['_selectedTriggers'] = ['feature_baseline_to_newly'];
    element['_initialSelectedFrequency'] = 'immediate';
    element['_initialSelectedTriggers'] = [];
    await element.updateComplete;
    expect(element.isDirty).to.be.true;

    // Use a promise that we can resolve manually to check intermediate state.
    const savePromise = new Promise(resolve => {
      (apiClient.createSubscription as sinon.SinonStub).callsFake(() => {
        resolve(mockInitialSubscription);
        return Promise.resolve(mockInitialSubscription);
      });
    });

    const saveOperation = element['_handleSave']();
    await element.updateComplete;

    // Check intermediate 'saving' state and loading property on button.
    expect(element['_actionState'].phase).to.equal('saving');
    const saveButton = element.shadowRoot?.querySelector<SlButton>(
      'sl-button[variant="primary"]',
    );
    expect(saveButton!.loading).to.be.true;

    await savePromise;
    await saveOperation;
    await element.updateComplete;

    expect(apiClient.createSubscription).to.have.been.calledWith('test-token', {
      saved_search_id: 'new-saved-search-id',
      channel_id: mockNotificationChannels[0].id,
      frequency: 'monthly',
      triggers: ['feature_baseline_to_newly'],
    });
    expect(eventSpy).to.have.been.calledOnce;

    // Check final 'success' state.
    expect(element['_actionState'].phase).to.equal('success');
    if (element['_actionState'].phase === 'success') {
      expect(element['_actionState'].message).to.equal('Subscription saved.');
    }
  });

  it('sets actionState correctly on successful update', async () => {
    const eventSpy = sandbox.spy();
    element.addEventListener('subscription-save-success', eventSpy);

    // Setup existing subscription
    element['_subscriptionsForSavedSearch'] = [mockInitialSubscription];
    element['_activeChannelId'] = mockInitialSubscription.channel_id;
    element['_selectedFrequency'] = 'immediate'; // Change frequency from 'weekly'
    element['_initialSelectedFrequency'] = mockInitialSubscription.frequency; // Set initial
    element['_initialSelectedTriggers'] = mockInitialSubscription.triggers.map(
      t => t.value,
    ) as backend['schemas']['SubscriptionTriggerWritable'][];
    element['_selectedTriggers'] = [
      ...mockInitialSubscription.triggers.map(t => t.value),
      'feature_baseline_to_widely',
    ] as backend['schemas']['SubscriptionTriggerWritable'][];

    await element.updateComplete;
    expect(element.isDirty).to.be.true;

    await element['_handleSave']();
    await element.updateComplete;

    expect(apiClient.updateSubscription).to.have.been.calledWith(
      mockInitialSubscription.id,
      'test-token',
      {
        frequency: 'immediate',
        triggers: [
          ...mockInitialSubscription.triggers.map(t => t.value),
          'feature_baseline_to_widely',
        ],
      },
    );
    expect(eventSpy).to.have.been.calledOnce;
    expect(element['_actionState'].phase).to.equal('success');
  });

  it('sets actionState correctly on save failure', async () => {
    (apiClient.createSubscription as sinon.SinonStub).returns(
      Promise.reject(new Error('Save failed')),
    );

    const eventSpy = sandbox.spy();
    element.addEventListener('subscription-save-error', eventSpy);

    element.savedSearchId = 'test-search-id';
    element['_activeChannelId'] = mockNotificationChannels[0].id;
    element['_selectedFrequency'] = 'monthly'; // Make it dirty
    element['_initialSelectedFrequency'] = 'immediate';

    await element['_handleSave']();
    await element.updateComplete;

    expect(eventSpy).to.have.been.calledOnce;
    expect(eventSpy.args[0][0].detail.message).to.equal('Save failed');
    expect(element['_actionState'].phase).to.equal('error');
    if (element['_actionState'].phase === 'error') {
      expect(element['_actionState'].message).to.contain('Save failed');
    }
  });

  it('sets actionState correctly on subscription limit exceeded error', async () => {
    const limitError = new ForbiddenError(
      'user has reached the maximum number of allowed subscriptions',
    );
    (apiClient.createSubscription as sinon.SinonStub).returns(
      Promise.reject(limitError),
    );

    const eventSpy = sandbox.spy();
    element.addEventListener('subscription-save-error', eventSpy);

    element.savedSearchId = 'test-search-id';
    element['_activeChannelId'] = mockNotificationChannels[0].id;
    element['_selectedFrequency'] = 'monthly'; // Make it dirty
    element['_initialSelectedFrequency'] = 'immediate';

    await element['_handleSave']();
    await element.updateComplete;

    expect(eventSpy).to.have.been.calledOnce;
    expect(element['_actionState'].phase).to.equal('error');
    if (element['_actionState'].phase === 'error') {
      expect(element['_actionState'].message).to.equal(
        'user has reached the maximum number of allowed subscriptions',
      );
    }
  });

  it('sets actionState correctly on successful delete', async () => {
    // Create a new element configured exactly for this test's needs.
    const deleteElement = await fixture<ManageSubscriptionsDialog>(html`
      <webstatus-manage-subscriptions-dialog
        .apiClient=${apiClient as APIClient}
        .userContext=${userContext}
        .subscriptionId=${'test-sub-id'}
      ></webstatus-manage-subscriptions-dialog>
    `);
    // Create a promise that will resolve when the success event is heard
    const eventPromise = new Promise<void>(resolve => {
      deleteElement.addEventListener('subscription-delete-success', () =>
        resolve(),
      );
    });

    deleteElement.open = true;
    await deleteElement.updateComplete;

    // 2. Get the dialog element and create a promise that resolves when it's fully shown
    const mainDialog =
      deleteElement.shadowRoot!.querySelector<SlDialog>('.dialog-main');
    const afterShowPromise = new Promise(resolve => {
      mainDialog!.addEventListener('sl-after-show', resolve, {once: true});
    });

    deleteElement.open = true;
    await afterShowPromise;
    await deleteElement['_loadingTask'].taskComplete;
    await deleteElement.updateComplete;

    const deleteButton = deleteElement.shadowRoot?.querySelector<SlButton>(
      '#confirm-unsubscribe',
    );
    expect(deleteButton).to.exist; // Ensure button exists.

    // Simulate clicking the delete button.
    deleteButton!.click();
    await deleteElement.updateComplete;

    expect(apiClient.deleteSubscription).to.have.been.calledWith(
      'test-sub-id',
      'test-token',
    );
    // Wait for event promise to resolve
    await eventPromise;

    expect(deleteElement['_actionState'].phase).to.equal('success');
    if (deleteElement['_actionState'].phase === 'success') {
      expect(deleteElement['_actionState'].message).to.equal(
        'Successfully unsubscribed.',
      );
    }
  });

  it('sets actionState correctly on delete failure', async () => {
    (apiClient.deleteSubscription as sinon.SinonStub).returns(
      Promise.reject(new Error('Delete failed')),
    );

    const eventSpy = sandbox.spy();
    element.addEventListener('subscription-delete-error', eventSpy);

    element.savedSearchId = '';
    element.subscriptionId = 'test-sub-id';

    await element['_handleDelete']();
    await element.updateComplete;

    expect(eventSpy).to.have.been.calledOnce;
    expect(eventSpy.args[0][0].detail.message).to.equal('Delete failed');
    expect(element['_actionState'].phase).to.equal('error');
    if (element['_actionState'].phase === 'error') {
      expect(element['_actionState'].message).to.contain('Delete failed');
    }
  });

  it('resets actionState to idle when success alert is dismissed', async () => {
    // Open the dialog to ensure content is rendered.
    element.open = true;
    await element.updateComplete;

    // Simulate a successful save to trigger the success alert.
    element['_activeChannelId'] = mockNotificationChannels[0].id;
    element['_selectedFrequency'] = 'monthly';
    element['_initialSelectedFrequency'] = 'immediate';
    await element.updateComplete;

    await element['_handleSave']();
    await element.updateComplete;

    // Expect success alert to be open.
    const successAlert = element.shadowRoot?.querySelector<SlAlert>(
      'sl-alert[variant="success"]',
    );
    expect(successAlert).to.exist;
    expect(successAlert!.open).to.be.true;

    successAlert!.dispatchEvent(new CustomEvent('sl-after-hide'));

    // Await the state update triggered by the event handler.
    await element.updateComplete;

    // Expect actionState to now be idle.
    expect(element['_actionState']).to.deep.equal({phase: 'idle'});
  });

  describe('_handleChannelChange with custom dialog', () => {
    let confirmDialog: SlDialog;

    beforeEach(async () => {
      element.open = true;
      await element.updateComplete;
      await element['_loadingTask'].run(); // Rerun to populate data.
      await element.updateComplete;

      // Manually set initial state for this test suite.
      // Note: `_switchChannel` is the method that sets the initial state.
      element['_switchChannel'](mockInitialSubscription.channel_id);
      await element.updateComplete;

      // Make it dirty by changing something on the initial channel.
      element['_selectedFrequency'] = 'immediate';
      await element.updateComplete;
      expect(element.isDirty).to.be.true;

      confirmDialog =
        element.shadowRoot!.querySelector<SlDialog>('.dialog-confirm')!;
      // Ensure the dialog is actually rendered and available
      expect(confirmDialog).to.exist;
      expect(confirmDialog).to.be.instanceOf(SlDialog);
    });

    it('prompts user via dialog when switching channels while dirty and cancels', async () => {
      const originalActiveChannelId = element['_activeChannelId'];
      const originalSelectedFrequency = element['_selectedFrequency'];

      // Attempt to switch channels.
      element['_handleChannelChange'](mockOtherSubscription.channel_id);
      await element.updateComplete;

      // Check that the confirmation dialog is now open.
      expect(confirmDialog.open).to.be.true;

      // Simulate clicking the "Cancel" button.
      const cancelButton = confirmDialog.querySelector<SlButton>(
        'sl-button:not([variant="primary"])',
      );
      cancelButton!.click();
      await element.updateComplete;

      // Dialog should be closed.
      expect(confirmDialog.open).to.be.false;
      // Should revert to original channel.
      expect(element['_activeChannelId']).to.equal(originalActiveChannelId);
      // Should keep original dirty changes.
      expect(element['_selectedFrequency']).to.equal(originalSelectedFrequency);
      expect(element.isDirty).to.be.true;
    });

    it('discards changes and switches channels when confirmed via dialog', async () => {
      // Attempt to switch channels.
      element['_handleChannelChange'](mockOtherSubscription.channel_id);
      await element.updateComplete;

      // Check that the confirmation dialog is now open.
      expect(confirmDialog.open).to.be.true;

      // Simulate clicking the "Discard" button.
      const discardButton = confirmDialog.querySelector<SlButton>(
        'sl-button[variant="primary"]',
      );
      discardButton!.click();
      await element.updateComplete;

      // Dialog should be closed.
      expect(confirmDialog.open).to.be.false;
      // Should switch to the new channel.
      expect(element['_activeChannelId']).to.equal(
        mockOtherSubscription.channel_id,
      );
      // Should have new settings from otherSubscription, thus no longer dirty.
      expect(element['_selectedFrequency']).to.equal(
        mockOtherSubscription.frequency,
      );
      expect(element.isDirty).to.be.false;
    });

    it('does not prompt user when switching channels while not dirty', async () => {
      // Make it not dirty.
      element['_selectedFrequency'] = mockInitialSubscription.frequency;
      await element.updateComplete;
      expect(element.isDirty).to.be.false;

      element['_handleChannelChange'](mockOtherSubscription.channel_id);
      await element.updateComplete;

      // Dialog should not have opened.
      expect(confirmDialog.open).to.be.false;
      expect(element['_activeChannelId']).to.equal(
        mockOtherSubscription.channel_id,
      );
      expect(element.isDirty).to.be.false;
    });
  });

  it('enables create button only when a trigger is selected', async () => {
    element.open = true;
    await element['_loadingTask'].run();
    await element.updateComplete;

    // Switch to a channel with no existing subscription.
    element['_switchChannel']('other-channel-id');
    await element.updateComplete;

    const createButton = element.shadowRoot?.querySelector<SlButton>(
      'sl-button[variant="primary"]',
    );
    expect(createButton).to.exist;
    expect(createButton!.disabled).to.be.true; // Initially disabled

    // Select a trigger.
    const triggerCheckbox =
      element.shadowRoot?.querySelector<SlCheckbox>('sl-checkbox');
    expect(triggerCheckbox).to.exist;
    triggerCheckbox!.checked = true;
    triggerCheckbox!.dispatchEvent(
      new CustomEvent('sl-change', {bubbles: true, composed: true}),
    );
    await element.updateComplete;

    expect(createButton!.disabled).to.be.false; // Should now be enabled
  });

  it('disables update button when all triggers are deselected on existing subscription', async () => {
    element.open = true;
    await element['_loadingTask'].run();
    await element.updateComplete;

    // 1. Ensure we are on an existing subscription
    element['_switchChannel'](mockInitialSubscription.channel_id);
    await element.updateComplete;

    // 2. Find the checkbox that IS currently checked (from mock data)
    const checkedBox = element.shadowRoot?.querySelector<SlCheckbox>(
      'sl-checkbox[checked]',
    );

    // 3. Uncheck it
    checkedBox?.click();
    await element.updateComplete;

    // 4. Verify selected triggers are empty
    expect(element['_selectedTriggers'].length).to.equal(0);

    // 5. Verify button is disabled
    const saveButton = element.shadowRoot?.querySelector<SlButton>(
      'sl-button[variant="primary"]',
    );
    expect(saveButton!.disabled).to.be.true;
  });
});
