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
import {Task, TaskStatus} from '@lit/task';
import {LitElement, type TemplateResult, html} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {type components} from 'webstatus.dev-backend';
import {convertToCSV} from '../utils/csv.js';

import {
  getColumnsSpec,
  getPageSize,
  getPaginationStart,
  getSearchQuery,
  getSortSpec,
  getWPTMetricView,
} from '../utils/urls.js';
import {
  type APIClient,
  type FeatureSortOrderType,
  type FeatureSearchType,
  FeatureWPTMetricViewType,
} from '../api/client.js';
import {apiClientContext} from '../contexts/api-client-context.js';
import './webstatus-overview-content.js';
import {TaskTracker} from '../utils/task-tracker.js';
import {ApiError, UnknownError} from '../api/errors.js';
import {CELL_DEFS} from './webstatus-overview-cells.js';
import {ColumnKey, parseColumnsSpec} from './webstatus-overview-cells.js';

@customElement('webstatus-overview-page')
export class OverviewPage extends LitElement {
  loadingTask: Task;

  @consume({context: apiClientContext})
  apiClient?: APIClient;

  @state()
  taskTracker: TaskTracker<components['schemas']['FeaturePage'], ApiError> = {
    status: TaskStatus.INITIAL, // Initial state
    error: null,
    data: null,
  };

  @state()
  location!: {search: string}; // Set by router.

  @state()
  // allFeaturesFetcher is either undefined or a function that returns
  // an array of all features via apiClient.getAllFeatures
  allFeaturesFetcher:
    | undefined
    | (() => Promise<components['schemas']['Feature'][]>) = undefined;

  constructor() {
    super();

    this.loadingTask = new Task(this, {
      args: () =>
        [this.apiClient, this.location] as [APIClient, {search: string}],
      task: async ([apiClient, routerLocation]): Promise<
        components['schemas']['FeaturePage']
      > => {
        this.allFeaturesFetcher = () => {
          return apiClient.getAllFeatures(
            getSearchQuery(routerLocation) as FeatureSearchType,
            getSortSpec(routerLocation) as FeatureSortOrderType,
            getWPTMetricView(routerLocation) as FeatureWPTMetricViewType
          );
        };
        return this._fetchFeatures(apiClient, routerLocation);
      },
      onComplete: page => {
        this.taskTracker = {
          status: TaskStatus.COMPLETE,
          error: null,
          data: page,
        };
      },
      onError: (error: unknown) => {
        if (error instanceof ApiError) {
          this.taskTracker = {
            status: TaskStatus.ERROR,
            error: error,
            data: null,
          };
        } else {
          // Should never reach here but let's handle it.
          this.taskTracker = {
            status: TaskStatus.ERROR,
            error: new UnknownError('unknown error fetching features'),
            data: null,
          };
        }
      },
    });

    // Set up listener of 'exportToCSV' event from webstatus-overview-filters.
    this.addEventListener('exportToCSV', () => {
      this.exportToCSV();
    });
  }

  async exportToCSV(): Promise<void> {
    if (!this.allFeaturesFetcher) {
      return;
    }
    // Fetch all pages of data via getAllFeatures
    const allFeatures = await this.allFeaturesFetcher();

    // Use CELL_DEFS to define the columns and
    // get the current (active) columns.
    const columnKeys = parseColumnsSpec(getColumnsSpec(this.location));
    const columns: string[] = [];

    const pushBrowserName = (name: string) => {
      columns.push(name);
      columns.push(`${name} WPT Stable Score`);
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
          pushBrowserName(name);
          break;
      }
      columns.push(name);
    });

    // Convert array of feature rows into array of arrays of strings,
    // in the same order as columns.
    const rows = allFeatures.map(feature => {
      const baselineStatus = feature.baseline?.status || '';
      const browserImpl = feature.browser_implementations!;
      const wptStableScores = feature.wpt?.stable || undefined;
      const row = [];
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
            row.push(browserImpl?.chrome?.date || '');
            row.push(String(wptStableScores?.chrome?.score) || '');
            break;
          case ColumnKey.StableEdge:
            row.push(browserImpl?.edge?.date || '');
            row.push(String(wptStableScores?.edge?.score) || '');
            break;
          case ColumnKey.StableFirefox:
            row.push(browserImpl?.firefox?.date || '');
            row.push(String(wptStableScores?.firefox?.score) || '');
            break;
          case ColumnKey.StableSafari:
            row.push(browserImpl?.safari?.date || '');
            row.push(String(wptStableScores?.safari?.score) || '');
            break;
        }
      }
      return row;
    });

    const csv = convertToCSV(columns, rows);

    // Create blob to download the csv.
    const blob = new Blob([csv], {type: 'text/csv'});
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = 'webstatus-feature-overview.csv';
    link.click();
  }

  async _fetchFeatures(
    apiClient: APIClient | undefined,
    routerLocation: {search: string}
  ): Promise<components['schemas']['FeaturePage']> {
    if (typeof apiClient !== 'object')
      return Promise.reject(new Error('APIClient is not initialized.'));
    const sortSpec = getSortSpec(routerLocation) as FeatureSortOrderType;
    const searchQuery = getSearchQuery(routerLocation) as FeatureSearchType;
    const offset = getPaginationStart(routerLocation);
    const pageSize = getPageSize(routerLocation);
    const wptMetricView = getWPTMetricView(
      routerLocation
    ) as FeatureWPTMetricViewType;
    return apiClient.getFeatures(
      searchQuery,
      sortSpec,
      wptMetricView,
      offset,
      pageSize
    );
  }

  render(): TemplateResult {
    return html`
      <webstatus-overview-content
        .location=${this.location}
        .taskTracker=${this.taskTracker}
      >
      </webstatus-overview-content>
    `;
  }
}
