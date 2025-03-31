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

import {LitElement, html, css, TemplateResult, PropertyValueMap} from 'lit';
import {customElement, property, state, query} from 'lit/decorators.js';
import {SHARED_STYLES} from '../css/shared-css.js';
import './webstatus-typeahead.js';
import {type WebstatusTypeahead} from './webstatus-typeahead.js';
import {formatOverviewPageUrl} from '../utils/urls.js';
import {navigateToUrl} from '../utils/app-router.js';
import {SavedSearchResponse, type APIClient} from '../api/client.js';
import {User} from 'firebase/auth';
import {toast} from '../utils/toast.js';
import {Task, TaskStatus} from '@lit/task';
import {type Bookmark} from '../utils/constants.js';
import {SlDialog} from '@shoelace-style/shoelace';

@customElement('webstatus-bookmark-editor')
export class WebstatusBookmarkEditor extends LitElement {
  @property({type: Object})
  location!: {search: string};

  @property({type: Object})
  apiClient?: APIClient;

  @property({type: Object})
  bookmark?: Bookmark;

  @property({type: Object})
  user: User | null | undefined;

  @property({type: Boolean})
  showDialog = false;

  @state()
  title = '';

  @state()
  description = '';

  @state()
  query = '';

  @query('webstatus-typeahead')
  typeahead!: WebstatusTypeahead;

  @state()
  saveTask = new Task(this, {
    autoRun: false,
    args: () =>
      [
        this.apiClient,
        this.title,
        this.description,
        this.query,
        this.user,
      ] as const,
    task: async ([
      apiClient,
      title,
      description,
      query,
      user,
    ]): Promise<SavedSearchResponse> => {
      const newBookmark = {
        name: title,
        description: description,
        query: query,
      };
      const token = await user!.getIdToken();
      return apiClient!.createSavedSearch(newBookmark, token);
    },
    onComplete: bookmark => {
      this.dispatchEvent(
        new CustomEvent('bookmark-saved', {
          detail: {bookmark},
          bubbles: true,
          composed: true,
        }),
      );
      this.closeModal();
    },
    onError: async (error: unknown) => {
      await toast(
        `Failed to save bookmark: ${error instanceof Error ? error.message : 'Unknown error'}`,
        'danger',
        'exclamation-triangle',
      );
    },
  });

  static get styles() {
    return [
      SHARED_STYLES,
      css`
        .vbox {
          display: flex;
          flex-direction: column;
          gap: 1rem;
        }
        .hbox {
          display: flex;
          gap: 1rem;
          justify-content: end;
        }
      `,
    ];
  }

  connectedCallback() {
    super.connectedCallback();
    this.title = this.bookmark?.name || '';
    this.description = this.bookmark?.description || '';
    this.query = this.bookmark?.query || '';
  }

  protected firstUpdated(): void {
    if (this.showDialog) this.showEditor();
    else this.closeEditor();
  }

  protected willUpdate(changedProperties: PropertyValueMap<this>): void {
    if (changedProperties.has('showDialog')) {
      if (this.showDialog) this.showEditor();
      else this.closeEditor();
    }
  }

  handleTitleChange(event: Event) {
    this.title = (event.target as HTMLInputElement).value;
  }

  handleDescriptionChange(event: Event) {
    this.description = (event.target as HTMLInputElement).value;
  }

  handleQueryChange(_: Event) {
    this.query = this.typeahead.value;
  }

  @query('sl-dialog')
  dialog?: SlDialog;

  showEditor() {
    if (this.dialog?.open === false) this.dialog?.show();
  }

  closeEditor() {
    if (this.dialog?.open === true) this.dialog?.hide();
  }

  handlePreview() {
    // Update the location object with the new query
    const newUrl = formatOverviewPageUrl(this.location, {
      q: this.query,
      start: 0,
    });
    navigateToUrl(newUrl);
  }

  async handleSave() {
    if (this.user === undefined || this.user === null) {
      await toast(
        'Please log in to save a bookmark.',
        'warning',
        'exclamation-triangle',
      );
      return;
    }
    void this.saveTask.run();
  }

  async closeModal() {
    const dialog = this.shadowRoot?.querySelector('sl-dialog');
    if (dialog?.hide) await dialog.hide();
    this.dispatchEvent(
      new CustomEvent('bookmark-canceled', {
        bubbles: true,
        composed: true,
      }),
    );
  }

  render(): TemplateResult {
    return html`
      <sl-dialog label="Bookmark Editor">
        <div class="vbox">
          <sl-input
            label="Title"
            value=${this.title}
            @sl-input=${this.handleTitleChange}
          ></sl-input>
          <sl-textarea
            label="Description"
            value=${this.description}
            @sl-input=${this.handleDescriptionChange}
          ></sl-textarea>
          <webstatus-typeahead
            label="Query"
            value=${this.query}
            @sl-change=${this.handleQueryChange}
          ></webstatus-typeahead>
          <div class="hbox">
            <sl-button @click=${this.handlePreview}>Preview</sl-button>
            <sl-button
              variant="primary"
              ?disabled=${this.saveTask.status === TaskStatus.PENDING}
              @click=${this.handleSave}
            >
              ${this.saveTask.status === TaskStatus.PENDING
                ? 'Saving...'
                : 'Save'}
            </sl-button>
            <sl-button @click=${this.closeModal}>Cancel</sl-button>
          </div>
        </div>
      </sl-dialog>
    `;
  }
}
