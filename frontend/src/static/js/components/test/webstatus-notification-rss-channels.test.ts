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

import {assert, fixture, html} from '@open-wc/testing';
import type {WebstatusNotificationRssChannels} from '../../components/webstatus-notification-rss-channels.js';
import '../../components/webstatus-notification-rss-channels.js';
import '../../components/webstatus-notification-panel.js';
import type {components} from 'webstatus.dev-backend';
import {APIClient} from '../../api/client.js';

describe('webstatus-notification-rss-channels', () => {
  it('displays "No RSS feeds configured." message', async () => {
    const el = await fixture<WebstatusNotificationRssChannels>(html`
      <webstatus-notification-rss-channels></webstatus-notification-rss-channels>
    `);
    const basePanel = el.shadowRoot!.querySelector(
      'webstatus-notification-panel',
    );
    assert.isNotNull(basePanel);
    const noChannelsText = basePanel!.querySelector(
      '[slot="content"] p',
    ) as HTMLParagraphElement;
    assert.isNotNull(noChannelsText);
    assert.include(noChannelsText.textContent, 'No RSS feeds configured.');
  });

  it('does not display "Create RSS channel" button', async () => {
    const el = await fixture<WebstatusNotificationRssChannels>(html`
      <webstatus-notification-rss-channels></webstatus-notification-rss-channels>
    `);

    const basePanel = el.shadowRoot!.querySelector(
      'webstatus-notification-panel',
    );
    assert.isNotNull(basePanel);

    const actionsSlot = basePanel!.querySelector('[slot="actions"]');
    if (actionsSlot) {
      const button = actionsSlot.querySelector('sl-button');
      assert.isNull(button);
    }
  });

  it('displays list of subscriptions', async () => {
    const mockApiClient = {
      getBaseUrl: () => 'http://localhost:8080',
    } as unknown as APIClient;
    const mockSubscriptions: components['schemas']['SubscriptionResponse'][] = [
      {
        id: 'sub-1',
        channel_id: 'channel-1',
        channel_type: 'rss',
        frequency: 'immediate',
        triggers: [],
        subscribable: {
          id: 'search-1',
          name: 'My Search 1',
        },
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
      {
        id: 'sub-2',
        channel_id: 'channel-2',
        channel_type: 'rss',
        frequency: 'weekly',
        triggers: [],
        subscribable: {
          id: 'search-2',
          name: 'My Search 2',
        },
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
    ];

    const el = await fixture<WebstatusNotificationRssChannels>(html`
      <webstatus-notification-rss-channels
        .subscriptions=${mockSubscriptions}
        .apiClient=${mockApiClient}
      ></webstatus-notification-rss-channels>
    `);

    const basePanel = el.shadowRoot!.querySelector(
      'webstatus-notification-panel',
    );
    assert.isNotNull(basePanel);

    const channelItems = basePanel!.querySelectorAll('.channel-item');
    assert.equal(channelItems.length, 2);

    assert.include(channelItems[0].textContent, 'My Search 1');
    assert.include(channelItems[0].textContent, '/v1/subscriptions/sub-1/rss');
    assert.include(channelItems[1].textContent, 'My Search 2');
    assert.include(channelItems[1].textContent, '/v1/subscriptions/sub-2/rss');
  });
});
