/**
 * Copyright 2026 Google LLC
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
import {customElement, state} from 'lit/decorators.js';
import {Task} from '@lit/task';
import {consume} from '@lit/context';
import {APIClient} from '../api/client.js';
import {apiClientContext} from '../contexts/api-client-context.js';
import {
  UserContext,
  firebaseUserContext,
} from '../contexts/firebase-user-context.js';
import {type components} from 'webstatus.dev-backend';
import {SHARED_STYLES} from '../css/shared-css.js';

type CodeSubscription = components['schemas']['CodeSubscriptionResponse'];
type SubscriptionOccurrence = components['schemas']['SubscriptionOccurrence'];
type SubscriptionTrigger = components['schemas']['SubscriptionTriggerWritable'];

@customElement('webstatus-code-subscriptions-page')
export class WebstatusCodeSubscriptionsPage extends LitElement {
  static styles = [
    SHARED_STYLES,
    css`
      :host {
        display: block;
        padding: var(--sl-spacing-large);
      }

      .container {
        max-width: 1200px;
        margin: 0 auto;
      }

      .header-controls {
        display: flex;
        justify-content: space-between;
        align-items: center;
        flex-wrap: wrap;
        gap: var(--sl-spacing-medium);
        margin-bottom: var(--sl-spacing-medium);
      }

      .repo-selector {
        min-width: 280px;
      }

      .admin-notice {
        display: flex;
        align-items: center;
        gap: var(--sl-spacing-small);
        background-color: var(--sl-color-neutral-100);
        color: var(--sl-color-neutral-700);
        padding: var(--sl-spacing-small) var(--sl-spacing-medium);
        border-radius: var(--sl-border-radius-medium);
        font-size: var(--sl-font-size-small);
        margin-bottom: var(--sl-spacing-medium);
      }

      .quota-card {
        background-color: var(--sl-color-neutral-50);
        border: 1px solid var(--sl-color-neutral-200);
        border-radius: var(--sl-border-radius-medium);
        padding: var(--sl-spacing-medium);
        margin-bottom: var(--sl-spacing-large);
      }

      .quota-text {
        display: flex;
        justify-content: space-between;
        margin-bottom: var(--sl-spacing-small);
        font-size: var(--sl-font-size-small);
        color: var(--sl-color-neutral-700);
      }

      .table-container {
        overflow-x: auto;
        border: 1px solid var(--sl-color-neutral-200);
        border-radius: var(--sl-border-radius-medium);
      }

      .subscriptions-table {
        width: 100%;
        border-collapse: collapse;
        text-align: left;
      }

      .subscriptions-table th,
      .subscriptions-table td {
        padding: var(--sl-spacing-medium);
        border-bottom: 1px solid var(--sl-color-neutral-200);
      }

      .subscriptions-table th {
        background-color: var(--sl-color-neutral-100);
        font-weight: var(--sl-font-weight-semibold);
      }

      .occurrences-list {
        margin: 0;
        padding-left: var(--sl-spacing-medium);
        font-size: var(--sl-font-size-small);
      }

      .occurrences-list li {
        margin-bottom: var(--sl-spacing-2x-small);
      }

      .status-badge {
        display: inline-block;
        padding: 2px 8px;
        border-radius: var(--sl-border-radius-pill);
        font-size: var(--sl-font-size-x-small);
        font-weight: var(--sl-font-weight-semibold);
        text-transform: uppercase;
      }

      .status-active {
        background-color: var(--sl-color-success-100);
        color: var(--sl-color-success-700);
      }

      .status-obsolete {
        background-color: var(--sl-color-neutral-200);
        color: var(--sl-color-neutral-600);
      }

      .last-scanned-text {
        font-size: var(--sl-font-size-small);
        color: var(--sl-color-neutral-600);
        white-space: nowrap;
      }

      .empty-state {
        text-align: center;
        padding: var(--sl-spacing-2x-large);
        color: var(--sl-color-neutral-500);
      }

      .login-prompt {
        text-align: center;
        padding: var(--sl-spacing-3x-large);
      }
    `,
  ];

  @consume({context: apiClientContext, subscribe: true})
  @state()
  apiClient?: APIClient;

  @consume({context: firebaseUserContext, subscribe: true})
  @state()
  userContext?: UserContext | null;

  @state()
  selectedRepo: string = 'GoogleChrome/webstatus.dev';

  public _dataTask = new Task(this, {
    task: async ([user, repo]) => {
      if (!user || !this.apiClient) {
        return null;
      }
      const token = await user.getIdToken();
      const activeRepo = repo || 'GoogleChrome/webstatus.dev';

      const subsResp = await this.apiClient.listCodeSubscriptions(
        'github',
        activeRepo,
        token,
      );
      const subscriptions: CodeSubscription[] = subsResp.data ?? [];

      return {
        repository: activeRepo,
        subscriptions,
      };
    },
    args: () =>
      [this.userContext?.user, this.selectedRepo, this.apiClient] as const,
  });

  private formatTrigger(trigger: SubscriptionTrigger | string): string {
    switch (trigger) {
      case 'feature_baseline_to_widely':
      case 'feature.baseline.promote_to_widely':
        return 'Widely Available';
      case 'feature_baseline_to_newly':
      case 'feature.baseline.promote_to_newly':
        return 'Newly Available';
      default:
        return String(trigger);
    }
  }

  private renderQuota(count: number): TemplateResult {
    const maxQuota = 500;
    const percentage = Math.min(100, Math.round((count / maxQuota) * 100));
    let variant: 'primary' | 'warning' | 'danger' = 'primary';
    if (percentage >= 100) {
      variant = 'danger';
    } else if (percentage >= 80) {
      variant = 'warning';
    }

    return html`
      <div class="quota-card">
        <div class="quota-text">
          <span><strong>Active Code Subscriptions</strong></span>
          <span>${count} / ${maxQuota} (${percentage}%)</span>
        </div>
        <sl-progress-bar
          value="${percentage}"
          variant="${variant}"
        ></sl-progress-bar>
      </div>
    `;
  }

  private getOccurrencesSummary(count: number): string {
    if (count === 1) {
      return '1 Occurrence';
    }
    return `${count} Occurrences`;
  }

  private renderOccurrences(sub: CodeSubscription): TemplateResult {
    if (!sub.occurrences || sub.occurrences.length === 0) {
      return html`<span>No occurrences recorded</span>`;
    }

    return html`
      <sl-details
        summary="${this.getOccurrencesSummary(sub.occurrences.length)}"
      >
        <ul class="occurrences-list">
          ${sub.occurrences.map((occ: SubscriptionOccurrence) => {
            const permalink = `https://github.com/${sub.repository_full_name}/blob/HEAD/${occ.file_path}#L${occ.line_number}`;
            return html`
              <li>
                <a href="${permalink}" target="_blank" rel="noopener">
                  ${occ.file_path}:L${occ.line_number}
                </a>
                <div><code>${occ.comment_snippet}</code></div>
              </li>
            `;
          })}
        </ul>
      </sl-details>
    `;
  }

  private renderSubscriptionRow(sub: CodeSubscription): TemplateResult {
    const formattedLastScanned = sub.updated_at
      ? new Date(sub.updated_at).toLocaleDateString(undefined, {
          year: 'numeric',
          month: 'short',
          day: 'numeric',
        })
      : 'Never';

    return html`
      <tr>
        <td>
          <strong>${sub.target_query}</strong>
        </td>
        <td>
          ${sub.triggers.map(
            (tr: SubscriptionTrigger) => html`
              <sl-badge variant="neutral">${this.formatTrigger(tr)}</sl-badge>
            `,
          )}
        </td>
        <td>${this.renderOccurrences(sub)}</td>
        <td>
          <span class="status-badge status-${sub.status.toLowerCase()}">
            ${sub.status}
          </span>
        </td>
        <td>
          <span class="last-scanned-text">${formattedLastScanned}</span>
        </td>
      </tr>
    `;
  }

  private renderSubscriptionsTable(
    subscriptions: readonly CodeSubscription[],
  ): TemplateResult {
    if (subscriptions.length === 0) {
      return html`
        <div class="table-container">
          <table class="subscriptions-table">
            <thead>
              <tr>
                <th>Target Feature</th>
                <th>Triggers</th>
                <th>Occurrences</th>
                <th>Status</th>
                <th>Last Scanned</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td colspan="5" class="empty-state">
                  No active code subscriptions found for this repository, or you
                  do not have admin permissions to access it.
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      `;
    }

    return html`
      <div class="table-container">
        <table class="subscriptions-table">
          <thead>
            <tr>
              <th>Target Feature</th>
              <th>Triggers</th>
              <th>Occurrences</th>
              <th>Status</th>
              <th>Last Scanned</th>
            </tr>
          </thead>
          <tbody>
            ${subscriptions.map(sub => this.renderSubscriptionRow(sub))}
          </tbody>
        </table>
      </div>
    `;
  }

  private renderWhenError(err: Error | null | undefined): TemplateResult {
    return html`
      <sl-alert variant="danger" open>
        <sl-icon slot="icon" name="exclamation-octagon"></sl-icon>
        <strong>Error loading code subscriptions:</strong> ${String(err)}
        <p
          style="margin: var(--sl-spacing-2x-small) 0 0 0; font-size: var(--sl-font-size-small);"
        >
          Please verify that you are logged in with GitHub and have admin
          permissions on this repository.
        </p>
      </sl-alert>
      <sl-button @click="${() => this._dataTask.run()}">Retry</sl-button>
    `;
  }

  render(): TemplateResult {
    if (this.userContext === null) {
      return html`
        <div class="login-prompt">
          <h2>Code Subscriptions</h2>
          <p>
            Sign in to view and manage code subscriptions for your repositories.
          </p>
        </div>
      `;
    }

    return html`
      <div class="container">
        <div class="header-controls">
          <h2>Code Subscriptions</h2>
          <div
            style="display: flex; gap: var(--sl-spacing-small); align-items: center; flex-wrap: wrap;"
          >
            <sl-input
              class="repo-selector"
              placeholder="owner/repo (e.g. GoogleChrome/webstatus.dev)"
              value="${this.selectedRepo}"
              @sl-change="${(e: Event) => {
                const input = e.target;
                if (
                  input &&
                  'value' in input &&
                  typeof input.value === 'string'
                ) {
                  this.selectedRepo = input.value.trim();
                }
              }}"
            >
              <sl-icon slot="prefix" name="search"></sl-icon>
            </sl-input>
            <sl-button
              href="https://github.com/apps/baselinebot/installations/new"
              target="_blank"
              variant="default"
              rel="noopener"
            >
              <sl-icon slot="prefix" name="github"></sl-icon>
              Connect Repository
            </sl-button>
          </div>
        </div>

        <div class="admin-notice">
          <sl-icon name="info-circle"></sl-icon>
          <span
            >Viewing and managing code subscriptions requires repository admin
            access.</span
          >
        </div>

        ${this._dataTask.render({
          pending: () => html`
            <sl-skeleton
              effect="sheen"
              style="height: 40px; margin-bottom: var(--sl-spacing-medium);"
            ></sl-skeleton>
            <sl-skeleton
              effect="sheen"
              style="height: 100px; margin-bottom: var(--sl-spacing-medium);"
            ></sl-skeleton>
            <sl-skeleton effect="sheen" style="height: 200px;"></sl-skeleton>
          `,
          error: error =>
            this.renderWhenError(
              error instanceof Error ? error : new Error(String(error)),
            ),
          complete: data => {
            if (!data) {
              return html`
                <div class="empty-state">
                  <p>No repository selected.</p>
                </div>
              `;
            }

            return html`
              ${this.renderQuota(data.subscriptions.length)}
              ${this.renderSubscriptionsTable(data.subscriptions)}
            `;
          },
        })}
      </div>
    `;
  }
}
