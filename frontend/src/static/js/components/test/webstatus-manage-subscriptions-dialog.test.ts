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

import {fixture, html} from '@open-wc/testing';
import {expect} from '@esm-bundle/chai';
import sinon from 'sinon';
import {APIClient} from '../../api/client.js';
import {User} from '../../contexts/firebase-user-context.js';
import '../webstatus-manage-subscriptions-dialog.js';
import {
  ManageSubscriptionsDialog,
} from '../webstatus-manage-subscriptions-dialog.js';
import {type components} from 'webstatus.dev-backend';

describe('webstatus-manage-subscriptions-dialog', () => {
  let sandbox: sinon.SinonSandbox;
  let apiClient: APIClient;
  let user: User;
  let element: ManageSubscriptionsDialog;

  const mockSavedSearch: components['schemas']['SavedSearchResponse'] = {
    id: 'test-search-id',
    name: 'Test Saved Search',
    query: 'is:test',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    permissions: {role: 'saved_search_owner'},
    bookmark_status: {status: 'bookmark_none'},
  };

  const mockNotificationChannels: components['schemas']['NotificationChannelResponse'][] = [
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

  const mockInitialSubscription:
    components['schemas']['SubscriptionResponse'] = {
    id: 'initial-sub-id',
    saved_search_id: 'test-search-id',
    channel_id: 'initial-channel-id',
    frequency: 'weekly',
    triggers: [{value: 'feature_baseline_to_newly'}],
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  };

  const mockOtherSubscription:
    components['schemas']['SubscriptionResponse'] = {
    id: 'other-sub-id',
    saved_search_id: 'test-search-id',
    channel_id: 'other-channel-id',
    frequency: 'monthly',
    triggers: [{value: 'feature_baseline_to_widely'}],
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  };

  beforeEach(async () => {
    sandbox = sinon.createSandbox();
    apiClient = {
      listNotificationChannels: sandbox.stub().resolves(mockNotificationChannels),
      getSavedSearchByID: sandbox.stub().resolves(mockSavedSearch),
      listSubscriptions: sandbox
        .stub()
        .resolves([mockInitialSubscription, mockOtherSubscription]),
      createSubscription: sandbox.stub().resolves(mockInitialSubscription),
      updateSubscription: sandbox.stub().resolves(mockInitialSubscription),
      deleteSubscription: sandbox.stub().resolves(undefined),
      getSubscription: sandbox.stub().resolves(mockInitialSubscription),
    } as any as APIClient;
    user = {getIdToken: async () => 'test-token'} as User;

    element = await fixture<ManageSubscriptionsDialog>(html`
      <webstatus-manage-subscriptions-dialog
        .apiClient=${apiClient as APIClient}
        .user=${user}
        saved-search-id="test-search-id"
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
  });

  it('shows content when opened', async () => {
    element.open = true;
    await element['_loadingTask'].run();
    await element.updateComplete;
    expect(element.shadowRoot?.querySelector('sl-spinner')).to.be.null;
    expect(element.shadowRoot?.textContent).to.include('Test Saved Search');
    expect(element.shadowRoot?.textContent).to.include('Notification channels');
  });


  it('fetches data when opened for a saved search', async () => {
    element.open = true;
    await element['_loadingTask'].run(); // Explicitly re-run the task

    // The beforeEach already triggers the loading task
    // We just need to assert the calls and state.
    expect(apiClient.listNotificationChannels).to.have.been.calledWith('test-token');
    expect(apiClient.getSavedSearchByID).to.have.been.calledWith(
      'test-search-id',
      'test-token'
    );
    expect(apiClient.listSubscriptions).to.have.been.calledWith('test-token');
    // Also verify that the dialog's internal state is updated
    expect(element['_notificationChannels']).to.deep.equal(mockNotificationChannels);
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

  it('dispatches SubscriptionSaveSuccessEvent on successful create', async () => {
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

    await element['_handleSave']();

    expect(apiClient.createSubscription).to.have.been.calledWith('test-token', {
      saved_search_id: 'new-saved-search-id',
      channel_id: mockNotificationChannels[0].id,
      frequency: 'monthly',
      triggers: ['feature_baseline_to_newly'],
    });
    expect(eventSpy).to.have.been.calledOnce;
  });

  it('dispatches SubscriptionSaveSuccessEvent on successful update', async () => {
    const eventSpy = sandbox.spy();
    element.addEventListener('subscription-save-success', eventSpy);

    // Setup existing subscription
    element['_subscriptionsForSavedSearch'] = [mockInitialSubscription];
    element['_activeChannelId'] = mockInitialSubscription.channel_id;
    element['_selectedFrequency'] = 'immediate'; // Change frequency from 'weekly'
    element['_initialSelectedFrequency'] = mockInitialSubscription.frequency; // Set initial
    element['_initialSelectedTriggers'] = mockInitialSubscription.triggers.map(t => t.value) as components['schemas']['SubscriptionTriggerWritable'][];
    element['_selectedTriggers'] = [...mockInitialSubscription.triggers.map(t => t.value), 'feature_baseline_to_widely'] as components['schemas']['SubscriptionTriggerWritable'][];

    await element.updateComplete;
    expect(element.isDirty).to.be.true;

    await element['_handleSave']();

    expect(apiClient.updateSubscription).to.have.been.calledWith(
      mockInitialSubscription.id,
      'test-token',
      {
        frequency: 'immediate',
        triggers: [...mockInitialSubscription.triggers.map(t => t.value), 'feature_baseline_to_widely'],
      }
    );
    expect(eventSpy).to.have.been.calledOnce;
  });

  it('dispatches SubscriptionSaveErrorEvent on save failure', async () => {
    (apiClient.createSubscription as sinon.SinonStub)
      .returns(Promise.reject(new Error('Save failed')));

    const eventSpy = sandbox.spy();
    element.addEventListener('subscription-save-error', eventSpy);

    element.savedSearchId = 'test-search-id';
    element['_activeChannelId'] = mockNotificationChannels[0].id;
    element['_selectedFrequency'] = 'monthly'; // Make it dirty
    element['_initialSelectedFrequency'] = 'immediate';

    await element['_handleSave']();

    expect(eventSpy).to.have.been.calledOnce;
    expect(eventSpy.args[0][0].detail.message).to.equal('Save failed');
  });

  it('dispatches SubscriptionDeleteSuccessEvent on successful delete', async () => {
    const eventSpy = sandbox.spy();
    element.addEventListener('subscription-delete-success', eventSpy);

    element.subscriptionId = 'test-sub-id';

    await element['_handleDelete']();

    expect(apiClient.deleteSubscription).to.have.been.calledWith('test-sub-id', 'test-token');
    expect(eventSpy).to.have.been.calledOnce;
  });

  it('dispatches SubscriptionDeleteErrorEvent on delete failure', async () => {
    (apiClient.deleteSubscription as sinon.SinonStub)
      .returns(Promise.reject(new Error('Delete failed')));

    const eventSpy = sandbox.spy();
    element.addEventListener('subscription-delete-error', eventSpy);

    element.subscriptionId = 'test-sub-id';

    await element['_handleDelete']();

    expect(eventSpy).to.have.been.calledOnce;
    expect(eventSpy.args[0][0].detail.message).to.equal('Delete failed');
  });

  describe('_handleChannelChange', () => {
    let confirmStub: sinon.SinonStub;

    beforeEach(async () => {
      confirmStub = sandbox.stub(window, 'confirm');
      // listSubscriptions is already stubbed in the top-level beforeEach

      element.savedSearchId = 'test-search-id';
      element.open = true;
      await element['_loadingTask'].taskComplete;

      // Manually set initial state for this test suite
      element['_activeChannelId'] = mockInitialSubscription.channel_id;
      element['_subscription'] = mockInitialSubscription;
      element['_selectedTriggers'] = mockInitialSubscription.triggers.map(t => t.value) as components['schemas']['SubscriptionTriggerWritable'][];
      element['_selectedFrequency'] = mockInitialSubscription.frequency;
      element['_initialSelectedTriggers'] = mockInitialSubscription.triggers.map(t => t.value) as components['schemas']['SubscriptionTriggerWritable'][];
      element['_initialSelectedFrequency'] = mockInitialSubscription.frequency;
      await element.updateComplete;

      // Make it dirty by changing something on the initial channel
      element['_selectedFrequency'] = 'immediate';
      await element.updateComplete;
      expect(element.isDirty).to.be.true;
    });

    it('prompts user to discard changes when switching channels while dirty (cancel)', async () => {
      confirmStub.returns(false); // User clicks cancel

      const originalActiveChannelId = element['_activeChannelId'];
      const originalSelectedFrequency = element['_selectedFrequency'];

      element['_handleChannelChange'](mockOtherSubscription.channel_id);
      await element.updateComplete;

      expect(confirmStub).to.have.been.calledOnce;
      // Should revert to original channel
      expect(element['_activeChannelId']).to.equal(originalActiveChannelId);
      // Should keep original dirty changes
      expect(element['_selectedFrequency']).to.equal(originalSelectedFrequency);
      expect(element.isDirty).to.be.true;
    });

    it('discards changes and switches channels when switching channels while dirty (ok)', async () => {
      confirmStub.returns(true); // User clicks OK

      element['_handleChannelChange'](mockOtherSubscription.channel_id);
      await element.updateComplete;

      expect(confirmStub).to.have.been.calledOnce;
      // Should switch to the new channel
      expect(element['_activeChannelId']).to.equal(mockOtherSubscription.channel_id);
      // Should have new settings from otherSubscription, thus no longer dirty
      expect(element['_selectedFrequency']).to.equal(mockOtherSubscription.frequency);
      expect(element.isDirty).to.be.false;
    });

    it('does not prompt user when switching channels while not dirty', async () => {
      element['_selectedFrequency'] = mockInitialSubscription.frequency; // Make it not dirty
      await element.updateComplete;
      expect(element.isDirty).to.be.false;

      element['_handleChannelChange'](mockOtherSubscription.channel_id);
      await element.updateComplete;

      expect(confirmStub).to.not.have.been.called;
      expect(element['_activeChannelId']).to.equal(mockOtherSubscription.channel_id);
      expect(element.isDirty).to.be.false;
    });
  });
});
