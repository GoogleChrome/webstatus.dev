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
import '../../components/webstatus-notification-panel.js';
import type {WebstatusNotificationPanel} from '../../components/webstatus-notification-panel.js';

describe('webstatus-notification-panel', () => {
  it('renders content in the content slot', async () => {
    const el = await fixture<WebstatusNotificationPanel>(html`
      <webstatus-notification-panel>
        <div slot="content">Test Content</div>
      </webstatus-notification-panel>
    `);
    const contentSlot = el.shadowRoot!.querySelector<HTMLSlotElement>(
      'slot[name="content"]',
    );
    assert.isNotNull(contentSlot);
    const assignedNodes = contentSlot.assignedNodes({
      flatten: true,
    }) as HTMLElement[];
    assert.include(assignedNodes[0].textContent, 'Test Content');
  });

  it('renders an icon in the icon slot', async () => {
    const el = await fixture<WebstatusNotificationPanel>(html`
      <webstatus-notification-panel>
        <sl-icon name="test-icon" slot="icon"></sl-icon>
      </webstatus-notification-panel>
    `);
    const iconSlot =
      el.shadowRoot!.querySelector<HTMLSlotElement>('slot[name="icon"]');
    assert.isNotNull(iconSlot);
    const assignedNodes = iconSlot.assignedNodes({
      flatten: true,
    }) as HTMLElement[];
    assert.equal(assignedNodes[0].tagName, 'SL-ICON');
    assert.equal(assignedNodes[0].getAttribute('name'), 'test-icon');
  });

  it('renders a title in the title slot', async () => {
    const el = await fixture<WebstatusNotificationPanel>(html`
      <webstatus-notification-panel>
        <span slot="title">Test Title</span>
      </webstatus-notification-panel>
    `);
    const titleSlot =
      el.shadowRoot!.querySelector<HTMLSlotElement>('slot[name="title"]');
    assert.isNotNull(titleSlot);
    const assignedNodes = titleSlot.assignedNodes({
      flatten: true,
    }) as HTMLElement[];
    assert.include(assignedNodes[0].textContent, 'Test Title');
  });

  it('renders actions in the actions slot', async () => {
    const el = await fixture<WebstatusNotificationPanel>(html`
      <webstatus-notification-panel>
        <sl-button slot="actions">Action Button</sl-button>
      </webstatus-notification-panel>
    `);
    const actionsSlot = el.shadowRoot!.querySelector<HTMLSlotElement>(
      'slot[name="actions"]',
    );
    assert.isNotNull(actionsSlot);
    const assignedNodes = actionsSlot.assignedNodes({
      flatten: true,
    }) as HTMLElement[];
    assert.equal(assignedNodes[0].tagName, 'SL-BUTTON');
    assert.include(assignedNodes[0].textContent, 'Action Button');
  });

  it('displays skeletons when loading is true and hides content', async () => {
    const el = await fixture<WebstatusNotificationPanel>(html`
      <webstatus-notification-panel .loading=${true}>
        <div slot="content">Test Content</div>
      </webstatus-notification-panel>
    `);
    const skeletons = el.shadowRoot!.querySelectorAll('sl-skeleton');
    assert.equal(skeletons.length, 2);
    const contentSlot = el.shadowRoot!.querySelector<HTMLSlotElement>(
      'slot[name="content"]',
    );
    assert.isNull(contentSlot);
  });

  it('hides skeletons when loading is false and shows content', async () => {
    const el = await fixture<WebstatusNotificationPanel>(html`
      <webstatus-notification-panel .loading=${false}>
        <div slot="content">Test Content</div>
      </webstatus-notification-panel>
    `);
    const skeletons = el.shadowRoot!.querySelectorAll('sl-skeleton');
    assert.equal(skeletons.length, 0);
    const contentSlot = el.shadowRoot!.querySelector<HTMLSlotElement>(
      'slot[name="content"]',
    );
    assert.isNotNull(contentSlot);
    const assignedNodes = contentSlot.assignedNodes({
      flatten: true,
    }) as HTMLElement[];
    assert.include(assignedNodes[0].textContent, 'Test Content');
  });
});
