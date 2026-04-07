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

import {css, html, PropertyValues} from 'lit';
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
type UpdateRequest = components['schemas']['UpdateNotificationChannelRequest'];
type UpdateMask = UpdateRequest['update_mask'][number];

@customElement('rss-config-form')
export class RssConfigForm
  extends ChannelConfigForm
  implements ChannelConfigComponent
{
  @property({type: Object}) channel?: ChannelResponse;

  @state() private _pendingName?: string;

  @query('#rss-name')
  private _nameInput!: SlInput;

  static styles = [
    SHARED_STYLES,
    css`
      :host {
        display: flex;
        flex-direction: column;
        gap: 16px;
      }
    `,
  ];

  isDirty(): boolean {
    const currentName = this.channel?.name ?? '';
    return this._pendingName !== undefined && this._pendingName !== currentName;
  }

  validate(): boolean {
    return this._nameInput.reportValidity();
  }

  getUpdate(): ChannelConfigUpdate {
    const updates: Partial<UpdateRequest> = {};
    const mask: UpdateMask[] = [];

    const nameToUse = this._nameInput ? this._nameInput.value : '';

    updates.name = nameToUse;
    mask.push('name');

    // For RSS, we signal the DB to allocate type: 'rss' on creation.
    if (!this.channel) {
      updates.config = {type: 'rss'};
      mask.push('config');
    }

    return {updates, mask};
  }

  private _handleInput() {
    this._pendingName = this._nameInput ? this._nameInput.value : undefined;
    this.dispatchEvent(
      new CustomEvent('change', {
        detail: this.getUpdate(),
        bubbles: true,
        composed: true,
      }),
    );
  }

  updated(changedProperties: PropertyValues<this>) {
    if (changedProperties.has('channel') && !this.channel) {
      this._pendingName = undefined;
    }
  }

  render() {
    const currentName = this.channel?.name ?? '';

    return html`
      <sl-input
        id="rss-name"
        label="Name"
        placeholder="e.g. My RSS Feed"
        .value=${this._pendingName ?? currentName}
        @sl-input=${this._handleInput}
        required
      ></sl-input>
      <div>RSS Feed (No additional configuration needed)</div>
    `;
  }
}
