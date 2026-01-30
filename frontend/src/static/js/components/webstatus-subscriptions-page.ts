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

import {LitElement, html, TemplateResult, css} from 'lit';
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
import {type components} from 'webstatus.dev-backend';
import {SHARED_STYLES} from '../css/shared-css.js';
import {FREQUENCY_DISPLAY_NAMES} from '../utils/format.js';

interface GetLocationFunction {
  (): Location;
}

@customElement('webstatus-subscriptions-page')
export class SubscriptionsPage extends LitElement {
  static styles = [
    SHARED_STYLES,
    css`
      .subscription-list {
        display: flex;
        flex-direction: column;
        gap: var(--sl-spacing-small);
      }
      .subscription-item {
        display: flex;
        align-items: center;
        gap: var(--sl-spacing-medium);
        padding: var(--sl-spacing-medium);
        border: 1px solid var(--sl-color-neutral-200);
        border-radius: var(--sl-border-radius-medium);
      }
      .subscription-details {
        flex: 1;
      }
      .subscription-actions {
        display: flex;
        gap: var(--sl-spacing-small);
      }
      .subscription-item sl-skeleton {
        height: 1.2em;
        margin-bottom: var(--sl-spacing-2x-small);
      }
      .login-prompt {
        text-align: center;
        padding: var(--sl-spacing-x-large);
        border: 1px solid var(--sl-color-neutral-200);
        border-radius: var(--sl-border-radius-medium);
      }
      .channel-info {
        gap: var(--sl-spacing-2x-small);
        align-items: center;
      }
    `,
  ];

  _loadingTask: Task;

  private renderSkeleton(): TemplateResult {
    return html`
      <div class="subscription-list">
        ${[...Array(3)].map(
          () => html`
            <div class="subscription-item">
              <div class="subscription-details">
                <sl-skeleton effect="sheen"></sl-skeleton>
                <sl-skeleton effect="sheen" style="width: 60%"></sl-skeleton>
              </div>
              <div class="subscription-actions">
                <sl-button size="small" disabled>Edit</sl-button>
                <sl-button size="small" variant="danger" outline disabled
                  >Delete</sl-button
                >
              </div>
            </div>
          `,
        )}
      </div>
    `;
  }

  private renderLoginPrompt(): TemplateResult {
    return html`
      <div class="login-prompt">
        <p>Please log in to manage your subscriptions.</p>
        <webstatus-login-button></webstatus-login-button>
      </div>
    `;
  }

  private _getChannelIcon(
    type?: components['schemas']['NotificationChannel']['type'],
  ): string {
    switch (type) {
      case 'email':
        return 'envelope';
      default:
        return 'bell';
    }
  }

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
  private _activeSavedSearchId: string | undefined = undefined;

  @state()
  private _subscriptions: components['schemas']['SubscriptionResponse'][] = [];

  @state()
  private _savedSearches: Map<
    string,
    components['schemas']['SavedSearchResponse']
  > = new Map();

  @state()
  private _notificationChannels: Map<
    string,
    components['schemas']['NotificationChannelResponse']
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

        const [subscriptions, savedSearches, notificationChannels] =
          await Promise.all([
            apiClient.listSubscriptions(token),
            apiClient.getAllUserSavedSearches(token),
            apiClient.listNotificationChannels(token),
          ]);

        this._subscriptions = subscriptions;
        this._savedSearches = new Map(savedSearches.map(ss => [ss.id, ss]));
        this._notificationChannels = new Map(
          notificationChannels.map(nc => [nc.id, nc]),
        );
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
    // We need to handle the user context before the loading task.
    if (this.userContext === undefined) {
      // Loading state, waiting for user context to be resolved.
      return this.renderSkeleton();
    } else if (this.userContext === null) {
      // User is logged out.
      return this.renderLoginPrompt();
    }

    // User is logged in, proceed with loading and rendering subscriptions.
    return html`
      <h1>My Subscriptions</h1>
      ${this._loadingTask.render({
        pending: () => this.renderSkeleton(),
        complete: () => this.renderSubscriptions(),
        error: e => html`Error: ${e}`,
      })}
      <webstatus-manage-subscriptions-dialog
        ?open=${this._isSubscriptionDialogOpen}
        .subscriptionId=${this._activeSubscriptionId ?? ''}
        .savedSearchId=${this._activeSavedSearchId ?? ''}
        @subscription-dialog-close=${this._handleDialogClose}
        @subscription-save-success=${this._handleSubscriptionSaveSuccess}
        @subscription-save-error=${this._handleSubscriptionSaveError}
        @subscription-delete-success=${this._handleSubscriptionDeleteSuccess}
        @subscription-delete-error=${this._handleSubscriptionDeleteError}
      >
      </webstatus-manage-subscriptions-dialog>
    `;
  }

  private _handleDialogClose() {
    this._isSubscriptionDialogOpen = false;
    this._activeSubscriptionId = undefined;
    this._activeSavedSearchId = undefined;
  }

  private renderSubscriptions(): TemplateResult {
    if (this._subscriptions.length === 0) {
      return html`<p>No subscriptions found.</p>`;
    }

    return html`
      <div class="subscription-list">
        ${this._subscriptions.map(sub => {
          const savedSearch = this._savedSearches.get(sub.saved_search_id);
          const channel = this._notificationChannels.get(sub.channel_id);
          return html`
            <div class="subscription-item">
              <div class="subscription-details">
                <strong>${savedSearch?.name ?? sub.saved_search_id}</strong
                ><br />
                <small class="hbox channel-info">
                  <sl-icon
                    name=${this._getChannelIcon(channel?.type)}
                  ></sl-icon>
                  <span>${channel?.name ?? sub.channel_id}</span> |
                  <span>${FREQUENCY_DISPLAY_NAMES[sub.frequency]}</span>
                </small>
              </div>
              <div class="subscription-actions">
                <sl-button
                  size="small"
                  @click=${() => this._openEditDialog(sub.id)}
                  >Edit</sl-button
                >
                <sl-button
                  size="small"
                  variant="danger"
                  outline
                  @click=${() => this._openDeleteDialog(sub.id)}
                  >Delete</sl-button
                >
              </div>
            </div>
          `;
        })}
      </div>
    `;
  }

  private _openEditDialog(subscriptionId: string) {
    const sub = this._subscriptions.find(s => s.id === subscriptionId);
    if (!sub) {
      // Should not happen, but handle gracefully.
      void this.toaster('Could not find subscription to edit.', 'danger');
      return;
    }
    this._activeSavedSearchId = sub.saved_search_id;
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
