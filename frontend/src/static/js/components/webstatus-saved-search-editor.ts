import {LitElement, html, css, type TemplateResult} from 'lit';
import {customElement, property, query, state} from 'lit/decorators.js';
import {SlDialog, SlInput} from '@shoelace-style/shoelace';
import {Bookmark, VOCABULARY} from '../utils/constants.js';
import './webstatus-typeahead.js';
import {WebstatusTypeahead} from './webstatus-typeahead.js';
import {Task} from '@lit/task';
import {APIClient, UpdateSavedSearchInput} from '../api/client.js';
import {apiClientContext} from '../contexts/api-client-context.js';
import {Toast} from '../utils/toast.js';
import {User, firebaseUserContext} from '../contexts/firebase-user-context.js';
import {consume} from '@lit/context';

type OperationType = 'save' | 'edit' | 'delete';

interface OperationConfig {
  label: string;
  render: () => TemplateResult;
  actionHandler: (bookmark?: Bookmark) => Promise<void>;
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
  bookmark?: Bookmark;

  @property({type: Boolean, reflect: true})
  isOpen = false;

  @property({type: String})
  operation: OperationType = 'save';

  @query('sl-input#name')
  nameInput!: SlInput;

  @query('sl-input#description')
  descriptionInput!: SlInput;

  @query('webstatus-typeahead')
  queryInput!: WebstatusTypeahead;

  @consume({context: apiClientContext, subscribe: true})
  @state()
  apiClient?: APIClient;

  @consume({context: firebaseUserContext, subscribe: true})
  @state()
  user: User | null | undefined;

  @query('sl-dialog')
  private _dialog?: SlDialog;

  private _currentTask?: Task;

  private operationConfigMap: {[key in OperationType]: OperationConfig} = {
    save: {
      label: 'Save New Bookmark',
      render: () => this.renderForm(),
      actionHandler: this.handleSave.bind(this),
    },
    edit: {
      label: 'Edit Bookmark',
      render: () => this.renderForm(),
      actionHandler: this.handleEdit.bind(this),
    },
    delete: {
      label: 'Delete Bookmark',
      render: () => html`<p>Are you sure you want to delete this bookmark?</p>`,
      actionHandler: this.handleDelete.bind(this),
    },
  };

  async open(operation: OperationType, bookmark?: Bookmark) {
    this.bookmark = bookmark;
    this.operation = operation;
    this.isOpen = true;
    await this._dialog?.show();
  }

  async close() {
    this.isOpen = false;
    await this._dialog?.hide();
  }

  async handleSave() {
    const isNameValid = this.nameInput.reportValidity();
    const isDescriptionValid = this.descriptionInput.reportValidity();

    if (isNameValid && isDescriptionValid) {
      const currentBookmark: Bookmark = {
        id: this.bookmark?.id,
        name: this.nameInput.value,
        description: this.descriptionInput.value,
        query: this.queryInput.value,
      };
      this._currentTask = new Task(this, {
        task: async ([bookmark, user, apiClient]) => {
          if (!apiClient) {
            throw new Error('API client not available.');
          }

          const token = await user?.getIdToken();
          if (!token) {
            throw new Error('User token not available.');
          }
          if (!bookmark) {
            throw new Error('Bookmark is required for save operation.');
          }
          return apiClient.createSavedSearch(token, {
            name: bookmark.name,
            description: bookmark.description,
            query: bookmark.query,
          });
        },
        args: () => [currentBookmark, this.user, this.apiClient],
        onComplete: async result => {
          this.dispatchEvent(
            new CustomEvent('bookmark-saved', {detail: result}),
          );
          await this.close();
        },
        onError: async (error: unknown) => {
          if (error instanceof Error) {
            await new Toast().toast(
              error?.message,
              'danger',
              'exclamation-triangle',
            );
          } else {
            console.error(error);
          }
        },
      });
      await this._currentTask.run();
    }
  }

