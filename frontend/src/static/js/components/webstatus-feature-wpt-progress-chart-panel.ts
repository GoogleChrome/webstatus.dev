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

import {customElement, property} from 'lit/decorators.js';
import {Task} from '@lit/task';
import {TemplateResult, html, nothing} from 'lit';
import {
  ALL_BROWSERS,
  BROWSER_ID_TO_COLOR,
  BROWSER_ID_TO_LABEL,
  BrowsersParameter,
  ChannelsParameter,
  STABLE_CHANNEL,
  WPTRunMetric,
} from '../api/client.js';
import {
  FetchFunctionConfig,
  WebstatusLineChartPanel,
} from './webstatus-line-chart-panel.js';

@customElement('webstatus-feature-wpt-progress-chart-panel')
export class WebstatusFeatureWPTProgressChartPanel extends WebstatusLineChartPanel {
  readonly featureSupportBrowsers: BrowsersParameter[] = ALL_BROWSERS;
  readonly featureSupportChannel: ChannelsParameter = STABLE_CHANNEL;
  readonly testView: 'subtest_counts' | 'test_counts' = 'subtest_counts';

  readonly testViewToString: Record<'subtest_counts' | 'test_counts', string> =
    {
      subtest_counts: 'subtests',
      test_counts: 'tests',
    };

  @property({type: String})
  featureId!: string;

  private _createFetchFunctionConfigs(
    startDate: Date,
    endDate: Date,
    featureId: string,
  ): FetchFunctionConfig<WPTRunMetric>[] {
    return this.featureSupportBrowsers.map(browser => ({
      label: BROWSER_ID_TO_LABEL[browser],
      fetchFunction: () =>
        this.apiClient.getFeatureStatsByBrowserAndChannel(
          featureId,
          browser,
          this.featureSupportChannel,
          startDate,
          endDate,
          this.testView,
        ),
      timestampExtractor: (dataPoint: WPTRunMetric): Date =>
        new Date(dataPoint.run_timestamp),
      valueExtractor: (dataPoint: WPTRunMetric): number =>
        dataPoint.test_pass_count || 0,
      tooltipExtractor: (dataPoint: WPTRunMetric): string =>
        `${BROWSER_ID_TO_LABEL[browser]}: ${dataPoint.test_pass_count!} of ${dataPoint.total_tests_count!}`,
    }));
  }

  createLoadingTask(): Task {
    return new Task(this, {
      args: () =>
        [this.startDate, this.endDate, this.featureId] as [Date, Date, string],
      task: async ([startDate, endDate, featureId]: [Date, Date, string]) => {
        if (
          featureId === undefined ||
          startDate === undefined ||
          endDate === undefined
        )
          return;
        await this._fetchAndAggregateData<WPTRunMetric>(
          this._createFetchFunctionConfigs(startDate, endDate, featureId),
          [
            // This additional series configuration calculates the "Total" series
            // by using the calculateMax method to find the maximum total_tests_count
            // across all browsers for each timestamp.
            {
              label: `Total number of ${this.testViewToString[this.testView]}`,
              calculator: this.calculateMax,
              cacheMap: new Map<string, WPTRunMetric>(),
              timestampExtractor: (dataPoint: WPTRunMetric): Date =>
                new Date(dataPoint.run_timestamp),
              valueExtractor: (dataPoint: WPTRunMetric): number =>
                dataPoint.total_tests_count || 0,
            },
          ],
        );
      },
    });
  }
  getPanelID(): string {
    return 'feature-wpt-implementation-progress';
  }
  getPanelText(): string {
    return 'Implementation progress';
  }
  renderControls(): TemplateResult {
    return html`${nothing}`;
  }

  override readonly hasMax: boolean = true;
  getDisplayDataChartOptionsInput(): {
    seriesColors: string[];
    vAxisTitle: string;
  } {
    // Compute seriesColors from selected browsers and BROWSER_ID_TO_COLOR
    const seriesColors = [...this.featureSupportBrowsers, 'total'].map(
      browser => {
        const browserKey = browser as keyof typeof BROWSER_ID_TO_COLOR;
        return BROWSER_ID_TO_COLOR[browserKey];
      },
    );

    return {
      seriesColors: seriesColors,
      vAxisTitle: `Number of ${this.testViewToString[this.testView]} passed`,
    };
  }
}
