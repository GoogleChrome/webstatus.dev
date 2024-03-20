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

//import {consume} from '@lit/context';
//import {Task} from '@lit/task';
import {LitElement, type TemplateResult, html, CSSResultGroup, css} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {SHARED_STYLES} from '../css/shared-css.js';
import {type components} from 'webstatus.dev-backend';

//import {type APIClient} from '../api/client.js';
//import {apiClientContext} from '../contexts/api-client-context.js';

// Should be able to do this instead:
// import {google} from '@types/google.visualization';

// From gviz.d.ts
export interface LoadOptions {
  packages?: string | string[];
  language?: string;
  callback?: Function;
  mapsApiKey?: string;
  safeMode?: boolean;
  /** not documented */
  debug?: boolean;
  /** not documented */
  pseudo?: boolean;
  /** not documented, looks for charts-version in url query params */
  enableUrlSettings?: boolean;
}

declare namespace google {
  namespace charts {
    /** Loads with `safeMode` enabled. */
    function safeLoad(options: LoadOptions): Promise<void>;
    function load(options: LoadOptions): Promise<void>;
    function load(
      version: string | number,
      options: LoadOptions
    ): Promise<void>;
    /** Legacy https://developers.google.com/chart/interactive/docs/basic_load_libs#updateloader */
    function load(
      visualization: 'visualization',
      version: string | number,
      options: LoadOptions
    ): Promise<void>;

    function setOnLoadCallback(handler: Function): void;
  }

  namespace visualization {
    /**
     * Value of a Cell.
     *
     * Note that undefined is not allowed and not all types use it yet.
     */
    type DataValue = number | string | boolean | Date | number[] | null;

    class DataInterface {}
    class DataTable implements DataInterface {
      addColumn(type: string, label?: string | null, id?: string): number;
      addRow(cellArray?: any[]): number;
      addRows(rows: any[][]): number;
    }

    function arrayToDataTable(data: any[], firstRowIsData?: boolean): DataTable;

    abstract class CoreChart {
      constructor(container: Element);
      // getChartLayoutInterface(): ChartLayoutInterface;
      getContainer(): Element;
      computeDiff(oldData: any, newData: any): any;
      draw(data: DataTable, opt_options?: any, opt_state?: any): void;
    }
    class LineChart extends CoreChart {}
  }
}

// const chromeStatsSample = {
//   data: [
//     {
//       run_timestamp: '2020-04-30T00:00:00Z',
//       test_pass_count: 987329,
//       total_tests_count: 2004705,
//     },
//     // ...
//   ],
//   metadata: {
//     next_page_token:
//       'eyJsYXN0X3RpbWVfc3RhcnQiOiIyMDIwLTA0LTMwVDAwOjAwOjAwWiIsImxhc3RfcnVuX2lkIjoyOTAwMH0',
//   },
// };

// const firefoxStatsSample = {
//   data: [
//     {
//       run_timestamp: '2020-04-30T00:00:00Z',
//       test_pass_count: 1019559,
//       total_tests_count: 2031833,
//     },
//     // ...
//   ],
//   metadata: {
//     next_page_token:
//       'eyJsYXN0X3RpbWVfc3RhcnQiOiIyMDIwLTA0LTI5VDAwOjAwOjAwWiIsImxhc3RfcnVuX2lkIjo5NTA4NDB9',
//   },
// };

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

  // Create data table rows from sample data
  createGlobalFeatureSupportData(): google.visualization.DataTable {
    const testData = this.globalFeatureSupport;

    const data = new google.visualization.DataTable();
    data.addColumn('date', 'Date');
    // addColumn for each browser in testData

    data.addColumn('number', 'Total');

    // Iterate through testData to create a row for each day.
    const rows = testData.map(dataRow => {
      const date = dataRow.run_timestamp;
      // Get passes for each browser ...
      const total = dataRow.total_tests_count;
      return [
        date,
        // browsers data
        total,
      ];
    });
    data.addRows(rows);
    return data;
  }

  // Create random data for globalFeatureSupport,
  // with first column being a date, ranging from Jan 2020 to now,
  // and the rest of the columns being for each browser and total,
  // with values, ranging from 5000 to 60000,
  // incrementing from previous values on random dates.
  // This is just to test the chart rendering.
  createRandomGlobalFeatureSupportData(): google.visualization.DataTable {
    const dataTable = new google.visualization.DataTable();
    dataTable.addColumn('date', 'Date');
    for (const browser of ['Chrome', 'Firefox', 'Safari', 'Edge']) {
      dataTable.addColumn('number', browser);
    }
    dataTable.addColumn('number', 'Total');

    const now = new Date();
    const startDate = new Date(2020, 0, 1);
    const start = new Date(startDate.getFullYear(), 0, 1);
    const end = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    const dateRange = end.getTime() - start.getTime();
    const numDays = Math.ceil(dateRange / (1000 * 60 * 60 * 24));

    const browsers = ['Chrome', 'Firefox', 'Safari', 'Edge'];
    const columns = ['Date', 'Chrome', 'Firefox', 'Safari', 'Edge', 'Total'];

    const dataRows = [];

    // Compute random starting value for each browser and total.
    const browserValues = [];
    let total = 0;
    for (const _browser of browsers) {
      browserValues.push(Math.floor(Math.random() * 10000));
      total += browserValues[browserValues.length - 1];
    }

    dataRows.push([start, ...browserValues, total]);

    for (let i = 1; i < numDays; i++) {
      // row is a date followed by numbers for each browser and total.
      const row: Array<google.visualization.DataValue> = [
        new Date(start.getTime() + i * (1000 * 60 * 60 * 24)),
      ];

      // Reset totoal for next row.
      total = 0;

      for (const _browser of browsers) {
        // Get previous value
        const previousValue = Number(
          dataRows[i - 1][columns.indexOf(_browser)]
        );
        // Decide whether to increment value from the previous value.
        const increment = Math.random() < 0.05;
        const value = increment
          ? previousValue + Math.floor(Math.random() * 1000)
          : previousValue;

        row.push(value);
        total += value;
      }
      row.push(total);
      dataRows.push(row);
    }
    dataTable.addRows(dataRows);

    return dataTable;
  }

  createGlobalFeatureSupportChart(): void {
    const data = this.createRandomGlobalFeatureSupportData();

    const options = {
      hAxis: {title: 'Feature', titleTextStyle: {color: '#333'}},
      vAxis: {minValue: 0},
      legend: {position: 'top'},
    };

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
            <sl-radio-button value="WPT">WPT</sl-radio-button>
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
      <sl-card class="halign-stretch" id="baseline-features">
        <div slot="header">Baseline features</div>
        <p class="under-construction">Small chart goes here...</p>
      </sl-card>
    `;
  }

  renderTimeToAvailability(): TemplateResult {
    return html`
      <sl-card class="halign-stretch" id="time-to-availibility">
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
