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
  getLegacySearchID,
  getSearchQuery,
  updatePageUrl,
} from '../utils/urls.js';
import {
  type APIClient,
  type FeatureSortOrderType,
  FeatureWPTMetricViewType,
  isFeatureSortOrderType,
  isWPTMetricViewType,
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
  SavedSearchScope,
} from '../contexts/app-bookmark-info-context.js';
import {UserSavedSearch} from '../utils/constants.js';

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

  _lastUserSavedSearch?: UserSavedSearch;

  constructor() {
    super();

    this.loadingTask = new Task(this, {
      autoRun: false,
      args: () =>
        [this.apiClient, this.location, this.appBookmarkInfo] as const,
      task: async ([apiClient, routerLocation, appBookmarkInfo]): Promise<
        components['schemas']['FeaturePage']
      > => {
        this.taskTracker = {
          status: TaskStatus.INITIAL,
          error: undefined,
          data: undefined,
        };
        this.currentLocation = this.location;
        return this._fetchFeatures(apiClient, routerLocation, appBookmarkInfo);
      },
      onComplete: page => {
        this.taskTracker = {
          status: TaskStatus.COMPLETE,
          error: undefined,
          data: page,
        };
      },
      // eslint-disable-next-line @typescript-eslint/no-restricted-types
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

  areUserSavedSearchesDifferent(
    current?: UserSavedSearch,
    incoming?: UserSavedSearch,
  ): boolean {
    // Could be a completely different search or a more recently updated search
    return (
      current?.id !== incoming?.id ||
      current?.updated_at !== incoming?.updated_at
    );
  }

  #upgradeLegacyUrl(): boolean {
    if (!this.appBookmarkInfo || !this.location) return false;

    const legacyId = getLegacySearchID(this.location);
    const globalSearches = this.appBookmarkInfo.globalSavedSearches || [];

    if (legacyId !== '') {
      const isGlobal = globalSearches.some(s => s.id === legacyId);
      const newQ = isGlobal ? `hotlist:${legacyId}` : `saved:${legacyId}`;
      updatePageUrl(window.location.pathname, this.location, {q: newQ});
      return true;
    }

    const currentQuery = getSearchQuery(this.location);
    const matchGlobal = globalSearches.find(
      s => s.query.trim() !== '' && s.query === currentQuery,
    );

    if (
      matchGlobal &&
      !currentQuery.startsWith('hotlist:') &&
      !currentQuery.startsWith('saved:')
    ) {
      updatePageUrl(window.location.pathname, this.location, {
        q: `hotlist:${matchGlobal.id}`,
      });
      return true;
    }

    return false;
  }

  protected willUpdate(changedProperties: PropertyValueMap<this>): void {
    if (
      changedProperties.has('apiClient') ||
      changedProperties.has('appBookmarkInfo')
    ) {
      if (this.apiClient === undefined) {
        return;
      }

      if (
        !savedSearchHelpers.isBusyLoadingSavedSearchInfo(this.appBookmarkInfo)
      ) {
        if (this.#upgradeLegacyUrl()) {
          return; // Wait for the URL change to trigger the next update
        }
      }

      const incomingCurrentSavedSearch =
        savedSearchHelpers.getCurrentSavedSearch(this.appBookmarkInfo);
      const userSavedSearch =
        incomingCurrentSavedSearch?.scope === SavedSearchScope.UserSavedSearch
          ? incomingCurrentSavedSearch.value
          : undefined;
      if (
        (this.currentLocation?.search !== this.location.search ||
          this.areUserSavedSearchesDifferent(
            this._lastUserSavedSearch,
            userSavedSearch,
          )) &&
        !savedSearchHelpers.isBusyLoadingSavedSearchInfo(this.appBookmarkInfo)
      ) {
        this._lastUserSavedSearch = userSavedSearch;
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
    let sortSpec: FeatureSortOrderType | undefined = undefined;
    const sortSpecRaw = getSortSpec(routerLocation);
    if (isFeatureSortOrderType(sortSpecRaw)) {
      sortSpec = sortSpecRaw;
    }
    let searchQuery: string = '';
    const query = savedSearchHelpers.getCurrentQuery(appBookmarkInfo);
    if (query) {
      searchQuery = query;
    }
    const offset = getPaginationStart(routerLocation);
    const pageSize = getPageSize(routerLocation);

    let wptMetricView: FeatureWPTMetricViewType | undefined = undefined;
    const wptMetricViewRaw = getWPTMetricView(routerLocation);
    if (isWPTMetricViewType(wptMetricViewRaw)) {
      wptMetricView = wptMetricViewRaw;
    }

    const resp = await apiClient.getFeatures(
      searchQuery,
      sortSpec,
      wptMetricView,
      offset,
      pageSize,
    );
    return {
      metadata: {
        total: resp.metadata.total || 0,
        next_page_token: resp.metadata.next_page_token,
      },
      data: resp.data,
    };
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
