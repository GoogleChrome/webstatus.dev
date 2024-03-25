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

//import {type APIClient} from '../api/client.js';
//import {apiClientContext} from '../contexts/api-client-context.js';

/** Map from browser-channel to global feature support. */
const browserChannelDataMap = new Map<
  string,
  Array<components['schemas']['WPTRunMetric']>
>();

const ALL_BROWSERS = ['chrome', 'firefox', 'safari', 'edge'];

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
    // Vary the rate randomly a small amount
    // rate = Math.min(1, Math.max(0.000001, rate * (0.95 + Math.random())));
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
function makeRandomDataForAllBrowserChannelCombos() {
  let rate = 0.5;
  const start = new Date(2021, 1, 1);
  const end = new Date(2024, 3, 31);
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
    for (const channel of ['stable']) {
      makeRandomDataForBrowserChannelCombo(
        totalTestsPerDay,
        start,
        browser,
        channel
      );
    }
  }
}

makeRandomDataForAllBrowserChannelCombos();

// Mocking data for the featuresLagging chart.
/** Map from browser-channel to features missing in only one browser. */
// const featuresLaggingDataMap = new Map<
//   string,
//   Array<components['schemas']['WPTRunMetric']>
//   >();

// /** Make random data for featuresLaggingDataMap.
//  */
// function makeRandomDataForFeaturesLagging(
//  ) {
//   const data: Array<components['schemas']['WPTRunMetric']> = [];
//   // data is computed from browserChannelDataMap.
//   const browsers = ALL_BROWSERS;

//   const numDays = browserChannelDataMap.values().next().value.length;
//   for (let i = 0; i < numDays; i++) {
//     // For each browser...
//     const dayData = browsers.map(
//       browser => browserChannelDataMap.get(`${browser}-stable`)![i]
//     );
//     // For each day, first compute the number of missing tests for each browser,
//     // which is the total minus the tests that passed.
//     const missingTests = dayData.map(
//       data => data.total_tests_count - data.test_pass_count
//     );

//     // Then guess if the missing tests for each browser are for the same feature
//     // as for other browsers.  For simplicity, guess that the number of
//     // tests that are missing only for each browser is a small fraction of all
//     // the missing tests for that browser.
//   }
//   featuresLaggingDataMap.set('all', data);
// }

// From google.visualization types, copied from gviz.d.ts
// Should be able to do this instead:

@customElement('webstatus-stats-page')
export class StatsPage extends LitElement {
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

        #global-feature-support-chart {
          min-height: 20em;
        }
      `,
    ];
  }

  async firstUpdated(): Promise<void> {
    // We can probably do this earlier, but this is a good place to start.
    google.charts.load('current', {
      packages: ['corechart'],
    });
    google.charts.setOnLoadCallback(() => {
      // Let's render a chart...
      this.createGlobalFeatureSupportChart();
    });
  }

  // Make a DataTable from the data in browserChannelDataMap
  createGlobalFeatureSupportDataTableFromMap(): google.visualization.DataTable {
    // Get the list of browsers from browserChannelDataMap
    const browsers = ALL_BROWSERS;
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

    const options = {
      hAxis: {title: 'Feature', titleTextStyle: {color: '#333'}},
      vAxis: {minValue: 0},
      legend: {position: 'top'},
      chartArea: {left: 60, right: 16},
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
          <sl-button href="#TODO">
            <sl-icon
              slot="prefix"
              name="calendar-blank"
              library="phosphor"
            ></sl-icon>
            Select range
          </sl-button>
          <sl-radio-group>
            <sl-radio-button value="WPT" checked>WPT</sl-radio-button>
            <sl-radio-button value="BCD">BCD</sl-radio-button>
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
          <sl-select>
            <sl-option>All browsers</sl-option>
            <sl-option>Chrome</sl-option>
            <sl-option>Firefox</sl-option>
          </sl-select>
        </div>
        <div id="global-feature-support-chart">Chart goes here...</div>
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
          <sl-select>
            <sl-option>All browsers</sl-option>
            <sl-option>Chrome</sl-option>
            <sl-option>Firefox</sl-option>
          </sl-select>
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
