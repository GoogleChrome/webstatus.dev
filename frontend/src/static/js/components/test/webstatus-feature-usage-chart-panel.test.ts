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
import {WebstatusFeatureUsageChartPanel} from '../webstatus-feature-usage-chart-panel.js';
import {APIClient, ChromeUsageStat} from '../../api/client.js';
import {WebstatusLineChartPanel} from '../webstatus-line-chart-panel.js';

import '../webstatus-feature-usage-chart-panel.js';

describe('WebstatusFeatureUsageChartPanel', () => {
  let el: WebstatusFeatureUsageChartPanel;
  let apiClientStub: SinonStubbedInstance<APIClient>;
  let populateDataForChartStub: SinonStub;
  const startDate = new Date('2024-01-01');
  const endDate = new Date('2024-01-31');

  beforeEach(async () => {
    apiClientStub = stub(new APIClient(''));
    populateDataForChartStub = stub(
      WebstatusLineChartPanel.prototype,
      '_populateDataForChart',
    );
    el = await fixture<WebstatusFeatureUsageChartPanel>(
      testHtml`<webstatus-feature-usage-chart-panel
      .startDate=${startDate}
      .endDate=${endDate}
        featureId="test-feature-id"
      ></webstatus-feature-usage-chart-panel>`,
    );
    el.apiClient = apiClientStub;
    await el.updateComplete;
  });

  afterEach(() => {
    populateDataForChartStub.restore();
  });

  it('renders the card', async () => {
    const card = el.shadowRoot!.querySelector('sl-card');
    expect(card).to.exist;
  });

  it('renders the panel header', async () => {
    const header = el.shadowRoot!.querySelector('[slot="header"]');
    expect(header).to.exist;
    expect(header!.textContent).to.contain('Feature Usage');
  });

  it('uses the correct dataFetchStartDate and dataFetchEndDate', async () => {
    // Start date should use the default dataFetchStartDateOffsetMsec and dataFetchEndDateOffsetMsec
    // Default dataFetchStartDateOffsetMsec is 30 days
    // Default dataFetchEndDateOffsetMsec is 0 days
    expect(el.dataFetchStartDate).to.deep.equal(new Date('2023-12-02'));
    expect(el.dataFetchEndDate).to.deep.equal(new Date('2024-01-31'));
  });

  it('calls _populateDataForChart with correct configurations', async () => {
    expect(populateDataForChartStub).to.have.been.calledOnce;
    const [fetchFunctionConfigs, additionalSeriesConfigs] =
      populateDataForChartStub.getCall(0).args;

    expect(fetchFunctionConfigs.length).to.equal(1); // Only 1 browser (Chrome)

    // Test Chrome configuration
    const chromeConfig = fetchFunctionConfigs[0];
    expect(chromeConfig.label).to.equal('Chrome');
    expect(chromeConfig.fetchFunction).to.be.a('function');
    const chromeTestDataPoint: ChromeUsageStat = {
      timestamp: '2024-01-01',
      usage: 0.5,
    };
    expect(chromeConfig.timestampExtractor(chromeTestDataPoint)).to.deep.equal(
      new Date('2024-01-01'),
    );
    expect(chromeConfig.valueExtractor(chromeTestDataPoint)).to.equal(50);
    expect(chromeConfig.tooltipExtractor!(chromeTestDataPoint)).to.equal(
      'Chrome: 50.0%',
    );

    // Test rounding to 1 decimal place
    const chromeTestDataPoint2: ChromeUsageStat = {
      timestamp: '2024-01-01',
      usage: 0.12345,
    };
    expect(chromeConfig.valueExtractor(chromeTestDataPoint2)).to.equal(12.3);
    expect(chromeConfig.tooltipExtractor!(chromeTestDataPoint2)).to.equal(
      'Chrome: 12.3%',
    );

    // Test rounding to 100%
    const chromeTestDataPoint3: ChromeUsageStat = {
      timestamp: '2024-01-01',
      usage: 1.0,
    };
    expect(chromeConfig.valueExtractor(chromeTestDataPoint3)).to.equal(100);
    expect(chromeConfig.tooltipExtractor!(chromeTestDataPoint3)).to.equal(
      'Chrome: 100%',
    );

    // Test undefined usage
    const chromeTestDataPoint4: ChromeUsageStat = {
      timestamp: '2024-01-01',
      usage: undefined,
    };
    expect(chromeConfig.valueExtractor(chromeTestDataPoint4)).to.equal(0);
    expect(chromeConfig.tooltipExtractor!(chromeTestDataPoint4)).to.equal(
      'Chrome: 0.0%',
    );

    // Test value between 0.01% and 0.1%
    const chromeTestDataPoint5: ChromeUsageStat = {
      timestamp: '2024-01-01',
      usage: 0.0009, // 0.09%
    };
    expect(chromeConfig.valueExtractor(chromeTestDataPoint5)).to.equal(0.09);
    expect(chromeConfig.tooltipExtractor!(chromeTestDataPoint5)).to.equal(
      'Chrome: 0.09%',
    );

    // Test value of 0.01%
    const chromeTestDataPoint6: ChromeUsageStat = {
      timestamp: '2024-01-01',
      usage: 0.0001, // 0.01%
    };
    expect(chromeConfig.valueExtractor(chromeTestDataPoint6)).to.equal(0.01);
    expect(chromeConfig.tooltipExtractor!(chromeTestDataPoint6)).to.equal(
      'Chrome: 0.01%',
    );

    // Test value less than 0.01%
    const chromeTestDataPoint7: ChromeUsageStat = {
      timestamp: '2024-01-01',
      usage: 0.00005, // 0.005%
    };
    expect(chromeConfig.valueExtractor(chromeTestDataPoint7)).to.equal(0.005);
    expect(chromeConfig.tooltipExtractor!(chromeTestDataPoint7)).to.equal(
      'Chrome: 0.005%',
    );

    // Assert that there are no additional series configurations
    expect(additionalSeriesConfigs).to.be.undefined;
  });

  it('generates chart options correctly', () => {
    const options = el.generateDisplayDataChartOptions();
    expect(options.vAxis?.title).to.equal('Usage (%)');

    // Check colors based on browsers displayed.
    // Only Chrome.
    expect(options.colors).eql(['#34A853']);
    expect(options.hAxis?.viewWindow?.min).to.deep.equal(el.startDate);
    const expectedEndDate = new Date(
      el.endDate.getTime() + 1000 * 60 * 60 * 24,
    );
    expect(options.hAxis?.viewWindow?.max).to.deep.equal(expectedEndDate);
  });
});
