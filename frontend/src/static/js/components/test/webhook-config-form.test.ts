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
import '@shoelace-style/shoelace/dist/components/input/input.js';
import '../webhook-config-form.js';
import {WebhookConfigForm} from '../webhook-config-form.js';
import type {components} from 'webstatus.dev-backend';
import {SlInput} from '@shoelace-style/shoelace';

describe('webhook-config-form', () => {
  let element: WebhookConfigForm;

  const mockChannel: components['schemas']['NotificationChannelResponse'] = {
    id: 'test-channel-id',
    type: 'webhook',
    name: 'Original Webhook Name',
    config: {type: 'webhook', url: 'https://hooks.slack.com/services/original'},
    status: 'enabled',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  };

  describe('Create Mode (No Initial Channel)', () => {
    beforeEach(async () => {
      element = await fixture<WebhookConfigForm>(html`
        <webhook-config-form></webhook-config-form>
      `);
    });

    it('renders empty inputs initially', async () => {
      expect(element).to.be.instanceOf(WebhookConfigForm);
      const nameInput =
        element.shadowRoot?.querySelector<SlInput>('#webhook-name');
      const urlInput =
        element.shadowRoot?.querySelector<SlInput>('#webhook-url');
      expect(nameInput?.value).to.equal('');
      expect(urlInput?.value).to.equal('');
    });

    it('is initially not dirty', async () => {
      expect(element.isDirty()).to.be.false;
    });

    it('becomes dirty when inputs are typed into', async () => {
      const nameInput =
        element.shadowRoot?.querySelector<SlInput>('#webhook-name');
      nameInput!.value = 'New Name';
      nameInput?.dispatchEvent(new CustomEvent('sl-input'));
      await element.updateComplete;

      expect(element.isDirty()).to.be.true;
    });

    it('emits a change event with full payload when input occurs', async () => {
      const changeSpy = sinon.spy();
      element.addEventListener('change', changeSpy);

      const nameInput =
        element.shadowRoot?.querySelector<SlInput>('#webhook-name');
      nameInput!.value = 'New Name';
      nameInput?.dispatchEvent(new CustomEvent('sl-input'));
      await element.updateComplete;

      expect(changeSpy).to.have.been.calledOnce;
      const detail = changeSpy.args[0][0].detail;
      // In create mode, both name and config masks are always forced because !this.channel
      expect(detail.mask).to.deep.equal(['name', 'config']);
      expect(detail.updates).to.deep.equal({
        name: 'New Name',
        config: {type: 'webhook', url: ''}, // URL hasn't been typed yet
      });
    });

    it('validates correctly by delegating to input elements', async () => {
      const nameInput =
        element.shadowRoot?.querySelector<SlInput>('#webhook-name');
      const urlInput =
        element.shadowRoot?.querySelector<SlInput>('#webhook-url');

      const nameStub = sinon.stub(nameInput!, 'reportValidity').returns(true);
      const urlStub = sinon.stub(urlInput!, 'reportValidity').returns(false);

      expect(element.validate()).to.be.false;
      expect(nameStub).to.have.been.calledOnce;
      expect(urlStub).to.have.been.calledOnce;
    });
  });

  describe('Edit Mode (With Initial Channel)', () => {
    beforeEach(async () => {
      element = await fixture<WebhookConfigForm>(html`
        <webhook-config-form .channel=${mockChannel}></webhook-config-form>
      `);
    });

    it('renders pre-filled inputs', async () => {
      const nameInput =
        element.shadowRoot?.querySelector<SlInput>('#webhook-name');
      const urlInput =
        element.shadowRoot?.querySelector<SlInput>('#webhook-url');
      expect(nameInput?.value).to.equal('Original Webhook Name');
      expect(urlInput?.value).to.equal(
        'https://hooks.slack.com/services/original',
      );
    });

    it('is initially not dirty', async () => {
      expect(element.isDirty()).to.be.false;
    });

    it('does not become dirty if typing the exact same value', async () => {
      const nameInput =
        element.shadowRoot?.querySelector<SlInput>('#webhook-name');
      nameInput!.value = 'Original Webhook Name';
      nameInput?.dispatchEvent(new CustomEvent('sl-input'));
      await element.updateComplete;

      expect(element.isDirty()).to.be.false;
    });

    it('only includes modified fields in the update payload mask', async () => {
      const changeSpy = sinon.spy();
      element.addEventListener('change', changeSpy);

      // We ONLY update the URL for an existing channel
      const urlInput =
        element.shadowRoot?.querySelector<SlInput>('#webhook-url');
      urlInput!.value = 'https://example.com/new-hook';
      urlInput?.dispatchEvent(new CustomEvent('sl-input'));
      await element.updateComplete;

      expect(changeSpy).to.have.been.calledOnce;
      const detail = changeSpy.args[0][0].detail;

      // Because we only typed in URL, name should NOT be in the mask
      expect(detail.mask).to.deep.equal(['config']);
      expect(detail.updates).to.deep.equal({
        config: {type: 'webhook', url: 'https://example.com/new-hook'},
      });
      expect(detail.updates.name).to.be.undefined;
    });
  });
});
