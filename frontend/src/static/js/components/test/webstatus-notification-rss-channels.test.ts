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

describe('webstatus-notification-rss-channels', () => {
  it('displays "Coming soon" message', async () => {
    const el = await fixture<WebstatusNotificationRssChannels>(html`
      <webstatus-notification-rss-channels></webstatus-notification-rss-channels>
    `);

    const basePanel = el.shadowRoot!.querySelector(
      'webstatus-notification-panel',
    );
    assert.isNotNull(basePanel);

    const comingSoonText = basePanel!.querySelector(
      '[slot="content"] p',
    ) as HTMLParagraphElement;
    assert.isNotNull(comingSoonText);
    assert.include(comingSoonText.textContent, 'Coming soon');
  });

  it('displays "Create RSS channel" button', async () => {
    const el = await fixture<WebstatusNotificationRssChannels>(html`
      <webstatus-notification-rss-channels></webstatus-notification-rss-channels>
    `);

    const basePanel = el.shadowRoot!.querySelector(
      'webstatus-notification-panel',
    );
    assert.isNotNull(basePanel);

    const createButton = basePanel!.querySelector(
      '[slot="actions"] sl-button',
    ) as HTMLButtonElement;
    assert.isNotNull(createButton);
    assert.include(
      createButton.textContent!.trim().replace(/\s+/g, ' '),
      'Create RSS channel',
    );
  });
});
