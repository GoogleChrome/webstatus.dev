/**
 * Copyright 2023 Google LLC
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

import {consume} from '@lit/context';
import {assert, fixture, html} from '@open-wc/testing';
import {LitElement, render} from 'lit';
import {customElement, property} from 'lit/decorators.js';

import {apiClientContext} from '../../contexts/api-client-context.js';
import '../webstatus-api-client-service.js';
import {type WebstatusAPIClientService} from '../webstatus-api-client-service.js';
import {APIClient} from '../../api/client.js';

describe('webstatus-api-client-service', () => {
  const apiUrl = 'http://localhost';
  it('can be added to the page with the api url', async () => {
    const component = await fixture<WebstatusAPIClientService>(
      html`<webstatus-api-client-service url=${apiUrl}>
      </webstatus-api-client-service>`
    );
    assert.exists(component);
    assert.equal(component.url, 'http://localhost');
    assert.exists(component.apiClient);
  });
  it('can have child components which are provided the api client via context', async () => {
    @customElement('fake-child-element')
    class FakeChildElement extends LitElement {
      @consume({context: apiClientContext, subscribe: true})
      @property({attribute: false})
      apiClient!: APIClient;
    }
    const root = document.createElement('div');
    document.body.appendChild(root);
    render(
      html` <webstatus-api-client-service .url=${apiUrl}>
        <fake-child-element></fake-child-element>
      </webstatus-api-client-service>`,
      root
    );
    const component = root.querySelector(
      'webstatus-api-client-service'
    ) as WebstatusAPIClientService;
    const childComponent = root.querySelector(
      'fake-child-element'
    ) as FakeChildElement;
    await component.updateComplete;
    await childComponent.updateComplete;

    assert.exists(component);
    assert.equal(component.url, 'http://localhost');
    assert.exists(component.apiClient);

    assert.exists(childComponent);
    assert.exists(childComponent.apiClient);
  });
});
