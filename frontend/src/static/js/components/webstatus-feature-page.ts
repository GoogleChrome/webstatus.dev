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
import {
  LitElement,
  type TemplateResult,
  html,
  CSSResultGroup,
  css,
  nothing,
} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {SHARED_STYLES} from '../css/shared-css.js';
import {type components} from 'webstatus.dev-backend';

import {type APIClient} from '../api/client.js';
import {formatFeaturePageUrl, formatOverviewPageUrl} from '../utils/urls.js';
import {apiClientContext} from '../contexts/api-client-context.js';
import {renderWPTScore} from './webstatus-overview-cells.js';

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

  location!: {params: {featureId: string}; search: string}; // Set by router.

  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        .crumbs {
          color: #aaa;
        }
        .crumbs a {
          text-decoration: none;
        }

        #nameAndOffsiteLinks {
          align-items: center;
        }

        .hbox,
        .vbox {
          gap: var(--content-padding-large);
        }

        .wptScore > div + div {
          margin-top: var(--content-padding-half);
        }
        .wptScore .icon {
          float: right;
        }
        .wptScore .score {
          font-size: 150%;
        }
        .wptScore .avail {
          color: var(--unimportant-text-color);
        }
        .chip.increased {
          background: var(--chip-background-increased);
          color: var(--chip-color-increased);
        }
        .chip.unchanged {
          background: var(--chip-background-unchanged);
          color: var(--chip-color-unchanged);
        }
        .chip.decreased {
          background: var(--chip-background-decreased);
          color: var(--chip-color-decreased);
        }

        .under-construction {
          min-height: 12em;
        }
      `,
    ];
  }

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

  renderNameAndOffsiteLinks(): TemplateResult {
    const mdnLink = '#TODO';
    const canIUseLink = '#TODO';
    return html`
      <div id="nameAndOffsiteLinks" class="hbox">
        <h1 class="halign-stretch">${this.feature!.name}</h1>
        <sl-button variant="default" href=${mdnLink}>
          <sl-icon slot="suffix" name="box-arrow-up-right"></sl-icon>
          MDN
        </sl-button>
        <sl-button variant="default" href=${canIUseLink}>
          <sl-icon slot="suffix" name="box-arrow-up-right"></sl-icon>
          CanIUse
        </sl-button>
      </div>
    `;
  }

  renderOneWPTCard(
    browser: components['parameters']['browserPathParam'],
    icon: string
  ): TemplateResult {
    const scorePart = this.feature
      ? renderWPTScore(this.feature, {search: ''}, {browser: browser})
      : nothing;

    return html`
      <sl-card class="halign-stretch wptScore">
        <img height="32" src="/public/img/${icon}" class="icon" />
        <div>${browser[0].toUpperCase() + browser.slice(1)}</div>
        <div class="score">
          ${scorePart}
          <span class="chip small increased">+1.2%</span>
        </div>
        <div class="avail">Available since ...</div>
      </sl-card>
    `;
  }

  renderWPTScores(): TemplateResult {
    return html`
      <section id="wpt-scores">
        <h3>Web platform test scores</h3>
        <div class="hbox" style="margin:0">
          ${this.renderOneWPTCard('chrome', 'chrome_32x32.png')}
          ${this.renderOneWPTCard('edge', 'edge_32x32.png')}
          ${this.renderOneWPTCard('firefox', 'firefox_32x32.png')}
          ${this.renderOneWPTCard('safari', 'safari_32x32.png')}
        </div>
      </section>
    `;
  }

  renderImplentationProgress(): TemplateResult {
    return html`
      <sl-card id="implementation-progress">
        <div slot="header">Implementation progress</div>
        <p class="under-construction">Chart goes here...</p>
      </sl-card>
    `;
  }

  renderCurrentBugs(): TemplateResult {
    return html`
      <sl-details id="current-bugs">
        <div slot="summary">Current bugs</div>
        <p class="under-construction">List goes here...</p>
      </sl-details>
    `;
  }

  renderAwarenes(): TemplateResult {
    return html`
      <sl-card class="halign-stretch" id="awarenss">
        <div slot="header">Awareness</div>
        <p class="under-construction">Small chart goes here...</p>
      </sl-card>
    `;
  }

  renderAdoption(): TemplateResult {
    return html`
      <sl-card class="halign-stretch" id="adoption">
        <div slot="header">Adoption</div>
        <p class="under-construction">Small chart goes here...</p>
      </sl-card>
    `;
  }

  renderGeneralInformation(): TemplateResult {
    return html`
      <sl-card id="general-information">
        <div slot="header">General information</div>
        <p class="under-construction">List goes here...</p>
      </sl-card>
    `;
  }

  renderWhenComplete(): TemplateResult {
    return html`
      <div class="vbox">
        ${this.renderCrumbs()} ${this.renderNameAndOffsiteLinks()}
        ${this.renderWPTScores()} ${this.renderImplentationProgress()}
        ${this.renderCurrentBugs()}
        <div class="hbox">
          ${this.renderAwarenes()} ${this.renderAdoption()}
        </div>
        ${this.renderGeneralInformation()}
      </div>
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
