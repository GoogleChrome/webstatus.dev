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
import {type TemplateResult, html, CSSResultGroup, css, nothing} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {ifDefined} from 'lit/directives/if-defined.js';
import {SHARED_STYLES} from '../css/shared-css.js';
import {type components} from 'webstatus.dev-backend';

import {
  FeatureWPTMetricViewType,
  type APIClient,
  type WPTRunMetric,
  BROWSER_LABEL_TO_ID,
  TEST_COUNT_METRIC_VIEW,
  SUBTEST_COUNT_METRIC_VIEW,
  DEFAULT_TEST_VIEW,
} from '../api/client.js';
import {
  formatFeaturePageUrl,
  formatOverviewPageUrl,
  getWPTMetricView,
} from '../utils/urls.js';
import {apiClientContext} from '../contexts/api-client-context.js';
import {
  BASELINE_CHIP_CONFIGS,
  renderBrowserQuality,
} from './webstatus-overview-cells.js';

import './webstatus-gchart';
import {BaseChartsPage} from './webstatus-base-charts-page.js';

import './webstatus-feature-wpt-progress-chart-panel.js';
import './webstatus-feature-usage-chart-panel.js';
import {DataFetchedEvent} from './webstatus-line-chart-panel.js';
import {NotFoundError} from '../api/errors.js';
// CanIUseData is a slimmed down interface of the data returned from the API.
interface CanIUseData {
  items?: {
    id?: string;
  }[];
}

@customElement('webstatus-feature-page')
export class FeaturePage extends BaseChartsPage {
  _loadingTask?: Task;

  _loadingMetadataTask: Task;

  @consume({context: apiClientContext})
  @state()
  apiClient!: APIClient;

  @state()
  featureSupport = new Map<string, Array<WPTRunMetric>>();

  @state()
  feature?: components['schemas']['Feature'] | undefined;

  @state()
  featureMetadata?: {can_i_use?: CanIUseData; description?: string} | undefined;

  featureId!: string;

  // Members that are used for testing with sinon.
  _getWPTMetricView: (options: {search: string}) => string = getWPTMetricView;

  static get styles(): CSSResultGroup {
    return [
      super.styles!,
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

        sl-card .card {
          height: 100%;
        }

        .wptScore {
          width: 16%;
        }
        .wptScore.baseline {
          width: 20%;
        }
        @media (max-width: 1100px) {
          .wptScore {
            width: 32%;
          }
          .wptScore.baseline {
            width: 50%;
          }
        }
        @media (max-width: 800px) {
          .wptScore {
            width: 60%;
          }
          .wptScore.baseline {
            width: 80%;
          }
        }

        .wptScore > div + div {
          margin-top: var(--content-padding-half);
        }
        .wptScore .icon {
          float: right;
        }
        .wptScore .score {
          font-size: 150%;
          white-space: nowrap;
        }
        .wptScore.baseline .score {
          font-size: 150%;
          white-space: wrap;
        }
        .wptScore .avail {
          color: var(--unimportant-text-color);
          font-size: 90%;
        }
        sl-skeleton {
          width: 4em;
        }
        sl-skeleton.icon {
          height: 2em;
          width: 2em;
        }
        .avail sl-skeleton {
          width: 8em;
        }
        .logo-button {
          gap: var(--content-padding-half);
          align-items: center;
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

        baseline-date {
          font-size: 0.8em;
        }

        #current-bugs li {
          list-style: none;
          margin-bottom: var(--content-padding);
        }

        #implementation-progress::part(base) {
          min-height: 350px;
        }

        #general-information .vbox {
          gap: var(--content-padding);
        }

        .info-section-header {
          font-weight: bold;
          width: 16em;
        }

        dt {
          font-weight: bold;
        }

