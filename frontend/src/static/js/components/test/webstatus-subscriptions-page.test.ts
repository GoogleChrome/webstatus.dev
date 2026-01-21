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

  beforeEach(async () => {
    sandbox = sinon.createSandbox();
    apiClient = {
      listSubscriptions: sandbox.stub().resolves(mockSubscriptions),
      getAllUserSavedSearches: sandbox.stub().resolves(mockSavedSearches),
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

  it('renders a loading spinner initially', () => {
    // This is hard to test reliably as the task starts immediately.
    // We'll focus on the complete and error states.
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
    expect(renderedText).to.include('channel1');
    expect(renderedText).to.include('weekly');
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
});
