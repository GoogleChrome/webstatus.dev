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

import {consume} from '@lit/context';
import {Task} from '@lit/task';
import {LitElement, html, css, TemplateResult, nothing} from 'lit';
import {customElement, property, query, state} from 'lit/decorators.js';
import {apiClientContext} from '../contexts/api-client-context.js';
import {APIClient} from '../api/client.js';
import {SHARED_STYLES} from '../css/shared-css.js';
import {type components} from 'webstatus.dev-backend';
import {
  UserContext,
  firebaseUserContext,
} from '../contexts/firebase-user-context.js';
import {SlCheckbox, SlDialog, SlRadioGroup} from '@shoelace-style/shoelace';

const FREQUENCY_CONFIG: ReadonlyArray<
  components['schemas']['SubscriptionFrequency']
> = ['immediate', 'weekly', 'monthly'];

// Translate the frequency values from the API to user-friendly display names.
const FREQUENCY_DISPLAY_NAMES: {
  [key in components['schemas']['SubscriptionFrequency']]: string;
} = {
  immediate: 'Each Change',
  weekly: 'Weekly',
  monthly: 'Monthly',
};

export class SubscriptionSaveSuccessEvent extends CustomEvent<void> {
  constructor() {
    super('subscription-save-success', {bubbles: true, composed: true});
  }
}

export class SubscriptionSaveErrorEvent extends CustomEvent<Error> {
  constructor(error: Error) {
    super('subscription-save-error', {
      bubbles: true,
      composed: true,
      detail: error,
    });
  }
}

export class SubscriptionDeleteSuccessEvent extends CustomEvent<void> {
  constructor() {
    super('subscription-delete-success', {bubbles: true, composed: true});
  }
}

export class SubscriptionDeleteErrorEvent extends CustomEvent<Error> {
  constructor(error: Error) {
    super('subscription-delete-error', {
      bubbles: true,
      composed: true,
      detail: error,
    });
  }
}

export class SubscriptionDialogCloseEvent extends CustomEvent<void> {
  constructor() {
    super('subscription-dialog-close', {bubbles: true, composed: true});
  }
}

function isSubscriptionFrequency(
  value: string,
): value is components['schemas']['SubscriptionFrequency'] {
  return ['immediate', 'weekly', 'monthly'].includes(value);
}

const SUBSCRIPTION_TRIGGERS: ReadonlyArray<
  components['schemas']['SubscriptionTriggerWritable']
> = [
  'feature_baseline_to_widely',
  'feature_baseline_to_newly',
  'feature_browser_implementation_any_complete',
  'feature_baseline_regression_to_limited',
] as const;

function isSubscriptionTrigger(
  value: string,
): value is components['schemas']['SubscriptionTriggerWritable'] {
  // Use a safe cast to readonly string[] to avoid TS issues
  const s: readonly string[] = SUBSCRIPTION_TRIGGERS;
  return s.includes(value);
}

type ActionState =
  | {phase: 'idle'}
  | {phase: 'saving'}
  | {phase: 'deleting'}
  | {phase: 'success'; message: string}
  | {phase: 'error'; message: string};

/**
 * A dialog for managing user subscriptions for saved searches.
 *
 * This component operates in two main modes, determined by the properties set:
 *
 * 1.  **Full Management Mode**: Triggered by providing a `saved-search-id`.
 *     This mode presents a full UI to create, update, or view subscriptions
 *     across different notification channels for that specific saved search.
 *     It is the primary interface for managing notifications.
 *
 * 2.  **Unsubscribe Mode**: Triggered by providing only a `subscription-id`
 *     (and no `saved-search-id`). This mode displays a simple confirmation
 *     dialog to unsubscribe (delete) from a single subscription. It is
 *     designed for use with direct unsubscribe links, such as those in
 *     notification emails.
 */
@customElement('webstatus-manage-subscriptions-dialog')
export class ManageSubscriptionsDialog extends LitElement {
  _loadingTask: Task;

  @consume({context: apiClientContext})
  @state()
  apiClient!: APIClient;

  @consume({context: firebaseUserContext, subscribe: true})
  @state()
  userContext: UserContext | null | undefined;

  @property({type: String})
  savedSearchId = '';

  @property({type: String})
  subscriptionId = '';

  @property({type: Boolean})
  open = false;
  /**
   * This property is controlled by the parent component.
   * The dialog does not set `this.open = false` directly.
   * Instead, it calls `this._mainDialog.hide()` which emits `sl-hide`,
   * expecting the parent to update this `open` property.
   */

