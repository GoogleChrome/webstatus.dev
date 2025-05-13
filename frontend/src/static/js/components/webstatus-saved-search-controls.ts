/**
 * Copyright 2025 Google LLC
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

import {LitElement, TemplateResult, css, html, nothing} from 'lit';
import {customElement, property, query, state} from 'lit/decorators.js';
import {APIClient} from '../contexts/api-client-context.js';
import {User} from '../contexts/firebase-user-context.js';
import {
  BookmarkOwnerRole,
  BookmarkStatusActive,
  SavedSearchOperationType,
  UserSavedSearch,
} from '../utils/constants.js';
import {WebstatusSavedSearchEditor} from './webstatus-saved-search-editor.js';

import './webstatus-saved-search-editor.js';
import {formatOverviewPageUrl, getOrigin} from '../utils/urls.js';
import {ifDefined} from 'lit/directives/if-defined.js';
import {Task, TaskStatus} from '@lit/task';
import {ApiError} from '../api/errors.js';
import {Toast} from '../utils/toast.js';
import {type WebstatusTypeahead} from './webstatus-typeahead.js';

@customElement('webstatus-saved-search-controls')
export class WebstatusSavedSearchControls extends LitElement {
  @property({type: Object})
  apiClient!: APIClient;

  @property({type: Object})
  user!: User;

  @property({type: Object})
  savedSearch?: UserSavedSearch;

  @property({type: Object})
  overviewPageQueryInput?: WebstatusTypeahead;

  @property({type: Object})
  location!: {search: string};

  @query('webstatus-saved-search-editor')
  savedSearchEditor!: WebstatusSavedSearchEditor;

  @state()
  private _bookmarkTask?: Task;

  // Members that are used for testing with sinon.
  _getOrigin: () => string = getOrigin;
  _formatOverviewPageUrl: (
    location: {search: string},
    overrides: {search_id?: string},
  ) => string = formatOverviewPageUrl;

  static styles = css`
    #bookmark-task-spinner {
      font-size: 1rem;
    }
  `;

  openSavedSearch(
    type: SavedSearchOperationType,
    savedSearch?: UserSavedSearch,
    overviewPageQueryInput?: string,
  ) {
    const event = new CustomEvent('open-saved-search-editor', {
      detail: {
        type,
        savedSearch,
        overviewPageQueryInput,
      },
      bubbles: true,
      composed: true,
    });
    this.dispatchEvent(event);
  }

  async handleBookmarkSavedSearch(
    savedSearch: UserSavedSearch,
    isBookmarkStatusActive: boolean,
  ) {
    this._bookmarkTask = new Task(this, {
      autoRun: false,
      task: async ([
        apiClient,
        user,
        savedSearchID,
        isBookmarkStatusActive,
      ]) => {
        const token = await user.getIdToken();
        if (isBookmarkStatusActive) {
          await apiClient.removeUserSavedSearchBookmark(savedSearchID, token);
        } else {
          await apiClient.putUserSavedSearchBookmark(savedSearchID, token);
        }

        // Return the new bookmark status
        return !isBookmarkStatusActive;
      },
      args: () => [
        this.apiClient,
        this.user,
        savedSearch.id,
        isBookmarkStatusActive,
      ],
      onComplete: async isNewBookmarkStatusActive => {
        if (isNewBookmarkStatusActive) {
          savedSearch.bookmark_status = {
            status: BookmarkStatusActive,
          };
          this.dispatchEvent(
            new CustomEvent('saved-search-bookmarked', {
              detail: savedSearch,
              bubbles: true,
              composed: true,
            }),
          );
        } else {
          this.dispatchEvent(
            new CustomEvent('saved-search-unbookmarked', {
              detail: savedSearch.id,
              bubbles: true,
              composed: true,
            }),
          );
        }
      },
      async onError(error: unknown) {
        let message: string;
        if (error instanceof ApiError) {
          message = error.message;
        } else {
          message =
            'Unknown error toggling bookmark status. Check console for details.';
          console.error(error);
        }
        await new Toast().toast(message, 'danger', 'exclamation-triangle');
      },
    });
    // Run manually to prevent changes to any variables from retriggering the task.
    await this._bookmarkTask.run();
  }

  renderBookmarkControl(
    savedSearch: UserSavedSearch,
    isOwner: boolean,
  ): TemplateResult {
    let bookmarkStatusIcon: 'star-fill' | 'star' = 'star';
    let bookmarkTooltipText: string = 'Bookmark this search';
    let bookmarkTooltipLabel: string = 'Bookmark';
    let bookmarkButtonDisabled: boolean = false;
    let isBookmarkStatusActive: boolean = false;
    if (savedSearch.bookmark_status?.status === BookmarkStatusActive) {
      bookmarkStatusIcon = 'star-fill';
      bookmarkTooltipText = 'Unbookmark this search';
      bookmarkTooltipLabel = 'Unbookmark';
      isBookmarkStatusActive = true;
    }
    if (isOwner) {
      bookmarkButtonDisabled = true;
      bookmarkTooltipText =
        'Users cannot remove the bookmark for saved searches they own';
    }
    const inProgress = this._bookmarkTask?.status === TaskStatus.PENDING;
    return html`
      <sl-tooltip content="${bookmarkTooltipText}">
        <sl-icon-button
          name="${bookmarkStatusIcon}"
          label="${bookmarkTooltipLabel}"
          .disabled=${bookmarkButtonDisabled || inProgress}
          @click=${ifDefined(isOwner)
            ? undefined
            : () =>
                this.handleBookmarkSavedSearch(
                  savedSearch,
                  isBookmarkStatusActive,
                )}
        ></sl-icon-button>
      </sl-tooltip>
      ${inProgress
        ? html`<sl-spinner id="bookmark-task-spinner"></sl-spinner>`
        : nothing}
    `;
  }

  renderActiveSavedSearchControls(
    savedSearch: UserSavedSearch,
  ): TemplateResult {
    const isOwner = savedSearch.permissions?.role === BookmarkOwnerRole;
    const shareableUrl = `${this._getOrigin()}${this._formatOverviewPageUrl(this.location, {search_id: savedSearch.id})}`;
    return html`
      <sl-copy-button
        value="${shareableUrl}"
        copy-label="Copy saved search URL to clipboard"
        success-label="Copied"
        error-label="Whoops, your browser doesn't support this!"
        ><sl-icon-button
          slot="copy-icon"
          name="share"
          label="Copy"
        ></sl-icon-button
        ><sl-icon-button
          slot="success-icon"
          name="share-fill"
          label="Copy Success"
        ></sl-icon-button>
      </sl-copy-button>
      ${this.renderBookmarkControl(savedSearch, isOwner)}
      ${isOwner
        ? html`
            <sl-tooltip content="Edit current saved search">
              <sl-icon-button
                name="pencil"
                label="Edit"
                @click=${() =>
                  this.openSavedSearch(
                    'edit',
                    this.savedSearch,
                    this.overviewPageQueryInput?.value,
                  )}
              ></sl-icon-button>
            </sl-tooltip>
            <sl-tooltip content="Delete saved search">
              <sl-icon-button
                name="trash"
                label="Delete"
                @click=${() => this.openSavedSearch('delete', this.savedSearch)}
              ></sl-icon-button>
            </sl-tooltip>
          `
        : nothing}
    `;
  }

  render() {
    return html`
      <div slot="anchor" class="popup-anchor saved-search-controls"></div>
      <div class="popup-content">
        <sl-tooltip content="Create a new saved search">
          <sl-icon-button
            name="floppy"
            data-testid="saved-search-save-button"
            label="Save"
            @click=${() => {
              this.openSavedSearch(
                'save',
                undefined,
                this.overviewPageQueryInput?.value,
              );
            }}
          ></sl-icon-button>
        </sl-tooltip>
        ${this.savedSearch !== undefined
          ? this.renderActiveSavedSearchControls(this.savedSearch)
          : nothing}
      </div>
    `;
  }
}
