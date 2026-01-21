/**
 * Copyright 2024 Google LLC
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

import {expect, fixture, html, waitUntil} from '@open-wc/testing';
import {WebstatusLogin} from '../webstatus-login.js';
import '../webstatus-login.js';
import {AuthConfig} from '../../contexts/firebase-auth-context.js';
import {Auth, AuthProvider} from 'firebase/auth';
import type {User as FirebaseUser} from 'firebase/auth';
import {UserContext} from '../../contexts/firebase-user-context.js';
import sinon from 'sinon';

describe('webstatus-login', () => {
  const authConfigStub: AuthConfig = {
    auth: {signOut: sinon.stub().resolves()} as unknown as Auth,
    signIn: sinon.stub().resolves(),
    provider: {} as AuthProvider,
    icon: 'github',
  };

  const firebaseUserStub: FirebaseUser = {
    email: 'test@example.com',
  } as FirebaseUser;

  it('renders nothing when firebaseAuthConfig is undefined', async () => {
    const component = await fixture<WebstatusLogin>(
      html`<webstatus-login></webstatus-login>`,
    );
    expect(component.shadowRoot?.textContent?.trim()).to.equal('');
  });

  it('renders the login button when user is undefined', async () => {
    const component = await fixture<WebstatusLogin>(
      html`<webstatus-login
        .firebaseAuthConfig=${authConfigStub}
      ></webstatus-login>`,
    );

    const loginButton = component.shadowRoot?.querySelector('sl-button');
    expect(loginButton).to.exist;
    expect(loginButton?.textContent?.trim()).to.equal('Log in');
    expect(loginButton).to.not.have.attribute('disabled');
  });

  it('triggers signIn when login button is clicked', async () => {
    const component = await fixture<WebstatusLogin>(
      html`<webstatus-login
        .firebaseAuthConfig=${authConfigStub}
      ></webstatus-login>`,
    );

    const loginButton = component.shadowRoot?.querySelector('sl-button');
    loginButton?.click();
    expect(authConfigStub.signIn).to.have.been.calledOnce;
  });

  it('renders the authenticated button when user is defined and idle', async () => {
    const userStub: UserContext = {
      user: firebaseUserStub,
      syncState: 'idle',
    };

    const component = await fixture<WebstatusLogin>(html`
      <webstatus-login
        .userContext=${userStub}
        .firebaseAuthConfig=${authConfigStub}
      ></webstatus-login>
    `);

    const dropdown = component.shadowRoot?.querySelector('sl-dropdown');
    expect(dropdown).to.exist;
    expect(dropdown?.textContent?.trim()).to.include('test@example.com');

    const button = dropdown?.querySelector('sl-button');
    expect(button).to.not.have.attribute('disabled');
    expect(button?.querySelector('sl-icon')).to.exist;
    expect(button?.querySelector('sl-spinner')).to.not.exist;

    const menuItem = component.shadowRoot?.querySelector('sl-menu-item');
    expect(menuItem?.textContent?.trim()).to.equal('Sign out');
  });

  it('renders syncing state correctly', async () => {
    const userStub: UserContext = {
      user: firebaseUserStub,
      syncState: 'syncing',
    };

    const component = await fixture<WebstatusLogin>(html`
      <webstatus-login
        .userContext=${userStub}
        .firebaseAuthConfig=${authConfigStub}
      ></webstatus-login>
    `);

    const button = component.shadowRoot?.querySelector('sl-button');
    expect(button).to.have.attribute('loading');
    expect(button).to.have.attribute('disabled');
  });

  it('renders error state correctly', async () => {
    const userStub: UserContext = {
      user: firebaseUserStub,
      syncState: 'error',
    };

    const component = await fixture<WebstatusLogin>(html`
      <webstatus-login
        .userContext=${userStub}
        .firebaseAuthConfig=${authConfigStub}
      ></webstatus-login>
    `);

    const button = component.shadowRoot?.querySelector('sl-button');
    expect(button).to.not.have.attribute('disabled');

    // Wait until the error icon exists and its 'name' attribute is set.
    await waitUntil(
      () =>
        component.shadowRoot
          ?.querySelector('sl-icon.error-icon')
          ?.getAttribute('name') === 'exclamation-triangle',
      'Error icon name attribute did not render in time',
    );
    const errorIcon = component.shadowRoot?.querySelector('sl-icon.error-icon');
    expect(errorIcon).to.exist;
    expect(errorIcon?.getAttribute('name')).to.equal('exclamation-triangle');
  });

  it('triggers signOut when logout button is clicked', async () => {
    const userStub: UserContext = {
      user: firebaseUserStub,
      syncState: 'idle',
    };

    const component = await fixture<WebstatusLogin>(html`
      <webstatus-login
        .userContext=${userStub}
        .firebaseAuthConfig=${authConfigStub}
      ></webstatus-login>
    `);

    const logoutButton = component.shadowRoot?.querySelector('sl-menu-item');
    logoutButton?.click();

    expect(authConfigStub.auth.signOut).to.have.been.calledOnce;
  });
});
