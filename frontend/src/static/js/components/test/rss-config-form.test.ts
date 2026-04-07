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
import '../rss-config-form.js';
import {RssConfigForm} from '../rss-config-form.js';
import type {components} from 'webstatus.dev-backend';
import {SlInput} from '@shoelace-style/shoelace';

describe('rss-config-form', () => {
  let element: RssConfigForm;

  const mockChannel: components['schemas']['NotificationChannelResponse'] = {
    id: 'test-channel-id',
    type: 'rss',
    name: 'Original RSS Name',
    config: {type: 'rss'},
    status: 'enabled',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  };

  describe('Create Mode (No Initial Channel)', () => {
    beforeEach(async () => {
      element = await fixture<RssConfigForm>(html`
        <rss-config-form></rss-config-form>
      `);
    });

    it('renders empty input initially', async () => {
      expect(element).to.be.instanceOf(RssConfigForm);
      const nameInput = element.shadowRoot?.querySelector<SlInput>('#rss-name');
      expect(nameInput?.value).to.equal('');
    });

    it('is initially not dirty', async () => {
      expect(element.isDirty()).to.be.false;
    });

    it('becomes dirty when inputs are typed into', async () => {
      const nameInput = element.shadowRoot?.querySelector<SlInput>('#rss-name');
      nameInput!.value = 'New Name';
      nameInput?.dispatchEvent(new CustomEvent('sl-input'));
      await element.updateComplete;

      expect(element.isDirty()).to.be.true;
    });

    it('emits a change event with full payload when input occurs', async () => {
      const changeSpy = sinon.spy();
      element.addEventListener('change', changeSpy);

      const nameInput = element.shadowRoot?.querySelector<SlInput>('#rss-name');
      nameInput!.value = 'New Name';
      nameInput?.dispatchEvent(new CustomEvent('sl-input'));
      await element.updateComplete;

      expect(changeSpy).to.have.been.calledOnce;
      const detail = changeSpy.args[0][0].detail;
      // In create mode, both name and config masks are forced
      expect(detail.mask).to.deep.equal(['name', 'config']);
      expect(detail.updates).to.deep.equal({
        name: 'New Name',
        config: {type: 'rss'},
      });
    });

    it('validates correctly by delegating to input elements', async () => {
      const nameInput = element.shadowRoot?.querySelector<SlInput>('#rss-name');

      const nameStub = sinon.stub(nameInput!, 'reportValidity').returns(true);

      expect(element.validate()).to.be.true;
      expect(nameStub).to.have.been.calledOnce;
    });

    it('fails validation when name is empty', async () => {
      const nameInput = element.shadowRoot?.querySelector<SlInput>('#rss-name');
      nameInput!.value = '';
      nameInput?.dispatchEvent(new CustomEvent('sl-input'));
      await element.updateComplete;

      expect(element.validate()).to.be.false;
    });
  });

  describe('Edit Mode (With Initial Channel)', () => {
    beforeEach(async () => {
      element = await fixture<RssConfigForm>(html`
        <rss-config-form .channel=${mockChannel}></rss-config-form>
      `);
    });

    it('renders pre-filled input', async () => {
      const nameInput = element.shadowRoot?.querySelector<SlInput>('#rss-name');
      expect(nameInput?.value).to.equal('Original RSS Name');
    });

    it('is initially not dirty', async () => {
      expect(element.isDirty()).to.be.false;
    });

    it('does not become dirty if typing the exact same value', async () => {
      const nameInput = element.shadowRoot?.querySelector<SlInput>('#rss-name');
      nameInput!.value = 'Original RSS Name';
      nameInput?.dispatchEvent(new CustomEvent('sl-input'));
      await element.updateComplete;

      expect(element.isDirty()).to.be.false;
    });

    it('only includes modified fields in the update payload mask', async () => {
      const changeSpy = sinon.spy();
      element.addEventListener('change', changeSpy);

      const nameInput = element.shadowRoot?.querySelector<SlInput>('#rss-name');
      nameInput!.value = 'Updated Name';
      nameInput?.dispatchEvent(new CustomEvent('sl-input'));
      await element.updateComplete;

      expect(changeSpy).to.have.been.calledOnce;
      const detail = changeSpy.args[0][0].detail;

      expect(detail.mask).to.deep.equal(['name']);
      expect(detail.updates).to.deep.equal({
        name: 'Updated Name',
      });
    });
  });
});
