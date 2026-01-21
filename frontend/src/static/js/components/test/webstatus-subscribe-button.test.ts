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

import {fixture, html} from '@open-wc/testing';
import {expect} from '@esm-bundle/chai';
import sinon from 'sinon';
import '../webstatus-subscribe-button.js';
import {
  SubscribeButton,
  SubscribeEvent,
} from '../webstatus-subscribe-button.js';

describe('webstatus-subscribe-button', () => {
  it('dispatches subscribe event on click', async () => {
    const savedSearchId = 'test-search-id';
    const element = await fixture<SubscribeButton>(html`
      <webstatus-subscribe-button
        saved-search-id=${savedSearchId}
      ></webstatus-subscribe-button>
    `);
    const eventSpy = sinon.spy();
    element.addEventListener('subscribe', eventSpy);

    element.shadowRoot?.querySelector('sl-button')?.click();

    expect(eventSpy).to.have.been.calledOnce;
    const event = eventSpy.args[0][0] as SubscribeEvent;
    expect(event.detail.savedSearchId).to.equal(savedSearchId);
  });
});
