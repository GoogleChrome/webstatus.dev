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

import {css, html} from 'lit';
import {customElement, property, state, query} from 'lit/decorators.js';
import {components} from 'webstatus.dev-backend';
import {SlInput} from '@shoelace-style/shoelace';
import {SHARED_STYLES} from '../css/shared-css.js';
import {
  ChannelConfigComponent,
  ChannelConfigUpdate,
  ChannelConfigForm,
} from './channel-config-types.js';

type ChannelResponse = components['schemas']['NotificationChannelResponse'];
type WebhookConfig = components['schemas']['WebhookConfig'];
type UpdateRequest = components['schemas']['UpdateNotificationChannelRequest'];
type UpdateMask = UpdateRequest['update_mask'][number];

@customElement('webhook-config-form')
export class WebhookConfigForm
  extends ChannelConfigForm
  implements ChannelConfigComponent
{
  @property({type: Object}) channel?: ChannelResponse;

  protected get config(): WebhookConfig | undefined {
    const config = this.channel?.config;
    return config?.type === 'webhook' ? config : undefined;
  }

  @state() private _pendingName?: string;
  @state() private _pendingUrl?: string;

  @query('#webhook-name')
  private _nameInput!: SlInput;

  @query('#webhook-url')
  private _urlInput!: SlInput;

  static styles = [
    SHARED_STYLES,
    css`
      :host {
        display: flex;
        flex-direction: column;
        gap: 16px;
      }
      .help-text {
        font-size: 12px;
        color: var(--unimportant-text-color);
        margin: 0;
      }
    `,
  ];

  isDirty(): boolean {
    const currentName = this.channel?.name ?? '';
    const currentUrl = this.config?.url ?? '';

    const nameChanged =
      this._pendingName !== undefined && this._pendingName !== currentName;
    const urlChanged =
      this._pendingUrl !== undefined && this._pendingUrl !== currentUrl;

    return nameChanged || urlChanged;
  }

  validate(): boolean {
    return this._nameInput.reportValidity() && this._urlInput.reportValidity();
  }

  getUpdate(): ChannelConfigUpdate {
    const updates: Partial<UpdateRequest> = {};
    const mask: UpdateMask[] = [];

    const currentName = this.channel?.name ?? '';
    const nameToUse = this._pendingName ?? currentName;

    const currentUrl = this.config?.url ?? '';
    const urlToUse = this._pendingUrl ?? currentUrl;

    const nameChanged =
      this._pendingName !== undefined && this._pendingName !== currentName;
    if (nameChanged || !this.channel) {
      updates.name = nameToUse;
      mask.push('name');
    }

    const urlChanged =
      this._pendingUrl !== undefined && this._pendingUrl !== currentUrl;
    if (urlChanged || !this.channel) {
      // For config updates, we must send the entire config object as it's a 'oneOf' in OpenAPI.
      updates.config = {
        type: 'webhook',
        url: urlToUse,
      };
      mask.push('config');
    }

    return {updates, mask};
  }

  private _handleInput() {
    this._pendingName = this._nameInput.value;
    this._pendingUrl = this._urlInput.value;
    this.dispatchEvent(
      new CustomEvent('change', {
        detail: this.getUpdate(),
        bubbles: true,
        composed: true,
      }),
    );
  }

  render() {
    const currentName = this.channel?.name ?? '';
    const currentUrl = this.config?.url ?? '';

    return html`
      <sl-input
        id="webhook-name"
        label="Name"
        placeholder="e.g. My Slack Webhook"
        .value=${this._pendingName ?? currentName}
        @sl-input=${this._handleInput}
        required
      ></sl-input>
      <sl-input
        id="webhook-url"
        label="Slack Webhook URL"
        placeholder="https://hooks.slack.com/services/..."
        .value=${this._pendingUrl ?? currentUrl}
        @sl-input=${this._handleInput}
        required
        type="url"
      ></sl-input>
      <p class="help-text">
        Currently only Slack incoming webhooks are supported.
      </p>
    `;
  }
}
