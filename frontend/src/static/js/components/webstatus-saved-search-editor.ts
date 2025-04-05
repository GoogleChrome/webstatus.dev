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

import {LitElement, html, css, type TemplateResult, nothing} from 'lit';
import {customElement, property, query, state} from 'lit/decorators.js';
import {SlAlert, SlButton, SlDialog, SlInput} from '@shoelace-style/shoelace';
import {UserSavedSearch, VOCABULARY} from '../utils/constants.js';
import './webstatus-typeahead.js';
import {WebstatusTypeahead} from './webstatus-typeahead.js';
import {Task, TaskStatus} from '@lit/task';
import {APIClient, UpdateSavedSearchInput} from '../api/client.js';
import {Toast} from '../utils/toast.js';
import {User} from '../contexts/firebase-user-context.js';
import {ApiError} from '../api/errors.js';

type OperationType = 'save' | 'edit' | 'delete';

interface OperationConfig {
  label: string;
  render: (inProgress: boolean) => TemplateResult;
  actionHandler: () => Promise<void>;
  primaryButtonText: string;
  buttonVariant: SlButton['variant'];
}

// SavedSearchInputConstraints come from components/schemas/SavedSearch in the openapi document.
const SavedSearchInputConstraints = {
  NameMinLength: 1,
  NameMaxLength: 32,
  // We drop the description if it is an empty string.
  DescriptionMaxLength: 256,
  QueryMinLength: 1,
  QueryMaxLength: 256,
};

@customElement('webstatus-saved-search-editor')
export class WebstatusSavedSearchEditor extends LitElement {
  static styles = css`
    .dialog-buttons {
      display: flex;
      justify-content: end;
      gap: 1em;
    }
    sl-input {
      padding-bottom: 1em;
    }
    webstatus-typeahead {
      padding-bottom: 1em;
    }
  `;

  @property({type: Object})
  savedSearch?: UserSavedSearch;

  @property({type: String})
  operation: OperationType = 'save';

  @property({type: Object})
  apiClient!: APIClient;

  @property({type: Object})
  user!: User;

  @property({type: Object})
  location!: {search: string};

  // This is the typehead from the overview page so that we can carry over the user's existing query.
  @state()
  overviewPageQueryInput?: WebstatusTypeahead;

  @query('sl-alert#editor-alert')
  editorAlert?: SlAlert;

  @query('sl-input#name')
  nameInput?: SlInput;

  @query('sl-textarea#description')
  descriptionInput?: SlInput;

  @query('webstatus-typeahead')
  queryInput?: WebstatusTypeahead;

  @query('sl-dialog')
  private _dialog?: SlDialog;

  @state()
  private _currentTask?: Task;

  private operationConfigMap: {[key in OperationType]: OperationConfig} = {
    save: {
      label: 'Save New Search',
      render: (inProgress: boolean) => this.renderForm(inProgress),
      actionHandler: this.handleSave.bind(this),
      primaryButtonText: 'Save',
      buttonVariant: 'primary',
    },
    edit: {
      label: 'Edit Saved Search',
      render: (inProgress: boolean) => this.renderForm(inProgress),
      actionHandler: this.handleEdit.bind(this),
      primaryButtonText: 'Save',
      buttonVariant: 'primary',
    },
    delete: {
      label: 'Delete Saved Search',
      render: (_: boolean) =>
        html`<p>Are you sure you want to delete this search?</p>`,
      actionHandler: this.handleDelete.bind(this),
      primaryButtonText: 'Delete',
      buttonVariant: 'danger',
    },
  };

  async open(
    operation: OperationType,
    savedSearch?: UserSavedSearch,
    overviewPageQueryInput?: WebstatusTypeahead,
  ) {
    this.savedSearch = savedSearch;
    this.operation = operation;
    this.overviewPageQueryInput = overviewPageQueryInput;
    await this._dialog?.show();
  }

