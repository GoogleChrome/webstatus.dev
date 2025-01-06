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
import {Task, TaskStatus} from '@lit/task';
import {
  LitElement,
  type TemplateResult,
  html,
  CSSResultGroup,
  css,
  nothing,
} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {ifDefined} from 'lit/directives/if-defined.js';
import {SHARED_STYLES} from '../css/shared-css.js';
import {type components} from 'webstatus.dev-backend';
import {SlMenu, SlMenuItem} from '@shoelace-style/shoelace/dist/shoelace.js';

import {
  ALL_BROWSERS,
  BROWSER_ID_TO_LABEL,
  BROWSER_ID_TO_COLOR,
  STABLE_CHANNEL,
  FeatureWPTMetricViewType,
  type APIClient,
  type BrowsersParameter,
  type ChannelsParameter,
  type WPTRunMetric,
  type ChromiumUsageStat,
} from '../api/client.js';
import {
  formatFeaturePageUrl,
  formatOverviewPageUrl,
  getDateRange,
  getWPTMetricView,
  updateFeaturePageUrl,
} from '../utils/urls.js';
import {apiClientContext} from '../contexts/api-client-context.js';
import {
  BASELINE_CHIP_CONFIGS,
  renderBrowserQuality,
} from './webstatus-overview-cells.js';

import './webstatus-loading-overlay.js';
import './webstatus-gchart';
import {WebStatusDataObj} from './webstatus-gchart.js';
import {NotFoundError} from '../api/errors.js';

type LoadingTaskType = '_loadingMetricsTask' | '_loadingUsageTask';

/** Generate a key for featureSupport. */
function featureSupportKey(
  browser: BrowsersParameter,
  channel?: ChannelsParameter,
): string {
  return `${browser}-${channel}`;
}

function isValidDate(d: Date): boolean {
  return !isNaN(d.getTime());
}

// CanIUseData is a slimmed down interface of the data returned from the API.
interface CanIUseData {
  items?: {
    id?: string;
  }[];
}

@customElement('webstatus-feature-page')
export class FeaturePage extends LitElement {
  _loadingTask?: Task;

  _loadingMetadataTask: Task;

  _loadingMetricsTask?: Task;

  _loadingUsageTask?: Task;

  @consume({context: apiClientContext})
  @state()
  apiClient!: APIClient;

  @state()
  featureSupportBrowsers: BrowsersParameter[] = ALL_BROWSERS;

  @state()
  featureUsageBrowsers: BrowsersParameter[] = ['chrome'];

  @state()
  // Default: Date.now() - 1 year.
  startDate: Date = new Date(Date.now() - 365 * 24 * 60 * 60 * 1000);

  @state()
  // Default: Date.now()).
  endDate: Date = new Date();

  // Map from browser-channel to  feature support.
  // The key is generated by featureSupportKey().
  @state()
  featureSupport = new Map<string, Array<WPTRunMetric>>();

  @state()
  featureUsage = new Map<string, Array<ChromiumUsageStat>>();

  @state()
  featureSupportChartOptions = {};

  @state()
  featureUsageChartOptions = {};

  @state()
  featureSupportChartDataObj: WebStatusDataObj | undefined;

  @state()
  featureUsageChartDataObj: WebStatusDataObj | undefined;

  @state()
  feature?: components['schemas']['Feature'] | undefined;

  @state()
  featureMetadata?: {can_i_use?: CanIUseData; description?: string} | undefined;

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
    // Get date range from query parameters.
    const dateRange = getDateRange({search: location.search});
    if (dateRange) {
      this.startDate = dateRange.start || this.startDate;
      this.endDate = dateRange.end || this.endDate;
    }
    this._loadingTask = new Task(this, {
      args: () => [this.apiClient, this.featureId],
      task: async ([apiClient, featureId]) => {
        if (typeof apiClient === 'object' && typeof featureId === 'string') {
          const wptMetricView = getWPTMetricView(
            location,
          ) as FeatureWPTMetricViewType;
          this.feature = await apiClient.getFeature(featureId, wptMetricView);
          return this.feature;
        }
        return Promise.reject('api client and/or featureId not set');
      },
    });
    // Temporarily to avoid the no-floating-promises error.
    void this._startFeatureSupportTask(false);
    void this._startFeatureUsageTask(false);

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

  updateUrl() {
    // Update the URL to include the current date range.
    const overrides = {dateRange: {start: this.startDate, end: this.endDate}};
    updateFeaturePageUrl({feature_id: this.featureId}, location, overrides);
  }

