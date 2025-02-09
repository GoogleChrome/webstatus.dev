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
import {SlMenu, SlMenuItem} from '@shoelace-style/shoelace/dist/shoelace.js';

import {
  ALL_BROWSERS,
  BROWSER_ID_TO_COLOR,
  type APIClient,
  type BrowsersParameter,
  type BrowserReleaseFeatureMetric,
  BaselineStatusMetric,
} from '../api/client.js';
import {apiClientContext} from '../contexts/api-client-context.js';
import {getFeaturesLaggingFlag} from '../utils/urls.js';

import './webstatus-gchart';
import {WebStatusDataObj} from './webstatus-gchart.js';
import {BaseChartsPage} from './webstatus-base-charts-page.js';

interface MetricData<T> {
  label: string;
  data: Array<T>;
  getTimestamp: (item: T) => Date; // Function to extract timestamp
  getData: (item: T) => number | undefined; // Function to extract data
}

@customElement('webstatus-stats-page')
export class StatsPage extends BaseChartsPage {
  @state()
  _loadingGFSTask: Task;

  @state()
  _loadingMissingOneTask: Task;

  @consume({context: apiClientContext})
  apiClient!: APIClient;

  @state()
  supportedBrowsers: BrowsersParameter[] = ALL_BROWSERS;

  @state()
  globalFeatureSupportChartOptions = {};

  @state()
  globalFeatureSupportChartDataObj: WebStatusDataObj | undefined;

  @state()
  missingOneImplementationChartDataObj: WebStatusDataObj | undefined;

  static get styles(): CSSResultGroup {
    return [
      super.styles!,
      css`
        .under-construction {
          min-height: 12em;
        }

        /*  Make the dropdown menu button icon rotate when the menu is open,
            so it looks like sl-select. */
        sl-dropdown > sl-button > sl-icon {
          rotate: 0deg;
          transition: var(--sl-transition-medium) rotate ease;
        }
        sl-dropdown[open] > sl-button > sl-icon {
          rotate: -180deg;
          transition: var(--sl-transition-medium) rotate ease;
        }
      `,
    ];
  }

  handleBrowserSelection(event: Event) {
    const menu = event.target as SlMenu;
    const menuItemsArray: Array<SlMenuItem> = Array.from(menu.children).filter(
      child => child instanceof SlMenuItem,
    ) as Array<SlMenuItem>;

    // Build the list of values of checked menu-items.
    this.supportedBrowsers = menuItemsArray
      .filter(menuItem => menuItem.checked)
      .map(menuItem => menuItem.value) as BrowsersParameter[];
    // Regenerate data and redraw.  We should instead just filter it.
  }

  globalFeatureSupportResizeObserver: ResizeObserver | null = null;

  missingOneImplementationResizeObserver: ResizeObserver | null = null;

  setupResizeObserver() {
    // Set up ResizeObserver one time to redraw chart when container resizes.
    if (!this.globalFeatureSupportResizeObserver) {
      const gfsChartElement = this.shadowRoot!.getElementById(
        'global-feature-support-chart',
      );
      if (!gfsChartElement) return;
      this.globalFeatureSupportResizeObserver = new ResizeObserver(() => {
        // TODO: trigger update based on resize.
      });
      this.globalFeatureSupportResizeObserver.observe(gfsChartElement);
    }

    if (!this.missingOneImplementationResizeObserver) {
      const gfsChartElement = this.shadowRoot!.getElementById(
        'missing-one-implementation-chart',
      );
      if (!gfsChartElement) return;
      this.missingOneImplementationResizeObserver = new ResizeObserver(() => {
        // TODO: trigger update based on resize.
      });
      this.missingOneImplementationResizeObserver.observe(gfsChartElement);
    }
  }

