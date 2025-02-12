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
  BaselineStatusMetric,
  ALL_BROWSERS,
  BrowsersParameter,
  BROWSER_ID_TO_COLOR,
} from '../api/client.js';
import {customElement, state} from 'lit/decorators.js';

@customElement('webstatus-stats-global-feature-chart-panel')
export class WebstatusStatsGlobalFeatureCountChartPanel extends WebstatusLineChartPanel {
  getDisplayDataChartOptionsInput(): {
    seriesColors: string[];
    vAxisTitle: string;
  } {
    // Compute seriesColors from selected browsers and BROWSER_ID_TO_COLOR
    const selectedBrowsers = this.supportedBrowsers;
    const seriesColors = [...selectedBrowsers, 'total'].map(browser => {
      const browserKey = browser as keyof typeof BROWSER_ID_TO_COLOR;
      return BROWSER_ID_TO_COLOR[browserKey];
    });

    return {
      seriesColors: seriesColors,
      vAxisTitle: 'Number of features supported',
    };
  }
  @state()
  supportedBrowsers: BrowsersParameter[] = ALL_BROWSERS;

  createLoadingTask(): Task {
    return new Task(this, {
      args: () => [this.apiClient, this.startDate, this.endDate] as const,
      task: async ([apiClient, startDate, endDate]) => {
        await this._fetchGlobalFeatureSupportData(
          apiClient,
          startDate,
          endDate,
        );
        return;
      },
    });
  }

  async _fetchGlobalFeatureSupportData(
    apiClient: APIClient,
    startDate: Date,
    endDate: Date,
  ) {
    if (typeof apiClient !== 'object') return;

    const browserMetricData: Array<
      LineChartMetricData<BrowserReleaseFeatureMetric>
    > = ALL_BROWSERS.map(browser => ({
      label: browser,
      data: [],
      getTimestamp: (dataPoint: BrowserReleaseFeatureMetric) =>
        new Date(dataPoint.timestamp),
      getValue: (dataPoint: BrowserReleaseFeatureMetric) => dataPoint.count,
    }));

    const maxMetricData: LineChartMetricData<BaselineStatusMetric> = {
      label: 'Total number of Baseline features',
      data: [],
      getTimestamp: (dataPoint: BaselineStatusMetric) =>
        new Date(dataPoint.timestamp),
      getValue: (dataPoint: BaselineStatusMetric) => dataPoint.count,
    };

    const allMetricData = [...browserMetricData, maxMetricData];
    const browserPromises = ALL_BROWSERS.map(async browser => {
      const browserData = browserMetricData.find(
        data => data.label === browser,
      );
      if (!browserData) return;

      for await (const page of apiClient.getFeatureCountsForBrowser(
        browser,
        startDate,
        endDate,
      )) {
        browserData.data.push(...page);
      }
    });

    const maxPromise = (async () => {
      for await (const page of apiClient.listAggregatedBaselineStatusCounts(
        startDate,
        endDate,
      )) {
        maxMetricData.data.push(...page);
      }
    })();

    await Promise.all([...browserPromises, maxPromise]);

    this.setDisplayDataFromMap(allMetricData);
  }
  getPanelID(): string {
    return 'global-feature-support';
  }
  getPanelText(): string {
    return 'Global feature support';
  }
  renderControls(): TemplateResult {
    return html`${nothing}`;
  }
}
