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

import {LitElement, html, TemplateResult} from 'lit';
import {customElement, property, state} from 'lit/decorators.js';
import {consume} from '@lit/context';
import {
  UserContext,
  firebaseUserContext,
} from '../contexts/firebase-user-context.js';
import {toast} from '../utils/toast.js';

import {
  SubscriptionDeleteErrorEvent,
  SubscriptionSaveErrorEvent,
} from './webstatus-manage-subscriptions-dialog.js';

import './webstatus-manage-subscriptions-dialog.js';

@customElement('webstatus-subscribe-button')
export class SubscribeButton extends LitElement {
  @consume({context: firebaseUserContext, subscribe: true})
  @state()
  userContext: UserContext | null | undefined;

  @property({type: String})
  savedSearchId = '';

  @property({attribute: false})
  toaster = toast;

  @state()
  private _isSubscriptionDialogOpen = false;

  render(): TemplateResult {
    if (!this.userContext || !this.savedSearchId) {
      return html``;
    }

    return html`
      <sl-button
        variant="primary"
        @click=${() => (this._isSubscriptionDialogOpen = true)}
      >
        <sl-icon slot="prefix" name="bell"></sl-icon>
        Subscribe
      </sl-button>

      <webstatus-manage-subscriptions-dialog
        ?open=${this._isSubscriptionDialogOpen}
        .savedSearchId=${this.savedSearchId}
        @subscription-dialog-close=${() =>
          (this._isSubscriptionDialogOpen = false)}
        @subscription-save-success=${this._handleSubscriptionSaveSuccess}
        @subscription-save-error=${this._handleSubscriptionSaveError}
        @subscription-delete-success=${this._handleSubscriptionDeleteSuccess}
        @subscription-delete-error=${this._handleSubscriptionDeleteError}
      >
      </webstatus-manage-subscriptions-dialog>
    `;
  }

  private _handleSubscriptionSaveSuccess() {
    this._isSubscriptionDialogOpen = false;
    void this.toaster('Subscription saved!', 'success');
  }

  private _handleSubscriptionSaveError(e: SubscriptionSaveErrorEvent) {
    void this.toaster(
      `Error saving subscription: ${e.detail.message}`,
      'danger',
    );
  }

  private _handleSubscriptionDeleteSuccess() {
    this._isSubscriptionDialogOpen = false;
    void this.toaster('Subscription deleted!', 'success');
  }

  private _handleSubscriptionDeleteError(e: SubscriptionDeleteErrorEvent) {
    void this.toaster(
      `Error deleting subscription: ${e.detail.message}`,
      'danger',
    );
  }
}
