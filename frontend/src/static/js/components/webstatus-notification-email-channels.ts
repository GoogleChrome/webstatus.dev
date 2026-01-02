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
import {customElement, property} from 'lit/decorators.js';
import {repeat} from 'lit/directives/repeat.js';
import {components} from 'webstatus.dev-backend';
import './webstatus-notification-panel.js';

type NotificationChannelResponse =
  components['schemas']['NotificationChannelResponse'];

@customElement('webstatus-notification-email-channels')
export class WebstatusNotificationEmailChannels extends LitElement {
  static styles = css`
    .channel-item {
      background-color: #f9f9f9;
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 8px 16px;
      border-bottom: 1px solid #e4e4e7;
    }

    .channel-item:last-child {
      border-bottom: none;
    }

    .channel-info {
      display: flex;
      flex-direction: column;
    }

    .channel-info .name {
      font-size: 14px;
    }

    .info-icon-button {
      font-size: 1.2rem;
    }
  `;

  @property({type: Array})
  channels: NotificationChannelResponse[] = [];

  @property({type: Boolean})
  loading = false;

  render() {
    return html`
      <webstatus-notification-panel .loading=${this.loading}>
        <sl-icon name="envelope" slot="icon"></sl-icon>
        <span slot="title">Email</span>
        <div slot="actions">
          <sl-tooltip
            content="The emails here are based on the verified emails from your GitHub account. Please complete verification, then logout of webstatus.dev and re-login to sync."
          >
            <sl-icon-button
              name="info-circle"
              label="Information"
              class="info-icon-button"
            ></sl-icon-button>
          </sl-tooltip>
        </div>
        <div slot="content">
          ${repeat(
            this.channels,
            channel => channel.id,
            channel => html`
              <div class="channel-item">
                <div class="channel-info">
                  <span class="name">${channel.value}</span>
                </div>
                ${channel.status === 'enabled'
                  ? html`<sl-badge variant="success" pill>Enabled</sl-badge>`
                  : html`<sl-badge variant="danger" pill>Disabled</sl-badge>`}
              </div>
            `,
          )}
        </div>
      </webstatus-notification-panel>
    `;
  }
}
