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

import {Task} from '@lit/task';
import {TemplateResult, html, nothing} from 'lit';
import {
  LineChartMetricData,
  WebstatusLineChartPanel,
} from './webstatus-line-chart-panel.js';
import {
  BrowserReleaseFeatureMetric,
  type APIClient,
  BrowsersParameter,
  BROWSER_ID_TO_COLOR,
  BROWSER_ID_TO_LABEL,
} from '../api/client.js';
import {customElement, state} from 'lit/decorators.js';

@customElement('webstatus-stats-missing-one-impl-chart-panel')
export class WebstatusStatsMissingOneImplChartPanel extends WebstatusLineChartPanel {
  @state()
  supportedBrowsers: BrowsersParameter[] = ['chrome', 'firefox', 'safari'];

  createLoadingTask(): Task {
    return new Task(this, {
      args: () =>
        [this.apiClient, this.startDate, this.endDate] as [
          APIClient,
          Date,
          Date,
        ],
      task: async ([apiClient, startDate, endDate]: [
        APIClient,
        Date,
        Date,
      ]) => {
        await this._fetchMissingOneImplemenationCounts(
          apiClient,
          startDate,
          endDate,
        );
        return;
      },
    });
  }

  getDisplayDataChartOptionsInput(): {
    seriesColors: string[];
    vAxisTitle: string;
  } {
    // Compute seriesColors from selected browsers and BROWSER_ID_TO_COLOR
    const selectedBrowsers = this.supportedBrowsers;
    const seriesColors = [...selectedBrowsers].map(browser => {
      const browserKey = browser as keyof typeof BROWSER_ID_TO_COLOR;
      return BROWSER_ID_TO_COLOR[browserKey];
    });

    return {
      seriesColors: seriesColors,
      vAxisTitle: 'Number of features missing',
    };
  }

  async _fetchMissingOneImplemenationCounts(
    apiClient: APIClient,
    startDate: Date,
    endDate: Date,
  ) {
    if (typeof apiClient !== 'object') return;

    const browserMetricData: Array<
      LineChartMetricData<BrowserReleaseFeatureMetric> & {
        browser: BrowsersParameter;
      }
    > = this.supportedBrowsers.map(browser => ({
      label: browser === 'chrome' ? 'Chromium' : BROWSER_ID_TO_LABEL[browser], // Special case for Chrome
      browser: browser,
      data: [],
      getTimestamp: (dataPoint: BrowserReleaseFeatureMetric) =>
        new Date(dataPoint.timestamp),
      getValue: (dataPoint: BrowserReleaseFeatureMetric) => dataPoint.count,
    }));
    const promises = this.supportedBrowsers.map(async browser => {
      const browserData = browserMetricData.find(
        data => data.browser === browser,
      );
      if (!browserData) return;

      const otherBrowsers = this.supportedBrowsers.filter(
        value => browser !== value,
      );
      for await (const page of apiClient.getMissingOneImplementationCountsForBrowser(
        browser,
        otherBrowsers,
        startDate,
        endDate,
      )) {
        browserData.data.push(...page);
      }
    });
    await Promise.all(promises); // Wait for all browsers to finish

    this.setDisplayDataFromMap(browserMetricData);
  }
  getPanelID(): string {
    return 'missing-one-implementation';
  }
  getPanelText(): string {
    return 'Features missing in only 1 browser';
  }
  renderControls(): TemplateResult {
    return html`${nothing}`;
  }
}
