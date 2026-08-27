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
import '../webstatus-code-subscriptions-page.js';
import {WebstatusCodeSubscriptionsPage} from '../webstatus-code-subscriptions-page.js';
import {type components} from 'webstatus.dev-backend';

describe('webstatus-code-subscriptions-page', () => {
  let sandbox: sinon.SinonSandbox;
  let apiClient: APIClient;
  let user: UserContext;
  let element: WebstatusCodeSubscriptionsPage;

  const mockSubscriptions: components['schemas']['CodeSubscriptionPage'] = {
    data: [
      {
        id: 'sub-1',
        vcs_provider: 'github',
        vcs_installation_id: '12345',
        vcs_repository_id: '67890',
        repository_owner: 'GoogleChrome',
        repository_name: 'webstatus.dev',
        repository_full_name: 'GoogleChrome/webstatus.dev',
        feature_id: 'css-subgrid',
        target_query: 'id:css-subgrid',
        triggers: ['feature_baseline_to_widely'],
        occurrences: [
          {
            file_path: 'src/grid.css',
            line_number: 42,
            comment_snippet:
              '/* TODO(baseline/subgrid): Remove flexbox fallback */',
          },
        ],
        occurrence_count: 1,
        status: 'ACTIVE',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
    ],
  };

  beforeEach(async () => {
    sandbox = sinon.createSandbox();
    apiClient = {
      listCodeSubscriptions: sandbox.stub().resolves(mockSubscriptions),
    } as unknown as APIClient;

    user = {
      user: {
        uid: 'user-1',
        email: 'test@example.com',
        getIdToken: sandbox.stub().resolves('test-token'),
      },
      syncState: 'idle',
    } as unknown as UserContext;

    element = await fixture<WebstatusCodeSubscriptionsPage>(html`
      <webstatus-code-subscriptions-page
        .apiClient=${apiClient}
        .userContext=${user}
      ></webstatus-code-subscriptions-page>
    `);
    await element.updateComplete;
  });

  afterEach(() => {
    sandbox.restore();
  });

  it('renders login prompt when user is unauthenticated (null)', async () => {
    element.userContext = null;
    await element.requestUpdate();
    await element.updateComplete;

    const loginPrompt = element.shadowRoot?.querySelector('.login-prompt');
    expect(loginPrompt).to.exist;
    expect(loginPrompt?.textContent).to.include('Sign in to view and manage');
  });

  it('renders subscriptions table and quota bar when data is loaded', async () => {
    await element._dataTask.taskComplete;
    await element.updateComplete;

    const quotaCard = element.shadowRoot?.querySelector('.quota-card');
    expect(quotaCard).to.exist;

    const table = element.shadowRoot?.querySelector('.subscriptions-table');
    expect(table).to.exist;
    expect(table?.textContent).to.include('id:css-subgrid');
  });

  it('renders empty subscriptions message when repo has no subscriptions', async () => {
    element.apiClient = {
      ...apiClient,
      listCodeSubscriptions: sandbox.stub().resolves({data: []}),
    } as unknown as APIClient;
    element._dataTask.run();
    await element._dataTask.taskComplete;
    await element.requestUpdate();
    await element.updateComplete;

    const emptyState = element.shadowRoot?.querySelector('.empty-state');
    expect(emptyState).to.exist;
    expect(emptyState?.textContent).to.include(
      'No active code subscriptions found',
    );
  });

  it('renders error alert when API fetch fails and retries on button click', async () => {
    element.apiClient = {
      ...apiClient,
      listCodeSubscriptions: sandbox.stub().rejects(new Error('Network error')),
    } as unknown as APIClient;
    element._dataTask.run();
    try {
      await element._dataTask.taskComplete;
    } catch {
      // Expected rejection
    }
    await element.requestUpdate();
    await element.updateComplete;

    const repoInput = element.shadowRoot?.querySelector('.repo-selector');
    expect(repoInput).to.exist;

    const errorAlert = element.shadowRoot?.querySelector(
      'sl-alert[variant="danger"]',
    );
    expect(errorAlert).to.exist;
    expect(errorAlert?.textContent).to.include(
      'Error loading code subscriptions',
    );

    const retryBtn = element.shadowRoot?.querySelector(
      'sl-button[variant="default"]',
    ) as HTMLElement;
    expect(retryBtn).to.exist;
  });
});
