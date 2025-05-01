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
import {WebstatusFeatureWPTProgressChartPanel} from '../webstatus-feature-wpt-progress-chart-panel.js';
import {
  APIClient,
  DEFAULT_TEST_VIEW,
  FeatureWPTMetricViewType,
  WPTRunMetric,
} from '../../api/client.js';
import '../webstatus-feature-wpt-progress-chart-panel.js';
import {WebstatusLineChartTabbedPanel} from '../webstatus-line-chart-tabbed-panel.js';

const startDate = new Date('2024-01-01');
const endDate = new Date('2024-01-31');
async function createFixtureElement(
  startDate: Date,
  endDate: Date,
  testView: FeatureWPTMetricViewType,
) {
  return await fixture<WebstatusFeatureWPTProgressChartPanel>(
    testHtml`<webstatus-feature-wpt-progress-chart-panel
    .startDate=${startDate}
    .endDate=${endDate}
    .testView=${testView}
      featureId="test-feature-id"
    ></webstatus-feature-wpt-progress-chart-panel>`,
  );
}

describe('WebstatusFeatureWPTProgressChartPanel', () => {
  let el: WebstatusFeatureWPTProgressChartPanel;
  let apiClientStub: SinonStubbedInstance<APIClient>;
  let populateDataForChartByViewStub: SinonStub;

  beforeEach(async () => {
    apiClientStub = stub(new APIClient(''));
    populateDataForChartByViewStub = stub(
      WebstatusLineChartTabbedPanel.prototype,
      '_populateDataForChartByView',
    );
    el = await createFixtureElement(startDate, endDate, DEFAULT_TEST_VIEW);
    el.apiClient = apiClientStub;
    await el.updateComplete;
  });

  afterEach(() => {
    populateDataForChartByViewStub.restore();
  });

  it('renders the card', async () => {
    const card = el.shadowRoot!.querySelector('sl-card');
    expect(card).to.exist;
  });

  it('renders the panel header', async () => {
    const header = el.shadowRoot!.querySelector('[slot="header"]');
    expect(header).to.exist;
    expect(header!.textContent).to.contain('Implementation progress');
  });

  it('uses the correct dataFetchStartDate and dataFetchEndDate', async () => {
    // Start date should use the default dataFetchStartDateOffsetMsec and dataFetchEndDateOffsetMsec
    // Default dataFetchStartDateOffsetMsec is 30 days
    // Default dataFetchEndDateOffsetMsec is 0 days
    expect(el.dataFetchStartDate).to.deep.equal(new Date('2023-12-02'));
    expect(el.dataFetchEndDate).to.deep.equal(new Date('2024-01-31'));
  });

  it('calls _populateDataForChartByView with correct configurations', async () => {
    // Call method for both desktop and mobile views.
    expect(populateDataForChartByViewStub).to.have.been.calledTwice;
    let [fetchFunctionConfigs, dataIndex, additionalSeriesConfigs] =
      populateDataForChartByViewStub.getCall(0).args;
    expect(fetchFunctionConfigs.length).to.equal(4); // 4 browsers
    expect(dataIndex).to.equal(0); // First view index

    // Test Chrome configuration
    const chromeConfig = fetchFunctionConfigs[0];
    expect(chromeConfig.label).to.equal('Chrome');
    expect(chromeConfig.fetchFunction).to.be.a('function');
    const chromeTestDataPoint: WPTRunMetric = {
      run_timestamp: '2024-01-01T12:34:56.789Z',
      total_tests_count: 10,
      test_pass_count: 5,
    };
    expect(chromeConfig.timestampExtractor(chromeTestDataPoint)).to.deep.equal(
      new Date('2024-01-01T13:00:00.000Z'), // Expecting the rounded timestamp
    );
    expect(chromeConfig.valueExtractor(chromeTestDataPoint)).to.equal(5);

    // Test Firefox configuration
    const firefoxConfig = fetchFunctionConfigs[1];
    expect(firefoxConfig.label).to.equal('Firefox');
    expect(firefoxConfig.fetchFunction).to.be.a('function');
    const firefoxTestDataPoint: WPTRunMetric = {
      run_timestamp: '2024-01-01',
      total_tests_count: 12,
      test_pass_count: 7,
    };
    expect(
      firefoxConfig.timestampExtractor(firefoxTestDataPoint),
    ).to.deep.equal(new Date('2024-01-01'));
    expect(firefoxConfig.valueExtractor(firefoxTestDataPoint)).to.equal(7);

    // Test Safari configuration
    const safariConfig = fetchFunctionConfigs[2];
    expect(safariConfig.label).to.equal('Safari');
    expect(safariConfig.fetchFunction).to.be.a('function');
    const safariTestDataPoint: WPTRunMetric = {
      run_timestamp: '2024-01-01',
      total_tests_count: 8,
      test_pass_count: 3,
    };
    expect(safariConfig.timestampExtractor(safariTestDataPoint)).to.deep.equal(
      new Date('2024-01-01'),
    );
    expect(safariConfig.valueExtractor(safariTestDataPoint)).to.equal(3);

    // Test Edge configuration
    const edgeConfig = fetchFunctionConfigs[3];
    expect(edgeConfig.label).to.equal('Edge');
    expect(edgeConfig.fetchFunction).to.be.a('function');
    const edgeTestDataPoint: WPTRunMetric = {
      run_timestamp: '2024-01-01',
      total_tests_count: 15,
      test_pass_count: 9,
    };
    expect(edgeConfig.timestampExtractor(edgeTestDataPoint)).to.deep.equal(
      new Date('2024-01-01'),
    );
    expect(edgeConfig.valueExtractor(edgeTestDataPoint)).to.equal(9);

    // Check additional series configurations
    expect(additionalSeriesConfigs).to.have.lengthOf(1);
    const totalConfig = additionalSeriesConfigs[0];
    expect(totalConfig.calculator).to.equal(el.calculateMax);
    const totalTestDataPoint: WPTRunMetric = {
      run_timestamp: '2024-01-01T12:34:56.789Z',
      total_tests_count: 15,
      test_pass_count: 9,
    };
    expect(totalConfig.timestampExtractor(totalTestDataPoint)).to.deep.equal(
      new Date('2024-01-01T13:00:00.000Z'), // Expecting the rounded timestamp
    );
    expect(totalConfig.valueExtractor(totalTestDataPoint)).to.equal(15);

    [fetchFunctionConfigs, dataIndex, additionalSeriesConfigs] =
      populateDataForChartByViewStub.getCall(1).args;

    expect(fetchFunctionConfigs.length).to.equal(3); // 3 browsers
    expect(dataIndex).to.equal(1); // Second view index

    // Test Chrome Android configuration
    const chromeAndroidConfig = fetchFunctionConfigs[0];
    expect(chromeAndroidConfig.label).to.equal('Chrome Android');
    expect(chromeAndroidConfig.fetchFunction).to.be.a('function');
    const chromeAndroidTestDataPoint: WPTRunMetric = {
      run_timestamp: '2024-01-01T12:34:56.789Z',
      total_tests_count: 10,
      test_pass_count: 5,
    };
    expect(
      chromeAndroidConfig.timestampExtractor(chromeAndroidTestDataPoint),
    ).to.deep.equal(
      new Date('2024-01-01T13:00:00.000Z'), // Expecting the rounded timestamp
    );
    expect(
      chromeAndroidConfig.valueExtractor(chromeAndroidTestDataPoint),
    ).to.equal(5);

    // Test Firefox configuration
    const firefoxAndroidConfig = fetchFunctionConfigs[1];
    expect(firefoxAndroidConfig.label).to.equal('Firefox Android');
    expect(firefoxAndroidConfig.fetchFunction).to.be.a('function');
    const firefoxAndroidTestDataPoint: WPTRunMetric = {
      run_timestamp: '2024-01-01',
      total_tests_count: 12,
      test_pass_count: 7,
    };
    expect(
      firefoxAndroidConfig.timestampExtractor(firefoxAndroidTestDataPoint),
    ).to.deep.equal(new Date('2024-01-01'));
    expect(
      firefoxAndroidConfig.valueExtractor(firefoxAndroidTestDataPoint),
    ).to.equal(7);

    // Test Safari configuration
    const safariIosConfig = fetchFunctionConfigs[2];
    expect(safariIosConfig.label).to.equal('Safari iOS');
    expect(safariIosConfig.fetchFunction).to.be.a('function');
    const safariIosTestDataPoint: WPTRunMetric = {
      run_timestamp: '2024-01-01',
      total_tests_count: 8,
      test_pass_count: 3,
    };
    expect(
      safariIosConfig.timestampExtractor(safariIosTestDataPoint),
    ).to.deep.equal(new Date('2024-01-01'));
    expect(safariIosConfig.valueExtractor(safariIosTestDataPoint)).to.equal(3);
  });

  it('generates chart options correctly', () => {
    const options = el.generateDisplayDataChartOptions();
    // Check colors based on browsers displayed.
    // 4 browsers and total.
    expect(options.colors).eql([
      '#34A853',
      '#F48400',
      '#4285F4',
      '#7851A9',
      '#888888',
    ]);
    expect(options.hAxis?.viewWindow?.min).to.deep.equal(el.startDate);
    const expectedEndDate = new Date(
      el.endDate.getTime() + 1000 * 60 * 60 * 24,
    );
    expect(options.hAxis?.viewWindow?.max).to.deep.equal(expectedEndDate);
  });

  describe('metric view specific tests', () => {
    it('generates metric view specific chart options correctly when view=test', async () => {
      populateDataForChartByViewStub.reset();
      el = await createFixtureElement(startDate, endDate, 'test_counts');
      el.apiClient = apiClientStub;
      await el.updateComplete;
      const options = el.generateDisplayDataChartOptions();
      expect(options.vAxis?.title).to.equal('Number of tests passed');
      // Call method for both desktop and mobile views.
      expect(populateDataForChartByViewStub).to.have.been.calledTwice;
      const additionalSeriesConfigs =
        populateDataForChartByViewStub.getCall(0).args[2];
      // Check additional series configurations
      expect(additionalSeriesConfigs).to.have.lengthOf(1);
      const totalConfig = additionalSeriesConfigs[0];
      expect(totalConfig.label).to.equal('Total number of tests');
    });
    it('generates metric view specific chart options correctly when view=subtest', async () => {
      populateDataForChartByViewStub.reset();
      el = await createFixtureElement(startDate, endDate, 'subtest_counts');
      el.apiClient = apiClientStub;
      await el.updateComplete;
      const options = el.generateDisplayDataChartOptions();
      expect(options.vAxis?.title).to.equal('Number of subtests passed');
      // Call method for both desktop and mobile views.
      expect(populateDataForChartByViewStub).to.have.been.calledTwice;
      const additionalSeriesConfigs =
        populateDataForChartByViewStub.getCall(0).args[2];
      // Check additional series configurations
      expect(additionalSeriesConfigs).to.have.lengthOf(1);
      const totalConfig = additionalSeriesConfigs[0];
      expect(totalConfig.label).to.equal('Total number of subtests');
    });
  });
});
