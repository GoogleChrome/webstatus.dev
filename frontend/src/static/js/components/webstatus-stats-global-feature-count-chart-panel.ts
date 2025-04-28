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
    const selectedBrowsers = this.browsersByView[this.currentView];
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
  supportedBrowsers: BrowsersParameter[] = ['chrome', 'firefox', 'safari'];

  @state()
  browsersByView: Array<Array<BrowsersParameter>> = [
    ['chrome', 'firefox', 'safari'],
    ['chrome_android', 'firefox_android', 'safari_ios'],
  ];

  @state()
  tabViews: Array<string> = ['Desktop', 'Mobile'];

  private _createFetchFunctionConfigs(
    startDate: Date,
    endDate: Date,
  ): Array<FetchFunctionConfig<BrowserReleaseFeatureMetric>[]> {
    return this.browsersByView.map(browsers =>
      browsers.map(browser => {
        const label =
          browser === 'chrome' ? 'Chrome/Edge' : BROWSER_ID_TO_LABEL[browser];
        return {
          label,
          fetchFunction: () =>
            this.apiClient.getFeatureCountsForBrowser(
              browser,
              startDate,
              endDate,
            ),
          timestampExtractor: (dataPoint: BrowserReleaseFeatureMetric) =>
            new Date(dataPoint.timestamp),
          valueExtractor: (dataPoint: BrowserReleaseFeatureMetric) =>
            dataPoint.count ?? 0,
        };
      }),
    );
  }

  createLoadingTask(): Task {
    return new Task(this, {
      args: () =>
        [this.dataFetchStartDate, this.dataFetchEndDate] as [Date, Date],
      task: async ([startDate, endDate]: [Date, Date]) => {
        const fetchFunctionConfigs = this._createFetchFunctionConfigs(
          startDate,
          endDate,
        );
        const promises = fetchFunctionConfigs.map((configs, i) => {
          return this._fetchAndAggregateData(
            [
              ...configs,
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
            ],
            i,
          );
        });
        await Promise.all(promises);
      },
    });
  }
  getPanelID(): string {
    return 'global-feature-support';
  }

  getPanelText(): string {
    return 'Global feature support';
  }

  getPanelDescription(): TemplateResult {
    return html`This chart displays the number of web features that are
      available in each browser, including the total Baseline features (newly or
      widely available on all major browsers). <br />Click on a line in the
      chart to see the exact number for the current browser version at any given
      time.`;
  }

  renderControls(): TemplateResult {
    return html`${nothing}`;
  }
}
