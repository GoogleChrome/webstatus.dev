import {
  CSSResultGroup,
  LitElement,
  TemplateResult,
  css,
  html,
  nothing,
} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {APIClient, apiClientContext} from '../contexts/api-client-context.js';
import {consume} from '@lit/context';
import {firebaseUserContext, User} from '../contexts/firebase-user-context.js';
import {Task} from '@lit/task';
import {SavedSearchResponseList} from '../api/client.js';

@customElement('webstatus-sidebar-menu-saved-searches-section')
export class WebstatusSidebarMenuSavedSearchesSection extends LitElement {
  @consume({context: apiClientContext})
  apiClient!: APIClient;

  @consume({context: firebaseUserContext, subscribe: true})
  user?: User;

  _task: Task;

  @state()
  savedSearches: SavedSearchResponseList = [];

  static get styles(): CSSResultGroup {
    return [
      css`
        sl-skeleton {
          width: 10rem;
        }
      `,
    ];
  }

  constructor() {
    super();
    this._task = new Task(this, {
      args: () => [this.apiClient, this.user] as const,
      task: async ([apiClient, user]) => {
        if (!user || !apiClient) {
          return [];
        }
        const token = await user.getIdToken();
        this.savedSearches = await apiClient.getAllUserSavedSearches(token);
        return this.savedSearches;
      },
    });
  }

  // Must render to light DOM, so sl-tree works as intended.
  createRenderRoot() {
    return this;
  }

  renderInitialSection(): TemplateResult {
    return html`${nothing}`;
  }

  renderLoadingSection(): TemplateResult {
    return html`
      <sl-tree-item><sl-skeleton effect="sheen"></sl-skeleton></sl-tree-item>
      <sl-tree-item><sl-skeleton effect="sheen"></sl-skeleton></sl-tree-item>
      <sl-tree-item><sl-skeleton effect="sheen"></sl-skeleton></sl-tree-item>
    `;
  }

  renderErrorSection(error: unknown): TemplateResult {
    if (!(error instanceof Error)) {
      console.log(JSON.stringify(error));
      return html`<div>Unknown error</div>`;
    }

    let message = 'Something went wrong';
    if (error.message) {
      message = error.message;
    }
    return html` <div>Unable to list saved searches for user: ${message}</div>`;
  }

  renderSuccessSection(): TemplateResult {
    return html`
      <sl-tree-item><sl-skeleton effect="sheen"></sl-skeleton></sl-tree-item>
      <sl-tree-item><sl-skeleton effect="sheen"></sl-skeleton></sl-tree-item>
      <sl-tree-item><sl-skeleton effect="sheen"></sl-skeleton></sl-tree-item>
    `;
  }

  renderSavedSearches(): TemplateResult {
    return this._task.render({
      complete: () => this.renderSuccessSection(),
      error: e => this.renderErrorSection(e),
      initial: () => this.renderInitialSection(),
      pending: () => this.renderLoadingSection(),
    });
  }

  render(): TemplateResult {
    return html`
      <sl-divider aria-hidden="true"></sl-divider>

      <sl-icon name="caret-right-fill" slot="expand-icon"></sl-icon>
      <sl-icon name="caret-right-fill" slot="collapse-icon"></sl-icon>
      <sl-tree-item>
        Bookmarked Searches ${this.renderSavedSearches()}
      </sl-tree-item>
    `;
  }
}
