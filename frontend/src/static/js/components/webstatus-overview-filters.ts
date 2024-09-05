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

import {consume} from '@lit/context';
import {LitElement, type TemplateResult, CSSResultGroup, css, html} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {type components} from 'webstatus.dev-backend';
import {ref, createRef} from 'lit/directives/ref.js';
import {
  formatOverviewPageUrl,
  getSearchQuery,
  getColumnsSpec,
  getSortSpec,
  getWPTMetricView,
} from '../utils/urls.js';

import {openColumnsDialog} from './webstatus-columns-dialog.js';
import {SHARED_STYLES} from '../css/shared-css.js';

import './webstatus-typeahead.js';
import {type WebstatusTypeahead} from './webstatus-typeahead.js';
import './webstatus-overview-table.js';
import {TaskStatus} from '@lit/task';

import {
  type APIClient,
  type FeatureSortOrderType,
  type FeatureSearchType,
  FeatureWPTMetricViewType,
  BROWSER_ID_TO_LABEL,
  CHANNEL_ID_TO_LABEL,
} from '../api/client.js';
import {apiClientContext} from '../contexts/api-client-context.js';

import {CELL_DEFS, getBrowserAndChannel} from './webstatus-overview-cells.js';
import {
  ColumnKey,
  parseColumnsSpec,
  BrowserChannelColumnKeys,
} from './webstatus-overview-cells.js';

import {downloadCSV} from '../utils/csv.js';
import {toast} from '../utils/toast.js';

