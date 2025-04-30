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

import {fixture, html as testHtml, expect} from '@open-wc/testing';
import {SinonStub, SinonStubbedInstance, stub} from 'sinon';
import {WebstatusStatsMissingOneImplChartPanel} from '../webstatus-stats-missing-one-impl-chart-panel.js'; // Path to your component
import {APIClient, BrowserReleaseFeatureMetric} from '../../api/client.js';
import {WebstatusLineChartPanel} from '../webstatus-line-chart-panel.js';
import {ChartSelectPointEvent} from '../webstatus-gchart.js';

import '../webstatus-stats-missing-one-impl-chart-panel.js';

describe('WebstatusStatsMissingOneImplChartPanel', () => {
  let el: WebstatusStatsMissingOneImplChartPanel;
  let apiClientStub: SinonStubbedInstance<APIClient>;
  let setDisplayDataFromMapStub: SinonStub;
  let fetchAndAggregateDataStub: SinonStub;
  const startDate = new Date('2024-01-01');
  const endDate = new Date('2024-01-31');

  beforeEach(async () => {
    apiClientStub = stub(new APIClient(''));
    setDisplayDataFromMapStub = stub(
      WebstatusLineChartPanel.prototype,
      'processDisplayDataFromMap',
    );
    fetchAndAggregateDataStub = stub(
      WebstatusLineChartPanel.prototype,
      '_populateDataForChart',
    );
    el = await fixture<WebstatusStatsMissingOneImplChartPanel>(
      testHtml`<webstatus-stats-missing-one-impl-chart-panel
      .startDate=${startDate}
      .endDate=${endDate}
      ></webstatus-stats-missing-one-impl-chart-panel>`,
    );
    el.apiClient = apiClientStub;
    await el.updateComplete;
  });

  afterEach(() => {
    setDisplayDataFromMapStub.restore();
    fetchAndAggregateDataStub.restore();
  });

  it('renders the card', async () => {
    const card = el.shadowRoot!.querySelector('sl-card');
    expect(card).to.exist;
  });

  it('renders the panel header', async () => {
    const header = el.shadowRoot!.querySelector('[slot="header"]');
    expect(header).to.exist;
    expect(header!.textContent).to.contain(
      'Features missing in only one browser',
    );
  });

  it('uses the correct dataFetchStartDate and dataFetchEndDate', async () => {
    // Start date should use the default dataFetchStartDateOffsetMsec and dataFetchEndDateOffsetMsec
    // Default dataFetchStartDateOffsetMsec is 30 days
    // Default dataFetchEndDateOffsetMsec is 0 days
    expect(el.dataFetchStartDate).to.deep.equal(new Date('2023-12-02'));
    expect(el.dataFetchEndDate).to.deep.equal(new Date('2024-01-31'));
  });

  it('calls _fetchAndAggregateData with correct configurations', async () => {
    expect(fetchAndAggregateDataStub).to.have.been.calledOnce;
    const [fetchFunctionConfigs] = fetchAndAggregateDataStub.getCall(0).args;

    expect(fetchFunctionConfigs.length).to.equal(3); // 3 browsers

    // Test Chrome configuration
    const chromeConfig = fetchFunctionConfigs[0];
    expect(chromeConfig.label).to.equal('Chrome/Edge');
    expect(chromeConfig.fetchFunction).to.be.a('function');
    await chromeConfig.fetchFunction();
    expect(
      apiClientStub.getMissingOneImplementationCountsForBrowser,
    ).to.have.been.calledWith(
      'chrome',
      ['firefox', 'safari'],
      new Date('2023-12-02'),
      new Date('2024-01-31'),
    );
    const chromeTestDataPoint: BrowserReleaseFeatureMetric = {
      timestamp: '2024-01-01',
      count: 10,
    };
    expect(chromeConfig.timestampExtractor(chromeTestDataPoint)).to.deep.equal(
      new Date('2024-01-01'),
    );
    expect(chromeConfig.valueExtractor(chromeTestDataPoint)).to.equal(10);

    // Test Firefox configuration
    const firefoxConfig = fetchFunctionConfigs[1];
    expect(firefoxConfig.label).to.equal('Firefox');
    expect(firefoxConfig.fetchFunction).to.be.a('function');
    await firefoxConfig.fetchFunction();
    expect(
      apiClientStub.getMissingOneImplementationCountsForBrowser,
    ).to.have.been.calledWith(
      'firefox',
      ['chrome', 'safari'],
      new Date('2023-12-02'),
      new Date('2024-01-31'),
    );
    const firefoxTestDataPoint: BrowserReleaseFeatureMetric = {
      timestamp: '2024-01-01',
      count: 9,
    };
    expect(
      firefoxConfig.timestampExtractor(firefoxTestDataPoint),
    ).to.deep.equal(new Date('2024-01-01'));
    expect(firefoxConfig.valueExtractor(firefoxTestDataPoint)).to.equal(9);

    // Test Safari configuration
    const safariConfig = fetchFunctionConfigs[2];
    expect(safariConfig.label).to.equal('Safari');
    expect(safariConfig.fetchFunction).to.be.a('function');
    await safariConfig.fetchFunction();
    expect(
      apiClientStub.getMissingOneImplementationCountsForBrowser,
    ).to.have.been.calledWith(
      'safari',
      ['chrome', 'firefox'],
      new Date('2023-12-02'),
      new Date('2024-01-31'),
    );
    const safariTestDataPoint: BrowserReleaseFeatureMetric = {
      timestamp: '2024-01-01',
      count: 7,
    };
    expect(safariConfig.timestampExtractor(safariTestDataPoint)).to.deep.equal(
      new Date('2024-01-01'),
    );
    expect(safariConfig.valueExtractor(safariTestDataPoint)).to.equal(7);
  });

  it('generates chart options correctly', () => {
    const options = el.generateDisplayDataChartOptions();
    expect(options.vAxis?.title).to.equal('Number of features missing');
    expect(options.colors).eql(['#34A853', '#F48400', '#4285F4']);
    expect(options.hAxis?.viewWindow?.min).to.deep.equal(el.startDate);
    const expectedEndDate = new Date(
      el.endDate.getTime() + 1000 * 60 * 60 * 24,
    );
    expect(options.hAxis?.viewWindow?.max).to.deep.equal(expectedEndDate);
  });

  it('renders missing one implementation features footer', async () => {
    apiClientStub.getMissingOneImplementationFeatures.resolves([
      {
        feature_id: 'grid',
      },
      {
        feature_id: 'html',
      },
      {
        feature_id: 'js',
      },
      {
        feature_id: 'bluetooth',
      },
    ]);
    const chart = el.shadowRoot!.querySelector(
      '#missing-one-implementation-chart',
    )!;

    const chartClickEvent: ChartSelectPointEvent = new CustomEvent(
      'point-selected',
      {
        detail: {
          label: 'Chrome',
          timestamp: new Date('2024-01-01'),
          value: 123,
        },
        bubbles: true,
      },
    );
    // Simulate point-selected event on the chart component
    chart.dispatchEvent(chartClickEvent);
    await el.updateComplete;

    // Assert that the task and renderer are set (no need to wait for the event)
    expect(el._pointSelectedTask).to.exist;
    await el._pointSelectedTask?.taskComplete;
    await el.updateComplete;

    expect(el._renderCustomPointSelectedSuccess).to.exist;
    await el.updateComplete;

    const header = el.shadowRoot!.querySelector(
      '#missing-one-implementation-list-header',
    );
    expect(header).to.exist;
    const expectedHeader = `
      <div slot="header" id="missing-one-implementation-list-header">
        Missing features on 2024-01-01 for Chrome:
        <a href="/?q=id%3Agrid+OR+id%3Ahtml+OR+id%3Ajs+OR+id%3Abluetooth">4 features</a>
      </div>
    `;
    expect(header).dom.to.equal(expectedHeader);

    const table = el.shadowRoot!.querySelector('.missing-features-table');
    expect(table).to.exist;

    const rows = table!
      .getElementsByTagName('tbody')[0]
      .getElementsByTagName('tr');
    expect(rows.length).to.equal(10, 'should have 10 rows');

    const firstRowCells = rows[0].querySelectorAll('td');
    const textContent = firstRowCells[0].textContent
      ?.split('\n')
      .map(word => word.trim())
      .filter(word => word.length > 0)
      .join(' ');
    expect(textContent).to.equal('grid TOP CSS', 'first row ID');
  });

  it('renders empty features footer', async () => {
    apiClientStub.getMissingOneImplementationFeatures.resolves([]);
    const chart = el.shadowRoot!.querySelector(
      '#missing-one-implementation-chart',
    )!;

    const chartClickEvent: ChartSelectPointEvent = new CustomEvent(
      'point-selected',
      {
        detail: {
          label: 'Chrome',
          timestamp: new Date('2024-01-01'),
          value: 123,
        },
        bubbles: true,
      },
    );
    // Simulate point-selected event on the chart component
    chart.dispatchEvent(chartClickEvent);
    await el.updateComplete;

    // Assert that the task and renderer are set (no need to wait for the event)
    expect(el._pointSelectedTask).to.exist;
    await el._pointSelectedTask?.taskComplete;
    await el.updateComplete;

    expect(el._renderCustomPointSelectedSuccess).to.exist;
    await el.updateComplete;

    const header = el.shadowRoot!.querySelector(
      '#missing-one-implementation-list-header',
    );
    expect(header).to.exist;
    const expectedHeader = `
      <div slot="header" id="missing-one-implementation-list-header">
        No missing features for on 2024-01-01 for
        Chrome
      </div>
    `;
    expect(header).dom.to.equal(expectedHeader);

    const table = el.shadowRoot!.querySelector('.missing-features-table');
    expect(table).to.not.exist;
  });

  it('assert correct getMissingOneImplementationFeatures calls', async () => {
    apiClientStub.getMissingOneImplementationFeatures.resolves([
      {
        feature_id: 'css',
      },
      {
        feature_id: 'html',
      },
      {
        feature_id: 'js',
      },
      {
        feature_id: 'bluetooth',
      },
    ]);
    const chart = el.shadowRoot!.querySelector(
      '#missing-one-implementation-chart',
    )!;

    const chartClickEvent: ChartSelectPointEvent = new CustomEvent(
      'point-selected',
      {
        detail: {
          label: 'Safari',
          timestamp: new Date('2024-01-01'),
          value: 123,
        },
        bubbles: true,
      },
    );
    // Simulate point-selected event on the chart component
    chart.dispatchEvent(chartClickEvent);
    await el.updateComplete;

    expect(el._pointSelectedTask).to.exist;
    expect(
      apiClientStub.getMissingOneImplementationFeatures,
    ).to.have.been.calledWith(
      'safari',
      ['chrome', 'firefox'],
      new Date('2024-01-01'),
    );

    const chartClickEventOne: ChartSelectPointEvent = new CustomEvent(
      'point-selected',
      {
        detail: {
          label: 'Firefox',
          timestamp: new Date('2024-01-01'),
          value: 123,
        },
        bubbles: true,
      },
    );
    chart.dispatchEvent(chartClickEventOne);
    await el.updateComplete;

    expect(el._pointSelectedTask).to.exist;
    expect(
      apiClientStub.getMissingOneImplementationFeatures,
    ).to.have.been.calledWith(
      'firefox',
      ['chrome', 'safari'],
      new Date('2024-01-01'),
    );

    const chartClickEventThree: ChartSelectPointEvent = new CustomEvent(
      'point-selected',
      {
        detail: {
          label: 'Chrome',
          timestamp: new Date('2024-01-01'),
          value: 123,
        },
        bubbles: true,
      },
    );
    chart.dispatchEvent(chartClickEventThree);
    await el.updateComplete;

    expect(el._pointSelectedTask).to.exist;
    expect(
      apiClientStub.getMissingOneImplementationFeatures,
    ).to.have.been.calledWith(
      'chrome',
      ['firefox', 'safari'],
      new Date('2024-01-01'),
    );
  });
});
