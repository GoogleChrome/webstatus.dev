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
import {WebstatusStatsGlobalFeatureCountChartPanel} from '../webstatus-stats-global-feature-count-chart-panel.js';
import {
  APIClient,
  BaselineStatusMetric,
  BrowserReleaseFeatureMetric,
} from '../../api/client.js';
import {WebstatusLineChartPanel} from '../webstatus-line-chart-panel.js';

import '../webstatus-stats-global-feature-count-chart-panel.js';

describe('WebstatusStatsGlobalFeatureCountChartPanel', () => {
  let el: WebstatusStatsGlobalFeatureCountChartPanel;
  let apiClientStub: SinonStubbedInstance<APIClient>;
  let setDisplayDataFromMapStub: SinonStub;
  let fetchAndAggregateDataStub: SinonStub;
  const startDate = new Date('2024-01-01');
  const endDate = new Date('2024-01-31');

  beforeEach(async () => {
    apiClientStub = stub(new APIClient(''));
    fetchAndAggregateDataStub = stub(
      WebstatusLineChartPanel.prototype,
      '_fetchAndAggregateData',
    );
    setDisplayDataFromMapStub = stub(
      WebstatusLineChartPanel.prototype,
      'setDisplayDataFromMap',
    );
    el =
      await fixture<WebstatusStatsGlobalFeatureCountChartPanel>(testHtml`<webstatus-stats-global-feature-chart-panel
.startDate=${startDate}
.endDate=${endDate}
    ></webstatus-stats-global-feature-chart-panel>`);
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
    expect(header!.textContent).to.contain('Global feature support');
  });

  it('calls _fetchAndAggregateData with correct arguments', async () => {
    expect(fetchAndAggregateDataStub).to.have.been.calledOnce;
    const [fetchFunctionConfigs, additionalSeriesConfigs] =
      fetchAndAggregateDataStub.getCall(0).args;
    expect(fetchFunctionConfigs.length).to.equal(5); // 4 browsers + total
    // Test Chrome configuration
    const chromeConfig = fetchFunctionConfigs[0];
    expect(chromeConfig.label).to.equal('Chrome');
    expect(chromeConfig.fetchFunction).to.be.a('function');
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
    const safariTestDataPoint: BrowserReleaseFeatureMetric = {
      timestamp: '2024-01-01',
      count: 7,
    };
    expect(safariConfig.timestampExtractor(safariTestDataPoint)).to.deep.equal(
      new Date('2024-01-01'),
    );
    expect(safariConfig.valueExtractor(safariTestDataPoint)).to.equal(7);

    // Test Edge configuration
    const edgeConfig = fetchFunctionConfigs[3];
    expect(edgeConfig.label).to.equal('Edge');
    expect(edgeConfig.fetchFunction).to.be.a('function');
    const edgeTestDataPoint: BrowserReleaseFeatureMetric = {
      timestamp: '2024-01-01',
      count: 8,
    };
    expect(edgeConfig.timestampExtractor(edgeTestDataPoint)).to.deep.equal(
      new Date('2024-01-01'),
    );
    expect(edgeConfig.valueExtractor(edgeTestDataPoint)).to.equal(8);

    // Test Total configuration
    const totalConfig = fetchFunctionConfigs[4];
    expect(totalConfig.label).to.equal('Total number of Baseline features');
    expect(totalConfig.fetchFunction).to.be.a('function');
    const totalTestDataPoint: BaselineStatusMetric = {
      timestamp: '2024-01-01',
      count: 20,
    };
    expect(totalConfig.timestampExtractor(totalTestDataPoint)).to.deep.equal(
      new Date('2024-01-01'),
    );
    expect(totalConfig.valueExtractor(totalTestDataPoint)).to.equal(20);

    expect(additionalSeriesConfigs).to.be.undefined;
  });

  it('generates chart options correctly', () => {
    const options = el.generateDisplayDataChartOptions();
    expect(options.vAxis?.title).to.equal('Number of features supported');

    // Check colors based on browsers displayed.
    // 4 browsers and total.
    expect(options.colors).eql([
      '#FF0000',
      '#F48400',
      '#4285F4',
      '#0F9D58',
      '#888888',
    ]);
    expect(options.hAxis?.viewWindow?.min).to.deep.equal(el.startDate);
    const expectedEndDate = new Date(
      el.endDate.getTime() + 1000 * 60 * 60 * 24,
    );
    expect(options.hAxis?.viewWindow?.max).to.deep.equal(expectedEndDate);
  });
});