const VOCABULARY = [
  {
    name: 'available_date:chrome:2023-01-01..2024-01-01',
    doc: 'Became available on Chrome between the given dates',
  },
  {
    name: 'available_date:edge:2023-01-01..2024-01-01',
    doc: 'Became available on Edge between the given dates',
  },
  {
    name: 'available_date:firefox:2023-01-01..2024-01-01',
    doc: 'Became available on Firefox between the given dates',
  },
  {
    name: 'available_date:safari:2023-01-01..2024-01-01',
    doc: 'Became available on Safari between the given dates',
  },
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
    name: 'group:',
    doc: 'Features in a group or its descendants. E.g., group:css',
  },
  {
    name: 'snapshot:',
    doc: 'Features in a snapshot. E.g., snapshot:ecmascript-5',
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
  @consume({context: apiClientContext})
  @state()
  apiClient?: APIClient;

  @state()
  location!: {search: string}; // Set by parent.

  // Whether the export button should be enabled based on export status.
  @state()
  exportDataStatus: TaskStatus = TaskStatus.INITIAL;

  @state()
  // A function that returns an array of all features via apiClient.getAllFeatures
  allFeaturesFetcher:
    | undefined
    | (() => Promise<components['schemas']['Feature'][]>) = undefined;

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

  connectedCallback(): void {
    super.connectedCallback();
    document.addEventListener('keyup', this.handleDocumentKeyUp);
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    document.removeEventListener('keyup', this.handleDocumentKeyUp);
  }

  handleDocumentKeyUp = (e: KeyboardEvent) => {
    const inInputContext = e
      .composedPath()
      .some(el =>
        ['INPUT', 'TEXTAREA', 'SL-POPUP', 'SL-DIALOG'].includes(
          (el as HTMLElement).tagName
        )
      );
    if (e.key === '/' && !inInputContext) {
      e.preventDefault();
      e.stopPropagation();
      (this.typeaheadRef?.value as WebstatusTypeahead).focus();
    }
  };

  gotoFilterQueryString(): void {
    const newUrl = formatOverviewPageUrl(this.location, {
      q: (this.typeaheadRef.value as WebstatusTypeahead).value,
      start: 0,
    });
    window.location.href = newUrl;
  }

  protected firstUpdated(): void {
    if (this.apiClient !== undefined) {
      // Perform any initializations once the apiClient is passed to us via context.
      // TODO. allFeaturesFetcher should be moved to a separate task.
      this.allFeaturesFetcher = () => {
        return this.apiClient!.getAllFeatures(
          getSearchQuery(this.location) as FeatureSearchType,
          getSortSpec(this.location) as FeatureSortOrderType,
          getWPTMetricView(this.location) as FeatureWPTMetricViewType
        );
      };
    }
  }

  async exportToCSV(
    completedCallback: (() => void) | undefined
  ): Promise<void> {
    if (!this.allFeaturesFetcher) {
      return;
    }

    // Fetch all pages of data via getAllFeatures
    this.allFeaturesFetcher()
      .then(allFeatures => {
        // Use CELL_DEFS to define the columns and
        // get the current (active) columns.
        const columns: string[] = [];
        const columnKeys = parseColumnsSpec(getColumnsSpec(this.location));

        const pushBrowserChannelName = (
          browserColumnKey: BrowserChannelColumnKeys
        ) => {
          const name = CELL_DEFS[browserColumnKey].nameInDialog;

          const {browser, channel} = getBrowserAndChannel(browserColumnKey);
          const browserLabel = BROWSER_ID_TO_LABEL[browser];
          const channelLabel = CHANNEL_ID_TO_LABEL[channel];

          if (channel === 'stable') {
            columns.push(name);
          }
          columns.push(`${browserLabel} WPT ${channelLabel} Score`);
        };

        columnKeys.forEach(columnKey => {
          const name = CELL_DEFS[columnKey].nameInDialog;
          switch (columnKey) {
            case ColumnKey.Name:
              columns.push(name);
              break;
            case ColumnKey.BaselineStatus:
              columns.push(name);
              break;
            case ColumnKey.StableChrome:
            case ColumnKey.StableEdge:
            case ColumnKey.StableFirefox:
            case ColumnKey.StableSafari:
            case ColumnKey.ExpChrome:
            case ColumnKey.ExpEdge:
            case ColumnKey.ExpFirefox:
            case ColumnKey.ExpSafari:
              pushBrowserChannelName(columnKey);
              break;
          }
        });

        // Convert array of feature rows into array of arrays of strings,
        // in the same order as columns.
        const rows = allFeatures.map(feature => {
          const baselineStatus = feature.baseline?.status || '';
          const browserImpl = feature.browser_implementations!;
          const wptData = feature.wpt;
          const row: string[] = [];

          const pushBrowserChannelValue = (
            browserColumnKey: BrowserChannelColumnKeys
          ) => {
            const {browser, channel} = getBrowserAndChannel(browserColumnKey);
            const browserImplDate = browserImpl && browserImpl[browser]?.date;
            const wptScore = wptData?.[channel]?.[browser]?.score;

            if (channel === 'stable') {
              row.push(browserImplDate || '');
            }
            row.push(String(wptScore) || '');
          };

          // Iterate over the current columns to get the values for each column.
          for (const key of columnKeys) {
            switch (key) {
              case ColumnKey.Name:
                row.push(feature.name);
                break;
              case ColumnKey.BaselineStatus:
                row.push(baselineStatus);
                break;
              case ColumnKey.StableChrome:
              case ColumnKey.StableEdge:
              case ColumnKey.StableFirefox:
              case ColumnKey.StableSafari:
              case ColumnKey.ExpChrome:
              case ColumnKey.ExpEdge:
              case ColumnKey.ExpFirefox:
              case ColumnKey.ExpSafari:
                pushBrowserChannelValue(key);
                break;
            }
          }
          return row;
        });

        downloadCSV(columns, rows, 'webstatus-feature-overview.csv')
          .catch(error => {
            toast(
              `Save file error: ${error.message}`,
              'danger',
              'exclamation-triangle'
            );
          })
          .finally(() => {
            completedCallback && completedCallback();
          });
      })
      .catch(error => {
        toast(
          `Download error: ${error.message}`,
          'danger',
          'exclamation-triangle'
        );
      })
      .finally(() => {
        completedCallback && completedCallback();
      });
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
      this.exportToCSV(() => {
        this.exportDataStatus = TaskStatus.COMPLETE;
      });
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