  async _fetchGlobalFeatureSupportData(
    apiClient: APIClient,
    startDate: Date,
    endDate: Date,
  ) {
    if (typeof apiClient !== 'object') return;

    const browserMetricData: Array<MetricData<BrowserReleaseFeatureMetric>> =
      ALL_BROWSERS.map(browser => ({
        label: browser,
        data: [],
        getTimestamp: (item: BrowserReleaseFeatureMetric) =>
          new Date(item.timestamp),
        getData: (item: BrowserReleaseFeatureMetric) => item.count,
      }));

    const maxMetricData: MetricData<BaselineStatusMetric> = {
      label: 'Total number of Baseline features',
      data: [],
      getTimestamp: (item: BaselineStatusMetric) => new Date(item.timestamp),
      getData: (item: BaselineStatusMetric) => item.count,
    };

    const allMetricData = [...browserMetricData, maxMetricData];

    const browserPromises = ALL_BROWSERS.map(async browser => {
      const browserData = browserMetricData.find(
        data => data.label === browser,
      );
      if (!browserData) return;

      for await (const page of apiClient.getFeatureCountsForBrowser(
        browser,
        startDate,
        endDate,
      )) {
        browserData.data.push(...page);
      }
    });

    const maxPromise = (async () => {
      for await (const page of apiClient.listAggregatedBaselineStatusCounts(
        startDate,
        endDate,
      )) {
        maxMetricData.data.push(...page);
      }
    })();

    await Promise.all([...browserPromises, maxPromise]);

    this.globalFeatureSupportChartDataObj =
      this.createDisplayDataFromMap(allMetricData);
  }

  async _fetchMissingOneImplemenationCounts(
    apiClient: APIClient,
    startDate: Date,
    endDate: Date,
  ) {
    if (typeof apiClient !== 'object') return;
    const browserMetricData: Array<MetricData<BrowserReleaseFeatureMetric>> =
      ALL_BROWSERS.map(browser => ({
        label: browser,
        data: [],
        getTimestamp: (item: BrowserReleaseFeatureMetric) =>
          new Date(item.timestamp),
        getData: (item: BrowserReleaseFeatureMetric) => item.count,
      }));
    const promises = ALL_BROWSERS.map(async browser => {
      const browserData = browserMetricData.find(
        data => data.label === browser,
      );
      if (!browserData) return;

      const otherBrowsers = ALL_BROWSERS.filter(value => browser !== value);
      for await (const page of apiClient.getMissingOneImplementationCountsForBrowser(
        browser,
        otherBrowsers,
        startDate,
        endDate,
      )) {
        browserData.data.push(...page);
      }
    });
    await Promise.all(promises); // Wait for all browsers to finish

    this.missingOneImplementationChartDataObj =
      this.createDisplayDataFromMap<BrowserReleaseFeatureMetric>(
        browserMetricData,
      );
  }

  // Make startDate and endDate reactive so that @lit/task can detect the changes.
  // TODO: Remove the @state decorator from start and end dates when we move the loading task into a non-page component.
  @state()
  override startDate!: Date;
  @state()
  override endDate!: Date;

  constructor() {
    super();

    this._loadingGFSTask = new Task(this, {
      args: () =>
        [this.apiClient, this.startDate, this.endDate] as [
          APIClient,
          Date,
          Date,
        ],
      task: async ([apiClient, startDate, endDate]: [
        APIClient,
        Date,
        Date,
      ]) => {
        await this._fetchGlobalFeatureSupportData(
          apiClient,
          startDate,
          endDate,
        );
        return;
      },
    });

    this._loadingMissingOneTask = new Task(this, {
      args: () =>
        [this.apiClient, this.startDate, this.endDate] as [
          APIClient,
          Date,
          Date,
        ],
      task: async ([apiClient, startDate, endDate]: [
        APIClient,
        Date,
        Date,
      ]) => {
        await this._fetchMissingOneImplemenationCounts(
          apiClient,
          startDate,
          endDate,
        );
        return;
      },
    });
  }