        dd {
          margin-bottom: var(--content-padding-large);
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
          const wptMetricView = getWPTMetricView(
            this.location,
          ) as FeatureWPTMetricViewType;
          this.feature = await apiClient.getFeature(featureId, wptMetricView);
          return this.feature;
        }
        return Promise.reject('api client and/or featureId not set');
      },
      onError: async error => {
        if (error instanceof NotFoundError) {
          const queryParam = this.featureId ? `?q=${this.featureId}` : '';

          // TODO: cannot use navigateToUrl because it creates a
          // circular dependency.
          // For now use the window href and revisit when navigateToUrl
          // is move to another location.
          window.location.href = `/errors-404/feature-not-found${queryParam}`;
        } else {
          console.error('Unexpected error in _loadingTask:', error);
        }
      },
    });

    this._loadingMetadataTask = new Task(this, {
      args: () => [this.apiClient, this.featureId],
      task: async ([apiClient, featureId]) => {
        if (typeof apiClient === 'object' && typeof featureId === 'string') {
          this.featureMetadata = await apiClient.getFeatureMetadata(featureId);
        }
        return this.featureMetadata;
      },
    });
  }

  override async firstUpdated(): Promise<void> {
    await super.firstUpdated();
    this.featureId =
      this.location.params['featureId']?.toString() || 'undefined';
  }

  render(): TemplateResult {
    return html`
      ${this._loadingTask?.render({
        complete: () => this.renderWhenComplete(),
        error: () => this.renderWhenError(),
        initial: () => this.renderWhenInitial(),
        pending: () => this.renderWhenPending(),
      })}
    `;
  }

  renderCrumbs(): TemplateResult {
    const overviewUrl = formatOverviewPageUrl(this.location);
    const canonicalFeatureUrl = this.feature
      ? formatFeaturePageUrl(this.feature!)
      : this.location;
    return html`
      <div class="crumbs">
        <a href=${overviewUrl}>Features overview</a>
        &rsaquo;
        <a href=${canonicalFeatureUrl} router-ignore
          >${this.feature?.name || this.featureId}</a
        >
      </div>
    `;
  }

  renderOffsiteLink(
    label: string,
    link: string | null,
    logo?: string,
    logoAlt?: string,
  ): TemplateResult {
    if (!link) {
      return html``;
    }
    return html`
      <sl-button variant="default" href=${link} target="_blank">
        <sl-icon slot="suffix" name="box-arrow-up-right"></sl-icon>
        <div class="hbox logo-button">
          ${logo
            ? html`<img
                src=${logo}
                alt="${ifDefined(logoAlt)}"
                width="24"
                height="24"
              />`
            : nothing}
          ${label}
        </div>
      </sl-button>
    `;
  }

  wptLinkMetricView(): string {
    const view = this._getWPTMetricView(this.location);
    switch (view) {
      case SUBTEST_COUNT_METRIC_VIEW:
        return 'subtest';
      case TEST_COUNT_METRIC_VIEW:
      default:
        return 'test';
    }
  }

  metricViewForRequests(): FeatureWPTMetricViewType {
    const view = this._getWPTMetricView(this.location);
    switch (view) {
      case SUBTEST_COUNT_METRIC_VIEW:
      case TEST_COUNT_METRIC_VIEW:
        return view;
      default:
        return DEFAULT_TEST_VIEW;
    }
  }

  buildWPTLink(feature?: {
    feature_id: string;
    wpt?: {stable?: object; experimental?: object};
  }): string | null {
    if (feature?.wpt?.stable === undefined) return null;
    const wptLinkURL = new URL('https://wpt.fyi/results');
    const query = `feature:${feature.feature_id} !is:tentative`;
    wptLinkURL.searchParams.append('label', 'master');
    wptLinkURL.searchParams.append('label', 'stable');
    wptLinkURL.searchParams.append('q', query);
    wptLinkURL.searchParams.append('view', this.wptLinkMetricView());
    return wptLinkURL.toString();
  }

  findCanIUseLink(data?: CanIUseData): string | null {
    // For now, only return a link if there is exactly one item.
    // For null or more than 1 item, return null.
    // TODO. Discuss what should happen if we have more than one id.
    if (!data || !data.items || data.items.length !== 1) {
      return null;
    }

    return `https://caniuse.com/${data.items[0].id}`;
  }

  renderNameDescriptionControls(): TemplateResult {
    return html`
      <div id="nameAndOffsiteLinks" class="hbox wrap">
        <div class="vbox">
          <h1>${this.feature?.name || this.featureId}</h1>
          ${this.renderDescription()}
        </div>
        <div class="spacer"></div>
        ${this.renderDateRangePicker()}
      </div>
    `;
  }

  renderDeltaChip(
    browser: components['parameters']['browserPathParam'],
  ): TemplateResult {
    const runs = this.featureSupport.get(browser);
    if (runs === undefined || runs.length === 0) {
      return html` <span class="chip small unchanged"></span> `;
    }

    // Runs are retrieved in descending chronological order.
    const mostRecentRun = runs[0];
    const oldestRun = runs[runs.length - 1];
    const mostRecentPercent =
      mostRecentRun.test_pass_count! / mostRecentRun.total_tests_count!;
    const oldestPercent =
      oldestRun.test_pass_count! / oldestRun.total_tests_count!;
    const delta = (mostRecentPercent - oldestPercent) * 100.0;
    let deltaStr = Number(delta).toFixed(1) + '%';
    let deltaClass = 'unchanged';
    if (delta > 0) {
      deltaStr = '+' + deltaStr;
      deltaClass = 'increased';
    } else if (delta < 0) {
      deltaClass = 'decreased';
    } else {
      deltaClass = 'unchanged';
    }
    return html` <span class="chip small ${deltaClass}">${deltaStr}</span> `;
  }

  renderBrowserImpl(
    browserImpl?: components['schemas']['BrowserImplementation'],
  ): TemplateResult {
    const sinceDate: string | undefined = browserImpl?.date;
    const sincePhrase =
      sinceDate && this.endDate > new Date(sinceDate)
        ? 'Available since'
        : 'Became available on';
    const sinceVersion: string | undefined = browserImpl?.version;
    const versionText = sinceVersion ? 'in version ' + sinceVersion : '';

    return html`
      ${sinceDate
        ? html`<div class="avail">
            ${sincePhrase} ${sinceDate} ${versionText}
          </div>`
        : nothing}
    `;
  }

  renderOneWPTCard(
    browser: components['parameters']['browserPathParam'],
    icon: string,
  ): TemplateResult {
    const scorePart = this.feature
      ? renderBrowserQuality(this.feature, {search: ''}, {browser: browser})
      : html`<sl-skeleton effect="sheen"></sl-skeleton>`;
    const browserImpl = this.feature?.browser_implementations?.[browser];

    return html`
      <sl-card class="halign-stretch wptScore">
        <img height="32" src="/public/img/${icon}" class="icon" />
        <div>${browser[0].toUpperCase() + browser.slice(1)}</div>
        <div class="score">${scorePart} ${this.renderDeltaChip(browser)}</div>
        ${this.renderBrowserImpl(browserImpl)}
      </sl-card>
    `;
  }

  renderBaselineCardWhenPending(): TemplateResult {
    return html`
      <sl-card class="halign-stretch wptScore baseline">
        <sl-skeleton effect="sheen" class="icon"></sl-skeleton>
        <div>Baseline</div>
        <div class="score"><sl-skeleton effect="sheen"></sl-skeleton></div>
        <div class="avail"><sl-skeleton effect="sheen"></sl-skeleton></div>
      </sl-card>
    `;
  }

  renderBaselineCard(): TemplateResult {
    if (this.feature === undefined) return this.renderBaselineCardWhenPending();
    const status = this.feature?.baseline?.status;
    if (status === undefined) return html``;

    const chipConfig = BASELINE_CHIP_CONFIGS[status];
    const sinceDate = this.feature?.baseline?.low_date;
    return html`
      <sl-card class="halign-stretch wptScore baseline">
        <img height="28" src="/public/img/${chipConfig.icon}" class="icon" />
        <div>Baseline</div>
        <div class="score">${chipConfig.word}</div>
        ${sinceDate
          ? html`<div class="avail">Baseline since ${sinceDate}</div>`
          : nothing}
      </sl-card>
    `;
  }

  renderDescription(): TemplateResult {
    if (this.featureMetadata?.description === undefined) {
      return html`${nothing}`;
    }

    return html`
      <div id="feature-description">
        <h3>${this.featureMetadata.description}</h3>
      </div>
    `;
  }

  renderWPTScores(): TemplateResult {
    return html`
      <section id="wpt-scores">
        <h3>Web platform test scores</h3>
        <div class="wptScores hbox wrap" style="margin:0">
          ${this.renderOneWPTCard('chrome', 'chrome_32x32.png')}
          ${this.renderOneWPTCard('edge', 'edge_32x32.png')}
          ${this.renderOneWPTCard('firefox', 'firefox_32x32.png')}
          ${this.renderOneWPTCard('safari', 'safari_32x32.png')}
          ${this.renderBaselineCard()}
        </div>
      </section>
    `;
  }

  renderImplentationProgress(): TemplateResult {
    return html`
      <webstatus-feature-wpt-progress-chart-panel
        .featureId=${this.featureId}
        .testView=${this.metricViewForRequests()}
        .startDate=${this.startDate}
        .endDate=${this.endDate}
        @data-fetch-complete=${this.handleWPTScoresFetched}
      ></webstatus-feature-wpt-progress-chart-panel>
    `;
  }

  handleWPTScoresFetched(e: DataFetchedEvent<WPTRunMetric>) {
    this.featureSupport = new Map(
      Array.from(e.detail, ([key, value]) => [
        BROWSER_LABEL_TO_ID[key],
        value.data,
      ]),
    );
  }

  renderFeatureUsage(): TemplateResult {
    return html`
      <webstatus-feature-usage-chart-panel
        .featureId=${this.featureId}
        .startDate=${this.startDate}
        .endDate=${this.endDate}
      >
      </webstatus-feature-usage-chart-panel>
    `;
  }

  renderBug(bugId: number): TemplateResult {
    return html`
      <li>
        <a href="#TODO" target="_blank">
          <img height="16" src="/public/img/chrome_24x24.png" />
          ${bugId}: Title of issue
        </a>
      </li>
    `;
  }

  renderCurrentBugs(): TemplateResult {
    return html`
      <sl-details id="current-bugs">
        <div slot="summary">Current bugs</div>
        <ul class="under-construction">
          ${[21830, 123412, 12983712, 1283, 987123, 12982, 1287].map(bugId =>
            this.renderBug(bugId),
          )}
        </ul>
      </sl-details>
    `;
  }

  renderWhenComplete(): TemplateResult {
    const wptLink = this.buildWPTLink(this.feature);
    const wptLogo = '/public/img/wpt-logo.svg';
    const canIUseLink = this.findCanIUseLink(this.featureMetadata?.can_i_use);

    return html`
      <div class="vbox">
        <div class="hbox wrap">
          ${this.renderCrumbs()}
          <div class="spacer"></div>

          <div class="hbox wrap">
            ${this.renderOffsiteLink(
              'WPT.fyi',
              wptLink,
              wptLogo,
              'WPT default view',
            )}
            ${this.renderOffsiteLink('MDN', null)}
            ${this.renderOffsiteLink('CanIUse', canIUseLink)}
          </div>
        </div>

        ${this.renderNameDescriptionControls()} ${this.renderWPTScores()}
        ${this.renderImplentationProgress()} ${this.renderFeatureUsage()}
      </div>
    `;

    // TODO: Fetch and display current bugs.
    //   ${this.renderCurrentBugs()}
  }

  renderWhenError(): TemplateResult {
    return html`Error when loading feature ${this.featureId}.`;
  }

  renderWhenInitial(): TemplateResult {
    return html`Preparing request for ${this.featureId}.`;
  }

  renderWhenPending(): TemplateResult {
    // Lower-level render functions check for missing this.feature.
    return this.renderWhenComplete();
  }
}
