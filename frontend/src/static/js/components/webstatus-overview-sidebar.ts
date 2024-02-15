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

import {LitElement, type TemplateResult, css, html} from 'lit';
import {customElement} from 'lit/decorators.js';

import './webstatus-overview-sidebar-menu.js';

@customElement('webstatus-overview-sidebar')
export class WebstatusOverviewSidebar extends LitElement {
  static readonly styles = css`
    .sidebar {
      display: flex;
      flex-direction: column;
      justify-content: flex-start;
      height: 100%;
      width: 288px;
      /* padding-right: 20px;
      padding-top: 10px; */
    }

    sl-tree#bottom-menu {
      margin-top: auto;
    }
    #sidebar-bottom {
      flex-grow: 2;
      display: flex;
    }

    sl-tree-item#theme-selector sl-select::part(combobox) {
      border: 0;
    }

    sl-tree-item#theme-selector::part(expand-button) {
      width: 0;
    }

    #theme-selector > sl-select > sl-icon {
      margin-inline-end: 8px;
    }
  `;

  render(): TemplateResult {
    return html`
      <div class="sidebar">
        <webstatus-overview-sidebar-menu></webstatus-overview-sidebar-menu>

        <sl-divider></sl-divider>

        <div id="sidebar-bottom">
          <sl-tree id="bottom-menu">
            <sl-tree-item>
              <sl-icon name="github"></sl-icon> Report an issue
            </sl-tree-item>
            <sl-tree-item id="theme-selector">
              <sl-select value="light">
                <sl-icon name="brightness-high" slot="prefix"></sl-icon>
                <sl-option value="light">
                  <sl-icon name="brightness-high" slot="prefix"></sl-icon>
                  Light theme</sl-option
                >
                <sl-option value="dark">
                  <sl-icon
                    name="brightness-high-fill"
                    style="color: black"
                    slot="prefix"
                  ></sl-icon>
                  Dark theme</sl-option
                >
              </sl-select>
            </sl-tree-item>
          </sl-tree>
        </div>
      </div>
    `;
  }
}
