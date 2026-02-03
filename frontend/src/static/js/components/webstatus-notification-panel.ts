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

@customElement('webstatus-notification-panel')
export class WebstatusNotificationPanel extends LitElement {
  static styles = css`
    .card {
      border: 1px solid #e4e4e7;
      border-radius: 4px;
      overflow: hidden;
    }

    .card-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 8px 16px;
    }

    .card-header .title {
      display: flex;
      align-items: center;
      gap: 8px;
      font-weight: bold;
      font-size: 16px;
    }

    .card-body {
      padding: 0 20px 20px 20px;
    }

    .loading-skeleton {
      display: flex;
      flex-direction: column;
      gap: 10px;
      padding: 16px;
    }
  `;

  @property({type: Boolean})
  loading = false;

  render() {
    return html`
      <div class="card">
        <div class="card-header">
          <div class="title">
            <slot name="icon"></slot>
            <slot name="title"></slot>
          </div>
          <div class="actions">
            <slot name="actions"></slot>
          </div>
        </div>
        <div class="card-body">
          ${this.loading
            ? html`
                <div class="loading-skeleton">
                  <sl-skeleton effect="sheen"></sl-skeleton>
                  <sl-skeleton effect="sheen"></sl-skeleton>
                </div>
              `
            : html`<slot name="content"></slot>`}
        </div>
      </div>
    `;
  }
}
