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
import {LitElement, type TemplateResult, html, PropertyValueMap} from 'lit';
import {customElement, state, property} from 'lit/decorators.js';
import {type components} from 'webstatus.dev-backend';

import {
  getPageSize,
  getPaginationStart,
  getSortSpec,
  getWPTMetricView,
} from '../utils/urls.js';
import {
  type APIClient,
  type FeatureSortOrderType,
  FeatureWPTMetricViewType,
} from '../api/client.js';
import {apiClientContext} from '../contexts/api-client-context.js';
import './webstatus-overview-content.js';
import {TaskTracker} from '../utils/task-tracker.js';
import {ApiError, UnknownError} from '../api/errors.js';
import {toast} from '../utils/toast.js';
import {
  appBookmarkInfoContext,
  AppBookmarkInfo,
  savedSearchHelpers,
} from '../contexts/app-bookmark-info-context.js';

@customElement('webstatus-overview-page')
export class OverviewPage extends LitElement {
  loadingTask: Task;

  @consume({context: apiClientContext})
  @state()
  apiClient?: APIClient;

  @state()
  taskTracker: TaskTracker<components['schemas']['FeaturePage'], ApiError> = {
    status: TaskStatus.INITIAL, // Initial state
    error: undefined,
    data: undefined,
  };

  @property({type: Object})
  location!: {search: string}; // Set by router.

  @state()
  currentLocation?: {search: string};

  @consume({context: appBookmarkInfoContext, subscribe: true})
  @state()
  appBookmarkInfo?: AppBookmarkInfo;

  constructor() {
    super();

    this.loadingTask = new Task(this, {
      autoRun: false,
      args: () =>
        [this.apiClient, this.location, this.appBookmarkInfo] as const,
      task: async ([apiClient, routerLocation, appBookmarkInfo]): Promise<
        components['schemas']['FeaturePage']
      > => {
        if (this.location.search !== this.currentLocation?.search) {
          // Reset taskTracker here due to a Task data cache issue.
          this.taskTracker = {
            status: TaskStatus.INITIAL,
            error: undefined,
            data: undefined,
          };
          this.currentLocation = this.location;
          return this._fetchFeatures(
            apiClient,
            routerLocation,
            appBookmarkInfo,
          );
        }
        return this.taskTracker.data ?? {metadata: {total: 0}, data: []};
      },
      onComplete: page => {
        this.taskTracker = {
          status: TaskStatus.COMPLETE,
          error: undefined,
          data: page,
        };
      },
      onError: async (error: unknown) => {
        if (error instanceof ApiError) {
          this.taskTracker = {
            status: TaskStatus.ERROR,
            error: error,
            data: undefined,
          };
          await toast(`${error.message}`, 'danger', 'exclamation-triangle');
        } else {
          // Should never reach here but let's handle it.
          this.taskTracker = {
            status: TaskStatus.ERROR,
            error: new UnknownError('unknown error fetching features'),
            data: undefined,
          };
        }
      },
    });
  }
  protected willUpdate(changedProperties: PropertyValueMap<this>): void {
    if (
      changedProperties.has('apiClient') ||
      changedProperties.has('location') ||
      changedProperties.has('appBookmarkInfo')
    ) {
      if (
        this.apiClient !== undefined &&
        this.currentLocation?.search !== this.location.search &&
        !savedSearchHelpers.isBusyLoadingSavedSearchInfo(
          this.appBookmarkInfo,
          this.location,
        )
      ) {
        void this.loadingTask.run();
      }
    }
  }

  async _fetchFeatures(
    apiClient: APIClient | undefined,
    routerLocation: {search: string},
    appBookmarkInfo?: AppBookmarkInfo,
  ): Promise<components['schemas']['FeaturePage']> {
    if (typeof apiClient !== 'object')
      return Promise.reject(new Error('APIClient is not initialized.'));
    const sortSpec = getSortSpec(routerLocation) as FeatureSortOrderType;
    let searchQuery: string = '';
    const query = savedSearchHelpers.getCurrentQuery(
      appBookmarkInfo,
      routerLocation,
    );
    if (query) {
      searchQuery = query;
    }
    const offset = getPaginationStart(routerLocation);
    const pageSize = getPageSize(routerLocation);
    const wptMetricView = getWPTMetricView(
      routerLocation,
    ) as FeatureWPTMetricViewType;
    return apiClient.getFeatures(
      searchQuery,
      sortSpec,
      wptMetricView,
      offset,
      pageSize,
    );
  }

  render(): TemplateResult {
    return html`
      <webstatus-overview-content
        .location=${this.location}
        .taskTracker=${this.taskTracker}
        .appBookmarkInfo=${this.appBookmarkInfo}
      >
      </webstatus-overview-content>
    `;
  }
}
