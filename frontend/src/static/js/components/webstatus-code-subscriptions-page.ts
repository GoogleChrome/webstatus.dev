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
import {SlSelect} from '@shoelace-style/shoelace';

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
        color: var(--sl-color-neutral-900);
      }

      .container {
        max-width: 1200px;
        margin: 0 auto;
        display: flex;
        flex-direction: column;
        gap: var(--sl-spacing-large);
      }

      .header-controls {
        display: flex;
        justify-content: space-between;
        align-items: center;
        flex-wrap: wrap;
        gap: var(--sl-spacing-medium);
      }

      .repo-selector {
        min-width: 320px;
      }

      .quota-card {
        background: var(--sl-color-neutral-50);
        border: 1px solid var(--sl-color-neutral-200);
        border-radius: var(--sl-border-radius-medium);
        padding: var(--sl-spacing-medium);
        display: flex;
        flex-direction: column;
        gap: var(--sl-spacing-small);
      }

      .quota-text {
        display: flex;
        justify-content: space-between;
        font-size: var(--sl-font-size-small);
        color: var(--sl-color-neutral-700);
      }

      .table-container {
        width: 100%;
        overflow-x: auto;
      }

      table.subscriptions-table {
        width: 100%;
        border-collapse: collapse;
        text-align: left;
      }

      table.subscriptions-table th,
      table.subscriptions-table td {
        padding: var(--sl-spacing-small) var(--sl-spacing-medium);
        border-bottom: 1px solid var(--sl-color-neutral-200);
      }

      table.subscriptions-table th {
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
  selectedRepoID?: string;

  public _dataTask = new Task(this, {
    task: async ([user, repoID]) => {
      if (!user || !this.apiClient) {
        return null;
      }
      const token = await user.getIdToken();
      const reposResp = await this.apiClient.listVCSRepositories(
        'github',
        token,
      );
      const repos = reposResp.data;

      if (!repos || repos.length === 0) {
        return {repositories: [], subscriptions: []};
      }

      const activeRepoID = repoID ?? repos[0]?.repository_id;
      if (!this.selectedRepoID && activeRepoID) {
        this.selectedRepoID = activeRepoID;
      }

      let subscriptions: CodeSubscription[] = [];
      if (activeRepoID) {
        const subsResp = await this.apiClient.listCodeSubscriptions(
          'github',
          activeRepoID,
          token,
        );
        subscriptions = subsResp.data;
      }

      return {
        repositories: repos,
        subscriptions,
      };
    },
    args: () => [this.userContext?.user, this.selectedRepoID] as const,
  });

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
    return html`
      <tr>
        <td>
          <strong>${sub.target_query}</strong>
        </td>
        <td>
          ${sub.triggers.map(
            (tr: SubscriptionTrigger) => html`
              <sl-badge variant="neutral">${tr}</sl-badge>
            `,
          )}
        </td>
        <td>${this.renderOccurrences(sub)}</td>
        <td>
          <span class="status-badge status-${sub.status.toLowerCase()}">
            ${sub.status}
          </span>
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
              </tr>
            </thead>
            <tbody>
              <tr>
                <td colspan="4" class="empty-state">
                  No active code subscriptions found in this repository.
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
      <div class="container">
        <sl-alert variant="danger" open>
          <sl-icon slot="icon" name="exclamation-octagon"></sl-icon>
          <strong>Error loading code subscriptions:</strong> ${String(err)}
        </sl-alert>
        <sl-button @click="${() => this._dataTask.run()}">Retry</sl-button>
      </div>
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

    const taskResult = this._dataTask.render({
      pending: () => html`
        <div class="container">
          <sl-skeleton effect="sheen" style="height: 40px;"></sl-skeleton>
          <sl-skeleton effect="sheen" style="height: 100px;"></sl-skeleton>
          <sl-skeleton effect="sheen" style="height: 200px;"></sl-skeleton>
        </div>
      `,
      error: error => this.renderWhenError(error instanceof Error ? error : new Error(String(error))),
      complete: data => {
        if (!data || data.repositories.length === 0) {
          return html`
            <div class="container">
              <h2>Code Subscriptions</h2>
              <div class="empty-state">
                <p>No connected GitHub repositories found.</p>
                <p>
                  Install the webstatus.dev GitHub App on your repositories to
                  enable automated code subscriptions.
                </p>
              </div>
            </div>
          `;
        }

        return html`
          <div class="container">
            <div class="header-controls">
              <h2>Code Subscriptions</h2>
              <sl-select
                class="repo-selector"
                value="${this.selectedRepoID ?? ''}"
                @sl-change="${(e: CustomEvent) => {
                  const select = e.target;
                  if (
                    select instanceof SlSelect &&
                    typeof select.value === 'string'
                  ) {
                    this.selectedRepoID = select.value;
                  }
                }}"
              >
                ${data.repositories.map(
                  repo => html`
                    <sl-option value="${repo.repository_id}">
                      ${repo.full_name}
                    </sl-option>
                  `,
                )}
              </sl-select>
            </div>

            ${this.renderQuota(data.subscriptions.length)}
            ${this.renderSubscriptionsTable(data.subscriptions)}
          </div>
        `;
      },
    });

    return html`${taskResult}`;
  }
}
