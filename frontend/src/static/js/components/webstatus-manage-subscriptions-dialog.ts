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
import {User, firebaseUserContext} from '../contexts/firebase-user-context.js';

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

@customElement('webstatus-manage-subscriptions-dialog')
export class ManageSubscriptionsDialog extends LitElement {
  _loadingTask: Task;

  @consume({context: apiClientContext})
  @state()
  apiClient!: APIClient;

  @consume({context: firebaseUserContext, subscribe: true})
  @state()
  user: User | null | undefined;

  @property({type: String, attribute: 'saved-search-id'})
  savedSearchId = '';

  @property({type: String, attribute: 'subscription-id'})
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

  get isDirty() {
    console.log('isDirty check:');
    console.log('  _selectedFrequency:', this._selectedFrequency);
    console.log('  _initialSelectedFrequency:', this._initialSelectedFrequency);
    console.log('  _selectedTriggers:', this._selectedTriggers);
    console.log('  _initialSelectedTriggers:', this._initialSelectedTriggers);

    if (this._selectedFrequency !== this._initialSelectedFrequency) {
      return true;
    }

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
      task: async ([apiClient, savedSearchId, subscriptionId, open]) => {
        if (!open || !apiClient || !this.user) {
          return;
        }
        const token = await this.user.getIdToken();

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

        if (subscriptionId) {
          // TODO: Fetch subscription details
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
    `;
  }

  renderContent(): TemplateResult {
    const confirmDeletion = this.subscriptionId && !this.savedSearchId;
    if (confirmDeletion) {
      return html`
        <p>Are you sure you want to unsubscribe?</p>
        <sl-button slot="footer" variant="danger" @click=${this._handleDelete}>
          Confirm Unsubscribe
        </sl-button>
      `;
    }

    return html`
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
                  const checkbox = e.target as HTMLInputElement;
                  if (checkbox.checked) {
                    this._selectedTriggers.push(trigger.value);
                  } else {
                    this._selectedTriggers = this._selectedTriggers.filter(
                      t => t !== trigger.value,
                    );
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
              const radioGroup = e.target as HTMLInputElement;
              this._selectedFrequency =
                radioGroup.value as components['schemas']['SubscriptionFrequency'];
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
        @click=${this._handleSave}
        >Save</sl-button
      >
    `;
  }

  private async _handleSave() {
    if (!this.user || !this.isDirty || !this._activeChannelId) {
      return;
    }
    try {
      const token = await this.user.getIdToken();
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
    } catch (e) {
      this.dispatchEvent(new SubscriptionSaveErrorEvent(e as Error));
    }
  }

  private async _handleDelete() {
    if (!this.subscriptionId || !this.user) {
      return;
    }
    try {
      const token = await this.user.getIdToken();
      await this.apiClient.deleteSubscription(this.subscriptionId, token);
      this.dispatchEvent(new SubscriptionDeleteSuccessEvent());
    } catch (e) {
      this.dispatchEvent(new SubscriptionDeleteErrorEvent(e as Error));
    }
  }

  private _handleChannelChange(channelId: string) {
    const previousActiveChannelId = this._activeChannelId;
    if (this.isDirty) {
      if (!confirm('You have unsaved changes. Discard them?')) {
        // If user cancels, prevent channel change. The UI will naturally
        // remain on the previously active radio button due to Lit's rendering.
        // Alternatives include explicitly re-checking the old radio button or
        // presenting a more advanced 'Save/Discard/Cancel' dialog.
        this._activeChannelId = previousActiveChannelId; // Explicitly revert UI
        return;
      }
    }

    this._activeChannelId = channelId;
    const sub = this._subscriptionsForSavedSearch.find(
      s => s.channel_id === channelId,
    );
    if (sub) {
      this._subscription = sub;
      this._activeChannelId = sub.channel_id;
      this._selectedTriggers = sub.triggers.map(
        t => t.value as components['schemas']['SubscriptionTriggerWritable'],
      );
      this._selectedFrequency = sub.frequency;
    } else {
      this._subscription = null;
      this._selectedTriggers = [];
      this._selectedFrequency = 'immediate';
    }
    this._initialSelectedTriggers = [...this._selectedTriggers];
    this._initialSelectedFrequency = this._selectedFrequency;
  }
}
