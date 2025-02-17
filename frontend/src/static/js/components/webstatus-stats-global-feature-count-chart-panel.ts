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
  FetchFunctionConfig,
  WebstatusLineChartPanel,
} from './webstatus-line-chart-panel.js';
import {
  BrowserReleaseFeatureMetric,
  BaselineStatusMetric,
  ALL_BROWSERS,
  BrowsersParameter,
  BROWSER_ID_TO_COLOR,
  BROWSER_ID_TO_LABEL,
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

  private _createFetchFunctionConfigs(
    startDate: Date,
    endDate: Date,
  ): FetchFunctionConfig<BrowserReleaseFeatureMetric>[] {
    return ALL_BROWSERS.map(browser => ({
      label: BROWSER_ID_TO_LABEL[browser],
      fetchFunction: () =>
        this.apiClient.getFeatureCountsForBrowser(browser, startDate, endDate),
      timestampExtractor: (dataPoint: BrowserReleaseFeatureMetric) =>
        new Date(dataPoint.timestamp),
      valueExtractor: (dataPoint: BrowserReleaseFeatureMetric) =>
        dataPoint.count ?? 0,
    }));
  }

  createLoadingTask(): Task {
    return new Task(this, {
      args: () => [this.startDate, this.endDate] as [Date, Date],
      task: async ([startDate, endDate]: [Date, Date]) => {
        await this._fetchAndAggregateData([
          ...this._createFetchFunctionConfigs(startDate, endDate),
          {
            // Additional fetch function config for the "Total" series
            label: 'Total number of Baseline features',
            fetchFunction: () =>
              this.apiClient.listAggregatedBaselineStatusCounts(
                startDate,
                endDate,
              ),
            timestampExtractor: (dataPoint: BaselineStatusMetric) =>
              new Date(dataPoint.timestamp),
            valueExtractor: (dataPoint: BaselineStatusMetric) =>
              dataPoint.count ?? 0,
          },
        ]);
      },
    });
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
