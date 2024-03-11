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
import {SlInput, SlMenu, SlMenuItem} from '@shoelace-style/shoelace';

import './webstatus-overview-table.js';

@customElement('webstatus-overview-filters')
export class WebstatusOverviewFilters extends LitElement {
  location!: {search: string}; // Set by parent.

  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        .all-filter-controls,
        .filter-by-feature-name,
        .filter-buttons {
          gap: var(--content-padding);
        }

        #baseline_since_button > sl-icon[name='calendar-blank'],
        #standards_track_button > sl-checkbox {
          padding-left: 1rem;
        }

        /** Filter query submit button pulses after changes. */
        @keyframes pulseBtn {
          0% {
            box-shadow: 0px 0px 0px 0px rgba(81, 73, 255, 1);
          }
          100% {
            box-shadow: 0px 0px 10px 2px rgba(81, 73, 255, 1);
          }
        }

        .glow-btn.changed {
          border-radius: 4px;
          animation-name: pulseBtn;
          animation-duration: 0.9s;
          animation-iteration-count: infinite;
          animation-direction: alternate;
          animation-timing-function: ease-in-out;
        }
      `,
    ];
  }

  parseFilterQueryString(filterQueryString: string): Map<string, string[]> {
    // Parse the filter query string into a map of filter keys and values.
    const filterQueryMap = new Map<string, string[]>();
    const filterQueryItems =
      filterQueryString.length > 0 ? filterQueryString.split(' ') : [];
    for (const filterQueryItem of filterQueryItems) {
      const [key, value] = filterQueryItem.split(':');
      // Parse each value as a comma separated list of values.
      const valueArray = value.split(',');
      filterQueryMap.set(key, valueArray);

      // Populate the menu items with the values from the filter query string.
      const menu = this.shadowRoot!.getElementById(key) as SlMenu;
      const menuChildren = menu.children;
      const menuItemsArray: Array<SlMenuItem> = Array.from(menuChildren).filter(
        child => child instanceof SlMenuItem
      ) as Array<SlMenuItem>;
      for (const menuItem of menuItemsArray) {
        menuItem.checked = valueArray.includes(menuItem.value);
      }
    }
    return filterQueryMap;
  }

  generateFilterQueryString(filterQueryMap: Map<string, string[]>): string {
    // Generate a filter query string from a map of filter keys and values.
    const filterQueryStringArray: string[] = [];
    for (const [key, valueArray] of filterQueryMap.entries()) {
      const valueString = valueArray.join(',');
      filterQueryStringArray.push(`${key}:${valueString}`);
    }
    const filterQueryString = filterQueryStringArray.join(' ');
    return filterQueryString;
  }

  filterQueryInput!: SlInput;
  filterQueryMap!: Map<string, string[]>;

  initializeFilterQueryMap(): Map<string, string[]> {
    // Initialize the filter query map with the values from the URL.
    // Get the filter query string from filter-query-input
    this.filterQueryInput = this.shadowRoot!.getElementById(
      'filter-query-input'
    ) as SlInput;
    const filterQueryString = (this.filterQueryInput.value || '').trim();

    this.filterQueryMap = this.parseFilterQueryString(filterQueryString);

    return this.filterQueryMap;
  }

  makeFilterSelectHandler(id: string): (event: Event) => void {
    return (event: Event) => {
      const menu = event.target as SlMenu;
      const menuChildren = menu.children;

      const menuItemsArray: Array<SlMenuItem> = Array.from(menuChildren).filter(
        child => child instanceof SlMenuItem
      ) as Array<SlMenuItem>;

      // Create a list of the currently checked sl-menu-items.
      const checkedItems = menuItemsArray.filter(menuItem => menuItem.checked);
      // Build a query string from the values of those items.
      const checkedItemsValues = checkedItems.map(menuItem => menuItem.value);

      // Update the filterQueryMap with the new values.
      this.filterQueryMap.set(id, checkedItemsValues);
      // Update the filterQuery input with the new filter query string.
      const filterQueryString = this.generateFilterQueryString(
        this.filterQueryMap
      );
      this.filterQueryInput.value = filterQueryString;

      // Activate the submit button glowing
      const submitButton = this.shadowRoot!.getElementById(
        'filter-submit-button'
      ) as SlInput;
      submitButton.classList.add('changed');
    };
  }

  firstUpdated(): void {
    this.initializeFilterQueryMap();
    // Add sl-select event handler to all sl-menu elements.
    const menuElements = Array.from(
      this.shadowRoot!.querySelectorAll('sl-menu')
    );
    for (const menuElement of menuElements) {
      const id = menuElement.id;
      menuElement.addEventListener(
        'sl-select',
        this.makeFilterSelectHandler(id)
      );
    }
  }

  render(): TemplateResult {
    const query = getSearchQuery(this.location);
    return html`
      <div class="vbox all-filter-controls">
        <div class="hbox filter-by-feature-name">
          <sl-input
            id="filter-query-input"
            class="halign-stretch"
            placeholder="Filter by ..."
            value="${query}"
          >
            <sl-button
              id="filter-submit-button"
              class="glow-btn"
              size="small"
              type="submit"
              slot="prefix"
              submit
              variant="success"
              outline
            >
              <sl-icon slot="prefix" name="search"></sl-icon>
            </sl-button>
          </sl-input>
        </div>

        <div class="hbox wrap filter-buttons">
          <sl-dropdown stay-open-on-select>
            <sl-button slot="trigger">
              <sl-icon slot="prefix" name="plus-circle"></sl-icon>
              Available on
            </sl-button>
            <sl-menu id="available_on">
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
          </sl-dropdown>

          <sl-dropdown stay-open-on-select>
            <sl-button slot="trigger">
              <sl-icon slot="prefix" name="plus-circle"></sl-icon>
              Not available on
            </sl-button>
            <sl-menu id="not_available_on">
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
          </sl-dropdown>

          <sl-button id="baseline_since_button">
            <sl-icon name="plus-circle"></sl-icon>
            Baseline since
            <sl-icon name="calendar-blank" library="phosphor"></sl-icon>
          </sl-button>

          <sl-dropdown stay-open-on-select>
            <sl-button slot="trigger">
              <sl-icon slot="prefix" name="plus-circle"></sl-icon>
              Baseline status
            </sl-button>
            <sl-menu id="baseline_status">
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
          </sl-dropdown>

          <sl-dropdown stay-open-on-select>
            <sl-button slot="trigger">
              <sl-icon slot="prefix" name="plus-circle"></sl-icon>
              Browser type
            </sl-button>
            <sl-menu id="browser_type">
              <sl-menu-item type="checkbox" value="stable-builds">
                Stable builds
              </sl-menu-item>
              <sl-menu-item type="checkbox" value="dev-builds">
                Dev builds
              </sl-menu-item>
            </sl-menu>
          </sl-dropdown>

          <sl-button id="standards_track_button">
            <sl-icon slot="prefix" name="plus-circle"></sl-icon>
            Standards track
            <sl-checkbox> </sl-checkbox>
          </sl-button>

          <sl-dropdown stay-open-on-select>
            <sl-button slot="trigger">
              <sl-icon slot="prefix" name="plus-circle"></sl-icon>
              Spec maturity
            </sl-button>
            <sl-menu id="spec_maturity">
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
          </sl-dropdown>

          <sl-dropdown stay-open-on-select>
            <sl-button slot="trigger">
              <sl-icon slot="prefix" name="plus-circle"></sl-icon>
              Web platform test score
            </sl-button>
            <sl-menu id="web_platform_test_score">
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
          </sl-dropdown>

          <sl-dropdown stay-open-on-select>
            <sl-button slot="trigger">
              <sl-icon
                slot="prefix"
                name="square-split-horizontal"
                library="phosphor"
              ></sl-icon>
              Columns
            </sl-button>
          </sl-dropdown>
        </div>
      </div>
    `;
  }
}
