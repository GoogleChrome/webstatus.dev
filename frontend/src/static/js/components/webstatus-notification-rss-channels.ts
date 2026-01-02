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
import {customElement} from 'lit/decorators.js';
import './webstatus-notification-panel.js';

@customElement('webstatus-notification-rss-channels')
export class WebstatusNotificationRssChannels extends LitElement {
  static styles = css`
    .card-body {
      padding: 20px;
      color: #71717a;
    }
  `;

  render() {
    return html`
      <webstatus-notification-panel>
        <sl-icon name="rss" slot="icon"></sl-icon>
        <span slot="title">RSS</span>
        <div slot="actions">
          <sl-button size="small" disabled
            ><sl-icon name="plus-lg" slot="prefix"></sl-icon>Create RSS
            channel</sl-button
          >
        </div>
        <div slot="content">
          <p>Coming soon</p>
        </div>
      </webstatus-notification-panel>
    `;
  }
}