  // Make a DataTable from the target data map.
  // TODO(kyleju): refactor this method acorss feature detail page
  // and stats page, https://github.com/GoogleChrome/webstatus.dev/issues/964.
  createDisplayDataFromMap<T>(
    metricDataArray: Array<MetricData<T>>,
  ): WebStatusDataObj {
    const dataObj: WebStatusDataObj = {cols: [], rows: []};
    dataObj.cols.push({type: 'date', label: 'Date', role: 'domain'});

    for (const metricData of metricDataArray) {
      dataObj.cols.push({
        type: 'number',
        label: metricData.label,
        role: 'data',
      });
    }

    const dateToDataMap = new Map<number, {[key: string]: number | null}>();

    for (const metricData of metricDataArray) {
      if (!Array.isArray(metricData.data)) continue;
      for (const item of metricData.data) {
        const timestamp = metricData.getTimestamp(item);
        const dateSeconds = timestamp.getTime();
        const dataValue = metricData.getData(item);

        if (!dateToDataMap.has(dateSeconds)) {
          dateToDataMap.set(dateSeconds, {});
        }
        const dateData = dateToDataMap.get(dateSeconds)!;
        dateData[metricData.label] = dataValue || null;
      }
    }

    const data = Array.from(dateToDataMap.entries()).sort(
      ([d1], [d2]) => d1 - d2,
    );

    for (const [dateSeconds, dateData] of data) {
      const date = new Date(dateSeconds);
      const row: [Date, ...Array<number | string | null>] = [date];

      for (const metricData of metricDataArray) {
        row.push(
          dateData[metricData.label] ? dateData[metricData.label] : null,
        );
      }
      dataObj.rows.push(row);
    }

    return dataObj;
  }

