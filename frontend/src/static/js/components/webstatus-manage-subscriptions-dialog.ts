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
import {LitElement, html, css, TemplateResult} from 'lit';
import {customElement, property, state} from 'lit/decorators.js';
import {apiClientContext} from '../contexts/api-client-context.js';
import {APIClient} from '../api/client.js';
import {SHARED_STYLES} from '../css/shared-css.js';
import {type components} from 'webstatus.dev-backend';
import {
  UserContext,
  firebaseUserContext,
} from '../contexts/firebase-user-context.js';
import {SlCheckbox, SlRadioGroup} from '@shoelace-style/shoelace';

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
        .dialog-overview {
          --sl-dialog-width: 80vw;
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
      },
    });
  }

  render(): TemplateResult {
    return html`
      <sl-dialog
        label="Manage notifications"
        class="dialog-overview"
        ?open=${this.open}
        @sl-hide=${() => (this.open = false)}
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
        @sl-hide=${() => (this._isConfirmDialogOpen = false)}
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

    return html`
      ${this.renderAlert()}
      <p>
        Select how and when you want to get updates for
        <strong
          >${this._subscription?.saved_search_id ??
          this._savedSearch?.name}</strong
        >.
      </p>

      <div class="hbox" style="gap: var(--sl-spacing-large)">
        <div class="vbox" style="flex: 1">
          <h3>Notification channels</h3>
          ${this._notificationChannels.map(
            channel => html`
              <sl-radio-button
                ?checked=${this._activeChannelId === channel.id}
                @sl-change=${() => {
                  this._handleChannelChange(channel.id);
                }}
                >${channel.name} (${channel.type})</sl-radio-button
              >
            `,
          )}
        </div>
        <div class="vbox" style="flex: 1">
          <h3>Triggers</h3>
          <p>Get an update when a feature...</p>
          ${ManageSubscriptionsDialog._TRIGGER_CONFIG.map(
            trigger => html`
              <sl-checkbox
                .checked=${this._selectedTriggers.includes(trigger.value)}
                @sl-change=${(e: CustomEvent) => {
                  const checkbox = e.target;
                  if (checkbox instanceof SlCheckbox) {
                    if (checkbox.checked) {
                      this._selectedTriggers.push(trigger.value);
                    } else {
                      this._selectedTriggers = this._selectedTriggers.filter(
                        t => t !== trigger.value,
                      );
                    }
                  }
                }}
                >...${trigger.label}</sl-checkbox
              >
            `,
          )}
        </div>
        <div class="vbox" style="flex: 1">
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
            <sl-radio-button value="immediate">Immediately</sl-radio-button>
            <sl-radio-button value="weekly">Weekly updates</sl-radio-button>
            <sl-radio-button value="monthly">Monthly updates</sl-radio-button>
          </sl-radio-group>
        </div>
      </div>

      <sl-button
        slot="footer"
        variant="primary"
        ?disabled=${!this.isDirty}
        .loading=${this._actionState.phase === 'saving'}
        @click=${this._handleSave}
        >Save</sl-button
      >
    `;
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
    if (!this.subscriptionId || !this.userContext) {
      return;
    }
    this._actionState = {phase: 'deleting'};
    try {
      const token = await this.userContext.user.getIdToken();
      await this.apiClient.deleteSubscription(this.subscriptionId, token);
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
      this._isConfirmDialogOpen = true;
      return;
    }
    this._switchChannel(channelId);
  }

  private _handleConfirmChange(confirmed: boolean) {
    this._isConfirmDialogOpen = false;
    if (confirmed) {
      this._switchChannel(this._pendingChannelId);
    }
    this._pendingChannelId = '';
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
