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

import {
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
  location!: { search: string; }; // Set by router.

  @state()
    // allFeaturesFetcher is either undefined or a function that returns
    // an array of all features via apiClient.getAllFeatures
  allFeaturesFetcher: undefined |
    (() => Promise<components['schemas']['Feature'][]>)
   = undefined;

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
            getWPTMetricView(routerLocation) as FeatureWPTMetricViewType,
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



  renderExportButton(): TemplateResult {
    const convertToCSV = (features: components['schemas']['Feature'][]):
      string => {
      const header = [
        'Feature',
        'Baseline status',
        'Browser Impl in Chrome',
        'Browser Impl in Edge',
        'Browser Impl in Firefox',
        'Browser Impl in Safari',
        'Browser Impl in Chrome Experimental',
        'Browser Impl in Edge Experimental',
        'Browser Impl in Firefox Experimental',
        'Browser Impl in Safari Experimental',
      ];
      const rows = features.map(feature => {
        const baselineStatus = feature.baseline!.status;
        const browserImpl = feature.browser_implementations!;
        return [
          feature.name,
          baselineStatus,
          browserImpl.chrome,
          browserImpl.edge,
          browserImpl.firefox,
          browserImpl.safari,
          browserImpl.chrome_experimental,
          browserImpl.edge_experimental,
          browserImpl.firefox_experimental,
          browserImpl.safari_experimental,
        ];
      });
      const csv = [header, ...rows].map(row => row.join(',')).join('\n');
      return csv;
    };

    const exportToCSV = async (): Promise<void> => {
      if (this.allFeaturesFetcher) {
        // Fetch all pages of data via getAllFeatures
        const allFeatures = await this.allFeaturesFetcher();

        // Convert data to csv
        const csv = convertToCSV(allFeatures);

        // Create blob to download the csv.
        const blob = new Blob([csv], { type: 'text/csv' });
        const url = window.URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.download = 'web-platform-tests-status.csv';
        link.click();
      }
    };

    return html`
      <sl-button
        @click=${exportToCSV}
      >
        <sl-icon slot="prefix" name="download"></sl-icon>
        Export to CSV
      </sl-button>
    `;
  }

  render(): TemplateResult {
    return html`
      ${ this.renderExportButton() }
      <webstatus-overview-content
        .location=${this.location}
        .taskTracker=${this.taskTracker}
      >
      </webstatus-overview-content>
    `;
  }
}
