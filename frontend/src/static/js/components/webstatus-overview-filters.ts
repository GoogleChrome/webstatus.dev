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
import {getSearchQuery} from '../utils/urls.js';
import {SHARED_STYLES} from '../css/shared-css.js';

import './webstatus-overview-table.js';

@customElement('webstatus-overview-filters')
export class WebstatusOverviewFilters extends LitElement {
  location!: {search: string}; // Set by parent.

  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        .filter-by-feature-name {
          gap: 1em;
        }
      `,
    ];
  }

  render(): TemplateResult {
    const query = getSearchQuery(this.location);
    return html`
      <div class="hbox space-between filter-by-feature-name">
        <sl-input
          class="hgrow"
          placeholder="Filter by feature name..."
          value="${query}"
        >
          <sl-icon name="search" slot="prefix"></sl-icon>
        </sl-input>

        <sl-dropdown>
          <sl-button slot="trigger">
            <sl-icon slot="prefix" name="filter"></sl-icon>
            Filter
          </sl-button>
          <sl-menu>
            <sl-menu-item value="available-on"
              >Available on
              <sl-menu slot="submenu">
                <sl-menu-item type="checkbox" value="chrome">
                  Chrome
                </sl-menu-item>
                <sl-menu-item type="checkbox" value="edge"> Edge </sl-menu-item>
                <sl-menu-item type="checkbox" value="firefox">
                  Firefox
                </sl-menu-item>
                <sl-menu-item type="checkbox" value="safari">
                  Safari
                </sl-menu-item>
              </sl-menu>
            </sl-menu-item>
            <sl-menu-item value="not-available-on">
              Not available on
              <sl-menu slot="submenu">
                <sl-menu-item type="checkbox" value="chrome">
                  Chrome
                </sl-menu-item>
                <sl-menu-item type="checkbox" value="edge"> Edge </sl-menu-item>
                <sl-menu-item type="checkbox" value="firefox">
                  Firefox
                </sl-menu-item>
                <sl-menu-item type="checkbox" value="safari">
                  Safari
                </sl-menu-item>
                <sl-divider></sl-divider>
                <sl-menu-item type="checkbox" value="not-in">
                  Not available in 1 browser
                </sl-menu-item>
              </sl-menu>
            </sl-menu-item>
            <sl-menu-item value="baseline-since">
              Baseline since
              <sl-icon
                slot="suffix"
                name="calendar-blank"
                library="phosphor"
              ></sl-icon>
            </sl-menu-item>
            <sl-menu-item value="baseline-status">
              Baseline status
              <sl-menu slot="submenu">
                <sl-menu-item type="checkbox" value="widely">
                  Widely available
                </sl-menu-item>
                <sl-menu-item type="checkbox" value="newly">
                  Newly available
                </sl-menu-item>
                <sl-menu-item type="checkbox" value="limited">
                  Limited availability
                </sl-menu-item>
              </sl-menu>
            </sl-menu-item>
            <sl-menu-item value="baseline-type"> Baseline type </sl-menu-item>
            <sl-menu-item type="checkbox" value="standards-track">
              Standards track
            </sl-menu-item>
            <sl-menu-item type="checkbox" value="spec-maturity">
              Spec maturity
              <sl-menu slot="submenu">
                <sl-menu-item type="checkbox" value="unknown">
                  Unknown
                </sl-menu-item>
                <sl-menu-item type="checkbox" value="proposed">
                  Proposed
                </sl-menu-item>
                <sl-menu-item type="checkbox" value="incubation">
                  Incubation
                </sl-menu-item>
                <sl-menu-item type="checkbox" value="working-draft">
                  Working draft
                </sl-menu-item>
                <sl-menu-item type="checkbox" value="living-standard">
                  Living standard
                </sl-menu-item>
              </sl-menu>
            </sl-menu-item>
            <sl-menu-item value="web-platform-test-score">
              Web platform test score
              <sl-menu slot="submenu">
                <sl-menu-item type="checkbox" value="chrome">
                  Chrome
                </sl-menu-item>
                <sl-menu-item type="checkbox" value="edge"> Edge </sl-menu-item>
                <sl-menu-item type="checkbox" value="firefox">
                  Firefox
                </sl-menu-item>
                <sl-menu-item type="checkbox" value="safari">
                  Safari
                </sl-menu-item>
              </sl-menu>
            </sl-menu-item>
            <sl-divider></sl-divider>
            <sl-menu-item value="clear-all"> Clear all </sl-menu-item>
          </sl-menu>
        </sl-dropdown>

        <sl-dropdown>
          <sl-button slot="trigger">
            <sl-icon
              slot="prefix"
              name="square-split-horizontal"
              library="phosphor"
            ></sl-icon>
          </sl-button>
        </sl-dropdown>
      </div>
    `;
  }
}
