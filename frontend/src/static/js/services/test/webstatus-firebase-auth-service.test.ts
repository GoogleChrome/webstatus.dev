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

import {assert, expect, fixture, html} from '@open-wc/testing';
import {WebstatusFirebaseAuthService} from '../webstatus-firebase-auth-service.js';
import {customElement, property} from 'lit/decorators.js';
import {consume, provide} from '@lit/context';
import {
  FirebaseApp,
  firebaseAppContext,
} from '../../contexts/firebase-app-context.js';
import {LitElement, TemplateResult, render} from 'lit';
import {FirebaseOptions} from 'firebase/app';
import sinon from 'sinon';
import '../webstatus-firebase-auth-service.js';
import {Auth, User} from 'firebase/auth';
import {
  AuthConfig,
  firebaseAuthContext,
} from '../../contexts/firebase-auth-context.js';
import {firebaseUserContext} from '../../contexts/firebase-user-context.js';

class FakeFirebaseApp implements FirebaseApp {
  name: string = '';
  options: FirebaseOptions = {};
  automaticDataCollectionEnabled: boolean = false;
}

@customElement('fake-parent-element')
class FakeParentElement extends LitElement {
  @provide({context: firebaseAppContext})
  @property({attribute: false})
  app?: FirebaseApp;

  render(): TemplateResult {
    return html`<slot></slot>`;
  }
}

describe('webstatus-firebase-auth-service', () => {
  const settings = {
    emulatorURL: '',
  };
  it('can be added to the page with the settings', async () => {
    const component = await fixture<WebstatusFirebaseAuthService>(
      html`<webstatus-firebase-auth-service .settings=${settings}>
      </webstatus-firebase-auth-service>`
    );
    assert.exists(component);
    assert.equal(component.settings, settings);
  });
  it('can receive the firebase app via context', async () => {
    const firebaseApp = new FakeFirebaseApp();

    const root = document.createElement('div');
    document.body.appendChild(root);

    render(
      html`
        <fake-parent-element>
          <webstatus-firebase-auth-service .settings=${settings}>
          </webstatus-firebase-auth-service>
        </fake-parent-element>
      `,
      root
    );
    const parentElement = root.querySelector<FakeParentElement>(
      'fake-parent-element'
    );
    assert.exists(parentElement);

    const component = root.querySelector<WebstatusFirebaseAuthService>(
      'webstatus-firebase-auth-service'
    );
    assert.exists(component);

    const initFirebaseAuthStub = sinon.stub(component, 'initFirebaseAuth');

    parentElement.app = firebaseApp;

    parentElement.requestUpdate();
    await parentElement.updateComplete;
    await component.updateComplete;

    expect(initFirebaseAuthStub).to.have.callCount(1);
    assert.equal(parentElement.app, firebaseApp);
    assert.equal(component.firebaseApp, parentElement.app);
  });

  it('initializes correctly with a Firebase Auth', async () => {
    @customElement('fake-child-auth-element-1')
    class FakeChildElement extends LitElement {
      @consume({context: firebaseAuthContext, subscribe: true})
      @property({attribute: false})
      firebaseAuthConfig?: AuthConfig;
    }
    const root = document.createElement('div');
    document.body.appendChild(root);
    render(
      html`<webstatus-firebase-auth-service .settings=${settings}
        ><fake-child-auth-element-1></fake-child-auth-element-1>
      </webstatus-firebase-auth-service>`,
      root
    );
    const component = root.querySelector<WebstatusFirebaseAuthService>(
      'webstatus-firebase-auth-service'
    );
    assert.exists(component);
    const childComponent = root.querySelector<FakeChildElement>(
      'fake-child-auth-element-1'
    );
    assert.exists(childComponent);
    const userStub = {} as User;
    const authStub = {
      onAuthStateChanged: (callback: (user?: User) => void) =>
        callback(userStub),
    } as Auth;
    component.authInitializer = () => authStub;
    const firebaseApp = new FakeFirebaseApp();
    component.firebaseApp = firebaseApp;
    await component.updateComplete;

    assert.exists(component.firebaseAuthConfig);
    assert.equal(
      component.firebaseAuthConfig?.auth,
      authStub,
      'auth should be set correctly'
    );
    assert.equal(
      component.firebaseAuthConfig?.icon,
      'github',
      'icon should be github'
    );
    // Ensure it gets it via context.
    assert.equal(
      component.firebaseAuthConfig,
      childComponent.firebaseAuthConfig
    );
  });

  it('sets the user when auth state changes', async () => {
    @customElement('fake-child-auth-element-2')
    class FakeChildElement extends LitElement {
      @consume({context: firebaseUserContext, subscribe: true})
      @property({attribute: false})
      user?: User;
    }
    const root = document.createElement('div');
    document.body.appendChild(root);
    render(
      html`<webstatus-firebase-auth-service .settings=${settings}
        ><fake-child-auth-element-2></fake-child-auth-element-2>
      </webstatus-firebase-auth-service>`,
      root
    );
    const component = root.querySelector<WebstatusFirebaseAuthService>(
      'webstatus-firebase-auth-service'
    );
    assert.exists(component);
    const childComponent = root.querySelector<FakeChildElement>(
      'fake-child-auth-element-2'
    );
    assert.exists(childComponent);

    const userStub = {} as User;
    const authStub = {
      onAuthStateChanged: (callback: (user: User | null) => void) =>
        callback(userStub),
    } as Auth;
    component.authInitializer = () => authStub;
    component.firebaseApp = {} as FirebaseApp;
    await component.updateComplete;

    // Ensure it gets the same user via context.
    assert.equal(component.user, userStub);
    assert.equal(childComponent.user, userStub);
  });

  it('calls emulatorConnector when emulatorURL is set', async () => {
    const testSettings = {
      // Set emulator URL
      emulatorURL: 'http://localhost:9099',
    };
    const root = document.createElement('div');
    document.body.appendChild(root);
    const emulatorConnectorStub = sinon.stub();
    const userStub = {} as User;
    const authStub = {
      onAuthStateChanged: (callback: (user?: User) => void) =>
        callback(userStub),
    } as Auth;

    render(
      html`
        <fake-parent-element>
          <webstatus-firebase-auth-service .settings=${testSettings}>
          </webstatus-firebase-auth-service>
        </fake-parent-element>
      `,
      root
    );
    const parentElement = root.querySelector<FakeParentElement>(
      'fake-parent-element'
    );
    assert.exists(parentElement);
    parentElement.app = new FakeFirebaseApp();
    const component = root.querySelector<WebstatusFirebaseAuthService>(
      'webstatus-firebase-auth-service'
    );

    assert.exists(component);
    component!.emulatorConnector = emulatorConnectorStub;
    component.authInitializer = () => authStub;

    await component.updateComplete;

    assert.isTrue(emulatorConnectorStub.calledOnce);
    expect(emulatorConnectorStub).to.have.been.calledWith(
      authStub,
      'http://localhost:9099'
    );
  });
});
