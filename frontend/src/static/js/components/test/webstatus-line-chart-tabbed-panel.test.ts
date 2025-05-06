/**
 * Copyright 2025 Google LLC
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

import {fixture, html as testHtml, expect, oneEvent} from '@open-wc/testing';
import {
  FetchFunctionConfig,
  AdditionalSeriesConfig,
} from '../webstatus-line-chart-panel.js';
import {WebstatusLineChartTabbedPanel} from '../webstatus-line-chart-tabbed-panel.js';
import {Task} from '@lit/task';
import {WebStatusDataObj} from '../webstatus-gchart.js';
import {TemplateResult, html} from 'lit';
import {customElement} from 'lit/decorators.js';
import {createMockIterator, taskUpdateComplete} from './test-helpers.js';
import {BrowsersParameter} from '../../api/client.js';

// Interface for the data used in LineChartMetricData
interface MetricDataPoint {
  date: Date;
  value: number;
}

@customElement('test-line-chart-tabbed-panel')
class TestLineChartPanel extends WebstatusLineChartTabbedPanel<BrowsersParameter> {
  readonly series: BrowsersParameter[] = ['chrome', 'firefox', 'safari'];
  browsersByView: BrowsersParameter[][] = [
    ['chrome', 'firefox', 'safari'],
    ['chrome_android', 'firefox_android', 'safari_ios'],
  ];
  tabViews: string[] = ['Desktop', 'Mobile'];

  resolveTask!: (value: WebStatusDataObj) => void;
  rejectTask!: (reason: Error) => void;
  resolvePointSelectedTask!: (value: unknown) => void;
  rejectPointSelectedTask!: (reason: Error) => void;

  createLoadingTask(): Task {
    return new Task(
      this,
      () =>
        new Promise((resolve, reject) => {
          this.rejectTask = reject; // Assign the reject function directly
          this.resolveTask = resolve; // Assign the resolve function directly
        }),
      () => [this.startDate, this.endDate],
    );
  }

  createPointSelectedTask(): {
    task: Task;
    renderSuccess?: () => TemplateResult;
  } {
    return {
      task: new Task(this, {
        args: () => [this.startDate, this.endDate],
        task: async () => {
          return new Promise((resolve, reject) => {
            this.resolvePointSelectedTask = resolve;
            this.rejectPointSelectedTask = reject;
          });
        },
      }),
      renderSuccess() {
        return html`Task Success`;
      },
    };
  }

  getPanelID(): string {
    return 'test-panel';
  }

  getPanelText(): string {
    return 'Test Panel';
  }

  getPanelDescription(): TemplateResult {
    return html`Test Description`;
  }

  renderControls(): TemplateResult {
    return html``;
  }

  getDisplayDataChartOptionsInput<BrowsersParameter>(
    _browsers: BrowsersParameter[],
  ): {
    seriesColors: Array<string>;
    vAxisTitle: string;
  } {
    return {seriesColors: ['blue'], vAxisTitle: 'Test Value'};
  }
}

describe('WebstatusLineChartPanel', () => {
  let el: TestLineChartPanel;

  beforeEach(async () => {
    el = await fixture(testHtml`<test-line-chart-tabbed-panel
    .startDate=${new Date('2024-01-01')}
    .endDate=${new Date('2024-01-31')}
    ></test-line-chart-tabbed-panel>`);
    await el.updateComplete;
  });

  it('renders the card', async () => {
    const card = el.shadowRoot!.querySelector('sl-card');
    expect(card).to.exist;
  });

  it('renders the panel header', async () => {
    const header = el.shadowRoot!.querySelector('[slot="header"]');
    expect(header).to.exist;
    expect(header!.textContent).to.contain('Test Panel');
  });

  it('renders the chart container when complete', async () => {
    el.resolveTask({cols: [], rows: []});
    await taskUpdateComplete();
    const chartContainer = el.shadowRoot!.querySelector(
      '#test-panel-0-complete',
    );
    expect(chartContainer).to.exist;

    const chart = el.shadowRoot!.querySelector('#test-panel-chart');
    expect(chart).to.exist;
  });

  describe('_populateDataForChartByView', () => {
    it('fetches data and applies additional series calculators', async () => {
      const fetchFunctionConfigs: FetchFunctionConfig<MetricDataPoint>[] = [
        {
          label: 'Metric 1',
          fetchFunction: () =>
            createMockIterator<MetricDataPoint>([
              {date: new Date('2024-01-01'), value: 10},
              {date: new Date('2024-01-02'), value: 20},
              {date: new Date('2024-01-04'), value: 35},
            ]),
          timestampExtractor: (dataPoint: MetricDataPoint) => dataPoint.date,
          valueExtractor: (dataPoint: MetricDataPoint) => dataPoint.value,
        },
        {
          label: 'Metric 2',
          fetchFunction: () =>
            createMockIterator<MetricDataPoint>([
              {date: new Date('2024-01-01'), value: 15},
              {date: new Date('2024-01-02'), value: 25},
              {date: new Date('2024-01-03'), value: 30},
            ]),
          timestampExtractor: (dataPoint: MetricDataPoint) => dataPoint.date,
          valueExtractor: (dataPoint: MetricDataPoint) => dataPoint.value,
        },
      ];

      const additionalSeriesConfigs: AdditionalSeriesConfig<MetricDataPoint>[] =
        [
          {
            label: 'Total',
            calculator: el.calculateMax,
            cacheMap: new Map(),
            timestampExtractor: (dataPoint: MetricDataPoint) => dataPoint.date,
            valueExtractor: (dataPoint: MetricDataPoint) => dataPoint.value,
          },
        ];

      await el._populateDataForChartByView(
        fetchFunctionConfigs,
        0,
        additionalSeriesConfigs,
      );
      await el.updateComplete;

      const fetchFunctionConfigsTwo: FetchFunctionConfig<MetricDataPoint>[] = [
        {
          label: 'Metric 3',
          fetchFunction: () =>
            createMockIterator<MetricDataPoint>([
              {date: new Date('2024-01-11'), value: 20},
              {date: new Date('2024-01-12'), value: 30},
              {date: new Date('2024-01-14'), value: 45},
            ]),
          timestampExtractor: (dataPoint: MetricDataPoint) => dataPoint.date,
          valueExtractor: (dataPoint: MetricDataPoint) => dataPoint.value,
        },
        {
          label: 'Metric 4',
          fetchFunction: () =>
            createMockIterator<MetricDataPoint>([
              {date: new Date('2024-01-11'), value: 25},
              {date: new Date('2024-01-12'), value: 35},
              {date: new Date('2024-01-13'), value: 40},
            ]),
          timestampExtractor: (dataPoint: MetricDataPoint) => dataPoint.date,
          valueExtractor: (dataPoint: MetricDataPoint) => dataPoint.value,
        },
      ];

      const additionalSeriesConfigsTwo: AdditionalSeriesConfig<MetricDataPoint>[] =
        [
          {
            label: 'Total',
            calculator: el.calculateMax,
            cacheMap: new Map(),
            timestampExtractor: (dataPoint: MetricDataPoint) => dataPoint.date,
            valueExtractor: (dataPoint: MetricDataPoint) => dataPoint.value,
          },
        ];

      await el._populateDataForChartByView(
        fetchFunctionConfigsTwo,
        1,
        additionalSeriesConfigsTwo,
      );
      await el.updateComplete;

      // Check if the data in the first view is populated correctly
      expect(el.dataByView).to.exist;
      expect(el.dataByView![0].cols).to.deep.equal([
        {type: 'date', label: 'Date', role: 'domain'},
        {type: 'number', label: 'Metric 1', role: 'data'},
        {type: 'number', label: 'Metric 2', role: 'data'},
        {type: 'number', label: 'Total', role: 'data'}, // Check for the additional 'Total' column
      ]);
      expect(el.dataByView![0].rows).to.deep.equal([
        [new Date('2024-01-01'), 10, 15, 15], // Total should be 15 (max of 10 and 15)
        [new Date('2024-01-02'), 20, 25, 25], // Total should be 25 (max of 20 and 25)
        [new Date('2024-01-03'), null, 30, 30], // Max should be 30
        [new Date('2024-01-04'), 35, null, 35], // Max should be 35
      ]);

      // Check if the data in the second view is populated correctly
      expect(el.dataByView).to.exist;
      expect(el.dataByView![1].cols).to.deep.equal([
        {type: 'date', label: 'Date', role: 'domain'},
        {type: 'number', label: 'Metric 3', role: 'data'},
        {type: 'number', label: 'Metric 4', role: 'data'},
        {type: 'number', label: 'Total', role: 'data'}, // Check for the additional 'Total' column
      ]);
      expect(el.dataByView![1].rows).to.deep.equal([
        [new Date('2024-01-11'), 20, 25, 25], // Total should be 25 (max of 20 and 25)
        [new Date('2024-01-12'), 30, 35, 35], // Total should be 35 (max of 30 and 35)
        [new Date('2024-01-13'), null, 40, 40], // Max should be 40
        [new Date('2024-01-14'), 45, null, 45], // Max should be 45
      ]);
    });

    it('dispatches data-fetch-starting and data-fetch-complete events', async () => {
      const fetchFunctionConfigs: FetchFunctionConfig<MetricDataPoint>[] = [
        {
          label: 'Metric 1',
          fetchFunction: () =>
            createMockIterator([{date: new Date('2024-01-01'), value: 10}]),
          timestampExtractor: (dataPoint: MetricDataPoint) => dataPoint.date,
          valueExtractor: (dataPoint: MetricDataPoint) => dataPoint.value,
        },
      ];

      const startingListener = oneEvent(el, 'data-fetch-starting');
      const completeListener = oneEvent(el, 'data-fetch-complete');

      await el._populateDataForChartByView(fetchFunctionConfigs, 0);

      await startingListener;
      const {detail} = await completeListener;
      expect(detail.get('Metric 1')!.data).to.deep.equal([
        {date: new Date('2024-01-01'), value: 10},
      ]);
    });
  });
});
