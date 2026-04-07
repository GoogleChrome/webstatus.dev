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
import '../webstatus-manage-notification-channel-dialog.js';
import {ManageNotificationChannelDialog} from '../webstatus-manage-notification-channel-dialog.js';
import type {components} from 'webstatus.dev-backend';
import {SlButton, SlDialog} from '@shoelace-style/shoelace';
import {ChannelConfigRegistry} from '../channel-config-registry.js';
import {ChannelConfigUpdate} from '../channel-config-types.js';
import '@shoelace-style/shoelace/dist/components/dialog/dialog.js';
import '@shoelace-style/shoelace/dist/components/button/button.js';

// Mock used to isolate nested form rendering and avoid needing to cast to access the component field.
class MockConfigForm extends HTMLElement {
  validate = () => true;
  getUpdate = () => ({
    updates: {name: 'Updated name'},
    mask: ['name'],
  });
}

if (!customElements.get('mock-config-form')) {
  customElements.define('mock-config-form', MockConfigForm);
}

describe('webstatus-manage-notification-channel-dialog', () => {
  let element: ManageNotificationChannelDialog;

  const mockChannel: components['schemas']['NotificationChannelResponse'] = {
    id: 'test-channel-id',
    type: 'webhook',
    name: 'My Webhook',
    config: {type: 'webhook', url: 'https://hooks.slack.com/services/test'},
    status: 'enabled',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  };

  beforeEach(async () => {
    element = await fixture<ManageNotificationChannelDialog>(html`
      <webstatus-manage-notification-channel-dialog></webstatus-manage-notification-channel-dialog>
    `);
  });

  afterEach(() => {
    sinon.restore();
  });

  it('renders correctly initially when closed', async () => {
    expect(element).to.be.instanceOf(ManageNotificationChannelDialog);
    expect(element.open).to.be.false;
    const dialog = element.shadowRoot?.querySelector<SlDialog>('sl-dialog');
    expect(dialog?.hasAttribute('open')).to.be.false;
  });

  it('sets title and button text correctly for create mode', async () => {
    element.open = true;
    element.mode = 'create';
    element.type = 'webhook';
    await element.updateComplete;

    const dialog = element.shadowRoot?.querySelector<SlDialog>('sl-dialog');
    expect(dialog?.label).to.equal('Create Webhook Channel');

    const saveBtn = element.shadowRoot?.querySelector(
      'sl-button[variant="primary"]',
    );
    expect(saveBtn?.textContent?.trim()).to.equal('Create');
  });

  it('sets title and button text correctly for edit mode', async () => {
    element.open = true;
    element.mode = 'edit';
    element.channel = mockChannel;
    await element.updateComplete;

    const dialog = element.shadowRoot?.querySelector<SlDialog>('sl-dialog');
    expect(dialog?.label).to.equal('Edit Webhook Channel');

    const saveBtn = element.shadowRoot?.querySelector(
      'sl-button[variant="primary"]',
    );
    expect(saveBtn?.textContent?.trim()).to.equal('Save');
  });

  it('disables save button in edit mode if there are no pending updates', async () => {
    element.open = true;
    element.mode = 'edit';
    element.channel = mockChannel;
    await element.updateComplete;

    const saveBtn = element.shadowRoot?.querySelector<SlButton>(
      'sl-button[variant="primary"]',
    );
    expect(saveBtn?.disabled).to.be.true;

    // Simulate an update bubbling up from the internal config form registry
    const updates: ChannelConfigUpdate = {
      updates: {name: 'New Name'},
      mask: ['name'],
    };
    element['_pendingUpdate'] = updates;
    await element.updateComplete;

    expect(saveBtn?.disabled).to.be.false;
  });

  it('never proactively disables create button based on pending updates', async () => {
    element.open = true;
    element.mode = 'create';
    await element.updateComplete;

    const saveBtn = element.shadowRoot?.querySelector<SlButton>(
      'sl-button[variant="primary"]',
    );
    expect(saveBtn?.disabled).to.be.false;
  });

  it('reflects loading state onto the primary action button', async () => {
    element.open = true;
    element.loading = true;
    await element.updateComplete;

    const saveBtn = element.shadowRoot?.querySelector<SlButton>(
      'sl-button[variant="primary"]',
    );
    expect(saveBtn?.loading).to.be.true;
  });

  it('emits sl-hide when the cancel button is clicked', async () => {
    const hideSpy = sinon.spy();
    element.addEventListener('sl-hide', hideSpy);

    const cancelBtn = element.shadowRoot?.querySelector<SlButton>(
      'sl-button:not([variant="primary"])',
    );
    cancelBtn?.click();

    expect(hideSpy).to.have.been.calledOnce;
  });

  it('clears pending updates internally when the dialog is closed', async () => {
    element.open = true;
    await element.updateComplete;

    const updates: ChannelConfigUpdate = {
      updates: {name: 'Foo'},
      mask: ['name'],
    };
    element['_pendingUpdate'] = updates;

    element.open = false; // Trigger Lit Element lifecycle update
    await element.updateComplete;

    expect(element['_pendingUpdate']).to.be.undefined;
  });

  it('does not emit save event if the nested config form fails validation', async () => {
    const renderStub = sinon
      .stub(ChannelConfigRegistry, 'renderConfig')
      .returns(html`
        <mock-config-form class="config-form"></mock-config-form>
      `);

    // Remove the beforeEach element to prevent overlay/backdrop conflicts
    const testElement = await fixture<ManageNotificationChannelDialog>(html`
      <webstatus-manage-notification-channel-dialog></webstatus-manage-notification-channel-dialog>
    `);

    testElement.open = true;
    await testElement.updateComplete;

    const mockForm =
      testElement.shadowRoot?.querySelector<MockConfigForm>('.config-form');
    mockForm!.validate = () => false;

    const saveSpy = sinon.spy();
    testElement.addEventListener('save', saveSpy);

    const saveBtn = testElement.shadowRoot?.querySelector<SlButton>(
      'sl-button[variant="primary"]',
    );
    saveBtn?.click();

    expect(saveSpy).to.not.have.been.called;
    renderStub.restore();
  });

  it('emits a save event packed with state details when the form is valid', async () => {
    const renderStub = sinon
      .stub(ChannelConfigRegistry, 'renderConfig')
      .returns(html`
        <mock-config-form class="config-form"></mock-config-form>
      `);

    const testElement = await fixture<ManageNotificationChannelDialog>(html`
      <webstatus-manage-notification-channel-dialog></webstatus-manage-notification-channel-dialog>
    `);

    testElement.open = true;
    testElement.mode = 'edit';
    testElement.channel = mockChannel;
    await testElement.updateComplete;

    const mockUpdate: ChannelConfigUpdate = {
      updates: {name: 'Updated name'},
      mask: ['name'],
    };
    testElement['_pendingUpdate'] = mockUpdate;
    // Await update to ensure button goes enabled.
    await testElement.updateComplete;

    const saveSpy = sinon.spy();
    testElement.addEventListener('save', saveSpy);

    const saveBtn = testElement.shadowRoot?.querySelector<SlButton>(
      'sl-button[variant="primary"]',
    );
    saveBtn?.click();

    expect(saveSpy).to.have.been.calledOnce;
    expect(saveSpy.args[0][0].detail).to.deep.equal({
      mode: 'edit',
      channelId: 'test-channel-id',
      ...mockUpdate,
    });
    renderStub.restore();
  });
});