  @state()
  private _notificationChannels: components['schemas']['NotificationChannelResponse'][] =
    [];

  @state()
  private _savedSearch: components['schemas']['SavedSearchResponse'] | null =
    null;

  @state()
  private _subscription: components['schemas']['SubscriptionResponse'] | null =
    null;

  @state()
  private _selectedTriggers: components['schemas']['SubscriptionTriggerWritable'][] =
    [];

  @state()
  private _selectedFrequency: components['schemas']['SubscriptionFrequency'] =
    'immediate';

  @state()
  private _initialSelectedTriggers: components['schemas']['SubscriptionTriggerWritable'][] =
    [];
  @state()
  private _initialSelectedFrequency: components['schemas']['SubscriptionFrequency'] =
    'immediate';
  @state()
  private _subscriptionsForSavedSearch: components['schemas']['SubscriptionResponse'][] =
    [];
  @state()
  private _activeChannelId: string | undefined = undefined;

  @state()
  private _actionState: ActionState = {phase: 'idle'};

  @state()
  private _isConfirmDialogOpen = false;

  @state()
  private _pendingChannelId = '';

  @state()
  private _isClosing = false;

  static _TRIGGER_CONFIG: {
    value: components['schemas']['SubscriptionTriggerWritable'];
    label: string;
  }[] = [
    {
      value: 'feature_baseline_to_widely',
      label: 'becomes widely available',
    },
    {
      value: 'feature_baseline_to_newly',
      label: 'becomes newly available',
    },
    {
      value: 'feature_browser_implementation_any_complete',
      label: 'gets a new browser implementation',
    },
    {
      value: 'feature_baseline_regression_to_limited',
      label: 'regresses to limited availability',
    },
  ];

  static get styles() {
    return [
      SHARED_STYLES,
      css`
        .dialog-content-layout {
          display: flex;
          flex-wrap: wrap;
          gap: var(--sl-spacing-large);
          align-items: stretch; /* Ensures columns are equal height */
        }

        .channel-list {
          flex: 1 1 300px;
          border: 1px solid var(--sl-color-neutral-200);
          border-radius: var(--sl-border-radius-medium);
          padding: var(--sl-spacing-small);
          max-height: 450px; /* Give the list a max height */
          overflow-y: auto; /* Allow scrolling if it exceeds max-height */
        }

        .channel-item {
          display: flex;
          align-items: center;
          gap: var(--sl-spacing-small);
          padding: var(--sl-spacing-small);
          cursor: pointer;
          border-radius: var(--sl-border-radius-medium);
          margin-bottom: var(--sl-spacing-2x-small);
          color: var(--sl-color-neutral-700);
        }
        .channel-item:hover {
          background-color: var(--sl-color-neutral-100);
        }
        .channel-item.selected {
          background-color: var(--sl-color-primary-100);
          font-weight: var(--sl-font-weight-bold);
          border-left: 3px solid var(--sl-color-primary-600);
          color: var(--sl-color-primary-800);
        }
        .channel-item.selected sl-icon {
          color: var(--sl-color-primary-600);
        }
        .settings-panel {
          flex: 2 1 400px;
          display: flex;
          flex-direction: column;
          gap: var(--sl-spacing-large);
        }

        .settings-panel sl-card {
          flex: 1 0 auto; /* Allow cards to grow but not shrink within the column */
        }
        .settings-panel sl-checkbox {
          display: block;
          margin-bottom: var(--sl-spacing-2x-small);
        }
        h3 {
          font-size: var(--sl-font-size-medium);
          font-weight: var(--sl-font-weight-bold);
          margin: 0 0 var(--sl-spacing-medium) 0;
          color: var(--sl-color-neutral-900);
        }
        .footer-actions {
          display: flex;
          justify-content: space-between;
          align-items: center;
          width: 100%;
          flex-wrap: wrap; /* Allow buttons to wrap on small screens */
          gap: var(--sl-spacing-small);
        }
        .channel-header {
          justify-content: space-between;
          align-items: center;
        }
        .subscription-indicator {
          color: var(--sl-color-success-500);
          font-size: var(--sl-font-size-x-small);
        }
      `,
    ];
  }

