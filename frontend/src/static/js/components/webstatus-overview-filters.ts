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
import {customElement, state} from 'lit/decorators.js';
import {formatOverviewPageUrl, getSearchQuery} from '../utils/urls.js';
import {openColumnsDialog} from './webstatus-columns-dialog.js';
import {SHARED_STYLES} from '../css/shared-css.js';
import {SlInput, SlMenu, SlMenuItem} from '@shoelace-style/shoelace';

import './webstatus-overview-table.js';

@customElement('webstatus-overview-filters')
export class WebstatusOverviewFilters extends LitElement {
  @state()
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

        #filter-input-input {
          --sl-input-spacing-medium: 0.875rem;
        }
        #filter-submit-button::part(base) {
          --sl-spacing-x-small: 0.275rem;
          --sl-input-height-small: 1.475rem;
        }

        /** Filter input submit button pulses after changes. */
        @keyframes pulseBtn {
          0% {
            box-shadow: 0px 0px 0px 0px var(--sl-color-success-600);
          }
          100% {
            box-shadow: 0px 0px 8px 2px var(--sl-color-success-600);
          }
        }

        .glow-btn {
          border-radius: 4px;
        }
        .glow-btn.changed {
          animation-name: pulseBtn;
          animation-duration: 0.9s;
          animation-iteration-count: infinite;
          animation-direction: alternate;
          animation-timing-function: ease-in-out;
        }
      `,
    ];
  }

  filterInput!: SlInput;
  filterInputMap!: Map<string, string[]>;

  // Initializes the filter input map with the values from the URL.
  // Gets the filter input string from filter-input-input
  initializeFilterInput(): void {
    this.filterInput = this.shadowRoot!.getElementById(
      'filter-input-input'
    ) as SlInput;
    const filterInputString = (this.filterInput.value || '').trim();
    this.filterInputMap = this.parseFilterInputString(filterInputString);

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

    // Set the sl-menu items based on the filterInputMap.
    for (const [key, valueArray] of this.filterInputMap.entries()) {
      const menuElement = this.shadowRoot!.getElementById(key) as SlMenu;
      if (menuElement == null) continue; // Skip for now.
      for (const value of valueArray) {
        const menuItem = menuElement.querySelector(
          `sl-menu-item[value="${value}"]`
        ) as SlMenuItem;
        if (menuItem) {
          menuItem.checked = true;
        }
      }
    }
  }

  // Parses the filter input string into a map of filter keys and values.
  parseFilterInputString(filterInputString: string): Map<string, string[]> {
    const filterInputMap = new Map<string, string[]>();
    if (filterInputString.length > 0) {
      // This parser does the inverse of generateFilterInputString.
      // Top-level is a list of ' AND '-separated clauses.
      const andClauseArray = filterInputString.split(' AND ');
      for (const orClausesString of andClauseArray) {
        // Each OR-clause is a list of ' OR '-separated clauses.
        // Strip optional parentheses from the OR-clause.
        const orClausesStringStripped = orClausesString.replace(
          /^\((.*)\)$/,
          '$1'
        );
        const orClauseArray = orClausesStringStripped.split(' OR ');
        let orKey = '';
        const valueArray = [];
        for (const orClauseString of orClauseArray) {
          // Each OR-clause is a key:value pair for the same key.
          const [key, value] = orClauseString.split(':');
          // Check that key matches orKey, if set.
          if (orKey && key !== orKey) {
            // This is a current limitation of the parser.
            throw new Error(
              `Unexpected key in filter input string: ${key} != ${orKey}`
            );
          }
          orKey = key;
          valueArray.push(value);
        }
        filterInputMap.set(orKey, valueArray);
      }
    }
    return filterInputMap;
  }

  // Generates a filter input string from a map of filter keys and values.
  generateFilterInputString(filterInputMap: Map<string, string[]>): string {
    const andClauseArray: string[] = [];
    for (const [key, orClauseArray] of filterInputMap.entries()) {
      if (orClauseArray.length > 0) {
        let orClauseString = orClauseArray
          .map((value: string) => `${key}:${value}`)
          .join(' OR ');
        if (orClauseArray.length > 1) orClauseString = `(${orClauseString})`;
        andClauseArray.push(orClauseString);
      }
    }
    const filterInputString = andClauseArray.join(' AND ');
    return filterInputString;
  }

  gotoFilterQueryString(): void {
    const newUrl = formatOverviewPageUrl(this.location, {
      q: this.filterInput.value,
      start: 0,
    });
    window.location.href = newUrl;
  }

  // Returns a handler for changes to a filter menu.
  makeFilterSelectHandler(id: string): (event: Event) => void {
    return (event: Event) => {
      const menu = event.target as SlMenu;
      const menuChildren = menu.children;

      const menuItemsArray: Array<SlMenuItem> = Array.from(menuChildren).filter(
        child => child instanceof SlMenuItem
      ) as Array<SlMenuItem>;

      // Create a list of the currently checked sl-menu-items.
      const checkedItems = menuItemsArray.filter(menuItem => menuItem.checked);
      // Build a input string from the values of those items.
      const checkedItemsValues = checkedItems.map(menuItem => menuItem.value);

      // Update the filterInputMap with the new values.
      this.filterInputMap.set(id, checkedItemsValues);
      // Update the filterInput with the new filter input string.
      const filterInputString = this.generateFilterInputString(
        this.filterInputMap
      );
      this.filterInput.value = filterInputString;

      // Activate the submit button glowing
      const submitButton = this.shadowRoot!.getElementById(
        'filter-submit-button'
      ) as SlInput;
      submitButton.classList.add('changed');
    };
  }

  firstUpdated(): void {
    this.initializeFilterInput();
  }

  handleSearchKey(event: KeyboardEvent) {
    if (event.code === 'Enter') {
      this.gotoFilterQueryString();
    }
  }

  render(): TemplateResult {
    const input = getSearchQuery(this.location);
    return html`
      <div class="vbox all-filter-controls">
        <div class="hbox filter-by-feature-name">
          <sl-input
            id="filter-input-input"
            class="halign-stretch"
            placeholder="Filter by ..."
            value="${input}"
            @keyup=${this.handleSearchKey}
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
              @click=${() => this.gotoFilterQueryString()}
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

          <sl-button
            slot="trigger"
            @click=${() => openColumnsDialog(this.location)}
          >
            <sl-icon
              slot="prefix"
              name="square-split-horizontal"
              library="phosphor"
            ></sl-icon>
            Columns
          </sl-button>
        </div>
      </div>
    `;
  }
}
