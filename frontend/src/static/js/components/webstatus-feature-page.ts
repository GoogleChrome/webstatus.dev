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
import {WebStatusDataObj} from './webstatus-gchart.js';

/** Generate a key for featureSupport. */
function featureSupportKey(
  browser: BrowsersParameter,
  channel?: ChannelsParameter
): string {
  return `${browser}-${channel}`;
}

@customElement('webstatus-feature-page')
export class FeaturePage extends LitElement {
  _loadingTask: Task;

  @consume({context: apiClientContext})
  @state()
  apiClient!: APIClient;

  @state()
  featureSupportBrowsers: BrowsersParameter[] = ALL_BROWSERS;

  @state()
  startDate: Date = new Date(2020, 0, 1); // Jan 1, 2020.

  @state()
  endDate: Date = new Date(); // Today

  // Map from browser-channel to  feature support.
  // The key is generated by featureSupportKey().
  @state()
  featureSupport = new Map<string, Array<WPTRunMetric>>();

  @state()
  featureSupportChartOptions = {};

  @state()
  featureSupportChartDataObj: WebStatusDataObj | undefined;

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

        #current-bugs li {
          list-style: none;
          margin-bottom: var(--content-padding);
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
            location
          ) as FeatureWPTMetricViewType;
          this.feature = await apiClient.getFeature(featureId, wptMetricView);
          await this._fetchFeatureSupportData(
            apiClient,
            this.startDate,
            this.endDate
          );
        }
        return this.feature;
      },
    });
  }

  handleBrowserSelection(event: Event) {
    const menu = event.target as SlMenu;
    const menuItemsArray: Array<SlMenuItem> = Array.from(menu.children).filter(
      child => child instanceof SlMenuItem
    ) as Array<SlMenuItem>;

    // Build the list of values of checked menu-items.
    this.featureSupportBrowsers = menuItemsArray
      .filter(menuItem => menuItem.checked)
      .map(menuItem => menuItem.value) as BrowsersParameter[];
    // Regenerate data and redraw.  We should instead just filter it.
    this._fetchFeatureSupportData(this.apiClient, this.startDate, this.endDate);
    this.generateFeatureSupportChartOptions();
  }

  handleStartDateChange(event: Event) {
    const currentStartDate = this.startDate;
    this.startDate = new Date((event.target as HTMLInputElement).value);
    if (this.startDate.getTime() === currentStartDate.getTime()) return;
  }

  handleEndDateChange(event: Event) {
    const currentEndDate = this.endDate;
    this.endDate = new Date((event.target as HTMLInputElement).value);
    if (this.endDate.getTime() === currentEndDate.getTime()) return;
  }

  // Make a DataTable from the data in featureSupport
  createFeatureSupportDataFromMap(): WebStatusDataObj {
    // Get the list of browsers from featureSupport
    const browsers = this.featureSupportBrowsers;
    const channel = 'stable';

    const dataObj: WebStatusDataObj = {cols: [], rows: []};
    dataObj.cols.push({type: 'date', label: 'Date'});
    for (const browser of browsers) {
      dataObj.cols.push({type: 'number', label: BROWSER_ID_TO_LABEL[browser]});
    }
    dataObj.cols.push({type: 'number', label: 'Total number of subtests'});

    // Map from date to an object with counts for each browser
    const dateToBrowserDataMap = new Map<number, {[key: string]: number}>();
    // Map from date to array of total_tests_count, the same for all browsers.
    const dateToTotalTestsCountMap = new Map<number, number>();

    // Merge data across all browsers into one array of rows.
    for (const browser of browsers) {
      const data = this.featureSupport.get(featureSupportKey(browser, channel));
      if (!data) continue;
      for (const row of data) {
        if (!row) continue;
        const dateSeconds = new Date(row.run_timestamp).getTime();
        const testPassCount = row.test_pass_count!;
        if (!dateToBrowserDataMap.has(dateSeconds)) {
          dateToBrowserDataMap.set(dateSeconds, {});
          dateToTotalTestsCountMap.set(dateSeconds, row.total_tests_count!);
        }
        const browserCounts = dateToBrowserDataMap.get(dateSeconds)!;
        browserCounts[browser] = testPassCount;
      }
    }

    // Create array of dateToBrowserDataMap entries and sort by dateSeconds
    const data = Array.from(dateToBrowserDataMap.entries()).sort(
      ([d1], [d2]) => d1 - d2
    );

    // For each date, add a row to the dataTable
    for (const datum of data) {
      const dateSeconds = datum[0];
      const date = new Date(dateSeconds);
      const browserCounts = datum[1];
      // Make an array of browser counts, in the order of browsers.
      // If the browser is not in the browserCounts, add null.
      const browserCountArray = browsers.map(browser => {
        return browserCounts[browser] || null;
      });
      const total = dateToTotalTestsCountMap.get(dateSeconds)!;
      dataObj.rows.push([date, ...browserCountArray, total]);
    }
    return dataObj;
  }

  generateFeatureSupportChartOptions(): google.visualization.LineChartOptions {
    // Compute seriesColors from selected browsers and BROWSER_ID_TO_COLOR
    const selectedBrowsers = this.featureSupportBrowsers;
    const seriesColors = [...selectedBrowsers, 'total'].map(browser => {
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
        title: 'Number of subtests passed',
        format: '#,###',
      },
      legend: {position: 'top'},
      colors: seriesColors,
      chartArea: {left: 100, right: 16, top: 40, bottom: 40},
      tooltip: {trigger: 'selection'},
      // Uncomment to allow multiple selection of points,
      // and all selected points will be summarized in one tooltip.
      // selectionMode: 'multiple',
    } as google.visualization.LineChartOptions;

    this.featureSupportChartOptions = options;
    return options;
  }

  async _fetchFeatureSupportData(
    apiClient: APIClient,
    startDate: Date,
    endDate: Date
  ) {
    if (typeof apiClient !== 'object') return;
    for (const browser of ALL_BROWSERS) {
      const channel = STABLE_CHANNEL;
      const wptRuns = await apiClient.getFeatureStatsByBrowserAndChannel(
        this.featureId,
        browser,
        channel,
        startDate,
        endDate
      );
      this.featureSupport.set(featureSupportKey(browser, channel), wptRuns);
    }
    this.featureSupportChartDataObj = this.createFeatureSupportDataFromMap();
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
    return this._loadingTask.render({
      complete: () => this.renderFeatureSupportChartWhenComplete(),
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

  renderOffsiteLink(
    label: string,
    link: string | null,
    logo?: string,
    logoAlt?: string
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

  renderNameAndOffsiteLinks(): TemplateResult {
    const wptLink =
      'https://wpt.fyi/results' +
      '?label=master&label=stable&aligned' +
      '&q=feature%3A' +
      this.feature!.feature_id;
    const wptLogo = '/public/img/wpt-logo.svg';

    return html`
      <div id="nameAndOffsiteLinks" class="hbox valign-items-end">
        <h1>${this.feature!.name}</h1>
        <div class="spacer"></div>
        <label
          >Start date
          <sl-input
            id="start-date"
            @sl-change=${this.handleStartDateChange}
            type="date"
            .valueAsDate="${this.startDate}"
          ></sl-input>
        </label>
        <label
          >End date
          <sl-input
            id="end-date"
            @sl-change=${this.handleEndDateChange}
            type="date"
            .valueAsDate="${this.endDate}"
          ></sl-input>
        </label>
        ${this.renderOffsiteLink(
          'WPT.fyi',
          wptLink,
          wptLogo,
          'WPT default view'
        )}
        ${this.renderOffsiteLink('MDN', null)}
        ${this.renderOffsiteLink('CanIUse', null)}
      </div>
    `;
  }

  renderDeltaChip(
      browser: components['parameters']['browserPathParam']
  ): TemplateResult {
      const channel = 'stable';
      const runs = this.featureSupport.get(featureSupportKey(browser, channel));
      if (runs === undefined) {
    return html`
<span class="chip small unchanged"></span>
`;
      }

      const firstRun = runs[0];
      const lastRun = runs[runs.length - 1];
      const firstPercent =
          firstRun.test_pass_count! / firstRun.total_tests_count!;
      const lastPercent =
          lastRun.test_pass_count! / lastRun.total_tests_count!;
      const delta = lastPercent - firstPercent;
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
    return html`
<span class="chip small ${deltaClass} ${browser}">${deltaStr}</span>
`;
  }

  renderOneWPTCard(
    browser: components['parameters']['browserPathParam'],
    icon: string
  ): TemplateResult {
    const scorePart = this.feature
      ? renderBrowserQuality(this.feature, {search: ''}, {browser: browser})
      : nothing;

    return html`
      <sl-card class="halign-stretch wptScore">
        <img height="32" src="/public/img/${icon}" class="icon" />
        <div>${browser[0].toUpperCase() + browser.slice(1)}</div>
        <div class="score">
          ${scorePart}
          ${this.renderDeltaChip(browser)}
        </div>
        <div class="avail">Available since ...</div>
      </sl-card>
    `;
  }

  renderBaselineCard(): TemplateResult {
    if (!this.feature) return html``;

    const status = this.feature.baseline?.status;

    if (status === undefined) return html``;

    const chipConfig = BASELINE_CHIP_CONFIGS[status];

    return html`
      <sl-card class="halign-stretch wptScore">
        <img height="28" src="/public/img/${chipConfig.icon}" class="icon" />
        <div>Baseline</div>
        <div class="score">${chipConfig.word}</div>
        <div class="avail">Baseline since ...</div>
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
            this.renderBug(bugId)
          )}
        </ul>
      </sl-details>
    `;
  }

  renderWhenComplete(): TemplateResult {
    return html`
      <div class="vbox">
        ${this.renderCrumbs()} ${this.renderNameAndOffsiteLinks()}
        ${this.renderWPTScores()} ${this.renderImplentationProgress()}
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
    return html`Loading ${this.featureId}.`;
  }
}