  async handleEdit() {
    const isNameValid = this.nameInput.reportValidity();
    const isDescriptionValid = this.descriptionInput.reportValidity();

    if (isNameValid && isDescriptionValid) {
      const currentBookmark: Bookmark = {
        id: this.bookmark?.id,
        name: this.nameInput.value,
        description: this.descriptionInput.value,
        query: this.queryInput.value,
      };
      this._currentTask = new Task(this, {
        task: async ([bookmark, user, apiClient]) => {
          if (!apiClient) {
            throw new Error('API client not available.');
          }

          const token = await user?.getIdToken();
          if (!token) {
            throw new Error('User token not available.');
          }
          if (!bookmark?.id) {
            throw new Error('Bookmark ID is required for edit operation.');
          }
          const update: UpdateSavedSearchInput = {
            id: bookmark.id,
            name: bookmark.name,
            description: bookmark.description,
            query: bookmark.query,
          };
          return apiClient.updateSavedSearch(update, token);
        },
        args: () => [currentBookmark, this.user, this.apiClient],
        onComplete: async result => {
          this.dispatchEvent(
            new CustomEvent('bookmark-saved', {detail: result}),
          );
          await this.close();
        },
        onError: async (error: unknown) => {
          if (error instanceof Error) {
            await new Toast().toast(
              error?.message,
              'danger',
              'exclamation-triangle',
            );
          } else {
            console.error(error);
          }
        },
      });
      await this._currentTask.run();
    }
  }

  async handleDelete() {
    if (!this.bookmark?.id) {
      console.error('Cannot delete a bookmark without an ID.');
      return;
    }
    this._currentTask = new Task(this, {
      task: async ([bookmark, user, apiClient]) => {
        if (!apiClient) {
          throw new Error('API client not available.');
        }

        const token = await user?.getIdToken();
        if (!token) {
          throw new Error('User token not available.');
        }
        if (!bookmark?.id) {
          throw new Error('Bookmark ID is required for delete operation.');
        }
        return apiClient.removeSavedSearchByID(bookmark.id, token);
      },
      args: () => [this.bookmark, this.user, this.apiClient],
      onComplete: async () => {
        this.dispatchEvent(
          new CustomEvent('bookmark-deleted', {
            detail: this.bookmark?.id,
          }),
        );
        await this.close();
      },
      onError: async (error: unknown) => {
        if (error instanceof Error) {
          await new Toast().toast(
            error?.message,
            'danger',
            'exclamation-triangle',
          );
        } else {
          console.error(error);
        }
      },
    });
    await this._currentTask.run();
  }

  async handleCancel() {
    this.dispatchEvent(new CustomEvent('bookmark-cancelled'));
    await this.close();
  }

  renderForm(): TemplateResult {
    return html`
      <sl-input
        id="name"
        label="Name"
        .value=${this.bookmark?.name ?? ''}
        required
      ></sl-input>
      <sl-input
        id="description"
        label="Description"
        placeholder="Optional Description"
        .value=${this.bookmark?.description ?? ''}
      ></sl-input>
      <webstatus-typeahead
        .vocabulary=${VOCABULARY}
        label="Query"
        .value=${this.bookmark?.query ?? ''}
      ></webstatus-typeahead>
    `;
  }

  render() {
    const config = this.operationConfigMap[this.operation];
    return html`
      <sl-dialog
        label="${config.label}"
        @sl-request-close=${async (event: Event) => {
          event.preventDefault();
          await this.handleCancel();
        }}
      >
        ${this._currentTask?.render({
          pending: () =>
            html`<sl-spinner
              style="font-size: var(--sl-font-size-xx-large);"
            ></sl-spinner>`,
          complete: () => html``,
          error: () => html``,
        }) ?? config.render()}
        <div class="dialog-buttons">
          <sl-button @click=${this.handleCancel}>Cancel</sl-button>
          <sl-button
            variant="primary"
            @click=${() => config.actionHandler(this.bookmark)}
            >${this.operation === 'delete' ? 'Delete' : 'Save'}</sl-button
          >
        </div>
      </sl-dialog>
    `;
  }
}
