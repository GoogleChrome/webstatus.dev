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

import {LitElement, css, html} from 'lit';
import {customElement, property, state} from 'lit/decorators.js';
import {repeat} from 'lit/directives/repeat.js';
import {consume} from '@lit/context';
import {apiClientContext} from '../contexts/api-client-context.js';
import {APIClient} from '../api/client.js';
import {
  UserContext,
  firebaseUserContext,
} from '../contexts/firebase-user-context.js';
import {components} from 'webstatus.dev-backend';
import {SHARED_STYLES} from '../css/shared-css.js';
import {toast} from '../utils/toast.js';
import './webstatus-notification-panel.js';

type SubscriptionResponse = components['schemas']['SubscriptionResponse'];

@customElement('webstatus-notification-rss-channels')
export class WebstatusNotificationRssChannels extends LitElement {
  @consume({context: apiClientContext})
  @state()
  apiClient!: APIClient;

  @consume({context: firebaseUserContext, subscribe: true})
  @state()
  userContext: UserContext | null | undefined;

  @property({type: Array})
  subscriptions: SubscriptionResponse[] = [];

  @state()
  private _isDeletingId: string | null = null;

  @state()
  private _isDeleteDialogOpen = false;

  @state()
  private _subscriptionToDelete?: SubscriptionResponse;

  static styles = [
    SHARED_STYLES,
    css`
      .channel-item {
        color: var(--default-color);
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 8px 16px;
        border-bottom: 1px solid var(--border-color);
      }

      .channel-item:last-child {
        border-bottom: none;
      }

      .channel-info {
        display: flex;
        flex-direction: column;
        overflow: hidden;
      }

      .channel-info .name {
        font-size: 14px;
        font-weight: bold;
      }

      .channel-info .url {
        font-size: 12px;
        color: var(--unimportant-text-color);
        margin-top: 4px;
        word-break: break-all;
      }

      .actions {
        display: flex;
        align-items: center;
        gap: 8px;
      }

      .empty-message {
        padding: 16px;
        color: var(--unimportant-text-color);
      }
    `,
  ];

  private _handleDeleteClick(sub: SubscriptionResponse) {
    this._subscriptionToDelete = sub;
    this._isDeleteDialogOpen = true;
  }

  private _closeDeleteDialog() {
    this._isDeleteDialogOpen = false;
    this._subscriptionToDelete = undefined;
  }

  private async _confirmDelete() {
    if (!this.userContext || !this._subscriptionToDelete) return;

    this._isDeletingId = this._subscriptionToDelete.id;
    try {
      const token = await this.userContext.user.getIdToken();
      await this.apiClient.deleteSubscription(
        this._subscriptionToDelete.id,
        token,
      );
      this.dispatchEvent(
        new CustomEvent('subscription-changed', {
          bubbles: true,
          composed: true,
        }),
      );
      this._closeDeleteDialog();
      void toast('Deleted RSS feed subscription.', 'success');
    } catch (e) {
      void toast(
        'Failed to delete RSS feed subscription. Please try again.',
        'danger',
        'exclamation-triangle',
      );
      console.error('Failed to delete RSS subscription', e);
    } finally {
      this._isDeletingId = null;
    }
  }

  render() {
    return html`
      <webstatus-notification-panel>
        <sl-icon name="rss" slot="icon"></sl-icon>
        <span slot="title">RSS Subscriptions</span>
        <div slot="content">
          ${this.subscriptions.length === 0
            ? html`<p class="empty-message">No RSS feeds configured.</p>`
            : repeat(
                this.subscriptions,
                sub => sub.id,
                sub => html`
                  <div class="channel-item">
                    <div class="channel-info">
                      <span class="name">${sub.subscribable.name}</span>
                      <span class="url"
                        >${new URL(
                          `${this.apiClient.getBaseUrl()}/v1/subscriptions/${sub.id}/rss`,
                          window.location.origin,
                        ).href}</span
                      >
                    </div>
                    <div class="actions">
                      <sl-button
                        label="Delete"
                        aria-label="Delete"
                        size="small"
                        variant="danger"
                        outline
                        circle
                        .loading=${this._isDeletingId === sub.id}
                        @click=${() => this._handleDeleteClick(sub)}
                      >
                        <sl-icon name="trash"></sl-icon>
                      </sl-button>
                    </div>
                  </div>
                `,
              )}
        </div>
      </webstatus-notification-panel>

      <sl-dialog
        .open=${this._isDeleteDialogOpen}
        label="Delete RSS Feed"
        @sl-hide=${this._closeDeleteDialog}
      >
        <p>
          Are you sure you want to delete the RSS feed for
          "${this._subscriptionToDelete?.subscribable.name}"?
        </p>
        <sl-button
          slot="footer"
          variant="default"
          @click=${this._closeDeleteDialog}
        >
          Cancel
        </sl-button>
        <sl-button
          slot="footer"
          variant="danger"
          .loading=${this._isDeletingId === this._subscriptionToDelete?.id}
          @click=${this._confirmDelete}
        >
          Delete
        </sl-button>
      </sl-dialog>
    `;
  }
}
