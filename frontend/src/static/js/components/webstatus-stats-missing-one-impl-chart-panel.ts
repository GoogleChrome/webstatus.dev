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
  BrowsersParameter,
  BROWSER_ID_TO_COLOR,
  BROWSER_ID_TO_LABEL,
} from '../api/client.js';
import {customElement, state} from 'lit/decorators.js';

@customElement('webstatus-stats-missing-one-impl-chart-panel')
export class WebstatusStatsMissingOneImplChartPanel extends WebstatusLineChartPanel {
  @state()
  supportedBrowsers: BrowsersParameter[] = ['chrome', 'firefox', 'safari'];

  private _createFetchFunctionConfigs(
    browsers: BrowsersParameter[],
    startDate: Date,
    endDate: Date,
  ): FetchFunctionConfig<BrowserReleaseFeatureMetric>[] {
    return browsers.map(browser => ({
      label: browser === 'chrome' ? 'Chromium' : BROWSER_ID_TO_LABEL[browser], // Special case for Chrome
      fetchFunction: () => {
        const otherBrowsers = browsers.filter(value => browser !== value);
        return this.apiClient.getMissingOneImplementationCountsForBrowser(
          browser,
          otherBrowsers,
          startDate,
          endDate,
        );
      },
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
        await this._fetchAndAggregateData(
          this._createFetchFunctionConfigs(
            this.supportedBrowsers,
            startDate,
            endDate,
          ),
        );
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

  getPanelID(): string {
    return 'missing-one-implementation';
  }
  getPanelText(): string {
    return 'Features missing in only one browser';
  }
  renderControls(): TemplateResult {
    return html`${nothing}`;
  }
}