  /**
   * Checks if the current subscription settings have been modified by the user.
   * @returns {boolean} True if there are unsaved changes.
   */
  get isDirty() {
    if (this._selectedFrequency !== this._initialSelectedFrequency) {
      return true;
    }

    // To accurately compare the triggers, we sort both arrays to ensure
    // the comparison is not affected by the order of elements.
    const sortedCurrent = [...this._selectedTriggers].sort();
    const sortedInitial = [...this._initialSelectedTriggers].sort();

    if (sortedCurrent.length !== sortedInitial.length) {
      return true;
    }

    for (let i = 0; i < sortedCurrent.length; i++) {
      if (sortedCurrent[i] !== sortedInitial[i]) {
        return true;
      }
    }

    return false;
  }

  constructor() {
    super();
    this._loadingTask = new Task(this, {
      args: () => [
        this.apiClient,
        this.savedSearchId,
        this.subscriptionId,
        this.open,
      ],
      task: async ([apiClient, savedSearchId, _subscriptionId, open]) => {
        // _subscriptionId is included in args to ensure the task re-runs if
        // the subscriptionId property changes, which is crucial for "Unsubscribe Mode".
        // It is prefixed with an underscore to indicate it's intentionally unused in the task body.
        if (!open || !apiClient || !this.userContext) {
          return;
        }
        const token = await this.userContext.user.getIdToken();

        const promises = [];
        promises.push(
          apiClient.listNotificationChannels(token).then(r => {
            this._notificationChannels = r || [];
          }),
        );

        if (savedSearchId) {
          promises.push(
            apiClient.getSavedSearchByID(savedSearchId, token).then(r => {
              this._savedSearch = r;
            }),
          );
          promises.push(
            apiClient.listSubscriptions(token).then(r => {
              this._subscriptionsForSavedSearch =
                r.filter(s => s.saved_search_id === savedSearchId) || [];
            }),
          );
        }

        await Promise.all(promises);

        // If a subscriptionId is provided (i.e., we are in "edit" mode),
        // find the corresponding subscription and pre-select its channel.
        if (this.subscriptionId) {
          const subToEdit = this._subscriptionsForSavedSearch.find(
            s => s.id === this.subscriptionId,
          );
          if (subToEdit) {
            this._switchChannel(subToEdit.channel_id);
          }
        }
      },
    });
  }

  @query('sl-dialog.dialog-main')
  _mainDialog!: SlDialog;

  render(): TemplateResult {
    return html`
      <sl-dialog
        label="Manage notifications"
        class="dialog-main"
        style="--width: min(90vw, 991px);"
        ?open=${this.open}
        @sl-request-close=${this._handleRequestClose}
      >
        ${this._loadingTask.render({
          pending: () => html`<sl-spinner></sl-spinner>`,
          complete: () => this.renderContent(),
          error: e => html`Error: ${e}`,
        })}
      </sl-dialog>

      <sl-dialog
        label="Unsaved Changes"
        class="dialog-confirm"
        ?open=${this._isConfirmDialogOpen}
        @sl-hide=${this._onConfirmDialogHide}
      >
        <p>You have unsaved changes. Do you want to discard them?</p>
        <sl-button
          slot="footer"
          @click=${() => this._handleConfirmChange(false)}
          >Cancel</sl-button
        >
        <sl-button
          slot="footer"
          variant="primary"
          @click=${() => this._handleConfirmChange(true)}
          >Discard</sl-button
        >
      </sl-dialog>
    `;
  }

  private _handleRequestClose(event: CustomEvent) {
    if (this.isDirty) {
      event.preventDefault();
      this._isClosing = true;
      this._isConfirmDialogOpen = true;
    }
  }

  private _onConfirmDialogHide(event: CustomEvent) {
    // Stop the hide event from bubbling up to the parent component.
    event.stopPropagation();
    this._isConfirmDialogOpen = false;
  }

