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

import {LitElement, TemplateResult, css, html} from 'lit';
import {customElement, property, state} from 'lit/decorators.js';
import {consume} from '@lit/context';
import {
  firebaseAuthContext,
  AuthConfig,
} from '../contexts/firebase-auth-context.js';
import {toast} from '../utils/toast.js';
import {SHARED_STYLES} from '../css/shared-css.js';

@customElement('webstatus-login-prompt-dialog')
export class WebstatusLoginPromptDialog extends LitElement {
  @property({type: Boolean})
  open = false;

  @property({type: String})
  savedSearchName = '';

  @consume({context: firebaseAuthContext, subscribe: true})
  @state()
  firebaseAuthConfig?: AuthConfig;

  static styles = [
    SHARED_STYLES,
    css`
      .content {
        display: flex;
        flex-direction: column;
        gap: var(--content-padding);
      }
      .footer {
        display: flex;
        justify-content: flex-end;
      }
    `,
  ];

  async _handleLogin() {
    if (this.firebaseAuthConfig) {
      try {
        await this.firebaseAuthConfig.signIn();
        this.dispatchEvent(
          new CustomEvent('login-success', {bubbles: true, composed: true}),
        );
        this.open = false;
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : 'unknown';
        await toast(
          `Failed to login: ${errorMessage}`,
          'danger',
          'exclamation-triangle',
        );
      }
    }
  }

  _handleClose() {
    this.open = false;
    this.dispatchEvent(
      new CustomEvent('prompt-close', {bubbles: true, composed: true}),
    );
  }

  render(): TemplateResult {
    return html`
      <sl-dialog
        .open=${this.open}
        label="Log in to subscribe"
        @sl-after-hide=${this._handleClose}
      >
        <div class="content">
          <p>
            You need an account to subscribe to
            <strong>${this.savedSearchName}</strong> and receive the latest
            updates.
          </p>
        </div>
        <div slot="footer" class="footer">
          <sl-button variant="primary" @click=${this._handleLogin}>
            <sl-icon slot="prefix" name="github"></sl-icon>
            Log in
          </sl-button>
        </div>
      </sl-dialog>
    `;
  }
}
