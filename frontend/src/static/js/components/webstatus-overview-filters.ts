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
import {ref, createRef} from 'lit/directives/ref.js';
import {formatOverviewPageUrl, getSearchQuery} from '../utils/urls.js';
import {openColumnsDialog} from './webstatus-columns-dialog.js';
import {SHARED_STYLES} from '../css/shared-css.js';

import './webstatus-typeahead.js';
import {type WebstatusTypeahead} from './webstatus-typeahead.js';
import './webstatus-overview-table.js';

const VOCABULARY = [
    {name: 'available_on:', doc: 'Available on a specific browser. E.g., available:chrome'},
    {name: 'baseline_date:', doc: 'Date the feature reached baseline: YYYY-MM-DD..YYYY-MM-DD'},
    {name: 'baseline_status:', doc: "Feature's baseline status: limited, newly, or widely"},
    {name: 'name:', doc: 'Find by name. E.g. name:grid. E.g., name:"CSS Grid"'},
];


@customElement('webstatus-overview-filters')
export class WebstatusOverviewFilters extends LitElement {
  typeaheadRef = createRef();

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

    gotoFilterQueryString(): void {
    const newUrl = formatOverviewPageUrl(this.location, {
      q: (this.typeaheadRef.value as WebstatusTypeahead).value,
      start: 0,
    });
    window.location.href = newUrl;
  }

  renderColumnButton(): TemplateResult {
    return html`
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
    `;
  }

  renderFilterInputBox(input: string): TemplateResult {
    return html`
      <webstatus-typeahead
        id="filter-input-input"
        ${ref(this.typeaheadRef)}
        class="halign-stretch"
        placeholder="Filter by ..."
        value="${input}"
        .vocabulary = ${VOCABULARY}
        @sl-change=${() => this.gotoFilterQueryString()}
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
      </webstatus-typeahead>
    `;
  }

  render(): TemplateResult {
    const query = getSearchQuery(this.location);
    return html`
      <div class="vbox all-filter-controls">
        <div class="hbox filter-by-feature-name">
          ${this.renderFilterInputBox(query)} ${this.renderColumnButton()}
        </div>
      </div>
    `;
  }
}
