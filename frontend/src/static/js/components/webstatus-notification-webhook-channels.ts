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
import {customElement, property, state, query} from 'lit/decorators.js';
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
import {SlInput} from '@shoelace-style/shoelace';
import {toast} from '../utils/toast.js';
import './webstatus-notification-panel.js';

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
  private _isCreateDialogOpen = false;

  @state()
  private _isSaving = false;

  @state()
  private _isDeletingId: string | null = null;

  @query('#webhook-name')
  private _nameInput!: SlInput;

  @query('#webhook-url')
  private _urlInput!: SlInput;

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

      sl-dialog::part(panel) {
        width: min(90vw, 500px);
      }

      .dialog-body {
        display: flex;
        flex-direction: column;
        gap: 16px;
      }
    `,
  ];

  private _openCreateDialog() {
    this._isCreateDialogOpen = true;
  }

  private _closeCreateDialog() {
    this._isCreateDialogOpen = false;
    this._nameInput.value = '';
    this._urlInput.value = '';
  }

  private async _handleCreate() {
    if (!this.userContext) return;
    const name = this._nameInput.value;
    const url = this._urlInput.value;

    if (!name || !url) {
      return;
    }

    this._isSaving = true;
    try {
      const token = await this.userContext.user.getIdToken();
      await this.apiClient.createNotificationChannel(token, {
        name,
        config: {
          type: 'webhook',
          url,
        },
      });
      this.dispatchEvent(
        new CustomEvent('channel-changed', {bubbles: true, composed: true}),
      );
      this._closeCreateDialog();
    } catch (e) {
      void toast(
        'Failed to create webhook channel. Please try again.',
        'danger',
        'exclamation-triangle',
      );
      console.error('Failed to create webhook channel', e);
    } finally {
      this._isSaving = false;
    }
  }

  private async _handleDelete(channelId: string) {
    if (!this.userContext) return;
    if (!confirm('Are you sure you want to delete this webhook channel?'))
      return;

    this._isDeletingId = channelId;
    try {
      const token = await this.userContext.user.getIdToken();
      await this.apiClient.deleteNotificationChannel(token, channelId);
      this.dispatchEvent(
        new CustomEvent('channel-changed', {bubbles: true, composed: true}),
      );
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
            ? html`<p
                style="padding: 16px; color: var(--unimportant-text-color);"
              >
                No webhook channels configured.
              </p>`
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
                        size="small"
                        variant="danger"
                        outline
                        circle
                        .loading=${this._isDeletingId === channel.id}
                        @click=${() => this._handleDelete(channel.id)}
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
        label="Create Webhook Channel"
        .open=${this._isCreateDialogOpen}
        @sl-hide=${this._closeCreateDialog}
      >
        <div class="dialog-body">
          <sl-input
            id="webhook-name"
            label="Name"
            placeholder="e.g. My Slack Webhook"
            required
          ></sl-input>
          <sl-input
            id="webhook-url"
            label="Slack Webhook URL"
            placeholder="https://hooks.slack.com/services/..."
            required
            type="url"
          ></sl-input>
          <p style="font-size: 12px; color: var(--unimportant-text-color);">
            Currently only Slack incoming webhooks are supported.
          </p>
        </div>
        <sl-button
          slot="footer"
          variant="primary"
          @click=${this._handleCreate}
          .loading=${this._isSaving}
        >
          Create
        </sl-button>
        <sl-button slot="footer" @click=${this._closeCreateDialog}>
          Cancel
        </sl-button>
      </sl-dialog>
    `;
  }
}
