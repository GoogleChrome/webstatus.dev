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
  BROWSER_ID_TO_LABEL,
  CHANNEL_ID_TO_LABEL,
} from '../api/client.js';
import {apiClientContext} from '../contexts/api-client-context.js';
import './webstatus-overview-content.js';
import {TaskTracker} from '../utils/task-tracker.js';
import {ApiError, UnknownError} from '../api/errors.js';
import {CELL_DEFS} from './webstatus-overview-cells.js';
import {
  ColumnKey,
  parseColumnsSpec,
  BrowserChannelColumnKeys,
} from './webstatus-overview-cells.js';
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

        const pushBrowserChannelName = (
          browserColumnKey: BrowserChannelColumnKeys
        ) => {
          const name = CELL_DEFS[browserColumnKey].nameInDialog;
          columns.push(name);

          const browser = CELL_DEFS[browserColumnKey].options.browser!;
          const browserLabel = BROWSER_ID_TO_LABEL[browser];
          const channel = CELL_DEFS[browserColumnKey].options.channel!;
          const channelLabel = CHANNEL_ID_TO_LABEL[channel];

          const wptName = `${browserLabel} WPT ${channelLabel} Score`;
          columns.push(wptName);
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
          const row: string[] = [];

          const pushBrowserChannelValue = (
            browserColumnKey: BrowserChannelColumnKeys
          ) => {
            const browser = CELL_DEFS[browserColumnKey].options.browser!;
            const channel = CELL_DEFS[browserColumnKey].options.channel!;
            const wptScore = feature.wpt?.[channel]?.[browser];
            row.push(browserImpl[browser]?.date || '');
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
