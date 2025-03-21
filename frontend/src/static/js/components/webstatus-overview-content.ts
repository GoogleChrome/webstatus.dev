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

import {
  LitElement,
  type TemplateResult,
  CSSResultGroup,
  css,
  html,
  nothing,
  PropertyValues,
} from 'lit';
import {Task, TaskStatus} from '@lit/task';
import {customElement, property, state} from 'lit/decorators.js';
import {type components} from 'webstatus.dev-backend';

import './webstatus-overview-filters.js';
import './webstatus-overview-table.js';
import './webstatus-overview-pagination.js';
import {SHARED_STYLES} from '../css/shared-css.js';
import {TaskTracker} from '../utils/task-tracker.js';
import {ApiError} from '../api/errors.js';
import {consume} from '@lit/context';
import {
  WebFeatureProgress,
  webFeatureProgressContext,
} from '../contexts/webfeature-progress-context.js';
import {Toast} from '../utils/toast.js';
import {getSearchID, getSearchQuery} from '../utils/urls.js';
import {Bookmark} from '../utils/constants.js';
import {
  AppBookmarkInfo,
  appBookmarkInfoContext,
} from '../contexts/app-bookmark-info-context.js';
import {APIClient, apiClientContext} from '../contexts/api-client-context.js';

const webFeaturesRepoUrl = 'https://github.com/web-platform-dx/web-features';

@customElement('webstatus-overview-content')
export class WebstatusOverviewContent extends LitElement {
  @property({type: Object})
  taskTracker: TaskTracker<components['schemas']['FeaturePage'], ApiError> = {
    status: TaskStatus.INITIAL, // Initial state
    error: null,
    data: null,
  };

  @property({type: Object})
  location!: {search: string}; // Set by parent.

  @consume({context: webFeatureProgressContext, subscribe: true})
  @state()
  webFeaturesProgress?: WebFeatureProgress;

  @consume({context: appBookmarkInfoContext, subscribe: true})
  @state()
  appBookmarkInfo: AppBookmarkInfo = {};

  @state()
  currentBookmark?: Bookmark;

  @consume({context: apiClientContext})
  apiClient?: APIClient;

  savedSearchTask: Task = new Task(this, {
    args: () => [this.location, this.apiClient] as const,
    task: async ([location, apiClient]) => {
      const searchID = getSearchID(location);
      if (!searchID || !apiClient) {
        return;
      }
      return await apiClient.getSavedSearchByID(searchID);
    },
    onError: async (error: unknown) => {
      let msg: string;
      if (error instanceof ApiError) {
        msg = error.message;
      } else {
        msg = 'Unknown message. Check console for details.';
        console.error(error);
      }
      const searchID = getSearchID(location);
      await new Toast().toast(
        `Error fetching saved search ID ${searchID}: ${msg}`,
        'danger',
        'exclamation-triangle',
      );
    },
    onComplete: async search => {
      if (!search) {
        return;
      }
      this.currentBookmark = search;
    },
  });

  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        .header-line {
          gap: 1em;
        }
        .stats-summary {
          color: var(--unimportant-text-color);
          margin-right: var(--content-padding);
        }
        .overview-description {
          color: var(--unimportant-text-color);
          margin-bottom: var(--content-padding);
        }
      `,
    ];
  }

  willUpdate(changedProperties: PropertyValues<this>) {
    if (changedProperties.has('appBookmarkInfo')) {
      // If we have a search ID, check the user saved searches first
      const searchID = getSearchID(this.location);
      if (searchID) {
        if (
          this.appBookmarkInfo.userSavedBookmarks?.status ===
          TaskStatus.COMPLETE
        ) {
          // Search for the search ID in the user saved searches
          const bookmark = this.appBookmarkInfo.userSavedBookmarks.data?.find(
            bookmark => bookmark.id === searchID,
          );
          if (bookmark) {
            this.currentBookmark = bookmark;
            return;
          }
          //
        }
        // TODO: Handle the status == ERROR case.
      }
    }
    if (this.currentBookmark === undefined) {
      // Otherwise, try to find the bookmark by query as long as it wasn't updated itself
      this.currentBookmark = this.getBookmarkFromQuery();
    }
  }

  getBookmarkFromQuery(): Bookmark | undefined {
    const currentQuery = getSearchQuery(this.location);
    return this.appBookmarkInfo?.globalBookmarks?.find(
      bookmark => bookmark.query === currentQuery,
    );
  }

  renderMappingPercentage(): TemplateResult {
    if (
      this.webFeaturesProgress === undefined ||
      this.webFeaturesProgress.isDisabled
    ) {
      return html``;
    }
    if (this.webFeaturesProgress.error) {
      // Temporarily to avoid the no-floating-promises error.
      void new Toast().toast(
        this.webFeaturesProgress.error,
        'danger',
        'exclamation-triangle',
      );
      return html``;
    }
    return html`Percentage of features mapped:&nbsp;
      <a href="${webFeaturesRepoUrl}">
        ${
          this.webFeaturesProgress.bcdMapProgress
            ? this.webFeaturesProgress.bcdMapProgress
            : 0 // The else case that returns 0 should not happen.
        }%
      </a>`;
  }

  renderCount(): TemplateResult {
    switch (this.taskTracker.status) {
      case TaskStatus.INITIAL:
      case TaskStatus.PENDING:
        return html`Loading features...`;
      case TaskStatus.COMPLETE:
        return html`
          <span class="stats-summary">
            ${this.taskTracker.data?.metadata.total ?? 0} features
          </span>
        `;
      case TaskStatus.ERROR:
        return html`Failed to load features`;
    }
  }

  render(): TemplateResult {
    const bookmark = this.currentBookmark;
    const pageTitle = bookmark ? bookmark.name : 'Features overview';
    const pageDescription = bookmark?.description;
    return html`
      <div class="main">
        <div class="hbox halign-items-space-between header-line">
          <h1 class="halign-stretch" id="overview-title">${pageTitle}</h1>
        </div>
        ${pageDescription
          ? html`<div class="hbox wrap" id="overview-description">
              <h3>${pageDescription}</h3>
            </div>`
          : nothing}
        <div class="hbox wrap">
          ${this.renderCount()}
          <div class="spacer"></div>
          <div id="mapping-percentage" class="hbox wrap">
            ${this.renderMappingPercentage()}
          </div>
        </div>
        <br />
        <webstatus-overview-filters
          .location=${this.location}
          .bookmark=${bookmark}
        ></webstatus-overview-filters>
        <br />

        <webstatus-overview-table
          .location=${this.location}
          .taskTracker=${this.taskTracker}
          .bookmark=${bookmark}
        >
        </webstatus-overview-table>
        <webstatus-overview-pagination
          .location=${this.location}
          .totalCount=${this.taskTracker.data?.metadata.total ?? 0}
        ></webstatus-overview-pagination>
      </div>
    `;
  }
}