  generatedisplayDataChartOptions(
    vAxisTitle: string,
  ): google.visualization.LineChartOptions {
    // Compute seriesColors from selected browsers and BROWSER_ID_TO_COLOR
    const selectedBrowsers = this.supportedBrowsers;
    const seriesColors = [...selectedBrowsers, 'total'].map(browser => {
      const browserKey = browser as keyof typeof BROWSER_ID_TO_COLOR;
      return BROWSER_ID_TO_COLOR[browserKey];
    });

    // Add one day to this.endDate.
    const endDate = new Date(this.endDate.getTime() + 1000 * 60 * 60 * 24);
    const options = {
      height: 300, // This is necessary to avoid shrinking to 0 or 18px.
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

      interpolateNulls: true,

      // Multiple selection of points will be summarized in one tooltip.
      tooltip: {trigger: 'selection'},
      selectionMode: 'multiple',
      aggregationTarget: 'category',

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

    this.globalFeatureSupportChartOptions = options;
    return options;
  }

  renderTitleAndControls(): TemplateResult {
    return html`
      <div id="titleAndControls" class="hbox">
        <h1>Statistics</h1>
        <div class="spacer"></div>
        ${this.renderDateRangePicker()}
      </div>
    `;
  }

  renderGlobalFeatureSupportChartWhenComplete(): TemplateResult {
    return html`
      <webstatus-gchart
        id="global-feature-support-chart"
        .hasMax=${false}
        .containerId="${'global-feature-support-chart-container'}"
        .chartType="${'LineChart'}"
        .dataObj="${this.globalFeatureSupportChartDataObj}"
        .options="${this.generatedisplayDataChartOptions(
          'Number of features supported',
        )}"
      >
        Loading chart...
      </webstatus-gchart>
    `;
  }

  renderGlobalFeatureSupportChart(): TemplateResult | undefined {
    return this._loadingGFSTask.render({
      complete: () => this.renderGlobalFeatureSupportChartWhenComplete(),
      error: () => this.renderChartWhenError(),
      initial: () => this.renderChartWhenInitial(),
      pending: () => this.renderChartWhenPending(),
    });
  }

  renderMissingOneImplementationChartWhenComplete(): TemplateResult {
    return html`
      <webstatus-gchart
        id="missing-one-implementation-chart"
        .hasMax=${false}
        .containerId="${'missing-one-implementation-chart-container'}"
        .chartType="${'LineChart'}"
        .dataObj="${this.missingOneImplementationChartDataObj}"
        .options="${this.generatedisplayDataChartOptions(
          'Number of features missing',
        )}"
      >
        Loading chart...
      </webstatus-gchart>
    `;
  }

  renderMissingOneImplementationChart(): TemplateResult | undefined {
    return this._loadingMissingOneTask.render({
      complete: () => this.renderMissingOneImplementationChartWhenComplete(),
      error: () =>
        html`<div id="missing-one-implementation-error">
          ${this.renderChartWhenError()}
        </div>`,
      initial: () =>
        html`<div id="missing-one-implementation-initial">
          ${this.renderChartWhenInitial()}
        </div>`,
      pending: () =>
        html`<div id="missing-one-implementation-pending">
          ${this.renderChartWhenPending()}
        </div>`,
    });
  }

  renderGlobalFeatureSupport(): TemplateResult {
    return html`
      <sl-card id="global-feature-support">
        <div slot="header" class="hbox">
          Global feature support
          <div class="spacer"></div>
          <sl-select>
            <sl-option>All features</sl-option>
            <sl-option>how to select?</sl-option>
          </sl-select>
          <sl-dropdown
            id="global-feature-support-browser-selector"
            multiple
            stay-open-on-select
            .value="${this.supportedBrowsers.join(' ')}"
          >
            <sl-button slot="trigger">
              <sl-icon slot="suffix" name="chevron-down"></sl-icon>
              Browsers
            </sl-button>
            <sl-menu @sl-select=${this.handleBrowserSelection}>
              <sl-menu-item type="checkbox" value="chrome">Chrome</sl-menu-item>
              <sl-menu-item type="checkbox" value="edge">Edge</sl-menu-item>
              <sl-menu-item type="checkbox" value="firefox"
                >Firefox</sl-menu-item
              >
              <sl-menu-item type="checkbox" value="safari">Safari</sl-menu-item>
            </sl-menu>
          </sl-dropdown>
        </div>
        <div>${this.renderGlobalFeatureSupportChart()}</div>
      </sl-card>
    `;
  }

  renderFeaturesLagging(): TemplateResult {
    return html`
      <sl-card id="features-lagging">
        <div slot="header" class="hbox">
          Features missing in only 1 browser
          <div class="spacer"></div>
          <sl-select>
            <sl-option>All features</sl-option>
            <sl-option>how to select?</sl-option>
          </sl-select>
          <sl-dropdown multiple value="Chrome Edge Firefox Safari">
            <sl-option>Chrome</sl-option>
            <sl-option>Edge</sl-option>
            <sl-option>Firefox</sl-option>
            <sl-option>Safari</sl-option>
          </sl-dropdown>
        </div>
        <div>${this.renderMissingOneImplementationChart()}</div>
      </sl-card>
    `;
  }

  renderBaselineFeatures(): TemplateResult {
    return html`
      <sl-card
        class="halign-stretch"
        id="baseline-features"
        style="display:none"
      >
        <div slot="header">Baseline features</div>
        <p class="under-construction">Small chart goes here...</p>
      </sl-card>
    `;
  }

  renderTimeToAvailability(): TemplateResult {
    return html`
      <sl-card
        class="halign-stretch"
        id="time-to-availibility"
        style="display:none"
      >
        <div slot="header">Time to availablity</div>
        <p class="under-construction">Small chart goes here...</p>
      </sl-card>
    `;
  }

  render(): TemplateResult {
    return html`
      <div class="vbox">
        ${this.renderTitleAndControls()} ${this.renderGlobalFeatureSupport()}
        ${getFeaturesLaggingFlag(this.location)
          ? this.renderFeaturesLagging()
          : nothing}
        <div class="hbox">
          ${this.renderBaselineFeatures()} ${this.renderTimeToAvailability()}
        </div>
      </div>
    `;
  }

  renderChartWhenError(): TemplateResult {
    return html`Error when loading stats.`;
  }

  renderChartWhenInitial(): TemplateResult {
    return html`Preparing request for stats.`;
  }

  renderChartWhenPending(): TemplateResult {
    return html`Loading stats.`;
  }
}
