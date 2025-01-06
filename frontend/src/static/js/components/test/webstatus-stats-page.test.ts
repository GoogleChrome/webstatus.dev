/**
 * Copyright 2024 Google LLC
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

import {expect, fixture, html} from '@open-wc/testing';
import sinon from 'sinon';
import {StatsPage} from '../webstatus-stats-page.js';
import '../webstatus-stats-page.js';
import {
  APIClient,
  BrowserReleaseFeatureMetric,
  BrowsersParameter,
} from '../../api/client.js';
import {WebstatusGChart} from '../webstatus-gchart.js';
import {TaskStatus} from '@lit/task';

function createMockIterator<T>(data: T[]) {
  return {
    [Symbol.asyncIterator]: () => ({
      next: async (): Promise<IteratorResult<T[]>> => {
        const value = data.shift();
        if (value) {
          return {
            value: [value],
            done: false,
          };
        } else {
          return {
            value: undefined,
            done: true,
          };
        }
      },
    }),
  };
}

describe('StatsPage', () => {
  let element: StatsPage;
  let apiClientStub: sinon.SinonStubbedInstance<APIClient>;
  const mockMissingOneCountData = new Map<
    BrowsersParameter,
    BrowserReleaseFeatureMetric[]
  >([
    [
      'chrome',
      [
        {timestamp: '2024-01-01T00:00:00.000Z', count: 10},
        {timestamp: '2024-01-02T00:00:00.000Z', count: 12},
      ],
    ],
    [
      'edge',
      [
        {timestamp: '2024-01-01T00:00:00.000Z', count: 8},
        {timestamp: '2024-01-02T00:00:00.000Z', count: 11},
      ],
    ],
    [
      'firefox',
      [
        {timestamp: '2024-01-01T00:00:00.000Z', count: 9},
        {timestamp: '2024-01-02T00:00:00.000Z', count: 10},
      ],
    ],
    [
      'safari',
      [
        {timestamp: '2024-01-01T00:00:00.000Z', count: 7},
        {timestamp: '2024-01-02T00:00:00.000Z', count: 13},
      ],
    ],
  ]);

  beforeEach(async () => {
    apiClientStub = sinon.createStubInstance<APIClient>(APIClient, {
      getMissingOneImplementationCountsForBrowser: sinon.stub(),
    });
  });

  afterEach(() => {
    sinon.restore();
  });

  describe('_fetchMissingOneImplemenationCounts', () => {
    beforeEach(async () => {
      element = await fixture(
        html`<webstatus-stats-page
          .apiClient=${apiClientStub}
          .location=${{search: ''}}
        ></webstatus-stats-page>`,
      );
    });
    it('should fetch and process data correctly', async () => {
      apiClientStub.getMissingOneImplementationCountsForBrowser.callsFake(
        browser => {
          const data = mockMissingOneCountData.get(browser)?.slice();
          return createMockIterator(data!);
        },
      );

      const startDate = new Date('2024-01-01T00:00:00.000Z');
      const endDate = new Date('2024-01-03T00:00:00.000Z');

      // Call _fetchMissingOneImplemenationCounts for each browser
      await element._fetchMissingOneImplemenationCounts(
        apiClientStub,
        startDate,
        endDate,
      );

      // Assertions for each browser
      expect(element.missingOneImplementationMap.get('chrome')).to.deep.equal(
        mockMissingOneCountData.get('chrome'),
      );
      expect(element.missingOneImplementationMap.get('edge')).to.deep.equal(
        mockMissingOneCountData.get('edge'),
      );
      expect(element.missingOneImplementationMap.get('firefox')).to.deep.equal(
        mockMissingOneCountData.get('firefox'),
      );
      expect(element.missingOneImplementationMap.get('safari')).to.deep.equal(
        mockMissingOneCountData.get('safari'),
      );
    });
  });

  describe('renderMissingOneImplementationChart', () => {
    let fetchMissingOneImplementationCountsStub: sinon.SinonStub;

    afterEach(async () => {
      fetchMissingOneImplementationCountsStub.restore();
    });
    it('should render the chart when the task is complete', async () => {
      // Successful fetch
      fetchMissingOneImplementationCountsStub = sinon
        .stub(StatsPage.prototype, '_fetchMissingOneImplemenationCounts')
        .resolves();
      element = await fixture(
        html`<webstatus-stats-page
          .apiClient=${apiClientStub}
          .location=${{search: '?show_features_lagging=true'}}
        ></webstatus-stats-page>`,
      );

      // Directly set the missingOneImplementationMap with mock data
      element.missingOneImplementationMap = mockMissingOneCountData;

      // Wait for the task to complete
      await element._loadingMissingOneTask.value;
      await element.updateComplete;

      const chartElement = element.shadowRoot!.querySelector<WebstatusGChart>(
        '#missing-one-implementation-chart',
      );
      expect(chartElement).to.exist;
      expect(chartElement!.dataObj).to.deep.equal(
        element.missingOneImplementationChartDataObj,
      );
      expect(chartElement!.options).to.deep.equal(
        element.generatedisplayDataChartOptions('Number of features missing'),
      );
      expect(element._loadingMissingOneTask.status).to.equal(
        TaskStatus.COMPLETE,
      );
    });

    it('should render loading indicators while pending', async () => {
      // Create a deferred promise that will be resolved later
      let resolvePending: () => void;
      const pendingPromise = new Promise<void>(resolve => {
        resolvePending = resolve;
      });

      // Stub the method to return the pending promise
      fetchMissingOneImplementationCountsStub = sinon
        .stub(StatsPage.prototype, '_fetchMissingOneImplemenationCounts')
        .returns(pendingPromise);
      element = await fixture(
        html`<webstatus-stats-page
          .apiClient=${apiClientStub}
          .location=${{search: '?show_features_lagging=true'}}
        ></webstatus-stats-page>`,
      );
      expect(element._loadingMissingOneTask.status).to.equal(
        TaskStatus.PENDING,
      );

      await element.updateComplete;

      const chartElement = element.shadowRoot!.querySelector(
        '#missing-one-implementation-chart',
      );
      expect(chartElement).to.not.exist;
      const loadingIndicator = element.shadowRoot!.querySelector(
        '#missing-one-implementation-pending',
      );
      expect(loadingIndicator).to.exist;
      expect(loadingIndicator?.textContent).to.contain('Loading stats.');

      resolvePending!();
    });

    it('should render an error message when the task fails', async () => {
      // Fetch fails
      fetchMissingOneImplementationCountsStub = sinon
        .stub(StatsPage.prototype, '_fetchMissingOneImplemenationCounts')
        .throws(new Error('API error'));
      element = await fixture(
        html`<webstatus-stats-page
          .apiClient=${apiClientStub}
          .location=${{search: '?show_features_lagging=true'}}
        ></webstatus-stats-page>`,
      );

      await element.updateComplete;

      const chartElement = element.shadowRoot!.querySelector(
        '#missing-one-implementation-chart',
      );
      expect(chartElement).to.not.exist;
      const errorMessageElement = element.shadowRoot!.querySelector(
        '#missing-one-implementation-error',
      );
      expect(errorMessageElement?.textContent).to.contain(
        'Error when loading stats.',
      );
      expect(element._loadingMissingOneTask.status).to.equal(TaskStatus.ERROR);
    });
  });
});
