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

// See https://github.com/DefinitelyTyped/DefinitelyTyped/blob/master/types/google.visualization/index.d.ts
/// <reference types="@types/google.visualization" />

import {consume} from '@lit/context';
import {Task} from '@lit/task';
import {LitElement, type TemplateResult, html, CSSResultGroup, css} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {SHARED_STYLES} from '../css/shared-css.js';
import {SlMenu, SlMenuItem} from '@shoelace-style/shoelace/dist/shoelace.js';

import {
  type APIClient,
  type BrowsersParameter,
  type ChannelsParameter,
  type WPTRunMetric,
} from '../api/client.js';
import {apiClientContext} from '../contexts/api-client-context.js';
import {gchartsContext} from '../contexts/gcharts-context.js';

// No way to get the values from the parameter types, so we have to
// redundantly specify them.
const ALL_BROWSERS: BrowsersParameter[] = [
  'chrome',
  'firefox',
  'safari',
  'edge',
];
const ALL_FEATURES: ChannelsParameter[] = ['stable'];

/** Generate a key for globalFeatureSupport. */
function globalFeatureSupportKey(
  browser: BrowsersParameter,
  channel?: ChannelsParameter
): string {
  return `${browser}-${channel}`;
}

@customElement('webstatus-stats-page')
export class StatsPage extends LitElement {
  @state()
  _loadingGFSTask: Task;

  @consume({context: gchartsContext, subscribe: true})
  @state()
  gchartsLibraryLoaded = false;

  @consume({context: apiClientContext})
  apiClient!: APIClient;

  @state()
  globalFeatureSupportBrowsers: BrowsersParameter[] = ALL_BROWSERS;

  @state()
  startDate: Date = new Date(2020, 0, 1); // Jan 1, 2020.

  @state()
  endDate: Date = new Date(); // Today

  // Map from browser-channel to global feature support.
  // The key is generated by globalFeatureSupportKey().
  @state()
  globalFeatureSupport = new Map<string, Array<WPTRunMetric>>();

  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        .hbox,
        .vbox {
          gap: var(--content-padding-large);
        }

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

