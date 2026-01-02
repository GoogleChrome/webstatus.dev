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
import type {WebstatusNotificationEmailChannels} from '../../components/webstatus-notification-email-channels.js';
import '../../components/webstatus-notification-email-channels.js';
import {components} from 'webstatus.dev-backend';
import {WebstatusNotificationPanel} from '../webstatus-notification-panel.js';

type NotificationChannelResponse =
  components['schemas']['NotificationChannelResponse'];

describe('webstatus-notification-email-channels', () => {
  it('renders email channels correctly', async () => {
    const mockChannels: NotificationChannelResponse[] = [
      {
        id: '1',
        type: 'email',
        value: 'test1@example.com',
        name: 'Email 1',
        status: 'enabled',
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-01T00:00:00Z',
      },
      {
        id: '2',
        type: 'email',
        value: 'test2@example.com',
        name: 'Email 2',
        status: 'disabled',
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-01T00:00:00Z',
      },
    ];

    const el = await fixture<WebstatusNotificationEmailChannels>(html`
      <webstatus-notification-email-channels
        .channels=${mockChannels}
      ></webstatus-notification-email-channels>
    `);

    const emailItems = el.shadowRoot!.querySelectorAll('.channel-item');
    assert.equal(emailItems.length, mockChannels.length);

    // Test first email channel
    const email1Name = emailItems[0].querySelector('.name');
    assert.include(email1Name!.textContent, 'test1@example.com');
    const email1Badge = emailItems[0].querySelector('sl-badge');
    assert.isNotNull(email1Badge);
    assert.include(email1Badge!.textContent, 'Enabled');

    // Test second email channel (disabled, so no badge)
    const email2Name = emailItems[1].querySelector('.name');
    assert.include(email2Name!.textContent, 'test2@example.com');
    const email2Badge = emailItems[1].querySelector('sl-badge');
    assert.isNotNull(email2Badge);
    assert.include(email2Badge!.textContent, 'Disabled');
  });

  it('passes loading state to the base panel', async () => {
    const el = await fixture<WebstatusNotificationEmailChannels>(html`
      <webstatus-notification-email-channels
        .loading=${true}
      ></webstatus-notification-email-channels>
    `);

    const basePanel = el.shadowRoot!.querySelector<WebstatusNotificationPanel>(
      'webstatus-notification-panel',
    );
    assert.isNotNull(basePanel);
    assert.isTrue(basePanel!.loading);
  });
});
