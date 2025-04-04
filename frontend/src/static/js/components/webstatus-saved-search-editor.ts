import {LitElement, html, css, type TemplateResult, nothing} from 'lit';
import {customElement, property, query, state} from 'lit/decorators.js';
import {SlButton, SlDialog, SlInput} from '@shoelace-style/shoelace';
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

@customElement('webstatus-saved-search-editor')
export class WebstatusSavedSearchEditor extends LitElement {
  static styles = css`
    .dialog-buttons {
      display: flex;
      justify-content: end;
      gap: 1rem;
    }
    sl-input {
      margin-bottom: 1rem;
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

  @query('sl-input#name')
  nameInput!: SlInput;

  @query('sl-input#description')
  descriptionInput!: SlInput;

  @query('webstatus-typeahead')
  queryInput!: WebstatusTypeahead;

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

  async open(operation: OperationType, savedSearch?: UserSavedSearch) {
    this.savedSearch = savedSearch;
    this.operation = operation;
    await this._dialog?.show();
  }

  async close() {
    this.nameInput.value = '';
    this.descriptionInput.value = '';
    this.queryInput.value = '';
    this._currentTask = undefined;
    await this._dialog?.hide();
  }

  async handleSave() {
    const isNameValid = this.nameInput.reportValidity();
    const isDescriptionValid = this.descriptionInput.reportValidity();
    if (isNameValid && isDescriptionValid) {
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
          this.nameInput.value,
          this.descriptionInput.value,
          this.queryInput.value,
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
    }
  }

  async handleEdit() {
    const isNameValid = this.nameInput.reportValidity();
    const isDescriptionValid = this.descriptionInput.reportValidity();

    if (isNameValid && isDescriptionValid && this.savedSearch) {
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
          this.nameInput.value,
          this.descriptionInput.value,
          this.queryInput.value,
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
    return html`
      <sl-input
        id="name"
        label="Name"
        .value=${this.savedSearch?.name ?? ''}
        required
        .disabled=${inProgress}
      ></sl-input>
      <sl-input
        id="description"
        label="Description"
        placeholder="Optional Description"
        .value=${this.savedSearch?.description ?? ''}
      ></sl-input>
      <webstatus-typeahead
        .vocabulary=${VOCABULARY}
        label="Query"
        .value=${this.savedSearch?.query ?? ''}
      ></webstatus-typeahead>
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
        ${inProgress ? html`<sl-spinner></sl-spinner>` : nothing}
        ${config.render(inProgress)}
        <div class="dialog-buttons">
          <sl-button @click=${this.handleCancel}>Cancel</sl-button>
          <sl-button
            variant="${config.buttonVariant}"
            @click=${() => config.actionHandler()}
            .disabled=${inProgress}
            .loading=${inProgress}
            >${config.primaryButtonText}</sl-button
          >
        </div>
      </sl-dialog>
    `;
  }
}