        #global-feature-support-chart {
          min-height: 20em;
        }
      `,
    ];
  }

  handleBrowserSelection(event: Event) {
    const menu = event.target as SlMenu;
    const menuItemsArray: Array<SlMenuItem> = Array.from(menu.children).filter(
      child => child instanceof SlMenuItem
    ) as Array<SlMenuItem>;

    // Build the list of values of checked menu-items.
    this.globalFeatureSupportBrowsers = menuItemsArray
      .filter(menuItem => menuItem.checked)
      .map(menuItem => menuItem.value) as BrowsersParameter[];
    // Regenerate data and redraw.  We should instead just filter it.
    this.drawGlobalFeatureSupportChart();
  }

  handleStartDateChange(event: Event) {
    const currentStartDate = this.startDate;
    this.startDate = new Date((event.target as HTMLInputElement).value);
    if (this.startDate.getTime() === currentStartDate.getTime()) return;
    // Regenerate data and redraw.  We should instead just filter it.
    this.drawGlobalFeatureSupportChart();
  }

  handleEndDateChange(event: Event) {
    const currentEndDate = this.endDate;
    this.endDate = new Date((event.target as HTMLInputElement).value);
    if (this.endDate.getTime() === currentEndDate.getTime()) return;
    // Regenerate data and redraw.  We should instead just filter it.
    this.drawGlobalFeatureSupportChart();
  }

  async _fetchGlobalFeatureSupportData(
    apiClient: APIClient,
    startDate: Date,
    endDate: Date
  ) {
    if (typeof apiClient !== 'object') return;
    for (const browser of ALL_BROWSERS) {
      for (const channel of ALL_FEATURES) {
        const wptRuns = await apiClient.getStatsByBrowserAndChannel(
          browser,
          channel,
          startDate,
          endDate
        );
        this.globalFeatureSupport.set(
          globalFeatureSupportKey(browser, channel),
          wptRuns
        );
      }
    }
  }

  setupGlobalFeatureSupportChart() {
    // Add window resize event handler to redraw the chart.
    window.addEventListener('resize', () => {
      this.drawGlobalFeatureSupportChart();
    });
  }

  constructor() {
    super();

    this._loadingGFSTask = new Task(this, {
      args: () =>
        [
          this.apiClient,
          this.startDate,
          this.endDate,
          this.gchartsLibraryLoaded,
        ] as [APIClient, Date, Date, boolean],
      task: async ([apiClient, startDate, endDate, gcLoaded]: [
        APIClient,
        Date,
        Date,
        boolean,
      ]) => {
        if (gcLoaded) {
          await this._fetchGlobalFeatureSupportData(
            apiClient,
            startDate,
            endDate
          );
        }
        return this.globalFeatureSupport;
      },
    });
  }

  async firstUpdated(): Promise<void> {}

  updated() {
    if (this.gchartsLibraryLoaded) {
      this.drawGlobalFeatureSupportChart();
    }
  }

  // Make a DataTable from the data in globalFeatureSupport
  createGlobalFeatureSupportDataTableFromMap(): google.visualization.DataTable {
    // Get the list of browsers from globalFeatureSupport
    const browsers = this.globalFeatureSupportBrowsers;
    const channel = 'stable';

    const dataTable = new google.visualization.DataTable();
    dataTable.addColumn('date', 'Date');
    for (const browser of browsers) {
      dataTable.addColumn('number', browser);
    }
    dataTable.addColumn('number', 'Total');

    // Map from date to an object with counts for each browser
    const dateToBrowserDataMap = new Map<number, {[key: string]: number}>();
    // Map from date to array of total_tests_count, the same for all browsers.
    const dateToTotalTestsCountMap = new Map<number, number>();

    // Merge data across all browsers into one array of rows.
    for (const browser of browsers) {
      const data = this.globalFeatureSupport.get(
        globalFeatureSupportKey(browser, channel)
      );
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
      dataTable.addRow([date, ...browserCountArray, total]);
    }
    return dataTable;
  }

  drawGlobalFeatureSupportChart(): void {
    const gfsChartElement = this.shadowRoot!.getElementById(
      'global-feature-support-chart'
    );
    if (!gfsChartElement) return;
    const datatable = this.createGlobalFeatureSupportDataTableFromMap();

    // Add 2 weeks to this.endDate.
    const endDate = new Date(this.endDate.getTime() + 1000 * 60 * 60 * 24 * 14);
    const options = {
      hAxis: {
        title: '',
        titleTextStyle: {color: '#333'},
        viewWindow: {min: this.startDate, max: endDate},
      },
      vAxis: {minValue: 0},
      legend: {position: 'top'},
      chartArea: {left: 60, right: 16, top: 40, bottom: 40},
    } as google.visualization.LineChartOptions;

    const chart = new google.visualization.LineChart(gfsChartElement);
    chart.draw(datatable, options);
  }

  renderTitleAndControls(): TemplateResult {
    return html`
      <div id="titleAndControls" class="hbox">
        <h1>Statistics</h1>
        <div class="spacer"></div>
        <div class="hbox wrap valign-items-center">
          <sl-checkbox>Show browser versions</sl-checkbox>
          <label
            >Start date
            <sl-input
              id="start-date"
              @sl-blur=${this.handleStartDateChange}
              type="date"
              .valueAsDate="${this.startDate}"
            ></sl-input>
          </label>
          <label
            >End date
            <sl-input
              id="end-date"
              @sl-blur=${this.handleEndDateChange}
              type="date"
              .valueAsDate="${this.endDate}"
            ></sl-input>
          </label>
          <sl-radio-group value="WPT">
            <sl-radio-button value="WPT">WPT</sl-radio-button>
            <sl-radio-button value="BCD" disabled>BCD</sl-radio-button>
          </sl-radio-group>
        </div>
      </div>
    `;
  }

  renderGlobalFeatureSupportChartWhenComplete(): TemplateResult {
    return html`
      <div
        id="global-feature-support-chart"
        style="padding: 0; margin: 0; border: 0"
      >
        Loading chart...
      </div>
    `;
  }

  renderGlobalFeatureSupportChart(): TemplateResult | undefined {
    if (!this.gchartsLibraryLoaded) return html`Loading chart library.`;
    return this._loadingGFSTask.render({
      complete: () => this.renderGlobalFeatureSupportChartWhenComplete(),
      error: () => this.renderChartWhenError(),
      initial: () => this.renderChartWhenInitial(),
      pending: () => this.renderChartWhenPending(),
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
            .value="${this.globalFeatureSupportBrowsers.join(' ')}"
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
        <div class="under-construction" id="features-lagging-chart">
          Chart goes here...
        </div>
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
        ${this.renderFeaturesLagging()}
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
