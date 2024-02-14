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

import {
  type APIClient,
  apiClientContext,
} from '../../contexts/api-client-context.js';
import {
  type AppSettings,
  appSettingsContext,
} from '../../contexts/settings-context.js';
import '../webstatus-app-settings.js';
import {type WebstatusAppSettings} from '../webstatus-app-settings.js';

describe('webstatus-app-settings', () => {
  const settings: AppSettings = {
    apiUrl: 'http://localhost',
    gsiClientId: 'testclientid',
  };
  it('can be added to the page with the settings', async () => {
    const component = await fixture<WebstatusAppSettings>(
      html`<webstatus-app-settings .appSettings=${settings}>
      </webstatus-app-settings>`
    );
    assert.exists(component);
    assert.equal(component.appSettings.apiUrl, 'http://localhost');
    assert.equal(component.appSettings.gsiClientId, 'testclientid');
  });
  it('can have child components which are provided the settings via context', async () => {
    @customElement('fake-child-element')
    class FakeChildElement extends LitElement {
      @consume({context: apiClientContext, subscribe: true})
      @property({attribute: false})
      apiClient!: APIClient;

      @consume({context: appSettingsContext, subscribe: true})
      @property({attribute: false})
      appSettings!: AppSettings;
    }
    const root = document.createElement('div');
    document.body.appendChild(root);
    render(
      html` <webstatus-app-settings .appSettings=${settings}>
        <fake-child-element></fake-child-element>
      </webstatus-app-settings>`,
      root
    );
    const component = root.querySelector(
      'webstatus-app-settings'
    ) as WebstatusAppSettings;
    const childComponent = root.querySelector(
      'fake-child-element'
    ) as FakeChildElement;
    await component.updateComplete;
    await childComponent.updateComplete;

    assert.exists(component);
    assert.equal(component.appSettings.apiUrl, 'http://localhost');
    assert.equal(component.appSettings.gsiClientId, 'testclientid');

    assert.exists(childComponent);
    assert.equal(childComponent.appSettings.apiUrl, 'http://localhost');
    assert.equal(childComponent.appSettings.gsiClientId, 'testclientid');
  });
});
