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
  WebstatusLineChartPanel,
  LineChartMetricData,
  FetchFunctionConfig,
  AdditionalSeriesConfig,
} from '../webstatus-line-chart-panel.js';
import {Task} from '@lit/task';
import {WebStatusDataObj} from '../webstatus-gchart.js';
import {TemplateResult, html} from 'lit';
import {customElement} from 'lit/decorators.js';
import {createMockIterator, taskUpdateComplete} from './test-helpers.test.js';

// Interface for the data used in LineChartMetricData
interface MetricDataPoint {
  date: Date;
  value: number;
}

@customElement('test-line-chart-panel')
class TestLineChartPanel extends WebstatusLineChartPanel {
  resolveTask!: (value: WebStatusDataObj) => void;
  rejectTask!: (reason: Error) => void;

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

  getPanelID(): string {
    return 'test-panel';
  }

  getPanelText(): string {
    return 'Test Panel';
  }

  renderControls(): TemplateResult {
    return html``;
  }

  getDisplayDataChartOptionsInput(): {
    seriesColors: Array<string>;
    vAxisTitle: string;
  } {
    return {seriesColors: ['blue'], vAxisTitle: 'Test Value'};
  }
}

describe('WebstatusLineChartPanel', () => {
  let el: TestLineChartPanel;

  beforeEach(async () => {
    el = await fixture(testHtml`<test-line-chart-panel
    .startDate=${new Date('2024-01-01')}
    .endDate=${new Date('2024-01-31')}
    ></test-line-chart-panel>`);
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
    const chartContainer = el.shadowRoot!.querySelector('#test-panel-complete');
    expect(chartContainer).to.exist;

    const chart = el.shadowRoot!.querySelector('#test-panel-chart');
    expect(chart).to.exist;
  });

  it('sets display data correctly', () => {
    const metricDataArray: Array<LineChartMetricData<MetricDataPoint>> = [
      {
        label: 'Metric 1',
        data: [
          {date: new Date('2024-01-01'), value: 10},
          {date: new Date('2024-01-02'), value: 20},
        ],
        getTimestamp: (dataPoint: MetricDataPoint) => dataPoint.date,
        getValue: (dataPoint: MetricDataPoint) => dataPoint.value,
      },
      {
        label: 'Metric 2',
        data: [
          {date: new Date('2024-01-01'), value: 15},
          {date: new Date('2024-01-02'), value: 25},
          {date: new Date('2024-01-03'), value: 30},
        ],
        getTimestamp: (dataPoint: MetricDataPoint) => dataPoint.date,
        getValue: (dataPoint: MetricDataPoint) => dataPoint.value,
      },
    ];

    el.setDisplayDataFromMap(metricDataArray);
    expect(el.data).to.exist;
    expect(el.data!.cols).to.deep.equal([
      {type: 'date', label: 'Date', role: 'domain'},
      {type: 'number', label: 'Metric 1', role: 'data'},
      {type: 'number', label: 'Metric 2', role: 'data'},
    ]);
    expect(el.data!.rows).to.deep.equal([
      [new Date('2024-01-01'), 10, 15], // Values for both metrics on the same date
      [new Date('2024-01-02'), 20, 25], // Values for both metrics on the same date
      [new Date('2024-01-03'), null, 30], // Metric 1 is null because it has no data for 2024-01-03
    ]);
  });

  it('generates chart options correctly', () => {
    const options = el.generateDisplayDataChartOptions();
    expect(options.vAxis?.title).to.equal('Test Value');
    expect(options.colors).to.deep.equal(['blue']);
    expect(options.hAxis?.viewWindow?.min).to.deep.equal(el.startDate);
    const expectedEndDate = new Date(
      el.endDate.getTime() + 1000 * 60 * 60 * 24,
    );
    expect(options.hAxis?.viewWindow?.max).to.deep.equal(expectedEndDate);
  });

  // Skip for now.
  // TODO. Revisit getting the element to stay in initial mode in the future.
  it.skip('renders initial state', async () => {
    const initialEl = await fixture<TestLineChartPanel>(
      testHtml`<test-line-chart-panel></test-line-chart-panel>`,
    );
    const initialMessage = initialEl.shadowRoot!.querySelector(
      '#test-panel-initial',
    );
    expect(initialMessage).to.exist;
    expect(initialMessage!.textContent).to.include(
      'Preparing request for stats.',
    );
  });

  it('renders pending state', async () => {
    const pendingEl = await fixture(
      testHtml`<test-line-chart-panel></test-line-chart-panel>`,
    );
    const pendingMessage = pendingEl.shadowRoot!.querySelector(
      '#test-panel-pending',
    );
    expect(pendingMessage).to.exist;
    expect(pendingMessage!.textContent).to.include('Loading chart');
  });

  it('renders error state', async () => {
    el.rejectTask(new Error('Test Error'));
    await taskUpdateComplete();
    const errorMessage = el.shadowRoot!.querySelector('#test-panel-error');
    expect(errorMessage).to.exist;
    expect(errorMessage!.textContent).to.include('Error when loading chart');
  });

  describe('_fetchAndAggregateData', () => {
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

      await el._fetchAndAggregateData(
        fetchFunctionConfigs,
        additionalSeriesConfigs,
      );
      await el.updateComplete;

      expect(el.data).to.exist;
      expect(el.data!.cols).to.deep.equal([
        {type: 'date', label: 'Date', role: 'domain'},
        {type: 'number', label: 'Metric 1', role: 'data'},
        {type: 'number', label: 'Metric 2', role: 'data'},
        {type: 'number', label: 'Total', role: 'data'}, // Check for the additional 'Total' column
      ]);
      expect(el.data!.rows).to.deep.equal([
        [new Date('2024-01-01'), 10, 15, 15], // Total should be 15 (max of 10 and 15)
        [new Date('2024-01-02'), 20, 25, 25], // Total should be 25 (max of 20 and 25)
        [new Date('2024-01-03'), null, 30, 30], // Max should be 30
        [new Date('2024-01-04'), 35, null, 35], // Max should be 35
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

      await el._fetchAndAggregateData(fetchFunctionConfigs);

      await startingListener;
      const {detail} = await completeListener;
      expect(detail.get('Metric 1')!.data).to.deep.equal([
        {date: new Date('2024-01-01'), value: 10},
      ]);
    });
  });
});
