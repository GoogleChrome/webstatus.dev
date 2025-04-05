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

import {LitElement, html, nothing} from 'lit';
import {customElement, property, query} from 'lit/decorators.js';
import {APIClient} from '../contexts/api-client-context.js';
import {User} from '../contexts/firebase-user-context.js';
import {
  BookmarkOwnerRole,
  BookmarkStatusActive,
  UserSavedSearch,
} from '../utils/constants.js';
import {WebstatusSavedSearchEditor} from './webstatus-saved-search-editor.js';

import './webstatus-saved-search-editor.js';

@customElement('webstatus-saved-search-controls')
export class WebstatusSavedSearchControls extends LitElement {
  @property({type: Object})
  apiClient!: APIClient;

  @property({type: Object})
  user!: User;

  @property({type: Object})
  savedSearch?: UserSavedSearch;

  @query('webstatus-saved-search-editor')
  savedSearchEditor!: WebstatusSavedSearchEditor;

  async openNewSavedSearchDialog() {
    await this.savedSearchEditor.open('save', undefined);
  }

  async openEditSavedSearchDialog() {
    await this.savedSearchEditor.open('edit', this.savedSearch);
  }

  async openDeleteSavedSearchDialog() {
    await this.savedSearchEditor.open('delete', this.savedSearch);
  }

  render() {
    let bookmarkStatusIcon: 'star-fill' | 'star' = 'star';
    let bookmarkTooltipText: string = 'Bookmark the saved search';
    let bookmarkTooltipLabel: string = 'Bookmark';
    let bookmarkButtonDisabled: boolean = false;
    if (this.savedSearch?.bookmark_status?.status === BookmarkStatusActive) {
      bookmarkStatusIcon = 'star-fill';
      bookmarkTooltipText = 'Unbookmark the saved search';
      bookmarkTooltipLabel = 'Unbookmark';
    }
    const isOwner = this.savedSearch?.permissions?.role === BookmarkOwnerRole;
    if (isOwner) {
      bookmarkButtonDisabled = true;
      bookmarkTooltipText =
        'Users cannot remove the bookmark for saved searches they own';
    }
    return html`
      <div slot="anchor" class="popup-anchor saved-search-controls"></div>
      <div class="popup-content">
        <sl-tooltip content="Create a new saved search">
          <sl-icon-button
            name="floppy"
            label="Save"
            @click=${() => this.openNewSavedSearchDialog()}
          ></sl-icon-button>
        </sl-tooltip>
        <sl-tooltip content="Copy saved search URL to clipboard">
          <sl-icon-button name="share" label="Share"></sl-icon-button>
        </sl-tooltip>
        <sl-tooltip content="${bookmarkTooltipText}">
          <sl-icon-button
            name="${bookmarkStatusIcon}"
            label="${bookmarkTooltipLabel}"
            .disabled=${bookmarkButtonDisabled}
          ></sl-icon-button>
        </sl-tooltip>
        ${isOwner
          ? html`
              <sl-tooltip content="Edit current saved search">
                <sl-icon-button
                  name="pencil"
                  label="Edit"
                  @click=${() => this.openEditSavedSearchDialog()}
                ></sl-icon-button>
              </sl-tooltip>
              <sl-tooltip content="Delete saved search">
                <sl-icon-button
                  name="trash"
                  label="Delete"
                  @click=${() => this.openDeleteSavedSearchDialog()}
                ></sl-icon-button>
              </sl-tooltip>
            `
          : nothing}
      </div>
      <webstatus-saved-search-editor
        .apiClient=${this.apiClient}
        .user=${this.user}
        .savedSearch=${this.savedSearch}
      ></webstatus-saved-search-editor>
    `;
  }
}