  async handleBrowserSelection(event: Event) {
    const menu = event.target as SlMenu;
    const menuItemsArray: Array<SlMenuItem> = Array.from(menu.children).filter(
      child => child instanceof SlMenuItem,
    ) as Array<SlMenuItem>;

    // Build the list of values of checked menu-items.
    this.featureSupportBrowsers = menuItemsArray
      .filter(menuItem => menuItem.checked)
      .map(menuItem => menuItem.value) as BrowsersParameter[];
    // Regenerate data and redraw.  We should instead just filter it.
    await this._startFeatureSupportTask(true);
    this.generateFeatureSupportChartOptions();
  }

  async handleStartDateChange(event: Event) {
    const currentStartDate = this.startDate;
    const newStartDate = new Date((event.target as HTMLInputElement).value);
    if (
      isValidDate(newStartDate) &&
      newStartDate.getTime() !== currentStartDate.getTime()
    ) {
      this.startDate = newStartDate;
      this.updateUrl();
      await Promise.all([
        this._startFeatureSupportTask(true),
        this._startFeatureUsageTask(true),
      ]);
    }
  }

  async handleEndDateChange(event: Event) {
    const currentEndDate = this.endDate;
    const newEndDate = new Date((event.target as HTMLInputElement).value);
    if (
      isValidDate(newEndDate) &&
      newEndDate.getTime() !== currentEndDate.getTime()
    ) {
      this.endDate = newEndDate;
      this.updateUrl();
      await Promise.all([
        this._startFeatureSupportTask(true),
        this._startFeatureUsageTask(true),
      ]);
    }
  }

  createDataFromMap<T>(
    data: Map<string, T[]>,
    browsers: BrowsersParameter[],
    browserDataExtractor: (
      data: Map<string, T[]>,
      browser: BrowsersParameter,
    ) => T[] | undefined,
    valueExtractor: (row: T) => [string, number?, number?],
    tooltipGenerator: (row: T, browser: BrowsersParameter) => string,
    totalLabel?: string,
  ): WebStatusDataObj {
    const dataObj: WebStatusDataObj = {cols: [], rows: []};
    dataObj.cols.push({type: 'date', label: 'Date', role: 'domain'});
    for (const browser of browsers) {
      const browserLabel = BROWSER_ID_TO_LABEL[browser];
      dataObj.cols.push({type: 'number', label: browserLabel, role: 'data'});
      dataObj.cols.push({
        type: 'string',
        label: `${browserLabel} tooltip`,
        role: 'tooltip',
      });
    }
    if (totalLabel) {
      dataObj.cols.push({
        type: 'number',
        label: totalLabel,
        role: 'data',
      });
    }

    // We build a map from each time slot for which any browser has data.
    // to an array of data for all browsers (in dateToBrowserDataMap)
    // along with the total_tests_count for that time.
    // Since times may be slightly different for data associated with each browser,
    // we round times to the nearest 1 hour as a compromise.
    // The total ought to be the same for all browsers,
    // but this is not the case due to upstream problems.
    // As a workaround, we will instead use the max of all the
    // browser's totals for each time slot.
    // So effectively, for each unique time slot, we merge the data
    // for all the browsers while computing the max of the total value for
    // each of the browsers.
    const dateToTotalMap = new Map<number, number>();

    // Map from date to an object with counts for each browser
    const dateToBrowserDataMap = new Map<
      number,
      {[key: string]: {tooltip: string; value: number}}
    >();

    for (const browser of browsers) {
      const browserData = browserDataExtractor(data, browser);
      if (!browserData) continue;
      for (const row of browserData) {
        if (!row) continue;
        const [dateString, value, totalValue] = valueExtractor(row);
        const timestampMs = new Date(dateString).getTime();
        // Round timestamp to the nearest hour.
        const msInHour = 1000 * 60 * 60 * 1;
        const roundedTimestamp = Math.round(timestampMs / msInHour) * msInHour;
        const tooltip = tooltipGenerator(row, browser);
        if (!dateToBrowserDataMap.has(roundedTimestamp)) {
          dateToBrowserDataMap.set(roundedTimestamp, {});
        }
        if (totalLabel) {
          const total = Math.max(
            dateToTotalMap.get(roundedTimestamp) || 0,
            totalValue || 0,
          );
          dateToTotalMap.set(roundedTimestamp, total);
        } else {
          dateToTotalMap.set(roundedTimestamp, 100);
        }
        const browserCounts = dateToBrowserDataMap.get(roundedTimestamp)!;
        browserCounts[browser] = {tooltip, value: value!};
      }
    }

    // Create array of dateToBrowserDataMap entries and sort by roundedTimestamp
    const browserData = Array.from(dateToBrowserDataMap.entries()).sort(
      ([d1], [d2]) => d1 - d2,
    );

    // For each date, add a row to the dataObj
    for (const datum of browserData) {
      const dateMs = datum[0];
      const date = new Date(dateMs);
      const browserCounts = datum[1];

      // Make an array of browser counts, in the order of selected browsers.
      // If the browser is not in the browserCounts, add null.
      const browserCountArray: Array<number | string | null> = [];
      browsers.forEach(browser => {
        const countAndTooltip = browserCounts[browser];
        if (countAndTooltip) {
          browserCountArray.push(countAndTooltip.value);
          browserCountArray.push(countAndTooltip.tooltip);
        } else {
          browserCountArray.push(null);
          browserCountArray.push(null);
        }
      });
      if (totalLabel) {
        const total = dateToTotalMap.get(dateMs)!;
        dataObj.rows.push([date, ...browserCountArray, total]);
      } else {
        dataObj.rows.push([date, ...browserCountArray]);
      }
    }
    return dataObj;
  }

