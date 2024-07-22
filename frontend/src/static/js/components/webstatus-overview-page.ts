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
import {downloadCSV} from '../utils/csv.js';

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
import {toast} from '../utils/toast.js';

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
  // A function that returns an array of all features via apiClient.getAllFeatures
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
          toast(`${error.message}`, 'danger', 'exclamation-triangle');
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
    this.addEventListener('exportToCSV', event => {
      const {detail} = event as CustomEvent<{
        callback: (() => void) | undefined;
      }>;
      this.exportToCSV(detail.callback);
    });
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

        const pushBrowserName = (name: string) => {
          columns.push(name);
          const pattern =
            /^Browser Implementation in (.+?)(?: (Experimental))?$/;
          const match = name.match(pattern);
          if (match) {
            const browser = match[1];
            const channel = match[2] ? 'Experimental' : 'Stable';
            const wptName = `${browser} WPT ${channel} Score`;
            columns.push(wptName);
          }
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

        downloadCSV(columns, rows, 'webstatus-feature-overview.csv')
          .then(() => {
            if (completedCallback) completedCallback();
          })
          .catch(error => {
            toast(
              `Save file error: ${error.message}`,
              'danger',
              'exclamation-triangle'
            );
          });
      })
      .catch(error => {
        toast(
          `Download error: ${error.message}`,
          'danger',
          'exclamation-triangle'
        );
      });
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
