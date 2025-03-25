/**
 * Copyright 2024 Google LLC
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

import {LitElement, html, type TemplateResult, CSSResultGroup, css} from 'lit';
import {customElement, property, state} from 'lit/decorators.js';
import {SHARED_STYLES} from '../css/shared-css.js';
import {GITHUB_REPO_ISSUE_LINK} from '../utils/constants.js';
import {getSearchQuery, formatFeaturePageUrl} from '../utils/urls.js';
import {consume} from '@lit/context';
import {APIClient, apiClientContext} from '../contexts/api-client-context.js';
import {Task, TaskStatus} from '@lit/task';
import {FeatureSortOrderType} from '../api/client.js';

@customElement('webstatus-not-found-error-page')
export class WebstatusNotFoundErrorPage extends LitElement {
  _loadingSimilarResults?: Task;

  @property({type: Object})
  location!: {search: string}; // Set by router.

  @consume({context: apiClientContext})
  @state()
  apiClient!: APIClient;

  @state()
  similarFeatures: {name: string; url: string}[] = [];

  constructor() {
    super();
    this._loadingSimilarResults = new Task(this, {
      args: () => [this.apiClient, getSearchQuery(this.location)],
      task: async ([apiClient, featureId]) => {
        if (!featureId) {
          this.similarFeatures = [];
          return;
        }
        try {
          const response = await apiClient.getFeatures(
            featureId,
            '' as FeatureSortOrderType,
            undefined,
            0,
            5,
          );
          const data = response.data;
          this.similarFeatures = Array.isArray(data)
            ? data.map(f => ({
                name: f.name,
                url: formatFeaturePageUrl(f),
              }))
            : [];
        } catch (fetchError) {
          console.error('Error fetching similar features:', fetchError);
          this.similarFeatures = [];
        }
      },
    });
  }

  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        #error-container {
          width: 100%;
          height: 100%;
          flex-direction: column;
          justify-content: center;
          align-items: center;
          display: inline-flex;
        }
        #error-header {
          align-self: stretch;
          height: 108px;
          flex-direction: column;
          justify-content: flex-start;
          align-items: center;
          gap: 12px;
          display: flex;
        }
        #error-status-code {
          color: #2563eb;
          font-size: 15px;
          font-weight: 700;
          line-height: 22.5px;
          word-wrap: break-word;
        }
        #error-actions {
          display: flex;
          flex-wrap: wrap;
          justify-content: center;
        }
        #error-headline {
          color: #1d2430;
          font-size: 32px;
          font-weight: 700;
          word-wrap: break-word;
        }
        #error-detailed-message {
          font-size: 15px;
          font-weight: 400;
          line-height: 22.5px;
          word-wrap: break-word;
        }

        .error-message {
          color: #6c7381;
        }
        .similar-features-container {
          text-align: left;
          padding: 12px;
          max-width: 400px;
        }
        .similar-results-header {
          color: #1a1a1a;
          font-weight: 500;
          margin-bottom: 6px;
        }
        .feature-list {
          list-style: none;
          padding: 0;
          margin: 0;
        }
        .feature-list li {
          padding: 6px 0;
        }
        .feature-list li a {
          text-decoration: none;
          color: #007bff;
          font-weight: 500;
        }
        .feature-list li a:hover {
          text-decoration: underline;
          color: #0056b3;
        }
      `,
    ];
  }

  protected render(): TemplateResult {
    const featureId = getSearchQuery(this.location);

    console.log('hello', featureId);

    return html`
      <div
        id="error-container"
        style="gap: ${this.similarFeatures.length > 0 ? '32px' : '48px'};"
      >
        <div id="error-header">
          <div id="error-status-code">404</div>
          <div id="error-headline">Page not found</div>
          <div id="error-detailed-message">
            ${featureId
              ? html`
                  We could not find Feature ID: <strong>${featureId}</strong>
                `
              : html`<span class="error-message"
                  >We couldn't find the page you're looking for.</span
                >`}
          </div>
        </div>

        ${featureId
          ? html`
              ${this._loadingSimilarResults?.status === TaskStatus.PENDING
                ? html`<p class="loading-message">
                    Loading similar features...
                  </p>`
                : this.similarFeatures.length > 0
                  ? html`
                      <div class="similar-features-container">
                        <p class="similar-results-header">
                          Here are some similar features:
                        </p>
                        <ul class="feature-list">
                          ${this.similarFeatures.map(
                            f =>
                              html`<li><a href="${f.url}">${f.name}</a></li>`,
                          )}
                        </ul>
                      </div>
                    `
                  : html`<p class="error-message">
                      No similar features found.
                    </p>`}
            `
          : ''}

        <div
          id="error-actions"
          style="gap: ${featureId && this.similarFeatures.length > 0
            ? '16px'
            : '32px'};"
        >
          ${featureId && this.similarFeatures.length > 0
            ? html`
                <sl-button
                  id="error-action-search-btn"
                  variant="primary"
                  href="/?q=${featureId}"
                >
                  Search for more similar features
                </sl-button>
              `
            : ''}
          <sl-button id="error-action-home-btn" variant="default" href="/">
            Go back home
          </sl-button>
          <sl-button
            id="error-action-report"
            variant="default"
            href="${GITHUB_REPO_ISSUE_LINK}"
            target="_blank"
          >
            <sl-icon name="github"></sl-icon>
            Report an issue
          </sl-button>
        </div>
      </div>
    `;
  }
}
