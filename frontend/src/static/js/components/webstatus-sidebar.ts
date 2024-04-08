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

import {LitElement, type TemplateResult, CSSResultGroup, css, html} from 'lit';
import {customElement} from 'lit/decorators.js';
import {SHARED_STYLES} from '../css/shared-css.js';

import './webstatus-sidebar-menu.js';

@customElement('webstatus-sidebar')
export class WebstatusSidebar extends LitElement {
  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        .sidebar {
          height: 100%;
          width: 288px;
        }

        sl-tree#bottom-menu {
          margin-top: auto;
        }

        sl-tree#bottom-menu a {
          color: inherit;
          text-decoration: none;
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
      `,
    ];
  }

  render(): TemplateResult {
    return html`
      <div class="sidebar vbox">
        <webstatus-sidebar-menu></webstatus-sidebar-menu>

        <sl-divider></sl-divider>
        <div class="spacer"></div>

        <div class="valign-stretch-2"></div>
          <sl-tree id="bottom-menu">
            <sl-tree-item>
              <sl-icon name="github"></sl-icon>
              <a href="https://github.com/GoogleChrome/webstatus.dev/issues/new"
                >Report an issue</a
              >
            </sl-tree-item>
          </sl-tree>
        </div>
      </div>
    `;

    // <sl-tree-item id="theme-selector">
    //   <sl-select value="light">
    //     <sl-icon name="brightness-high" slot="prefix"></sl-icon>
    //     <sl-option value="light">
    //       <sl-icon name="brightness-high" slot="prefix"></sl-icon>
    //       Light theme</sl-option
    //     >
    //     <sl-option value="dark">
    //       <sl-icon
    //         name="brightness-high-fill"
    //         style="color: black"
    //         slot="prefix"
    //       ></sl-icon>
    //       Dark theme</sl-option
    //     >
    //   </sl-select>
    // </sl-tree-item>
  }
}
