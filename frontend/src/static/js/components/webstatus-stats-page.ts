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

//import {consume} from '@lit/context';
//import {Task} from '@lit/task';
import {LitElement, type TemplateResult, html, CSSResultGroup, css} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {SHARED_STYLES} from '../css/shared-css.js';
import {type components} from 'webstatus.dev-backend';
import {SlMenu, SlMenuItem} from '@shoelace-style/shoelace/dist/shoelace.js';

//import {type APIClient} from '../api/client.js';
//import {apiClientContext} from '../contexts/api-client-context.js';

const ALL_BROWSERS = ['Chrome', 'Firefox', 'Safari', 'Edge'];
const ALL_FEATURES = ['stable'];

/** Map from browser-channel to global feature support. */
const browserChannelDataMap = new Map<
  string,
  Array<components['schemas']['WPTRunMetric']>
>();

/** Make random data for browserChannelDataMap */
function makeRandomDataForBrowserChannelCombo(
  totalTestsPerDay: Array<number>,
  start: Date,
  browser: string,
  channel: string
) {
  const data: Array<components['schemas']['WPTRunMetric']> = [];
  const numDays = totalTestsPerDay.length;

  // Compute random rate for this browser between 0 and 1.
  const rate = Math.random();

  let testPassCount = totalTestsPerDay[0] * rate;
  for (let i = 0; i < numDays; i++) {
    // newTestsPass is a small random fraction of the totalTestsPerDay not yet passed.
    const unpassedTests = Math.abs(totalTestsPerDay[i] - testPassCount);
    let newTestsPass = Math.floor(
      ((Math.random() * unpassedTests) / 1000) * rate
    );
    if (Math.random() < 0.01) {
      newTestsPass +=
        Math.floor(((Math.random() * unpassedTests) / 10) * rate) +
        Math.floor(
          ((Math.random() * unpassedTests) / 500) *
            rate *
            Math.floor((Math.random() * unpassedTests * rate) / 500)
        );
    }

    // testPassCount is previous testPassCount + newTestsPass
    testPassCount = testPassCount + newTestsPass;
    // Can never be more than 90% of the total.
    testPassCount = Math.min(totalTestsPerDay[i] * 0.9, testPassCount);

    data.push({
      run_timestamp: new Date(
        start.getTime() + i * (1000 * 60 * 60 * 24)
      ).toISOString(),
      test_pass_count: testPassCount,
      total_tests_count: totalTestsPerDay[i],
    });
  }
  browserChannelDataMap.set(`${browser}-${channel}`, data);
}

// Generate data for all browser/channel combos
function makeRandomDataForAllBrowserChannelCombos(start: Date, end: Date) {
  let rate = 0.5;
  const dateRange = end.getTime() - start.getTime();
  const numDays = Math.ceil(dateRange / (1000 * 60 * 60 * 24));

  const totalTestsPerDay: Array<number> = [];

  // Create random totalTestsPerDay
  for (let i = 0; i < numDays; i++) {
    // Vary the rate randomly a small amount
    rate = Math.min(1, Math.max(0.000001, rate * (0.95 + Math.random() / 10)));
    let newTests = 1;
    // Occasionally add a random number of tests.
    if (Math.random() < 0.01) {
      newTests +=
        Math.floor(Math.random() * 20000 * rate) +
        Math.floor(Math.random() * 100 * (1 - rate)) *
          Math.floor(Math.random() * 100 * (1 - rate));
    }

    totalTestsPerDay[i] = totalTestsPerDay[i - 1] || 5000;
    totalTestsPerDay[i] += newTests;
  }

  for (const browser of ALL_BROWSERS) {
    for (const channel of ALL_FEATURES) {
      makeRandomDataForBrowserChannelCombo(
        totalTestsPerDay,
        start,
        browser,
        channel
      );
    }
  }
}

@customElement('webstatus-stats-page')
export class StatsPage extends LitElement {
  @state()
  globalFeatureSupportBrowsers: string[] = ALL_BROWSERS;

  @state()
  startDate: Date = new Date(2021, 1, 1);

  @state()
  endDate: Date = new Date(2024, 4, 1);

  @state()
  globalFeatureSupport: Array<components['schemas']['WPTRunMetric']> = [];

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

  setupGlobalFeatureSupportBrowsersHandler() {
    // Get the global feature support data browser selector.
    const browserSelectorMenu = this.shadowRoot!.querySelector(
      '#global-feature-support-browser-selector sl-menu'
    ) as SlMenu;
    // Add a listener to the browserSelectorMenu to update the list of
    // browsers in globalFeatureSupportBrowsers.
    browserSelectorMenu.addEventListener('sl-select', event => {
      const menu = event.target as SlMenu;
      const menuItemsArray: Array<SlMenuItem> = Array.from(
        menu.children
      ).filter(child => child instanceof SlMenuItem) as Array<SlMenuItem>;

      // Build the list of values of checked menu-items.
      this.globalFeatureSupportBrowsers = menuItemsArray
        .filter(menuItem => menuItem.checked)
        .map(menuItem => menuItem.value);
      // console.info(`globalFeatureSupportBrowsers: ${this.globalFeatureSupportBrowsers}`);
      // Regenerate data and redraw.  We should instead just filter it.
      this.setupGlobalFeatureSupportChart();
    });
  }

