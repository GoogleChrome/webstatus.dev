/**
 * Copyright 2025 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may not use this file except in compliance with the License.
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
import {consume} from '@lit/context';
import {APIClient, apiClientContext} from '../contexts/api-client-context.js';
import {Task} from '@lit/task';
import {FeatureWPTMetricViewType} from '../api/client.js';
import {formatFeaturePageUrl, getWPTMetricView} from '../utils/urls.js';

type NewFeature = {name: string; url: string};

@customElement('webstatus-feature-gone-split-page')
export class WebstatusFeatureGoneSplitPage extends LitElement {
  _newFeatures?: Task<[APIClient, string], NewFeature[]>;

  @property({type: Object})
  location!: {search: string}; // Set by router.

  @consume({context: apiClientContext})
  @state()
  apiClient!: APIClient;

  constructor() {
    super();
    this._newFeatures = new Task<[APIClient, string], NewFeature[]>(this, {
      args: () => {
        const params = new URLSearchParams(this.location.search);
        const newFeatures = params.get('new_features') || '';
        return [this.apiClient, newFeatures];
      },
      task: async ([apiClient, newFeatures]) => {
        if (!newFeatures) return [];
        const featureIds = newFeatures.split(',');
        const wptMetricView = getWPTMetricView(
          this.location,
        ) as FeatureWPTMetricViewType;
        const features = await Promise.all(
          featureIds.map(id => apiClient.getFeature(id, wptMetricView)),
        );
        return features.map(f => ({
          name: f.name,
          url: formatFeaturePageUrl(f),
        }));
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
          gap: 32px;
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
          color: var(--status-code-color);
          font-size: 15px;
          font-weight: 700;
          line-height: 22.5px;
          word-wrap: break-word;
        }
        #error-actions {
          display: flex;
          flex-wrap: wrap;
          justify-content: center;
          gap: var(--content-padding);
        }
        #error-headline {
          color: var(--heading-color);
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
          color: var(--unimportant-text-color);
        }
        .new-features-container {
          text-align: left;
          padding: 12px;
          max-width: 400px;
        }
        .new-results-header {
          color: var(--default-color);
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
          color: var(--link-color);
          font-weight: 500;
        }
        .feature-list li a:hover {
          text-decoration: underline;
          color: var(--link-hover-color);
        }
      `,
    ];
  }

  private _renderErrorHeader(): TemplateResult {
    return html`
      <div id="error-header">
        <div id="error-status-code">410</div>
        <div id="error-headline">Feature Gone</div>
        <div id="error-detailed-message">
          <span class="error-message">
            This feature has been split into multiple new features.
          </span>
        </div>
      </div>
    `;
  }

  private _renderNewFeatures(
    features: NewFeature[] | undefined,
  ): TemplateResult {
    if (!features?.length) {
      return html`<p class="error-message">No new features found.</p>`;
    }
    return html`
      <div class="new-features-container">
        <p class="new-results-header">Please see the following new features:</p>
        <ul class="feature-list">
          ${features.map(f => html`<li><a href="${f.url}">${f.name}</a></li>`)}
        </ul>
      </div>
    `;
  }

  private _renderActionButtons(): TemplateResult {
    return html`
      <div id="error-actions">
        <sl-button id="error-action-home-btn" variant="primary" href="/">
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
    `;
  }

  protected render(): TemplateResult {
    return html`
      <div id="error-container">
        ${this._renderErrorHeader()}
        ${this._newFeatures?.render({
          initial: () =>
            html`<p class="loading-message">Loading new features...</p>`,
          pending: () =>
            html`<p class="loading-message">Loading new features...</p>`,
          complete: features =>
            html` ${this._renderNewFeatures(features)}
            ${this._renderActionButtons()}`,
          error: error =>
            html`<p class="error-message">
              Oops, something went wrong: ${error}
            </p>`,
        })}
      </div>
    `;
  }
}
