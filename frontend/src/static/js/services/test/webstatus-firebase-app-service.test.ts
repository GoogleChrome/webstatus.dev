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

import '../webstatus-firebase-app-service.js';
import {type WebstatusFirebaseAppService} from '../webstatus-firebase-app-service.js';
import {
  FirebaseApp,
  firebaseAppContext,
} from '../../contexts/firebase-app-context.js';

describe('webstatus-firebase-app-service', () => {
  const settings = {
    apiKey: 'testapikey',
    authDomain: 'testauthdomain',
  };
  it('can be added to the page with the settings', async () => {
    const component = await fixture<WebstatusFirebaseAppService>(
      html`<webstatus-firebase-app-service .settings=${settings}>
      </webstatus-firebase-app-service>`,
    );
    assert.exists(component);
    assert.equal(component.settings, settings);
    assert.exists(component.firebaseApp);
  });
  it('can have child components which are provided the settings via context', async () => {
    @customElement('fake-child-element')
    class FakeChildElement extends LitElement {
      @consume({context: firebaseAppContext, subscribe: true})
      @property({attribute: false})
      firebaseApp?: FirebaseApp;
    }
    const root = document.createElement('div');
    document.body.appendChild(root);
    render(
      html` <webstatus-firebase-app-service .settings=${settings}>
        <fake-child-element></fake-child-element>
      </webstatus-firebase-app-service>`,
      root,
    );
    const component = root.querySelector(
      'webstatus-firebase-app-service',
    ) as WebstatusFirebaseAppService;
    const childComponent = root.querySelector(
      'fake-child-element',
    ) as FakeChildElement;
    await component.updateComplete;
    await childComponent.updateComplete;

    assert.exists(component);
    assert.exists(childComponent);
    assert.exists(component.firebaseApp);
    assert.equal(component.firebaseApp, childComponent.firebaseApp);
  });
});
