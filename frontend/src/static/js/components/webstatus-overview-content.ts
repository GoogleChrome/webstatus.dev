/**
 * Copyright 2023 Google LLC
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

import { LitElement, type TemplateResult, css, html } from 'lit'
import { customElement } from 'lit/decorators.js'

@customElement('webstatus-overview-content')
export class WebstatusOverviewContent extends LitElement {
  static styles = css`
    .data-table {
      width: 100%;
    }
    th {
      text-align: left;
    }
  `
  render(): TemplateResult {
    return html`
      <div class="main">
        <div class="filters">
          <select>
            <option value="all">All</option>
            <option value="active">Active</option>
            <option value="inactive">Inactive</option>
          </select>
        </div>
        <table class="data-table">
          <thead>
            <tr>
              <th>Feature Name</th>
              <th>Baseline Status</th>
              <th>WPT Scores</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td><a href="/features/1">Container queries</a></td>
              <td><img height="24" src="/public/img/cross.svg" /></td>
              <td>
                <img src="/public/img/chrome-dev_24x24.png" /> 100%
                <img src="/public/img/firefox-nightly_24x24.png" /> 100%
                <img src="/public/img/safari-preview_24x24.png" /> 100%
              </td>
            </tr>
            <tr>
              <td><a href="/features/1">Flexbox</a></td>
              <td><img height="24" src="/public/img/check.svg" /></td>
              <td>
                <img src="/public/img/chrome-dev_24x24.png" /> 100%
                <img src="/public/img/firefox-nightly_24x24.png" /> 100%
                <img src="/public/img/safari-preview_24x24.png" /> 100%
              </td>
            </tr>
            <tr>
              <td><a href="/features/1">Grid</a></td>
              <td><img height="24" src="/public/img/cross.svg" /></td>
              <td>
                <img src="/public/img/chrome-dev_24x24.png" /> 100%
                <img src="/public/img/firefox-nightly_24x24.png" /> 100%
                <img src="/public/img/safari-preview_24x24.png" /> 100%
              </td>
            </tr>
          </tbody>
        </table>
        <button>Modify Columns</button>
      </div>
    `
  }
}