  async close() {
    this._currentTask = undefined;
    this.overviewPageQueryInput = undefined;
    this.savedSearch = undefined;
    await this._dialog?.hide();
  }

  async handleSave() {
    const isNameValid = this.nameInput!.reportValidity();
    const isDescriptionValid = this.descriptionInput!.reportValidity();
    const isQueryValid = this.isQueryValid();
    if (isNameValid && isDescriptionValid && isQueryValid) {
      await this.editorAlert?.hide();
      this._currentTask = new Task(this, {
        autoRun: false,
        task: async ([name, description, query, user, apiClient]) => {
          const token = await user!.getIdToken();
          return apiClient!.createSavedSearch(token, {
            name: name,
            description: description !== '' ? description : undefined,
            query: query,
          });
        },
        args: () => [
          this.nameInput!.value,
          this.descriptionInput!.value,
          this.queryInput!.value,
          this.user,
          this.apiClient,
        ],
        onComplete: async result => {
          this.dispatchEvent(
            new CustomEvent('saved-search-saved', {
              detail: result,
              bubbles: true,
              composed: true,
            }),
          );
          await this.close();
        },
        onError: async (error: unknown) => {
          let message: string;
          if (error instanceof ApiError) {
            message = error.message;
          } else {
            message =
              'Unknown error saving saved search. Check console for details.';
            console.error(error);
          }
          await new Toast().toast(message, 'danger', 'exclamation-triangle');
        },
      });
      await this._currentTask.run();
    } else {
      await this.editorAlert?.show();
    }
  }

  isQueryValid(): boolean {
    if (this.queryInput) {
      // TODO: Figure out a way to configure the form constraints on typeahead constraint
      // I also tried to set the constraints up in the firstUpdated callback but the child is not rendered yet.
      // Also, setting the custom validity message does not work because the typeahead renders the sl-input in a shadow DOM.
      // Moving the typeahead to the light dom with createRenderRoot messes up the style of the dropdown.
      // Until then, check manually.
      // Once that is resolved, we can get rid of the sl-alert component below.
      if (
        this.queryInput.value.length <
          SavedSearchInputConstraints.QueryMinLength ||
        this.queryInput.value.length >
          SavedSearchInputConstraints.QueryMaxLength
      ) {
        return false;
      } else {
        return true;
      }
    }

    return false;
  }

  async handleEdit() {
    const isNameValid = this.nameInput!.reportValidity();
    const isDescriptionValid = this.descriptionInput!.reportValidity();
    const isQueryValid = this.isQueryValid();
    if (isNameValid && isDescriptionValid && isQueryValid && this.savedSearch) {
      await this.editorAlert?.hide();
      this._currentTask = new Task(this, {
        autoRun: false,
        task: async ([
          savedSearch,
          name,
          description,
          query,
          user,
          apiClient,
        ]) => {
          const token = await user.getIdToken();
          const update: UpdateSavedSearchInput = {
            id: savedSearch.id,
            name: name !== savedSearch.name ? name : undefined,
            description:
              description !== savedSearch.description && description !== ''
                ? description
                : undefined,
            query: query !== savedSearch.query ? query : undefined,
          };
          return apiClient!.updateSavedSearch(update, token);
        },
        args: () => [
          this.savedSearch!,
          this.nameInput!.value,
          this.descriptionInput!.value,
          this.queryInput!.value,
          this.user,
          this.apiClient,
        ],
        onComplete: async result => {
          this.dispatchEvent(
            new CustomEvent('saved-search-edited', {
              detail: result,
              bubbles: true,
              composed: true,
            }),
          );
          await this.close();
        },
        onError: async (error: unknown) => {
          let message: string;
          if (error instanceof ApiError) {
            message = error.message;
          } else {
            message =
              'Unknown error editing saved search. Check console for details.';
            console.error(error);
          }
          await new Toast().toast(message, 'danger', 'exclamation-triangle');
        },
      });
      await this._currentTask.run();
    } else {
      await this.editorAlert?.show();
    }
  }

