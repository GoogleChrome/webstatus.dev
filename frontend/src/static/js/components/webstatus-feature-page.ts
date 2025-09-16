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
  BROWSER_ID_TO_LABEL,
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
  getBaselineChipConfig,
  renderBrowserQuality,
} from './webstatus-overview-cells.js';

import './webstatus-gchart';
import {BaseChartsPage} from './webstatus-base-charts-page.js';

import './webstatus-feature-wpt-progress-chart-panel.js';
import './webstatus-feature-usage-chart-panel.js';
import {DataFetchedEvent} from './webstatus-line-chart-panel.js';
import {
  FeatureGoneSplitError,
  FeatureMovedError,
  NotFoundError,
} from '../api/errors.js';
import {formatDeveloperUpvotesMessages} from '../utils/format.js';
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

  @state()
  oldFeatureId?: string;

  @state()
  _isMoved = false;

  @state()
  _newFeatureId?: string;

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
          width: 22%;
        }
        .wptScore.baseline {
          width: 22%;
        }
        @media (max-width: 1100px) {
          .wptScore {
            width: 32%;
          }
        }
        @media (max-width: 800px) {
          .wptScore {
            width: 60%;
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

        .discouraged-info li {
          margin-left: var(--content-padding);
        }

        .discouraged-info img {
          width: 20px;
        }
      `,
    ];
  }

  constructor() {
    super();
    this._loadingTask = new Task(this, {
      args: () => [this.apiClient, this.featureId],
      task: async ([apiClient, featureId]) => {
        if (typeof apiClient !== 'object' || typeof featureId !== 'string') {
          return Promise.reject('api client and/or featureId not set');
        }
        try {
          const wptMetricView = getWPTMetricView(
            this.location,
          ) as FeatureWPTMetricViewType;
          const feature = await apiClient.getFeature(featureId, wptMetricView);
          this.feature = feature;
          return feature;
        } catch (error) {
          if (error instanceof FeatureMovedError) {
            this.handleMovedFeature(featureId, error);
            // The task can now complete successfully with the new feature data.
            return error.feature;
          }
          // For other errors, re-throw them to be handled by onError.
          throw error;
        }
      },
      onError: async error => {
        // FeatureMovedError is now handled in the task, so it won't appear here.
        if (error instanceof NotFoundError) {
          const queryParam = this.featureId ? `?q=${this.featureId}` : '';

          // TODO: cannot use navigateToUrl because it creates a
          // circular dependency.
          // For now use the window href and revisit when navigateToUrl
          // is move to another location.
          window.location.href = `/errors-404/feature-not-found${queryParam}`;
        } else if (error instanceof FeatureGoneSplitError) {
          const newFeatureIds = error.newFeatureIds.join(',');
          const queryParam = newFeatureIds
            ? `?new_features=${newFeatureIds}`
            : '';
          window.location.href = `/errors-410/feature-gone-split${queryParam}`;
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

  handleMovedFeature(oldFeatureId: string, error: FeatureMovedError) {
    const newFeature = error.feature;
    const newFeatureId = error.newFeatureId;

    // Set component state to render the new feature.
    this.feature = newFeature;
    this.featureId = newFeatureId;
    this.oldFeatureId = oldFeatureId; // Used to show a redirect notice.

    // Update browser URL and history.
    const newUrl = `/features/${newFeatureId}?redirected_from=${oldFeatureId}`;
    history.pushState(null, '', newUrl);

    // Update the canonical URL in the document head for SEO.
    document.head.querySelector('link[rel="canonical"]')?.remove();
    const canonical = document.createElement('link');
    canonical.rel = 'canonical';
    // The canonical URL should be clean, without the 'redirected_from' param.
    canonical.href = `/features/${newFeatureId}`;
    document.head.appendChild(canonical);

    // Update the page title.
    document.title = newFeature.name || newFeatureId;
  }

  override async firstUpdated(): Promise<void> {
    await super.firstUpdated();
    this.featureId =
      this.location.params['featureId']?.toString() || 'undefined';
    const urlParams = new URLSearchParams(this.location.search);
    this.oldFeatureId = urlParams.get('redirected_from') || undefined;
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

  renderRedirectNotice(): TemplateResult {
    if (!this.oldFeatureId) {
      return html`${nothing}`;
    }

    return html`
      <sl-alert variant="primary" open closable>
        <sl-icon slot="icon" name="info-circle"></sl-icon>
        You have been redirected from an old feature ID
        (<strong>${this.oldFeatureId}</strong>).
      </sl-alert>
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

  renderDeveloperSignal(
    signal?: components['schemas']['FeatureDeveloperSignals'],
  ): TemplateResult {
    if (!signal?.link || !signal?.upvotes) {
      return html`${nothing}`;
    }

    const messages = formatDeveloperUpvotesMessages(signal.upvotes);

    return html`
      <sl-tooltip content=${messages.message}>
        <sl-button
          href=${signal.link}
          target="_blank"
          variant="default"
          size="small"
          aria-label="${messages.shortMessage}"
        >
          <sl-icon slot="prefix" name="hand-thumbs-up"></sl-icon>
          ${messages.shorthandNumber}
        </sl-button>
      </sl-tooltip>
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

  renderDiscouragedNotice(
    discouragedDetails?: components['schemas']['FeatureDiscouragedInfo'],
  ): TemplateResult {
    if (!discouragedDetails) {
      return html`${nothing}`;
    }
    // If there are links to documentation, build a section for that
    const accordingTo = discouragedDetails.according_to;
    let accordingToSection: TemplateResult = html`${nothing}`;
    if (accordingTo && accordingTo.length > 0) {
      accordingToSection = html`
        <br />
        For the rationale, see:
        <ul>
          ${accordingTo.map(
            f => html`<li><a href="${f.link}">${f.link}</a></li>`,
          )}
        </ul>
      `;
    }

    // If there are alternatives, build a section for that
    const alternatives = discouragedDetails.alternatives;
    let alternativesSection: TemplateResult = html`${nothing}`;
    if (alternatives && alternatives.length > 0) {
      alternativesSection = html`
        <br />
        Consider using the following features instead:
        <ul>
          ${alternatives.map(
            f => html`<li><a href="/features/${f.id}">${f.id}</a></li>`,
          )}
        </ul>
      `;
    }

    return html`
      <div class="hbox">
        <sl-alert variant="neutral" open class="discouraged-info">
          <img
            src="/public/img/discouraged.svg"
            class="discouraged-icon"
            slot="icon"
          />
          <h3>Discouraged</h3>
          Avoid using this feature in new projects. This feature may be a
          candidate for removal from web standards or browsers.
          ${accordingToSection} ${alternativesSection}
        </sl-alert>
      </div>
    `;
  }

  renderNameDescriptionControls(): TemplateResult {
    return html`
      <div id="nameAndOffsiteLinks" class="hbox wrap">
        <div class="vbox">
          <div class="hbox valign-items-center">
            <h1>${this.feature?.name || this.featureId}</h1>
            ${this.renderDeveloperSignal(this.feature?.developer_signals)}
          </div>
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
        <div>${BROWSER_ID_TO_LABEL[browser]}</div>
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

    const chipConfig = getBaselineChipConfig(status, this.feature.discouraged);
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
          ${this.renderOneWPTCard('chrome_android', 'chrome_32x32.png')}
          ${this.renderOneWPTCard('firefox_android', 'firefox_32x32.png')}
          ${this.renderOneWPTCard('safari_ios', 'safari_32x32.png')}
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
    if (this.featureSupport === undefined) {
      this.featureSupport = new Map();
    }
    for (const [key, value] of e.detail) {
      this.featureSupport.set(BROWSER_LABEL_TO_ID[key], value.data);
    }
    this.requestUpdate();
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
        ${this.renderRedirectNotice()}
        ${this.renderDiscouragedNotice(this.feature?.discouraged)}
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
