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
  PropertyValueMap,
} from 'lit';
import {TaskStatus} from '@lit/task';
import {customElement, property, query, state} from 'lit/decorators.js';
import {type components} from 'webstatus.dev-backend';

import './webstatus-overview-data-loader.js';
import './webstatus-overview-filters.js';
import './webstatus-overview-pagination.js';
import './webstatus-subscribe-button.js';
import {SHARED_STYLES} from '../css/shared-css.js';
import {TaskTracker} from '../utils/task-tracker.js';
import {ApiError} from '../api/errors.js';
import {
  AppBookmarkInfo,
  savedSearchHelpers,
  SavedSearchScope,
} from '../contexts/app-bookmark-info-context.js';
import {consume} from '@lit/context';
import {
  UserContext,
  firebaseUserContext,
} from '../contexts/firebase-user-context.js';
import {APIClient, apiClientContext} from '../contexts/api-client-context.js';
import {WebstatusSavedSearchEditor} from './webstatus-saved-search-editor.js';
import {
  formatOverviewPageUrl,
  getEditSavedSearch,
  getOrigin,
  updatePageUrl,
} from '../utils/urls.js';
import {
  OpenSavedSearchEvent,
  SavedSearchOperationType,
  UserSavedSearch,
} from '../utils/constants.js';

@customElement('webstatus-overview-content')
export class WebstatusOverviewContent extends LitElement {
  @property({type: Object})
  taskTracker: TaskTracker<components['schemas']['FeaturePage'], ApiError> = {
    status: TaskStatus.INITIAL, // Initial state
    error: undefined,
    data: undefined,
  };

  @property({type: Object})
  location!: {search: string}; // Set by parent.

  @property({type: Object})
  appBookmarkInfo?: AppBookmarkInfo;

  @property({type: String})
  activeQuery: string = '';

  @property({type: Object})
  savedSearch?: UserSavedSearch;

  @consume({context: apiClientContext})
  @state()
  apiClient?: APIClient;

  @consume({context: firebaseUserContext, subscribe: true})
  @state()
  userContext: UserContext | null | undefined;

  @query('webstatus-saved-search-editor')
  savedSearchEditor!: WebstatusSavedSearchEditor;

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

  connectedCallback(): void {
    super.connectedCallback();
    this.openSavedSearch = this.openSavedSearch.bind(this);
    this.addEventListener('open-saved-search-editor', this.openSavedSearch);
  }

  disconnectedCallback() {
    this.removeEventListener('open-saved-search-editor', this.openSavedSearch);
    super.disconnectedCallback();
  }

  async openSavedSearchDialog(
    type: SavedSearchOperationType,
    savedSearch?: UserSavedSearch,
    overviewPageQueryInput?: string,
  ) {
    this.savedSearch = savedSearch;
    void this.savedSearchEditor.open(type, savedSearch, overviewPageQueryInput);
  }

  async openSavedSearch(e: CustomEventInit<OpenSavedSearchEvent>) {
    void this.openSavedSearchDialog(
      e.detail!.type,
      e.detail!.savedSearch,
      e.detail!.overviewPageQueryInput,
    );
  }

  protected willUpdate(changedProperties: PropertyValueMap<this>): void {
    if (
      changedProperties.has('location') ||
      changedProperties.has('appBookmarkInfo')
    ) {
      this.activeQuery = savedSearchHelpers.getCurrentQuery(
        this.appBookmarkInfo,
      );
      const search = savedSearchHelpers.getCurrentSavedSearch(
        this.appBookmarkInfo,
      );
      // Allow resetting of active search.
      if (search === undefined) {
        this.savedSearch = undefined;
      }
      // If the search is a user search, store it. Ignore Global Saved Searches
      if (search?.scope === SavedSearchScope.UserSavedSearch) {
        this.savedSearch = search.value;
      }
    }
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

  // Members that are used for testing with sinon.
  _getOrigin: () => string = getOrigin;
  _getEditSavedSearch: (location: {search: string}) => boolean =
    getEditSavedSearch;
  _updatePageUrl: (
    pathname: string,
    location: {search: string},
    overrides: {edit_saved_search?: boolean},
  ) => void = updatePageUrl;
  _formatOverviewPageUrl: (
    location: {search: string},
    overrides: {search_id?: string},
  ) => string = formatOverviewPageUrl;

  protected async updated(
    _changedProperties: PropertyValueMap<this>,
  ): Promise<void> {
    if (
      this._getEditSavedSearch(this.location) &&
      !this.savedSearchEditor.isOpen() &&
      this.savedSearch
    ) {
      void this.openSavedSearchDialog(
        'edit',
        this.savedSearch,
        this.savedSearch.query,
      );
      this._updatePageUrl('', this.location, {edit_saved_search: undefined});
    }
  }

  render(): TemplateResult {
    const savedSearch = savedSearchHelpers.getCurrentSavedSearch(
      this.appBookmarkInfo,
    );
    const pageTitle = savedSearch
      ? savedSearch.value.name
      : 'Features overview';
    const pageDescription = savedSearch?.value.description;
    const userSavedSearch =
      savedSearch?.scope === SavedSearchScope.UserSavedSearch
        ? savedSearch
        : undefined;

    return html` <div class="main">
        <div class="hbox halign-items-space-between header-line">
          <h1 class="halign-stretch" id="overview-title">${pageTitle}</h1>
          ${userSavedSearch
            ? html`<webstatus-subscribe-button
                .savedSearchId=${userSavedSearch.value.id}
              ></webstatus-subscribe-button>`
            : nothing}
        </div>
        ${pageDescription
          ? html`<div class="hbox wrap" id="overview-description">
              <h3>${pageDescription}</h3>
            </div>`
          : nothing}
        <div class="hbox">${this.renderCount()}</div>
        <br />
        <webstatus-overview-filters
          .location=${this.location}
          .appBookmarkInfo=${this.appBookmarkInfo}
          .activeQuery=${this.activeQuery}
          .savedSearch=${userSavedSearch?.value}
          .userContext=${this.userContext}
          .apiClient=${this.apiClient}
        ></webstatus-overview-filters>
        <br />

        <webstatus-overview-data-loader
          .location=${this.location}
          .taskTracker=${this.taskTracker}
          .savedSearch=${savedSearch}
        >
        </webstatus-overview-data-loader>
        <webstatus-overview-pagination
          .location=${this.location}
          .totalCount=${this.taskTracker.data?.metadata.total ?? 0}
        ></webstatus-overview-pagination>
      </div>
      <webstatus-saved-search-editor
        .apiClient=${this.apiClient!}
        .userContext=${this.userContext!}
        .savedSearch=${userSavedSearch?.value}
        .location=${this.location}
      ></webstatus-saved-search-editor>`;
  }
}
