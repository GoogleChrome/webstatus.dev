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
import {
  SavedSearchResponse,
  type APIClient,
  UpdateSavedSearchInput,
} from '../api/client.js';
import {User} from 'firebase/auth';
import {toast} from '../utils/toast.js';
import {Task, TaskStatus} from '@lit/task';
import {type Bookmark} from '../utils/constants.js';
import {SlDialog, SlInput, SlTextarea} from '@shoelace-style/shoelace';

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

  @query('webstatus-typeahead')
  queryField!: WebstatusTypeahead;

  @query('sl-input')
  nameField!: SlInput;

  @query('sl-textarea')
  descriptionField!: SlTextarea;

  @state()
  updateTask = new Task(this, {
    autoRun: false,
    args: () =>
      [
        this.apiClient,
        this.nameField.value,
        this.descriptionField.value,
        this.queryField.value,
        this.user,
        this.bookmark,
      ] as const,
    task: async ([
      apiClient,
      name,
      description,
      query,
      user,
      bookmark,
    ]): Promise<SavedSearchResponse> => {
      const updatedBookmark: UpdateSavedSearchInput = {
        id: bookmark?.id!,
      };
      if (name !== undefined && name !== bookmark?.name)
        updatedBookmark.name = name;
      if (description !== undefined && description !== bookmark?.description)
        updatedBookmark.description = description;
      if (query !== undefined && query !== bookmark?.query)
        updatedBookmark.query = query;
      const token = await user!.getIdToken();
      return apiClient!.updateSavedSearch(updatedBookmark, token);
    },
    onComplete: bookmark => {
      this.dispatchEvent(
        new CustomEvent('search-saved', {
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
    window.addEventListener('beforeunload', this.handleBeforeUnload);
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    window.removeEventListener('beforeunload', this.handleBeforeUnload);
  }

  handleBeforeUnload = (event: BeforeUnloadEvent) => {
    if (this.hasUnsavedChanges()) {
      event.preventDefault();
      event.returnValue = '';
    }
  };

  hasUnsavedChanges(): boolean {
    return (
      this.nameField.value !== this.bookmark?.name ||
      this.descriptionField.value !== this.bookmark?.description ||
      this.queryField.value !== this.bookmark?.query
    );
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

  @query('sl-dialog')
  dialog?: SlDialog;

  showEditor() {
    if (this.dialog?.open === false) this.dialog?.show();
  }

  closeEditor() {
    if (this.dialog?.open === true) this.dialog?.hide();
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
    void this.updateTask.run();
  }

  async closeModal() {
    const dialog = this.shadowRoot?.querySelector('sl-dialog');
    if (dialog?.hide) await dialog.hide();
    this.dispatchEvent(
      new CustomEvent('search-canceled', {
        bubbles: true,
        composed: true,
      }),
    );
  }

  render(): TemplateResult {
    return html`
      <sl-dialog label="Bookmark Editor">
        <div class="vbox">
          <form>
            <sl-input
              label="Name"
              value=${this.bookmark?.name ?? ''}
            ></sl-input>
            <sl-textarea
              label="Description"
              value=${this.bookmark?.description ?? ''}
            ></sl-textarea>
            <webstatus-typeahead
              label="Query"
              value=${this.bookmark?.query ?? ''}
            ></webstatus-typeahead>
            <div class="hbox">
              <sl-button
                variant="primary"
                ?disabled=${this.updateTask.status === TaskStatus.PENDING}
                @click=${this.handleSave}
              >
                ${this.updateTask.status === TaskStatus.PENDING
                  ? 'Saving...'
                  : 'Save'}
              </sl-button>
              <sl-button @click=${this.closeModal}>Cancel</sl-button>
            </div>
          </form>
        </div>
      </sl-dialog>
    `;
  }
}
