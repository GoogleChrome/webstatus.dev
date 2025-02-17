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
import {APIClient, ChromiumUsageStat} from '../../api/client.js';
import {WebstatusLineChartPanel} from '../webstatus-line-chart-panel.js';

import '../webstatus-feature-usage-chart-panel.js';

describe('WebstatusFeatureUsageChartPanel', () => {
  let el: WebstatusFeatureUsageChartPanel;
  let apiClientStub: SinonStubbedInstance<APIClient>;
  let fetchAndAggregateDataStub: SinonStub;
  const startDate = new Date('2024-01-01');
  const endDate = new Date('2024-01-31');

  beforeEach(async () => {
    apiClientStub = stub(new APIClient(''));
    fetchAndAggregateDataStub = stub(
      WebstatusLineChartPanel.prototype,
      '_fetchAndAggregateData',
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
    fetchAndAggregateDataStub.restore();
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

  it('calls _fetchAndAggregateData with correct configurations', async () => {
    expect(fetchAndAggregateDataStub).to.have.been.calledOnce;
    const [fetchFunctionConfigs, additionalSeriesConfigs] =
      fetchAndAggregateDataStub.getCall(0).args;

    expect(fetchFunctionConfigs.length).to.equal(1); // Only 1 browser (Chrome)

    // Test Chrome configuration
    const chromeConfig = fetchFunctionConfigs[0];
    expect(chromeConfig.label).to.equal('Chrome');
    expect(chromeConfig.fetchFunction).to.be.a('function');
    const chromeTestDataPoint: ChromiumUsageStat = {
      timestamp: '2024-01-01',
      usage: 0.5,
    };
    expect(chromeConfig.timestampExtractor(chromeTestDataPoint)).to.deep.equal(
      new Date('2024-01-01'),
    );
    expect(chromeConfig.valueExtractor(chromeTestDataPoint)).to.equal(0.5);
    expect(chromeConfig.tooltipExtractor!(chromeTestDataPoint)).to.equal(
      'Chrome: 50%',
    );

    // Assert that there are no additional series configurations
    expect(additionalSeriesConfigs).to.be.undefined;
  });

  it('generates chart options correctly', () => {
    const options = el.generateDisplayDataChartOptions();
    expect(options.vAxis?.title).to.equal('Usage (%)');

    // Check colors based on browsers displayed.
    // Only Chrome.
    expect(options.colors).eql(['#FF0000']);
    expect(options.hAxis?.viewWindow?.min).to.deep.equal(el.startDate);
    const expectedEndDate = new Date(
      el.endDate.getTime() + 1000 * 60 * 60 * 24,
    );
    expect(options.hAxis?.viewWindow?.max).to.deep.equal(expectedEndDate);
  });
});
