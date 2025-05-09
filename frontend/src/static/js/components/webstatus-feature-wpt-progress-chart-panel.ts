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
  BROWSER_ID_TO_COLOR,
  BROWSER_ID_TO_LABEL,
  BrowsersParameter,
  ChannelsParameter,
  FeatureWPTMetricViewType,
  STABLE_CHANNEL,
  WPTRunMetric,
} from '../api/client.js';
import {FetchFunctionConfig} from './webstatus-line-chart-panel.js';
import {WebstatusLineChartTabbedPanel} from './webstatus-line-chart-tabbed-panel.js';

@customElement('webstatus-feature-wpt-progress-chart-panel')
export class WebstatusFeatureWPTProgressChartPanel extends WebstatusLineChartTabbedPanel<BrowsersParameter> {
  readonly series: BrowsersParameter[] = [];

  readonly browsersByView: BrowsersParameter[][] = [
    ['chrome', 'firefox', 'safari', 'edge'],
    ['chrome_android', 'firefox_android', 'safari_ios'],
  ];

  readonly tabViews = ['Desktop', 'Mobile'];

  readonly featureSupportChannel: ChannelsParameter = STABLE_CHANNEL;

  @property({type: String})
  testView!: FeatureWPTMetricViewType;

  readonly testViewToString: Record<FeatureWPTMetricViewType, string> = {
    subtest_counts: 'subtests',
    test_counts: 'tests',
  };

  @property({type: String})
  featureId!: string;

  /**
   * Extracts and rounds the timestamp from a WPTRunMetric to the nearest hour.
   * This is necessary because timestamps from different browsers may be slightly
   * different, and rounding them to the nearest hour provides a consistent
   * baseline for comparison.
   *
   * Additionally, rounding addresses inconsistencies in the reported
   * total_tests_count across different browsers for the same timestamp, which
   * can occur due to upstream data issues.
   *
   * @param dataPoint The WPTRunMetric data point.
   * @returns The rounded timestamp as a Date object.
   */
  private _timestampExtractor(dataPoint: WPTRunMetric): Date {
    const timestampMs = new Date(dataPoint.run_timestamp).getTime();
    // Round timestamp to the nearest hour.
    const msInHour = 1000 * 60 * 60 * 1;
    const roundedTimestamp = Math.round(timestampMs / msInHour) * msInHour;
    return new Date(roundedTimestamp);
  }

  private _createFetchFunctionConfigs(
    startDate: Date,
    endDate: Date,
    featureId: string,
    testView: FeatureWPTMetricViewType,
  ): FetchFunctionConfig<WPTRunMetric>[][] {
    return this.browsersByView.map(browsers =>
      browsers.map(browser => ({
        label: BROWSER_ID_TO_LABEL[browser],
        fetchFunction: () =>
          this.apiClient.getFeatureStatsByBrowserAndChannel(
            featureId,
            browser,
            this.featureSupportChannel,
            startDate,
            endDate,
            testView,
          ),
        timestampExtractor: this._timestampExtractor,
        valueExtractor: (dataPoint: WPTRunMetric): number =>
          dataPoint.test_pass_count || 0,
        tooltipExtractor: (dataPoint: WPTRunMetric): string =>
          `${BROWSER_ID_TO_LABEL[browser]}: ${dataPoint.test_pass_count!} of ${dataPoint.total_tests_count!}`,
      })),
    );
  }

  createLoadingTask(): Task {
    return new Task(this, {
      args: () =>
        [
          this.dataFetchStartDate,
          this.dataFetchEndDate,
          this.featureId,
          this.testView,
        ] as [Date, Date, string, FeatureWPTMetricViewType],
      task: async ([startDate, endDate, featureId, testView]: [
        Date,
        Date,
        string,
        FeatureWPTMetricViewType,
      ]) => {
        if (
          featureId === undefined ||
          startDate === undefined ||
          endDate === undefined ||
          testView === undefined
        )
          return;
        const fetchFunctionConfigs = this._createFetchFunctionConfigs(
          startDate,
          endDate,
          featureId,
          testView,
        );
        await Promise.all(
          fetchFunctionConfigs.map((configs, i) =>
            this._populateDataForChartByView(configs, i, [
              // This additional series configuration calculates the "Total" series
              // by using the calculateMax method to find the maximum total_tests_count
              // across all browsers for each timestamp.
              {
                label: `Total number of ${this.testViewToString[testView]}`,
                calculator: this.calculateMax,
                cacheMap: new Map<string, WPTRunMetric>(),
                timestampExtractor: this._timestampExtractor,
                valueExtractor: (dataPoint: WPTRunMetric): number =>
                  dataPoint.total_tests_count || 0,
              },
            ]),
          ),
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

  getPanelDescription(): TemplateResult {
    return html`This chart displays the number of
      <a href="https://web-platform-tests.org/" target="_blank"
        >Web Platform Tests</a
      >
      that are available to measure the support of this feature, as well as the
      pass rates of the feature in each major browser.`;
  }

  getTabTooltip(tab: string): TemplateResult {
    if (tab === 'Mobile') {
      return html` <sl-tooltip
        content="${'Work is underway to enable stable channel mobile test results.'}"
        placement="right"
      >
        ${tab}&nbsp;<sl-icon
          class="icon-button__icon"
          aria-hidden="true"
          name="exclamation-triangle"
          library="default"
        ></sl-icon
      ></sl-tooltip>`;
    }
    return super.getTabTooltip(tab);
  }

  renderControls(): TemplateResult {
    return html`${nothing}`;
  }

  override readonly hasMax: boolean = true;
  getDisplayDataChartOptionsInput<BrowsersParameter>(
    browsers: BrowsersParameter[],
  ): {
    seriesColors: string[];
    vAxisTitle: string;
  } {
    // Compute seriesColors from selected browsers and BROWSER_ID_TO_COLOR
    const seriesColors = [...browsers, 'total'].map(browser => {
      const browserKey = browser as keyof typeof BROWSER_ID_TO_COLOR;
      return BROWSER_ID_TO_COLOR[browserKey];
    });

    return {
      seriesColors: seriesColors,
      vAxisTitle: `Number of ${this.testViewToString[this.testView]} passed`,
    };
  }
}