  // Make a DataTable from the data in featureSupport
  createFeatureSupportDataFromMap(): WebStatusDataObj {
    return this.createDataFromMap(
      this.featureSupport,
      this.featureSupportBrowsers,
      (data, browser) => data.get(featureSupportKey(browser, 'stable')),
      row => [row.run_timestamp, row.test_pass_count!, row.total_tests_count!],
      (row, browser) =>
        `${BROWSER_ID_TO_LABEL[browser]}: ${row.test_pass_count!} of ${row.total_tests_count!}`,
      'Total number of subtests',
    );
  }

  // Make a DataTable from the data in featureUsage
  createFeatureUsageDataFromMap(): WebStatusDataObj {
    return this.createDataFromMap(
      this.featureUsage,
      this.featureUsageBrowsers,
      (data, browser) => data.get(browser),
      row => [row.timestamp, row.usage ? row.usage * 100 : 0],
      (row, browser) =>
        `${BROWSER_ID_TO_LABEL[browser]}: ${row.usage ? row.usage * 100 : 0}%`,
    );
  }

  generateFeatureChartOptions(
    browsers: BrowsersParameter[],
    vAxisTitle: string,
  ): google.visualization.ComboChartOptions {
    // Compute seriesColors from selected browsers and BROWSER_ID_TO_COLOR
    const seriesColors = [...browsers, 'total'].map(browser => {
      const browserKey = browser as keyof typeof BROWSER_ID_TO_COLOR;
      return BROWSER_ID_TO_COLOR[browserKey];
    });

    // Add one day to this.endDate.
    const endDate = new Date(this.endDate.getTime() + 1000 * 60 * 60 * 24);
    const options = {
      height: 300, // This is necessary to avoid shrinking to 0 or 18px.
      interpolateNulls: true,
      hAxis: {
        title: '',
        titleTextStyle: {color: '#333'},
        viewWindow: {min: this.startDate, max: endDate},
      },
      vAxis: {
        minValue: 0,
        title: vAxisTitle,
        format: '#,###',
      },
      legend: {position: 'top'},
      colors: seriesColors,
      chartArea: {left: 100, right: 16, top: 40, bottom: 40},

      // Enable explorer mode
      explorer: {
        actions: ['dragToZoom', 'rightClickToReset'],
        axis: 'horizontal',
        keepInBounds: true,
        maxZoomIn: 4,
        maxZoomOut: 4,
        zoomDelta: 0.01,
      },
    } as google.visualization.LineChartOptions;

    this.featureSupportChartOptions = options;
    return options;
  }

  generateFeatureSupportChartOptions(): google.visualization.ComboChartOptions {
    return this.generateFeatureChartOptions(
      this.featureSupportBrowsers,
      'Number of subtests passed',
    );
  }

  generateFeatureUsageChartOptions(): google.visualization.ComboChartOptions {
    return this.generateFeatureChartOptions(
      this.featureUsageBrowsers,
      'Usage (%)',
    );
  }

