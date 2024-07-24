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
import {TaskStatus} from '@lit/task';

const VOCABULARY = [
  {
    name: 'available_on:chrome',
    doc: 'Features available on Chrome',
  },
  {
    name: 'available_on:edge',
    doc: 'Features available on Edge',
  },
  {
    name: 'available_on:firefox',
    doc: 'Features available on Firefox',
  },
  {
    name: 'available_on:safari',
    doc: 'Features available on Safari',
  },
  {
    name: 'baseline_date:2023-01-01..2024-01-01',
    doc: 'Reached baseline between the given dates',
  },
  {
    name: 'baseline_status:limited',
    doc: 'Features that are not yet in baseline',
  },
  {
    name: 'baseline_status:newly',
    doc: 'Features newly added to baseline',
  },
  {
    name: 'baseline_status:widely',
    doc: 'Features in baseline and widely available',
  },
  {
    name: 'name:',
    doc: 'Find by substring of the name. E.g., name:grid',
  },
  {
    name: 'name:"a substring"',
    doc: 'Find by substring of the name. E.g., name:"CSS Grid"',
  },
  {
    name: 'OR',
    doc: 'Combine query terms with a logical-OR',
  },
  {
    name: '-',
    doc: 'Negate search term with a leading minus',
  },
];

@customElement('webstatus-overview-filters')
export class WebstatusOverviewFilters extends LitElement {
  typeaheadRef = createRef();

  @state()
  location!: {search: string}; // Set by parent.

  // Whether the export button should be enabled based on export status.
  @state()
  exportDataStatus: TaskStatus = TaskStatus.INITIAL;

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
        id="columns-button"
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
        .vocabulary=${VOCABULARY}
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

  renderExportButton(): TemplateResult {
    const exportToCSV = () => {
      this.exportDataStatus = TaskStatus.PENDING;

      // dispatch an event via CustomEvent
      const event = new CustomEvent('exportToCSV', {
        bubbles: true,
        composed: true,
        cancelable: true,
        detail: {
          callback: () => {
            this.exportDataStatus = TaskStatus.COMPLETE;
          },
        },
      });
      this.dispatchEvent(event);
    };

    return html`
      <sl-button
        @click=${exportToCSV}
        ?loading=${this.exportDataStatus === TaskStatus.PENDING}
        ?disabled=${this.exportDataStatus === TaskStatus.PENDING}
      >
        <sl-icon slot="prefix" name="download"></sl-icon>
        Export to CSV
      </sl-button>
    `;
  }

  render(): TemplateResult {
    const query = getSearchQuery(this.location);
    return html`
      <div class="vbox all-filter-controls">
        <div class="hbox filter-by-feature-name">
          ${this.renderFilterInputBox(query)} ${this.renderColumnButton()}
          ${this.renderExportButton()}
        </div>
      </div>
    `;
  }
}
