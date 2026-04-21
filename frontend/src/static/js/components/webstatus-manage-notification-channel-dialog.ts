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

import {LitElement, PropertyValues, css, html} from 'lit';
import {customElement, property, state, query} from 'lit/decorators.js';
import {components} from 'webstatus.dev-backend';
import {SHARED_STYLES} from '../css/shared-css.js';
import {
  ChannelConfigUpdate,
  ChannelConfigComponent,
} from './channel-config-types.js';
import {ChannelConfigRegistry} from './channel-config-registry.js';

type ChannelType = components['schemas']['NotificationChannel']['type'];
type ChannelResponse = components['schemas']['NotificationChannelResponse'];

@customElement('webstatus-manage-notification-channel-dialog')
export class ManageNotificationChannelDialog extends LitElement {
  @property({type: Boolean}) open = false;
  @property() mode: 'create' | 'edit' = 'create';
  @property() type: ChannelType = 'webhook';
  @property({type: Object}) channel?: ChannelResponse;
  @property({type: Boolean}) loading = false;

  @state() private _pendingUpdate?: ChannelConfigUpdate;

  @query('.config-form')
  private _configForm!: ChannelConfigComponent;

  static styles = [
    SHARED_STYLES,
    css`
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

  private _handleHide() {
    this.dispatchEvent(new CustomEvent('sl-hide'));
  }

  private _handleSave() {
    if (this._configForm && !this._configForm.validate()) return;

    const update = this._configForm.getUpdate();
    const detail = {
      mode: this.mode,
      channelId: this.channel?.id,
      ...update,
    };
    this.dispatchEvent(
      new CustomEvent('save', {
        detail: detail,
      }),
    );
  }

  render() {
    return html`
      <sl-dialog
        label="${this.mode === 'create' ? 'Create' : 'Edit'} ${this.type
          .charAt(0)
          .toUpperCase() + this.type.slice(1)} Channel"
        .open=${this.open}
        @sl-hide=${this._handleHide}
      >
        <div class="dialog-body">
          ${ChannelConfigRegistry.renderConfig(
            this.mode === 'edit' ? this.channel!.type : this.type,
            this.channel,
            u => (this._pendingUpdate = u),
          )}
        </div>
        <sl-button
          slot="footer"
          variant="primary"
          @click=${this._handleSave}
          .loading=${this.loading}
          .disabled=${this.mode === 'edit' &&
          (!this._pendingUpdate ||
            Object.keys(this._pendingUpdate.updates).length === 0)}
        >
          ${this.mode === 'create' ? 'Create' : 'Save'}
        </sl-button>
        <sl-button slot="footer" @click=${this._handleHide}>Cancel</sl-button>
      </sl-dialog>
    `;
  }

  updated(changedProperties: PropertyValues<this>) {
    if (changedProperties.has('open') && !this.open) {
      this._pendingUpdate = undefined;
    }
  }
}
