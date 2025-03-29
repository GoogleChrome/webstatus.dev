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
  ChartDataPoint,
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
  // Worst case there are 470 days between releases for Edge
  // https://github.com/mdn/browser-compat-data/blob/92d6876b420b0e6e69eb61256ed04827c9889063/browsers/edge.json#L53-L66
  // Set offset to -500 days.
  override dataFetchStartDateOffsetMsec: number = -500 * 24 * 60 * 60 * 1000;
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
  ): FetchFunctionConfig<ChartDataPoint, BrowserReleaseFeatureMetric>[] {
    return ALL_BROWSERS.map(browser => ({
      label: BROWSER_ID_TO_LABEL[browser],
      fetchFunction: () =>
        this.apiClient.getFeatureCountsForBrowser(browser, startDate, endDate),
      toChartDataPoint: (
        dataPoint: BrowserReleaseFeatureMetric,
      ): ChartDataPoint => {
        return {
          value: dataPoint.count ?? 0,
          timestamp: new Date(dataPoint.timestamp),
        };
      },
    }));
  }

  createLoadingTask(): Task {
    return new Task(this, {
      args: () =>
        [this.dataFetchStartDate, this.dataFetchEndDate] as [Date, Date],
      task: async ([startDate, endDate]: [Date, Date]) => {
        const baselineConfig: FetchFunctionConfig<
          ChartDataPoint,
          BaselineStatusMetric
        > = {
          label: 'Total number of Baseline features',
          fetchFunction: () =>
            this.apiClient.listAggregatedBaselineStatusCounts(
              startDate,
              endDate,
            ),
          toChartDataPoint: (
            dataPoint: BaselineStatusMetric,
          ): ChartDataPoint => {
            return {
              value: dataPoint.count ?? 0,
              timestamp: new Date(dataPoint.timestamp),
            };
          },
        };
        await this._fetchAndAggregateData<ChartDataPoint, unknown>([
          ...this._createFetchFunctionConfigs(startDate, endDate),
          baselineConfig,
        ]);
        // await this._fetchAndAggregateData<ChartDataPoint, BrowserReleaseFeatureMetric|BaselineStatusMetric>([
        //   ...this._createFetchFunctionConfigs(startDate, endDate),
        //   {
        //     // Additional fetch function config for the "Total" series
        //     label: 'Total number of Baseline features',
        //     fetchFunction: () =>
        //       this.apiClient.listAggregatedBaselineStatusCounts(
        //         startDate,
        //         endDate,
        //       ),
        //     toChartDataPoint: (dataPoint: BaselineStatusMetric):ChartDataPoint => {
        //       return {
        //         value: dataPoint.count ?? 0,
        //         timestamp: new Date(dataPoint.timestamp),
        //       };
        //     },
        //   },
        // ]);
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