  renderContent(): TemplateResult {
    const confirmDeletion = this.subscriptionId && !this.savedSearchId;
    if (confirmDeletion) {
      return html`
        ${this.renderAlert()}
        <p>Are you sure you want to unsubscribe?</p>
        <sl-button
          id="confirm-unsubscribe"
          slot="footer"
          variant="danger"
          @click=${this._handleDelete}
          .loading=${this._actionState.phase === 'deleting'}
        >
          Confirm Unsubscribe
        </sl-button>
      `;
    }

    const isNewSubscription = this._subscription === null;

    return html`
      ${this.renderAlert()}
      <p>
        Select how and when you want to get updates for
        <strong>${this._savedSearch?.name}</strong>.
      </p>

      <div class="dialog-content-layout">
        <div class="channel-list">
          <h3>Notification channels</h3>
          ${this._notificationChannels.map(
            channel => html`
              <div
                class="channel-item ${this._activeChannelId === channel.id
                  ? 'selected'
                  : ''}"
                @click=${() => this._handleChannelChange(channel.id)}
                role="radio"
                aria-checked=${this._activeChannelId === channel.id}
                tabindex="0"
              >
                <sl-icon name=${this._getChannelIcon(channel.type)}></sl-icon>
                <span>${channel.name}</span>
                ${this._subscriptionsForSavedSearch.some(
                  s => s.channel_id === channel.id,
                )
                  ? html`<sl-icon
                      class="subscription-indicator"
                      name="circle-fill"
                    ></sl-icon>`
                  : nothing}
              </div>
            `,
          )}
        </div>

        <div class="settings-panel">
          ${this._activeChannelId
            ? html`
                <sl-card>
                  <h3>Triggers</h3>
                  <p>Get an update when a feature...</p>
                  ${ManageSubscriptionsDialog._TRIGGER_CONFIG.map(
                    trigger => html`
                      <sl-checkbox
                        .checked=${this._selectedTriggers.includes(
                          trigger.value,
                        )}
                        @sl-change=${(e: CustomEvent) => {
                          const checkbox = e.target;
                          if (checkbox instanceof SlCheckbox) {
                            if (checkbox.checked) {
                              this._selectedTriggers = [
                                ...this._selectedTriggers,
                                trigger.value,
                              ];
                            } else {
                              this._selectedTriggers =
                                this._selectedTriggers.filter(
                                  t => t !== trigger.value,
                                );
                            }
                          }
                        }}
                        >...${trigger.label}</sl-checkbox
                      >
                    `,
                  )}
                </sl-card>

