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

import {expect, fixture, html} from '@open-wc/testing';
import {WebstatusLogin} from '../webstatus-login.js';
import '../webstatus-login.js';
import {AuthConfig} from '../../contexts/firebase-auth-context.js';
import {Auth, AuthProvider, User} from 'firebase/auth';
import sinon from 'sinon';

describe('webstatus-login', () => {
  it('renders nothing when firebaseAuthConfig is undefined', async () => {
    const component = await fixture<WebstatusLogin>(
      html`<webstatus-login></webstatus-login>`
    );
    expect(component.shadowRoot?.textContent?.trim()).to.equal('');
  });

  it('renders the login button when user is undefined', async () => {
    const authConfigStub: AuthConfig = {
      auth: {} as Auth,
      signIn: sinon.stub(),
      provider: {} as AuthProvider,
      icon: 'github',
    };

    const component = await fixture<WebstatusLogin>(
      html`<webstatus-login
        .firebaseAuthConfig=${authConfigStub}
      ></webstatus-login>`
    );

    expect(
      component.shadowRoot?.querySelector('sl-button')?.textContent?.trim()
    ).to.equal('Log in');
  });

  it('triggers signIn when login button is clicked', async () => {
    const authConfigStub: AuthConfig = {
      auth: {} as Auth,
      signIn: sinon.stub().resolves(),
      provider: {} as AuthProvider,
      icon: 'github',
    };
    const component = await fixture<WebstatusLogin>(
      html`<webstatus-login
        .firebaseAuthConfig=${authConfigStub}
      ></webstatus-login>`
    );

    const loginButton = component.shadowRoot?.querySelector('sl-button');
    loginButton?.click();
    expect(authConfigStub.signIn).to.have.been.calledOnce;
  });

  it('renders the authenticated button when user is defined', async () => {
    const authConfigStub: AuthConfig = {
      auth: {} as Auth,
      signIn: sinon.stub(),
      provider: {} as AuthProvider,
      icon: 'github',
    };
    const userStub: User = {
      email: 'test@example.com',
      // ... other necessary User properties
    } as User;

    const component = await fixture<WebstatusLogin>(html`
      <webstatus-login
        .user=${userStub}
        .firebaseAuthConfig=${authConfigStub}
      ></webstatus-login>
    `);

    expect(
      component.shadowRoot?.querySelector('sl-dropdown')?.textContent?.trim()
    ).to.include('test@example.com');
    expect(
      component.shadowRoot?.querySelector('sl-menu-item')?.textContent?.trim()
    ).to.equal('Sign out');
  });

  it('triggers signOut when logout button is clicked', async () => {
    const authStub = {
      signOut: sinon.stub().resolves(),
    } as unknown as Auth;
    const authConfigStub: AuthConfig = {
      auth: authStub,
      signIn: sinon.stub(),
      provider: {} as AuthProvider,
      icon: 'github',
    };
    const userStub: User = {
      email: 'test@example.com',
    } as User;

    const component = await fixture<WebstatusLogin>(html`
      <webstatus-login
        .user=${userStub}
        .firebaseAuthConfig=${authConfigStub}
      ></webstatus-login>
    `);

    const logoutButton = component.shadowRoot?.querySelector('sl-menu-item');
    logoutButton?.click();

    expect(authStub.signOut).to.have.been.calledOnce;
  });
});