  async handleDelete() {
    this._currentTask = new Task(this, {
      autoRun: false,
      task: async ([savedSearchID, user, apiClient]) => {
        const token = await user!.getIdToken();
        await apiClient!.removeSavedSearchByID(savedSearchID!, token);
        return savedSearchID!;
      },
      args: () => [this.savedSearch?.id, this.user, this.apiClient],
      onComplete: async savedSearchID => {
        this.dispatchEvent(
          new CustomEvent('saved-search-deleted', {
            detail: savedSearchID,
            bubbles: true,
            composed: true,
          }),
        );
        await this.close();
      },
      onError: async (error: unknown) => {
        let message: string;
        if (error instanceof ApiError) {
          message = error.message;
        } else {
          message =
            'Unknown error deleting saved search. Check console for details.';
          console.error(error);
        }
        await new Toast().toast(message, 'danger', 'exclamation-triangle');
      },
    });
    await this._currentTask.run();
  }

  async handleCancel() {
    this.dispatchEvent(
      new CustomEvent('saved-search-cancelled', {
        bubbles: true,
        composed: true,
      }),
    );
    await this.close();
  }

  renderForm(inProgress: boolean): TemplateResult {
    let query: string;
    if (this.overviewPageQueryInput && this.overviewPageQueryInput.value) {
      query = this.overviewPageQueryInput.value;
    } else if (this.savedSearch) {
      query = this.savedSearch.query;
    } else {
      query = '';
    }
    return html`
      <sl-input
        id="name"
        label="Name"
        .value=${this.savedSearch?.name ?? ''}
        helpText="Title of the search"
        required
        minlength=${SavedSearchInputConstraints.NameMinLength}
        maxlength=${SavedSearchInputConstraints.NameMaxLength}
        .disabled=${inProgress}
      ></sl-input>
      <sl-textarea
        id="description"
        label="Description"
        placeholder="Description"
        helpText="Optional Description"
        maxlength=${SavedSearchInputConstraints.DescriptionMaxLength}
        .value=${this.savedSearch?.description ?? ''}
      ></sl-textarea>
      <webstatus-typeahead
        .vocabulary=${VOCABULARY}
        .label=${'Query'}
        value=${query}
      ></webstatus-typeahead>
      <!-- TODO: See comment in isQueryValid. Until then we show our own validation message -->
      <div class="editor-alert">
        <sl-alert id="editor-alert" variant="danger" duration="3000" closable>
          <sl-icon slot="icon" name="exclamation-octagon"></sl-icon>
          Please check that you provided at least a name and query before
          submitting. The name must be between
          ${SavedSearchInputConstraints.NameMinLength} and
          ${SavedSearchInputConstraints.NameMaxLength} characters long, and the
          query must be between ${SavedSearchInputConstraints.QueryMinLength}
          and ${SavedSearchInputConstraints.QueryMaxLength} characters long.
        </sl-alert>
      </div>
    `;
  }

  render() {
    const config = this.operationConfigMap[this.operation];
    const inProgress = this._currentTask?.status === TaskStatus.PENDING;
    return html`
      <sl-dialog
        label="${config.label}"
        @sl-request-close=${async (event: Event) => {
          event.preventDefault();
          await this.handleCancel();
        }}
      >
        <form
          id="editor-form"
          @submit=${async (e: Event) => {
            e.preventDefault();
            await config.actionHandler();
          }}
        >
          ${inProgress ? html`<sl-spinner></sl-spinner>` : nothing}
          ${config.render(inProgress)}
          <div class="dialog-buttons">
            <sl-button @click=${this.handleCancel}>Cancel</sl-button>
            <sl-button
              variant="${config.buttonVariant}"
              type="submit"
              .disabled=${inProgress}
              .loading=${inProgress}
              >${config.primaryButtonText}</sl-button
            >
          </div>
        </form>
      </sl-dialog>
    `;
  }
}