  async _fetchAndAggregateData<T>(
    apiClient: APIClient,
    fetchFunction: (browser: BrowsersParameter) => AsyncIterable<T[]>,
    data: Map<string, T[]>,
    createChartData: () => void,
    browserDataReference: (browser: BrowsersParameter) => string,
    browsers: BrowsersParameter[],
  ) {
    if (typeof apiClient !== 'object') return;

    createChartData();
    const promises = browsers.map(async browser => {
      for await (const page of fetchFunction(browser)) {
        // Append the new data to existing data
        const existingData = data.get(browserDataReference(browser)) || [];
        data.set(browserDataReference(browser), [...existingData, ...page]);

        createChartData();
      }
    });

    await Promise.all(promises); // Wait for all browsers to finish
  }

  async _fetchFeatureSupportData(
    apiClient: APIClient,
    featureId: string,
    startDate: Date,
    endDate: Date,
  ) {
    await this._fetchAndAggregateData<WPTRunMetric>(
      apiClient,
      (browser: BrowsersParameter) =>
        apiClient.getFeatureStatsByBrowserAndChannel(
          featureId,
          browser,
          STABLE_CHANNEL,
          startDate,
          endDate,
        ),
      this.featureSupport,
      () =>
        (this.featureSupportChartDataObj =
          this.createFeatureSupportDataFromMap()),
      browser => featureSupportKey(browser, STABLE_CHANNEL),
      this.featureSupportBrowsers,
    );
  }

  async _fetchFeatureUsageData(
    apiClient: APIClient,
    featureId: string,
    startDate: Date,
    endDate: Date,
  ) {
    await this._fetchAndAggregateData<ChromiumUsageStat>(
      apiClient,
      (_: BrowsersParameter) =>
        apiClient.getChromiumDailyUsageStats(featureId, startDate, endDate),
      this.featureUsage,
      () =>
        (this.featureUsageChartDataObj = this.createFeatureUsageDataFromMap()),
      browser => browser,
      this.featureUsageBrowsers,
    );
  }

  async firstUpdated(): Promise<void> {
    // TODO(jrobbins): Use routerContext instead of this.location so that
    // nested components could also access the router.
    this.featureId = this.location.params.featureId;
  }

  render(): TemplateResult {
    return html`
      <webstatus-loading-overlay .status="${this._loadingMetricsTask?.status}">
      </webstatus-loading-overlay>
      ${this._loadingTask?.render({
        complete: () => this.renderWhenComplete(),
        error: error => {
          if (error instanceof NotFoundError) {
            // TODO: cannot use navigateToUrl because it creates a
            // circular dependency.
            // For now use the window href and revisit when navigateToUrl
            // is move to another location.
            window.location.href = '/errors-404/feature-not-found';
          }
          return this.renderWhenError();
        },
        initial: () => this.renderWhenInitial(),
        pending: () => this.renderWhenPending(),
      })}
    `;
  }

  renderFeatureSupportChartWhenComplete(): TemplateResult {
    return html`
      <webstatus-gchart
        id="feature-support-chart"
        .containerId="${'feature-support-chart-container'}"
        .chartType="${'LineChart'}"
        .dataObj="${this.featureSupportChartDataObj}"
        .options="${this.generateFeatureSupportChartOptions()}"
      >
        Loading chart...
      </webstatus-gchart>
    `;
  }