                <sl-card>
                  <h3>Frequency</h3>
                  <sl-radio-group
                    name="frequency"
                    .value=${this._selectedFrequency}
                    @sl-change=${(e: CustomEvent) => {
                      const radioGroup = e.target;
                      if (radioGroup instanceof SlRadioGroup) {
                        const value = radioGroup.value;
                        if (isSubscriptionFrequency(value)) {
                          this._selectedFrequency = value;
                        }
                      }
                    }}
                  >
                    ${FREQUENCY_CONFIG.map(
                      f =>
                        html`<sl-radio value=${f}
                          >${FREQUENCY_DISPLAY_NAMES[f]}</sl-radio
                        >`,
                    )}
                  </sl-radio-group>
                </sl-card>
              `
            : html`<sl-card>
                <p>
                  Please select a notification channel to configure its
                  settings.
                </p>
              </sl-card>`}
        </div>
      </div>

      <div slot="footer" class="footer-actions">
        <sl-button variant="text">Manage notification channels</sl-button>
        <div class="hbox" style="gap: var(--sl-spacing-small)">
          ${!isNewSubscription
            ? html`<sl-button
                variant="danger"
                outline
                .loading=${this._actionState.phase === 'deleting'}
                @click=${this._handleDelete}
                >Delete Subscription</sl-button
              >`
            : nothing}
          <sl-button
            variant="primary"
            ?disabled=${!this._activeChannelId ||
            this._selectedTriggers.length === 0 ||
            (!this.isDirty && !isNewSubscription)}
            .loading=${this._actionState.phase === 'saving'}
            @click=${this._handleSave}
            >${isNewSubscription
              ? 'Create Subscription'
              : 'Save preferences'}</sl-button
          >
        </div>
      </div>
    `;
  }

  private _getChannelIcon(
    type: components['schemas']['NotificationChannel']['type'],
  ): string {
    switch (type) {
      case 'email':
        return 'envelope';
      default:
        return 'bell';
    }
  }

  private async _handleSave() {
    if (!this.userContext || !this.isDirty || !this._activeChannelId) {
      return;
    }
    this._actionState = {phase: 'saving'};
    try {
      const token = await this.userContext.user.getIdToken();
      const existingSub = this._subscriptionsForSavedSearch.find(
        s => s.channel_id === this._activeChannelId,
      );

      if (existingSub) {
        // Update
        const updates: {
          triggers?: components['schemas']['SubscriptionTriggerWritable'][];
          frequency?: components['schemas']['SubscriptionFrequency'];
        } = {};
        if (this._selectedFrequency !== this._initialSelectedFrequency) {
          updates.frequency = this._selectedFrequency;
        }
        const triggersChanged =
          this._selectedTriggers.length !==
            this._initialSelectedTriggers.length ||
          [...this._selectedTriggers].sort().join(',') !==
            [...this._initialSelectedTriggers].sort().join(',');

        if (triggersChanged) {
          updates.triggers = this._selectedTriggers;
        }

        await this.apiClient.updateSubscription(existingSub.id, token, updates);
      } else {
        // Create
        await this.apiClient.createSubscription(token, {
          saved_search_id: this.savedSearchId,
          channel_id: this._activeChannelId,
          frequency: this._selectedFrequency,
          triggers: this._selectedTriggers,
        });
      }
      this.dispatchEvent(new SubscriptionSaveSuccessEvent());
      this._actionState = {
        phase: 'success',
        message: 'Subscription saved.',
      };
    } catch (e) {
      const error = e instanceof Error ? e : new Error('Unknown error saving');
      this.dispatchEvent(new SubscriptionSaveErrorEvent(error));
      this._actionState = {
        phase: 'error',
        message: `Error saving subscription: ${error.message}`,
      };
    }
  }

  private async _handleDelete() {
    const subscriptionIdToDelete =
      this.subscriptionId || this._subscription?.id;
    if (!subscriptionIdToDelete || !this.userContext) {
      return;
    }
    this._actionState = {phase: 'deleting'};
    try {
      const token = await this.userContext.user.getIdToken();
      await this.apiClient.deleteSubscription(subscriptionIdToDelete, token);
      this.dispatchEvent(new SubscriptionDeleteSuccessEvent());
      this._actionState = {
        phase: 'success',
        message: 'Successfully unsubscribed.',
      };
    } catch (e) {
      const error =
        e instanceof Error ? e : new Error('Unknown error unsubscribing');
      this.dispatchEvent(new SubscriptionDeleteErrorEvent(error));
      this._actionState = {
        phase: 'error',
        message: `Error unsubscribing: ${error.message}`,
      };
    }
  }

  private renderAlert(): TemplateResult {
    if (this._actionState.phase === 'success') {
      return html`
        <sl-alert
          variant="success"
          open
          closable
          @sl-after-hide=${() => {
            this._actionState = {phase: 'idle'};
          }}
        >
          <sl-icon slot="icon" name="check2-circle"></sl-icon>
          ${this._actionState.message}
        </sl-alert>
      `;
    } else if (this._actionState.phase === 'error') {
      return html`
        <sl-alert
          variant="danger"
          open
          closable
          @sl-after-hide=${() => {
            this._actionState = {phase: 'idle'};
          }}
        >
          <sl-icon slot="icon" name="exclamation-octagon"></sl-icon>
          ${this._actionState.message}
        </sl-alert>
      `;
    }
    return html``;
  }

  private _handleChannelChange(channelId: string) {
    if (this.isDirty) {
      this._pendingChannelId = channelId;
      this._isClosing = false; // Explicitly set intent to switch, not close.
      this._isConfirmDialogOpen = true;
      return;
    }
    this._switchChannel(channelId);
  }

  private _handleConfirmChange(confirmed: boolean) {
    this._isConfirmDialogOpen = false;
    if (confirmed) {
      if (this._isClosing) {
        // User wanted to close the dialog and confirmed discarding changes.
        void this._mainDialog.hide();
        this.dispatchEvent(new SubscriptionDialogCloseEvent());
      } else if (this._pendingChannelId) {
        // User wanted to switch channels and confirmed discarding changes.
        this._switchChannel(this._pendingChannelId);
      }
    }
    // Reset intent flags regardless of choice.
    this._pendingChannelId = '';
    this._isClosing = false;
  }

  private _switchChannel(channelId: string) {
    this._activeChannelId = channelId;
    const sub = this._subscriptionsForSavedSearch.find(
      s => s.channel_id === channelId,
    );
    if (sub) {
      this._subscription = sub;
      this._activeChannelId = sub.channel_id;
      this._selectedTriggers = sub.triggers
        .map(t => t.value)
        .filter(isSubscriptionTrigger);
      this._selectedFrequency = sub.frequency;
    } else {
      this._subscription = null;
      this._selectedTriggers = [];
      this._selectedFrequency = 'immediate';
    }
    this._initialSelectedTriggers = [...this._selectedTriggers];
    this._initialSelectedFrequency = this._selectedFrequency;
    this._actionState = {phase: 'idle'};
  }
}
