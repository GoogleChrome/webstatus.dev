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
import {
  APIClient,
  BrowserReleaseFeatureMetric,
  BrowsersParameter,
} from '../../api/client.js';
import {SlMenu, SlMenuItem} from '@shoelace-style/shoelace';
import {
  LineChartMetricData,
  WebstatusLineChartPanel,
} from '../webstatus-line-chart-panel.js';

import '../webstatus-stats-missing-one-impl-chart-panel.js';
import {createMockIterator, taskUpdateComplete} from './test-helpers.test.js';

describe('WebstatusStatsMissingOneImplChartPanel', () => {
  let el: WebstatusStatsMissingOneImplChartPanel;
  let apiClientStub: SinonStubbedInstance<APIClient>;
  let setDisplayDataFromMapStub: SinonStub;

  beforeEach(async () => {
    apiClientStub = stub(new APIClient(''));
    setDisplayDataFromMapStub = stub(
      WebstatusLineChartPanel.prototype,
      'setDisplayDataFromMap',
    );
    el = await fixture<WebstatusStatsMissingOneImplChartPanel>(
      testHtml`<webstatus-stats-missing-one-impl-chart-panel
      .startDate=${new Date('2024-01-01')}
      .endDate=${new Date('2024-01-31')}
      ></webstatus-stats-missing-one-impl-chart-panel>`,
    );
    el.apiClient = apiClientStub;
    await el.updateComplete;
  });

  afterEach(() => {
    setDisplayDataFromMapStub.restore();
  });

  it('renders the card', async () => {
    const card = el.shadowRoot!.querySelector('sl-card');
    expect(card).to.exist;
  });

  it('renders the panel header', async () => {
    const header = el.shadowRoot!.querySelector('[slot="header"]');
    expect(header).to.exist;
    expect(header!.textContent).to.contain(
      'Features missing in only 1 browser',
    );
  });

  it('fetches data and calls setDisplayDataFromMap', async () => {
    const mockMissingOneCountData = new Map<
      BrowsersParameter,
      BrowserReleaseFeatureMetric[]
    >([
      ['chrome', [{timestamp: '2024-01-01', count: 10}]],
      ['edge', [{timestamp: '2024-01-01', count: 8}]],
      ['firefox', [{timestamp: '2024-01-01', count: 9}]],
      ['safari', [{timestamp: '2024-01-01', count: 7}]],
    ]);

    apiClientStub.getMissingOneImplementationCountsForBrowser.callsFake(
      browser => {
        const data = mockMissingOneCountData.get(browser)?.slice();
        return createMockIterator(data!);
      },
    );

    await el._task?.run();
    await el.updateComplete;
    await taskUpdateComplete();

    expect(setDisplayDataFromMapStub.calledOnce).to.be.true;
    const args = setDisplayDataFromMapStub.firstCall.args;
    expect(args.length).to.equal(1);

    const metricDataArray = args[0] as Array<
      LineChartMetricData<{
        timestamp: string;
        count?: number | undefined;
      }>
    >;

    const browserToDataMap = new Map<
      string,
      {
        timestamp: string;
        count?: number | undefined;
      }[]
    >();
    metricDataArray.forEach(item => {
      browserToDataMap.set(item.label, item.data);
    });

    const expectedMap = new Map([
      ['chrome', [{timestamp: '2024-01-01', count: 10}]],
      ['edge', [{timestamp: '2024-01-01', count: 8}]],
      ['firefox', [{timestamp: '2024-01-01', count: 9}]],
      ['safari', [{timestamp: '2024-01-01', count: 7}]],
    ]);
    expect(browserToDataMap).to.deep.equal(expectedMap);
  });

  it('generates chart options correctly', () => {
    const options = el.generateDisplayDataChartOptions();
    expect(options.vAxis?.title).to.equal('Number of features missing');
    // Only browsers.
    expect(options.colors).eql(['#FF0000', '#F48400', '#4285F4', '#0F9D58']);
    expect(options.hAxis?.viewWindow?.min).to.deep.equal(el.startDate);
    const expectedEndDate = new Date(
      el.endDate.getTime() + 1000 * 60 * 60 * 24,
    );
    expect(options.hAxis?.viewWindow?.max).to.deep.equal(expectedEndDate);
  });

  it('handles browser selection', async () => {
    const dropdown = el.shadowRoot!.querySelector(
      '#missing-one-implementation-browser-selector',
    ) as SlMenu;
    const menuItems = Array.from(dropdown.querySelectorAll('sl-menu-item'));

    // Simulate selecting Chrome and Firefox
    const chromeItem = menuItems.find(
      item => item.value === 'chrome',
    ) as SlMenuItem;
    const firefoxItem = menuItems.find(
      item => item.value === 'firefox',
    ) as SlMenuItem;

    if (chromeItem) chromeItem.click();
    if (firefoxItem) firefoxItem.click();

    await el.updateComplete;

    expect(el.supportedBrowsers).to.deep.equal(['chrome', 'firefox']);
  });
});