  renderFeatureSupportChart(): TemplateResult | undefined {
    return this._loadingMetricsTask?.render({
      complete: () => this.renderFeatureSupportChartWhenComplete(),
      error: () => this.renderWhenError(),
      initial: () => this.renderWhenInitial(),
      pending: () => this.renderWhenPending(),
    });
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

  buildWPTLink(feature?: {
    feature_id: string;
    wpt?: {stable?: object; experimental?: object};
  }): string | null {
    if (feature?.wpt?.stable === undefined) return null;
    const wptLinkURL = new URL('https://wpt.fyi/results');
    const query = `feature:${feature.feature_id} !is:tentative`;
    wptLinkURL.searchParams.append('label', 'master');
    wptLinkURL.searchParams.append('label', 'stable');
    wptLinkURL.searchParams.append('aligned', '');
    wptLinkURL.searchParams.append('q', query);
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
        <div class="hbox wrap">
          <label>
            Start date
            <sl-input
              id="start-date"
              @sl-change=${this.handleStartDateChange}
              type="date"
              .valueAsDate="${this.startDate}"
            ></sl-input>
          </label>
          <label>
            End date
            <sl-input
              id="end-date"
              @sl-change=${this.handleEndDateChange}
              type="date"
              .valueAsDate="${this.endDate}"
            ></sl-input>
          </label>
        </div>
      </div>
    `;
  }

  renderDeltaChip(
    browser: components['parameters']['browserPathParam'],
  ): TemplateResult {
    const channel = 'stable';
    const runs = this.featureSupport.get(featureSupportKey(browser, channel));
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
      <sl-card id="implementation-progress">
        <div class="hbox">
          <div slot="header">Implementation progress</div>
          <div class="spacer"></div>
          <div class="hbox wrap valign-items-end">
            <sl-dropdown
              style="display:none"
              id="feature-support-browser-selector"
              multiple
              stay-open-on-select
              .value="${this.featureSupportBrowsers.join(' ')}"
            >
              <sl-button slot="trigger">
                <sl-icon slot="suffix" name="chevron-down"></sl-icon>
                Browsers
              </sl-button>
              <sl-menu @sl-select=${this.handleBrowserSelection}>
                <sl-menu-item type="checkbox" value="chrome"
                  >Chrome</sl-menu-item
                >
                <sl-menu-item type="checkbox" value="edge">Edge</sl-menu-item>
                <sl-menu-item type="checkbox" value="firefox"
                  >Firefox</sl-menu-item
                >
                <sl-menu-item type="checkbox" value="safari"
                  >Safari</sl-menu-item
                >
              </sl-menu>
            </sl-dropdown>
          </div>
        </div>

        <webstatus-gchart
          id="feature-support-chart"
          .containerId="${'feature-support-chart-container'}"
          .chartType="${'LineChart'}"
          .dataObj="${this.featureSupportChartDataObj}"
          .options="${this.generateFeatureSupportChartOptions()}"
          .dataLoadingStatus="${this._loadingMetricsTask?.status ??
          TaskStatus.INITIAL}"
        >
          Loading chart...
        </webstatus-gchart>
      </sl-card>
    `;
  }

  renderFeatureUsage(): TemplateResult {
    const urlParams = new URLSearchParams(window.location.search);
    if (urlParams.get('showUsageChart') === null) {
      return html``;
    }
    return html`
      <sl-card id="feature-usage">
        <div class="hbox">
          <div slot="header">Feature Usage</div>
          <div class="spacer"></div>
          <div class="spacer"></div>
          <div class="hbox wrap valign-items-end">
            <sl-dropdown
              style="display:none"
              id="feature-usage-browser-selector"
              multiple
              stay-open-on-select
              .value="${this.featureUsageBrowsers.join(' ')}"
            >
              <sl-button slot="trigger">
                <sl-icon slot="suffix" name="chevron-down"></sl-icon>
                Browsers
              </sl-button>
              <sl-menu @sl-select=${this.handleBrowserSelection}>
                <sl-menu-item type="checkbox" value="chrome"
                  >Chrome</sl-menu-item
                >
              </sl-menu>
            </sl-dropdown>
          </div>
        </div>

        <webstatus-gchart
          id="feature-usage-chart"
          .containerId="${'feature-usage-chart-container'}"
          .chartType="${'LineChart'}"
          .dataObj="${this.featureUsageChartDataObj}"
          .options="${this.generateFeatureUsageChartOptions()}"
          .dataLoadingStatus="${this._loadingUsageTask?.status ??
          TaskStatus.INITIAL}"
        >
          Loading chart...
        </webstatus-gchart>
      </sl-card>
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

  private async _startDataFetchingTask<T extends LoadingTaskType>(
    manualRun: boolean,
    dataFetcher: (
      apiClient: APIClient,
      featureId: string,
      startDate: Date,
      endDate: Date,
    ) => Promise<void>,
    taskType: T,
  ) {
    this[taskType]?.abort(); // Access the task property using bracket notation.

    this[taskType] = new Task(this, {
      // Assign the new task to the correct property
      args: () => [this.apiClient, this.featureId],
      task: async ([apiClient, featureId]) => {
        if (typeof apiClient === 'object' && typeof featureId === 'string') {
          await dataFetcher(apiClient, featureId, this.startDate, this.endDate);
        }
      },
    });

    if (manualRun) {
      this[taskType]!.autoRun = false; // Non-null assertion is safe here
      await this[taskType]!.run();
    }
  }

  private async _startFeatureSupportTask(manualRun: boolean) {
    await this._startDataFetchingTask(
      manualRun,
      this._fetchFeatureSupportData,
      '_loadingMetricsTask',
    );
  }

  private async _startFeatureUsageTask(manualRun: boolean) {
    await this._startDataFetchingTask(
      manualRun,
      this._fetchFeatureUsageData,
      '_loadingUsageTask',
    );
  }
}
