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
import {toast} from '../utils/toast.js';
import '../services/webstatus-webfeature-progress-service.js';

@customElement('webstatus-overview-page')
export class OverviewPage extends LitElement {
  loadingTask: Task;

  @consume({context: apiClientContext})
  @state()
  apiClient?: APIClient;

  @state()
  taskTracker: TaskTracker<components['schemas']['FeaturePage'], ApiError> = {
    status: TaskStatus.INITIAL, // Initial state
    error: null,
    data: null,
  };

  @state()
  location!: {search: string}; // Set by router.

  constructor() {
    super();

    this.loadingTask = new Task(this, {
      args: () =>
        [this.apiClient, this.location] as [APIClient, {search: string}],
      task: async ([apiClient, routerLocation]): Promise<
        components['schemas']['FeaturePage']
      > => {
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
      <webstatus-webfeature-progress-service>
        <webstatus-overview-content
          .location=${this.location}
          .taskTracker=${this.taskTracker}
        >
        </webstatus-overview-content>
      </webstatus-webfeature-progress-service>
    `;
  }
}
