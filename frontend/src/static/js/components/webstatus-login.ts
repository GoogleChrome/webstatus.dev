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
import {LitElement, type TemplateResult, html, nothing} from 'lit';
import {customElement, state} from 'lit/decorators.js';

import {
  FirebaseUser,
  firebaseUserContext,
} from '../contexts/firebase-user-context.js';
import {
  AuthConfig,
  firebaseAuthContext,
} from '../contexts/firebase-auth-context.js';
import {toast} from '../utils/toast.js';

@customElement('webstatus-login')
export class WebstatusLogin extends LitElement {
  @consume({context: firebaseAuthContext, subscribe: true})
  @state()
  firebaseAuthConfig?: AuthConfig;

  @consume({context: firebaseUserContext, subscribe: true})
  @state()
  user?: FirebaseUser;

  handleLogInClick(authConfig: AuthConfig) {
    if (this.user === undefined) {
      authConfig.signIn().catch(async error => {
        await toast(
          `Failed to login: ${error.message ?? 'unknown'}`,
          'danger',
          'exclamation-triangle',
        );
      });
      return;
    }
  }

  handleLogOutClick(authConfig: AuthConfig) {
    authConfig.auth.signOut().catch(async error => {
      await toast(
        `Failed to logout: ${error.message ?? 'unknown'}`,
        'danger',
        'exclamation-triangle',
      );
    });
  }

  renderLoginButton(authConfig: AuthConfig): TemplateResult {
    return html`
      <sl-button
        variant="default"
        @click=${() => this.handleLogInClick(authConfig)}
      >
        <sl-icon slot="prefix" name="${authConfig.icon}"></sl-icon>
        Log in
      </sl-button>
    `;
  }

  renderAuthenticatedButton(
    user: FirebaseUser,
    authConfig: AuthConfig
  ): TemplateResult {
    return html`
      <sl-dropdown>
        <sl-button slot="trigger" caret
          ><sl-icon slot="prefix" name="${authConfig.icon}"></sl-icon
          >${user?.gitHubUsername || user.email}</sl-button
        >
        <sl-menu>
          <sl-menu-item @click=${() => this.handleLogOutClick(authConfig)}
            >Sign out</sl-menu-item
          >
        </sl-menu>
      </sl-dropdown>
    `;
  }

  render(): TemplateResult {
    // Firebase auth not loaded yet.
    if (this.firebaseAuthConfig === undefined) {
      return html`${nothing}`;
    }

    // Unauthenticated user.
    if (this.user === undefined) {
      return this.renderLoginButton(this.firebaseAuthConfig);
    }

    // Authenticated user.
    return this.renderAuthenticatedButton(this.user, this.firebaseAuthConfig);
  }
}
