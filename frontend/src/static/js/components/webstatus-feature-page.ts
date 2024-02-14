/**
 * Copyright 2023 Google LLC
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
import {LitElement, type TemplateResult, css, html} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {type components} from 'webstatus.dev-backend';

import {type APIClient} from '../api/client.js';
import {formatFeaturePageUrl, formatOverviewPageUrl} from '../utils/urls.js';
import {apiClientContext} from '../contexts/api-client-context.js';

@customElement('webstatus-feature-page')
export class FeaturePage extends LitElement {
  _loadingTask: Task;

  @consume({context: apiClientContext})
  @state()
  apiClient!: APIClient;

  @state()
  feature?: components['schemas']['Feature'] | undefined;

  @state()
  featureId!: string;

  location!: {params: {featureId: string}, search: string}; // Set by router.

  static styles = css`
    .crumbs {
      color: #aaa;
    }
    .crumbs a {
      text-decoration: none;
    }
  `;

  constructor() {
    super();
    this._loadingTask = new Task(this, {
      args: () => [this.apiClient, this.featureId],
      task: async ([apiClient, featureId]) => {
        if (typeof apiClient === 'object' && typeof featureId === 'string') {
          this.feature = await apiClient.getFeature(featureId);
        }
        return this.feature;
      },
    });
  }

  async firstUpdated(): Promise<void> {
    // TODO(jrobbins): Use routerContext instead of this.location so that
    // nested components could also access the router.
    this.featureId = this.location.params.featureId;
  }

  render(): TemplateResult | undefined {
    return this._loadingTask.render({
      complete: () => this.renderWhenComplete(),
      error: () => this.renderWhenError(),
      initial: () => this.renderWhenInitial(),
      pending: () => this.renderWhenPending(),
    });
  }

    renderCrumbs(): TemplateResult {
    const overviewUrl = formatOverviewPageUrl(this.location);
    const canonicalFeatureUrl = formatFeaturePageUrl(this.feature!);
    return html`
      <div class="crumbs">
        <a href=${overviewUrl}>Feature overview</a>
        &rsaquo;
        <a href=${canonicalFeatureUrl}>${this.feature!.name}</a>
      </div>
    `;
  }

  renderWhenComplete(): TemplateResult {
    return html`
      ${this.renderCrumbs()}
      <h1>${this.feature!.name}</h1>
      spec size: ${this.feature!.spec?.length || 0}
      <br />
      Specs:
    `;
  }

  renderWhenError(): TemplateResult {
    return html`Error when loading feature ${this.featureId}.`;
  }

  renderWhenInitial(): TemplateResult {
    return html`Preparing request for ${this.featureId}.`;
  }

  renderWhenPending(): TemplateResult {
    return html`Loading ${this.featureId}.`;
  }
}
