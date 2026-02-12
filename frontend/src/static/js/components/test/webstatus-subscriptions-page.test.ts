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

import {expect, fixture, html} from '@open-wc/testing';
import sinon from 'sinon';
import {APIClient} from '../../api/client.js';
import {UserContext} from '../../contexts/firebase-user-context.js';
import '../webstatus-subscriptions-page.js';
import {SubscriptionsPage} from '../webstatus-subscriptions-page.js';
import {type components} from 'webstatus.dev-backend';

function mockLocation() {
  let search = '';
  return {
    setSearch: (s: string) => {
      search = s;
    },
    getLocation: (): Location => ({search}) as Location,
  };
}

describe('webstatus-subscriptions-page', () => {
  let sandbox: sinon.SinonSandbox;
  let apiClient: APIClient;
  let user: UserContext;
  let element: SubscriptionsPage;
  let mockLocationHelper: ReturnType<typeof mockLocation>;

  const mockSubscriptions: components['schemas']['SubscriptionResponse'][] = [
    {
      id: 'sub1',
      saved_search_id: 'search1',
      channel_id: 'channel1',
      frequency: 'weekly',
      triggers: [],
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    },
  ];

  const mockSavedSearches: components['schemas']['SavedSearchResponse'][] = [
    {
      id: 'search1',
      name: 'Test Search 1',
      query: 'is:test',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      permissions: {role: 'saved_search_owner'},
      bookmark_status: {status: 'bookmark_none'},
    },
  ];
  const mockNotificationChannels: components['schemas']['NotificationChannelResponse'][] =
    [
      {
        id: 'channel1',
        name: 'test@example.com',
        type: 'email',
        config: {type: 'email', address: 'test@example.com'},
        created_at: new Date().toISOString(),
        status: 'enabled',
        updated_at: new Date().toISOString(),
      },
    ];

  beforeEach(async () => {
    sandbox = sinon.createSandbox();
    apiClient = {
      listSubscriptions: sandbox.stub().resolves(mockSubscriptions),
      getAllUserSavedSearches: sandbox.stub().resolves(mockSavedSearches),
      listNotificationChannels: sandbox
        .stub()
        .resolves(mockNotificationChannels),
    } as unknown as APIClient;
    user = {user: {getIdToken: async () => 'test-token'}} as UserContext;
    mockLocationHelper = mockLocation();

    element = await fixture<SubscriptionsPage>(html`
      <webstatus-subscriptions-page
        .apiClient=${apiClient}
        .userContext=${user}
        .getLocation=${mockLocationHelper.getLocation}
      ></webstatus-subscriptions-page>
    `);
    element.toaster = sandbox.stub();
    await element.updateComplete;
  });

  afterEach(() => {
    sandbox.restore();
  });

  it('renders a skeleton when user context is loading', async () => {
    element.userContext = undefined;
    await element.updateComplete;
    const skeleton = element.shadowRoot?.querySelector('.subscription-item');
    expect(skeleton).to.exist;
    expect(skeleton?.querySelector('sl-skeleton')).to.exist;
  });

  it('renders a login prompt when user is logged out', async () => {
    element.userContext = null;
    await element.updateComplete;
    const loginPrompt = element.shadowRoot?.querySelector('.login-prompt');
    expect(loginPrompt).to.exist;
    expect(loginPrompt?.textContent).to.include(
      'Please log in to manage your subscriptions.',
    );
  });

  it('fetches and renders subscriptions', async () => {
    await element['_loadingTask'].taskComplete;
    await element.updateComplete;

    expect(apiClient.listSubscriptions).to.have.been.calledWith('test-token');
    expect(apiClient.getAllUserSavedSearches).to.have.been.calledWith(
      'test-token',
    );
    const renderedText = element.shadowRoot?.textContent;
    expect(renderedText).to.include('Test Search 1');
    expect(renderedText).to.include('test@example.com');
    expect(renderedText).to.include('Weekly'); // Check for title-cased
  });

  it('renders the correct channel icon', async () => {
    await element['_loadingTask'].taskComplete;
    await element.requestUpdate();
    await element.updateComplete;

    const item = element.shadowRoot?.querySelector('.subscription-item');
    const icon = item?.querySelector('sl-icon');

    expect(icon).to.exist;
    expect(icon?.getAttribute('name')).to.equal('envelope');
  });

  it('opens dialog on unsubscribe link', async () => {
    mockLocationHelper.setSearch('?unsubscribe=test-sub-id');
    // willUpdate is called before update, so we need to trigger an update.
    element.requestUpdate();
    await element.updateComplete;
    expect(element['_isSubscriptionDialogOpen']).to.be.true;
    expect(element['_activeSubscriptionId']).to.equal('test-sub-id');
  });

  it('refreshes on subscription save event', async () => {
    await element['_loadingTask'].taskComplete;
    await element.updateComplete;
    const runSpy = sandbox.spy(element['_loadingTask'], 'run');

    const dialog = element.shadowRoot?.querySelector(
      'webstatus-manage-subscriptions-dialog',
    );
    dialog?.dispatchEvent(new CustomEvent('subscription-save-success'));

    expect(runSpy).to.have.been.calledOnce;
  });

  it('refreshes on subscription delete event', async () => {
    await element['_loadingTask'].taskComplete;
    await element.updateComplete;
    const runSpy = sandbox.spy(element['_loadingTask'], 'run');

    const dialog = element.shadowRoot?.querySelector(
      'webstatus-manage-subscriptions-dialog',
    );
    dialog?.dispatchEvent(new CustomEvent('subscription-delete-success'));

    expect(runSpy).to.have.been.calledOnce;
  });
  it('renders empty state message when no subscriptions exist', async () => {
    // Override the stub to return empty array
    (apiClient.listSubscriptions as sinon.SinonStub).resolves([]);

    // Re-initialize element or force task rerun (easier to create new fixture)
    element = await fixture<SubscriptionsPage>(html`
      <webstatus-subscriptions-page
        .apiClient=${apiClient}
        .userContext=${user}
        .getLocation=${mockLocationHelper.getLocation}
      ></webstatus-subscriptions-page>
    `);

    await element['_loadingTask'].taskComplete;
    await element.updateComplete;

    const message = element.shadowRoot?.querySelector('p');
    expect(message?.textContent).to.include('No subscriptions found');
    expect(element.shadowRoot?.querySelector('.subscription-list')).to.not
      .exist;
  });
  it('renders error message when API fails', async () => {
    const errorMsg = 'Network error';
    (apiClient.listSubscriptions as sinon.SinonStub).rejects(
      new Error(errorMsg),
    );

    element = await fixture<SubscriptionsPage>(html`
      <webstatus-subscriptions-page
        .apiClient=${apiClient}
        .userContext=${user}
        .getLocation=${mockLocationHelper.getLocation}
      ></webstatus-subscriptions-page>
    `);

    try {
      await element['_loadingTask'].taskComplete;
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
    } catch (_e: unknown) {
      // Ignored. Even with the '_' prefix, TypeScript still complains if we don't use the variable at all.
      // Despite our eslint rule to the contrary.
    }

    await element.updateComplete;

    const text = element.shadowRoot?.textContent;
    expect(text).to.include(`Error: Error: ${errorMsg}`);
  });
  it('opens edit dialog when edit button is clicked', async () => {
    await element['_loadingTask'].taskComplete;
    await element.updateComplete;

    const editButton = element.shadowRoot?.querySelector<HTMLElement>(
      '.subscription-actions sl-button:first-child',
    );
    editButton?.click();
    await element.updateComplete;

    expect(element['_isSubscriptionDialogOpen']).to.be.true;
    expect(element['_activeSubscriptionId']).to.equal('sub1');
    expect(element['_activeSavedSearchId']).to.equal('search1');
  });

  it('opens delete dialog when delete button is clicked', async () => {
    await element['_loadingTask'].taskComplete;
    await element.updateComplete;

    const deleteButton = element.shadowRoot?.querySelector<HTMLElement>(
      '.subscription-actions sl-button[variant="danger"]',
    );
    deleteButton?.click();
    await element.updateComplete;

    expect(element['_isSubscriptionDialogOpen']).to.be.true;
    expect(element['_activeSubscriptionId']).to.equal('sub1');

    // Confirm that we are in delete mode by checking that activeSavedSearchId is undefined.
    expect(element['_activeSavedSearchId']).to.be.undefined;
  });
});