  setupDateRangeHandler() {
    const startDateInput = this.shadowRoot!.querySelector(
      '#start-date'
    ) as HTMLInputElement;
    startDateInput.addEventListener('sl-blur', event => {
      const currentStartDate = this.startDate;
      this.startDate = new Date((event.target as HTMLInputElement).value);
      if (this.startDate.getTime() === currentStartDate.getTime()) return;
      // Regenerate data and redraw.  We should instead just filter it.
      this.setupGlobalFeatureSupportChart();
    });
    const endDateInput = this.shadowRoot!.querySelector(
      '#end-date'
    ) as HTMLInputElement;
    endDateInput.addEventListener('sl-blur', event => {
      const currentEndDate = this.endDate;
      this.endDate = new Date((event.target as HTMLInputElement).value);
      if (this.endDate.getTime() === currentEndDate.getTime()) return;
      // Regenerate data and redraw.  We should instead just filter it.
      this.setupGlobalFeatureSupportChart();
    });
  }

  setupGlobalFeatureSupportChart() {
    makeRandomDataForAllBrowserChannelCombos(this.startDate, this.endDate);

    google.charts.load('current', {
      packages: ['corechart'],
    });
    google.charts.setOnLoadCallback(() => {
      // Let's render a chart...
      this.createGlobalFeatureSupportChart();
    });

    // Add window resize event handler to redraw the chart.
    window.addEventListener('resize', () => {
      this.createGlobalFeatureSupportChart();
    });
  }

  async firstUpdated(): Promise<void> {
    this.setupGlobalFeatureSupportBrowsersHandler();
    this.setupDateRangeHandler();
    this.setupGlobalFeatureSupportChart();
  }

  // Make a DataTable from the data in browserChannelDataMap
  createGlobalFeatureSupportDataTableFromMap(): google.visualization.DataTable {
    // Get the list of browsers from browserChannelDataMap
    const browsers = this.globalFeatureSupportBrowsers;
    const channel = 'stable';

    const dataTable = new google.visualization.DataTable();
    dataTable.addColumn('date', 'Date');
    for (const browser of browsers) {
      dataTable.addColumn('number', browser);
    }
    dataTable.addColumn('number', 'Total');

    // Map from date to array of counts for each browser
    const dateToBrowserDataMap = new Map<number, Array<number>>();
    // Map from date to array of total_tests_count, the same for all browsers.
    const dateToTotalTestsCountMap = new Map<number, number>();

    // Merge data across all browsers into one array of rows.
    for (const browser of browsers) {
      const data = browserChannelDataMap.get(`${browser}-${channel}`);
      if (!data) continue;
      for (const row of data) {
        if (!row) continue;
        const dateSeconds = new Date(row.run_timestamp).getTime();
        const testPassCount = row.test_pass_count!;
        if (!dateToBrowserDataMap.has(dateSeconds)) {
          dateToBrowserDataMap.set(dateSeconds, [testPassCount]);
          dateToTotalTestsCountMap.set(dateSeconds, row.total_tests_count!);
        } else {
          dateToBrowserDataMap.get(dateSeconds)!.push(testPassCount);
        }
      }
    }

    // Sort the dateToBrowserDataMap by dateSeconds
    const data = Array.from(dateToBrowserDataMap.entries()).sort(
      ([d1], [d2]) => d1 - d2
    );

    // For each date, add a row to the dataTable
    for (const row of data) {
      const dateSeconds = row[0];
      const date = new Date(dateSeconds);
      const browserCounts = row[1];
      const total = dateToTotalTestsCountMap.get(dateSeconds)!;
      dataTable.addRow([date, ...browserCounts, total]);
    }
    return dataTable;
  }

  createGlobalFeatureSupportChart(): void {
    const data = this.createGlobalFeatureSupportDataTableFromMap();

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

    const chart = new google.visualization.LineChart(
      this.shadowRoot!.getElementById('global-feature-support-chart')!
    );
    chart.draw(data, options);
  }

  render(): TemplateResult | undefined {
    return this.renderWhenComplete();
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
              type="date"
              .valueAsDate="${this.startDate}"
            ></sl-input>
          </label>
          <label
            >End date
            <sl-input
              id="end-date"
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
            <sl-menu>
              <sl-menu-item type="checkbox" value="Chrome">Chrome</sl-menu-item>
              <sl-menu-item type="checkbox" value="Edge">Edge</sl-menu-item>
              <sl-menu-item type="checkbox" value="Firefox"
                >Firefox</sl-menu-item
              >
              <sl-menu-item type="checkbox" value="Safari">Safari</sl-menu-item>
            </sl-menu>
          </sl-dropdown>
        </div>
        <div>
          <div
            id="global-feature-support-chart"
            style="padding: 0; margin: 0; border: 0"
          >
            Loading chart...
          </div>
        </div>
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

  renderWhenComplete(): TemplateResult {
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
}
