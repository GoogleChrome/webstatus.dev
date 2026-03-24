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
import './webstatus-manage-notification-channel-dialog.js';

type NotificationChannelResponse =
  components['schemas']['NotificationChannelResponse'];

@customElement('webstatus-notification-webhook-channels')
export class WebstatusNotificationWebhookChannels extends LitElement {
  @consume({context: apiClientContext})
  @state()
  apiClient!: APIClient;

  @consume({context: firebaseUserContext, subscribe: true})
  @state()
  userContext: UserContext | null | undefined;

  @property({type: Array})
  channels: NotificationChannelResponse[] = [];

  @state()
  private _isManageDialogOpen = false;

  @state()
  private _manageDialogMode: 'create' | 'edit' = 'create';

  @state()
  private _selectedChannel?: NotificationChannelResponse;

  @state()
  private _isSaving = false;

  @state()
  private _isDeletingId: string | null = null;

  @state()
  private _isDeleteDialogOpen = false;

  @state()
  private _channelToDelete?: NotificationChannelResponse;

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
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
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

  private _openCreateDialog() {
    this._manageDialogMode = 'create';
    this._selectedChannel = undefined;
    this._isManageDialogOpen = true;
  }

  private _openEditDialog(channel: NotificationChannelResponse) {
    this._manageDialogMode = 'edit';
    this._selectedChannel = channel;
    this._isManageDialogOpen = true;
  }

  private _closeManageDialog() {
    this._isManageDialogOpen = false;
  }

  private async _handleSave(e: CustomEvent) {
    if (!this.userContext) {
      return;
    }
    const {mode, channelId, updates} = e.detail;

    this._isSaving = true;
    try {
      const token = await this.userContext.user.getIdToken();
      if (mode === 'create') {
        const resp = await this.apiClient.createNotificationChannel(
          token,
          updates,
        );
        void toast(`Created webhook channel "${resp.name}".`, 'success');
      } else {
        const resp = await this.apiClient.updateNotificationChannel(
          token,
          channelId,
          updates,
        );
        void toast(`Updated webhook channel "${resp.name}".`, 'success');
      }
      this.dispatchEvent(
        new CustomEvent('channel-changed', {bubbles: true, composed: true}),
      );
      this._closeManageDialog();
    } catch (e) {
      void toast(
        `Failed to ${mode} webhook channel. Please try again.`,
        'danger',
        'exclamation-triangle',
      );
      console.error(`Failed to ${mode} webhook channel`, e);
    } finally {
      this._isSaving = false;
    }
  }

  private _handleDeleteClick(channel: NotificationChannelResponse) {
    this._channelToDelete = channel;
    this._isDeleteDialogOpen = true;
  }

  private _closeDeleteDialog() {
    this._isDeleteDialogOpen = false;
    this._channelToDelete = undefined;
  }

  private async _confirmDelete() {
    if (!this.userContext || !this._channelToDelete) return;

    this._isDeletingId = this._channelToDelete.id;
    try {
      const token = await this.userContext.user.getIdToken();
      await this.apiClient.deleteNotificationChannel(
        token,
        this._channelToDelete.id,
      );
      this.dispatchEvent(
        new CustomEvent('channel-changed', {bubbles: true, composed: true}),
      );
      this._closeDeleteDialog();
    } catch (e) {
      void toast(
        'Failed to delete webhook channel. Please try again.',
        'danger',
        'exclamation-triangle',
      );
      console.error('Failed to delete webhook channel', e);
    } finally {
      this._isDeletingId = null;
    }
  }

  render() {
    return html`
      <webstatus-notification-panel>
        <sl-icon name="webhook" slot="icon"></sl-icon>
        <span slot="title">Webhook</span>
        <div slot="actions">
          <sl-button size="small" @click=${this._openCreateDialog}>
            <sl-icon name="plus-lg" slot="prefix"></sl-icon>
            Create Webhook channel
          </sl-button>
        </div>
        <div slot="content">
          ${this.channels.length === 0
            ? html`<p class="empty-message">No webhook channels configured.</p>`
            : repeat(
                this.channels,
                channel => channel.id,
                channel => html`
                  <div class="channel-item">
                    <div class="channel-info">
                      <span class="name">${channel.name}</span>
                      <span class="url"
                        >${channel.config.type === 'webhook'
                          ? channel.config.url
                          : ''}</span
                      >
                    </div>
                    <div class="actions">
                      ${channel.status === 'enabled'
                        ? html`<sl-badge variant="success" pill
                            >Enabled</sl-badge
                          >`
                        : html`<sl-badge variant="danger" pill
                            >Disabled</sl-badge
                          >`}
                      <sl-button
                        label="Edit"
                        aria-label="Edit"
                        size="small"
                        variant="neutral"
                        outline
                        circle
                        @click=${() => this._openEditDialog(channel)}
                      >
                        <sl-icon name="pencil"></sl-icon>
                      </sl-button>
                      <sl-button
                        label="Delete"
                        aria-label="Delete"
                        size="small"
                        variant="danger"
                        outline
                        circle
                        .loading=${this._isDeletingId === channel.id}
                        @click=${() => this._handleDeleteClick(channel)}
                      >
                        <sl-icon name="trash"></sl-icon>
                      </sl-button>
                    </div>
                  </div>
                `,
              )}
        </div>
      </webstatus-notification-panel>

      <webstatus-manage-notification-channel-dialog
        .open=${this._isManageDialogOpen}
        .mode=${this._manageDialogMode}
        .channel=${this._selectedChannel}
        .loading=${this._isSaving}
        @sl-hide=${this._closeManageDialog}
        @save=${this._handleSave}
      >
      </webstatus-manage-notification-channel-dialog>

      <sl-dialog
        .open=${this._isDeleteDialogOpen}
        label="Delete Webhook Channel"
        @sl-hide=${this._closeDeleteDialog}
      >
        <p>Are you sure you want to delete this webhook channel?</p>
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
          .loading=${this._isDeletingId === this._channelToDelete?.id}
          @click=${this._confirmDelete}
        >
          Delete
        </sl-button>
      </sl-dialog>
    `;
  }
}
