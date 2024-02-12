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
import './webstatus-overview-table.js'

@customElement('webstatus-overview-content')
export class WebstatusOverviewContent extends LitElement {
  static styles = css`
    .stats-summary {
      color: #6c7381;
    }
  `

  render(): TemplateResult {
    return html`
      <div class="main">
        <h2>Features overview</h2>
        <span class="stats-summary">
          <sl-icon library="phosphor" name="list-magnifying-glass"></sl-icon>
          1433 features</span
        >

        <sl-input placeholder="Filter by feature name...">
          <sl-icon name="search" slot="prefix"></sl-icon>
        </sl-input>
        <sl-button
          ><sl-icon slot="prefix" name="filter"></sl-icon>Filter</sl-button
        >
        <sl-button>
          <sl-icon
            slot="prefix"
            name="square-split-horizontal"
            library="phosphor"
          ></sl-icon>
        </sl-button>

        <div class="filters">
          <select>
            <option value="all">All</option>
            <option value="active">Active</option>
            <option value="inactive">Inactive</option>
          </select>
        </div>

        <webstatus-overview-table></webstatus-overview-table>
        <button>Modify Columns</button>
      </div>
    `
  }
}
