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
import {
  LitElement,
  type TemplateResult,
  CSSResultGroup,
  css,
  html,
  PropertyValueMap,
  nothing,
} from 'lit';
import {customElement, property, state} from 'lit/decorators.js';
import {type components} from 'webstatus.dev-backend';
import {ref, createRef} from 'lit/directives/ref.js';
import {
  formatOverviewPageUrl,
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
import './webstatus-saved-search-controls.js';

import {CSVUtils} from '../utils/csv.js';
import {Toast} from '../utils/toast.js';
import {navigateToUrl} from '../utils/app-router.js';
import {
  AppBookmarkInfo,
  SavedSearchScope,
  savedSearchHelpers,
} from '../contexts/app-bookmark-info-context.js';
import {UserSavedSearch, VOCABULARY} from '../utils/constants.js';
import {User, firebaseUserContext} from '../contexts/firebase-user-context.js';

const WEBSTATUS_FEATURE_OVERVIEW_CSV_FILENAME =
  'webstatus-feature-overview.csv';

@customElement('webstatus-overview-filters')
export class WebstatusOverviewFilters extends LitElement {
  typeaheadRef = createRef<WebstatusTypeahead>();
  @consume({context: apiClientContext})
  @state()
  apiClient?: APIClient;

  @property({type: Object})
  location!: {search: string};

  @property({type: Object})
  appBookmarkInfo?: AppBookmarkInfo;

  @consume({context: firebaseUserContext, subscribe: true})
  @state()
  user: User | null | undefined;

  _activeQuery: string = '';

  _activeUserSavedSearch?: UserSavedSearch | undefined;

  // Whether the export button should be enabled based on export status.
  @state()
  exportDataStatus: TaskStatus = TaskStatus.INITIAL;

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

  protected willUpdate(changedProperties: PropertyValueMap<this>): void {
    if (
      changedProperties.has('location') ||
      changedProperties.has('appBookmarkInfo')
    ) {
      this._activeQuery = savedSearchHelpers.getCurrentQuery(
        this.appBookmarkInfo,
      );
      const search = savedSearchHelpers.getCurrentSavedSearch(
        this.appBookmarkInfo,
      );
      // Allow resetting of active search.
      if (search === undefined) {
        this._activeUserSavedSearch = undefined;
      }
      // If the search is a user search, store it. Ignore Global Saved Searches
      if (search?.scope === SavedSearchScope.UserSavedSearch) {
        this._activeUserSavedSearch = search.value;
      }
    }
  }

  handleDocumentKeyUp = (e: KeyboardEvent) => {
    const inInputContext = e
      .composedPath()
      .some(el =>
        ['INPUT', 'TEXTAREA', 'SL-POPUP', 'SL-DIALOG'].includes(
          (el as HTMLElement).tagName,
        ),
      );
    if (e.key === '/' && !inInputContext) {
      e.preventDefault();
      e.stopPropagation();
      this.typeaheadRef?.value?.focus();
    }
  };

  gotoFilterQueryString(): void {
    const newUrl = formatOverviewPageUrl(this.location, {
      q: this.typeaheadRef.value?.value,
      start: 0,
    });
    navigateToUrl(newUrl);
  }

  protected firstUpdated(): void {
    if (this.apiClient !== undefined) {
      // Perform any initializations once the apiClient is passed to us via context.
      // TODO. allFeaturesFetcher should be moved to a separate task.
      this.allFeaturesFetcher = () => {
        return this.apiClient!.getAllFeatures(
          savedSearchHelpers.getCurrentQuery(this.appBookmarkInfo),
          getSortSpec(this.location) as FeatureSortOrderType,
          getWPTMetricView(this.location) as FeatureWPTMetricViewType,
        );
      };
    }
  }

  async exportToCSV(): Promise<void> {
    if (!this.allFeaturesFetcher) {
      return;
    }

    let allFeatures: components['schemas']['Feature'][];
    try {
      allFeatures = await this.allFeaturesFetcher();
    } catch (error) {
      if (error instanceof Error) {
        throw new Error(`Download features error: ${error?.message}`);
      }
      throw new Error('Unknown error fetching features.');
    }

    // Use CELL_DEFS to define the columns and
    // get the current (active) columns.
    const columns: string[] = [];
    const columnKeys = parseColumnsSpec(getColumnsSpec(this.location));

    const pushBrowserChannelName = (
      browserColumnKey: BrowserChannelColumnKeys,
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
        case ColumnKey.ChromeUsage:
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
      const chromeUsage = feature.usage?.chrome?.daily?.toString() || '';
      const row: string[] = [];

      const pushBrowserChannelValue = (
        browserColumnKey: BrowserChannelColumnKeys,
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
          case ColumnKey.ChromeUsage:
            row.push(chromeUsage);
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

    try {
      await CSVUtils.downloadCSV(
        columns,
        rows,
        WEBSTATUS_FEATURE_OVERVIEW_CSV_FILENAME,
      );
    } catch (error) {
      if (error instanceof Error) {
        throw new Error(`Save file error: ${error.message}`);
      }
      throw new Error('Unknown error downloading csv');
    }
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
        data-testid="overview-query-input"
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
      ${this.user && this.apiClient
        ? this.renderSavedSearchControls(this.user, this.apiClient)
        : nothing}
    `;
  }

  renderSavedSearchControls(user: User, apiClient: APIClient): TemplateResult {
    if (this.typeaheadRef.value === undefined) {
      return html``;
    }
    return html`
      <sl-popup
        placement="top-end"
        autoSize="horizontal"
        distance="5"
        active
        .anchor=${this.typeaheadRef.value}
      >
        <div slot="anchor" class="popup-anchor saved-search-controls"></div>
        <div class="popup-content">
          <webstatus-saved-search-controls
            .user=${user}
            .apiClient=${apiClient}
            .savedSearch=${this._activeUserSavedSearch}
            .location=${this.location}
            .overviewPageQueryInput=${this.typeaheadRef.value}
          ></webstatus-saved-search-controls>
        </div>
      </sl-popup>
    `;
  }

  renderExportButton(): TemplateResult {
    const handleExportToCSV = () => {
      this.exportDataStatus = TaskStatus.PENDING;
      this.exportToCSV()
        .then(() => {
          this.exportDataStatus = TaskStatus.COMPLETE;
        })
        .catch(async error => {
          await new Toast().toast(
            error?.message,
            'danger',
            'exclamation-triangle',
          );
          this.exportDataStatus = TaskStatus.ERROR;
        });
    };

    return html`
      <sl-button
        id="export-to-csv-button"
        @click=${handleExportToCSV}
        ?loading=${this.exportDataStatus === TaskStatus.PENDING}
        ?disabled=${this.exportDataStatus === TaskStatus.PENDING}
      >
        <sl-icon slot="prefix" name="download"></sl-icon>
        Export to CSV
      </sl-button>
    `;
  }

  render(): TemplateResult {
    return html`
      <div class="vbox all-filter-controls">
        <div class="hbox filter-by-feature-name">
          ${this.renderFilterInputBox(this._activeQuery)}
          ${this.renderColumnButton()} ${this.renderExportButton()}
        </div>
      </div>
    `;
  }
}
