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

import {LitElement, html, TemplateResult} from 'lit';
import {customElement, state, property} from 'lit/decorators.js';
import {Task} from '@lit/task';
import {consume} from '@lit/context';
import {APIClient} from '../api/client.js';
import {apiClientContext} from '../contexts/api-client-context.js';
import {
  UserContext,
  firebaseUserContext,
} from '../contexts/firebase-user-context.js';
import {toast} from '../utils/toast.js';
import {
  SubscriptionSaveErrorEvent,
  SubscriptionDeleteErrorEvent,
} from './webstatus-manage-subscriptions-dialog.js';
import {ifDefined} from 'lit/directives/if-defined.js';
import {type components} from 'webstatus.dev-backend';

interface GetLocationFunction {
  (): Location;
}

@customElement('webstatus-subscriptions-page')
export class SubscriptionsPage extends LitElement {
  _loadingTask: Task;

  @consume({context: apiClientContext})
  @state()
  apiClient!: APIClient;

  @consume({context: firebaseUserContext, subscribe: true})
  @state()
  userContext: UserContext | null | undefined;

  @property({attribute: false})
  getLocation: GetLocationFunction = () => window.location;

  @property({attribute: false})
  toaster = toast;

  @state()
  private _isSubscriptionDialogOpen = false;

  @state()
  private _activeSubscriptionId: string | undefined = undefined;

  @state()
  private _subscriptions: components['schemas']['SubscriptionResponse'][] = [];

  @state()
  private _savedSearches: Map<
    string,
    components['schemas']['SavedSearchResponse']
  > = new Map();

  constructor() {
    super();
    this._loadingTask = new Task(this, {
      args: () => [this.apiClient, this.userContext],
      task: async ([apiClient, userContext]) => {
        if (!apiClient || !userContext) {
          return;
        }
        const token = await userContext.user.getIdToken();

        const [subscriptions, savedSearches] = await Promise.all([
          apiClient.listSubscriptions(token),
          apiClient.getAllUserSavedSearches(token),
        ]);

        this._subscriptions = subscriptions;
        this._savedSearches = new Map(savedSearches.map(ss => [ss.id, ss]));
      },
    });
  }

  willUpdate() {
    const urlParams = new URLSearchParams(this.getLocation().search);
    const unsubscribeToken = urlParams.get('unsubscribe');
    if (unsubscribeToken) {
      this._activeSubscriptionId = unsubscribeToken;
      this._isSubscriptionDialogOpen = true;
    }
  }

  render(): TemplateResult {
    return html`
      <h1>My Subscriptions</h1>
      ${this._loadingTask.render({
        pending: () => html`<sl-spinner></sl-spinner>`,
        complete: () => this.renderSubscriptions(),
        error: e => html`Error: ${e}`,
      })}
      <webstatus-manage-subscriptions-dialog
        ?open=${this._isSubscriptionDialogOpen}
        subscription-id=${ifDefined(this._activeSubscriptionId)}
        @sl-hide=${() => (this._isSubscriptionDialogOpen = false)}
        @subscription-save-success=${this._handleSubscriptionSaveSuccess}
        @subscription-save-error=${this._handleSubscriptionSaveError}
        @subscription-delete-success=${this._handleSubscriptionDeleteSuccess}
        @subscription-delete-error=${this._handleSubscriptionDeleteError}
      >
      </webstatus-manage-subscriptions-dialog>
    `;
  }

  private renderSubscriptions(): TemplateResult {
    if (this._subscriptions.length === 0) {
      return html`<p>No subscriptions found.</p>`;
    }

    return html`
      <ul>
        ${this._subscriptions.map(sub => {
          const savedSearch = this._savedSearches.get(sub.saved_search_id);
          return html`
            <li>
              <strong>${savedSearch?.name ?? sub.saved_search_id}</strong>
              (Channel: ${sub.channel_id}, Frequency: ${sub.frequency})
              <sl-button
                size="small"
                @click=${() => this._openEditDialog(sub.id)}
                >Edit</sl-button
              >
              <sl-button
                size="small"
                @click=${() => this._openDeleteDialog(sub.id)}
                >Delete</sl-button
              >
            </li>
          `;
        })}
      </ul>
    `;
  }

  private _openEditDialog(subscriptionId: string) {
    this._activeSubscriptionId = subscriptionId;
    this._isSubscriptionDialogOpen = true;
  }

  private _openDeleteDialog(subscriptionId: string) {
    this._activeSubscriptionId = subscriptionId;
    // In this case, we're initiating a delete from the list, not an unsubscribe link.
    // The dialog itself will handle the confirmation internally.
    this._isSubscriptionDialogOpen = true;
    // The dialog should internally check if it's a delete scenario via subscriptionId only
    // and render the confirmation view if savedSearchId is not present.
  }

  private _handleSubscriptionSaveSuccess() {
    this._isSubscriptionDialogOpen = false;
    void this.toaster('Subscription saved!', 'success');
    void this._loadingTask.run();
  }

  private _handleSubscriptionSaveError(e: SubscriptionSaveErrorEvent) {
    void this.toaster(
      `Error saving subscription: ${e.detail.message}`,
      'danger',
    );
  }

  private _handleSubscriptionDeleteSuccess() {
    this._isSubscriptionDialogOpen = false;
    void this.toaster('Subscription deleted!', 'success');
    void this._loadingTask.run();
  }

  private _handleSubscriptionDeleteError(e: SubscriptionDeleteErrorEvent) {
    void this.toaster(
      `Error deleting subscription: ${e.detail.message}`,
      'danger',
    );
  }
}
