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

import {html, TemplateResult} from 'lit';
import {type components} from 'webstatus.dev-backend';
import './webhook-config-form.js';

import {ChannelConfigUpdate} from './channel-config-types.js';

type ChannelType = components['schemas']['NotificationChannel']['type'];
type ChannelResponse = components['schemas']['NotificationChannelResponse'];

export const ChannelConfigRegistry = {
  renderConfig(
    type: ChannelType,
    channel: ChannelResponse | undefined,
    onUpdate: (update: ChannelConfigUpdate) => void,
  ): TemplateResult {
    switch (type) {
      case 'webhook':
        return html`<webhook-config-form
          class="config-form"
          .channel=${channel}
          @change=${(e: CustomEvent<ChannelConfigUpdate>) => onUpdate(e.detail)}
        ></webhook-config-form>`;
      case 'email':
        return html`<div>
          Email:
          ${channel?.config.type === 'email' ? channel.config.address : ''}
          (Verified)
        </div>`;
      default:
        return html`<p>Unsupported channel type: ${type}</p>`;
    }
  },
};
