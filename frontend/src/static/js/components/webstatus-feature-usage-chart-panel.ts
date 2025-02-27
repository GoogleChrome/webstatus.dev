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
import {WebstatusLineChartPanel} from './webstatus-line-chart-panel.js';
import {Task} from '@lit/task';
import {TemplateResult, html, nothing} from 'lit';
import {
  BROWSER_ID_TO_COLOR,
  BrowsersParameter,
  ChromiumUsageStat,
} from '../api/client.js';

@customElement('webstatus-feature-usage-chart-panel')
export class WebstatusFeatureUsageChartPanel extends WebstatusLineChartPanel {
  @property({type: String})
  featureId!: string;
  createLoadingTask(): Task {
    return new Task(this, {
      args: () =>
        [this.dataFetchStartDate, this.dataFetchEndDate, this.featureId] as [
          Date,
          Date,
          string,
        ],
      task: async ([startDate, endDate, featureId]: [Date, Date, string]) => {
        if (
          featureId === undefined ||
          startDate === undefined ||
          endDate === undefined
        )
          return;
        await this._fetchAndAggregateData<ChromiumUsageStat>([
          {
            label: 'Chrome',
            fetchFunction: () =>
              this.apiClient.getChromiumDailyUsageStats(
                featureId,
                startDate,
                endDate,
              ),
            timestampExtractor: (dataPoint: ChromiumUsageStat): Date =>
              new Date(dataPoint.timestamp),
            valueExtractor: (dataPoint: ChromiumUsageStat): number =>
              dataPoint.usage ? dataPoint.usage * 100 : 0,
            tooltipExtractor: (dataPoint: ChromiumUsageStat): string =>
              `Chrome: ${dataPoint.usage ? dataPoint.usage * 100 : 0}%`,
          },
        ]);
      },
    });
  }
  getPanelID(): string {
    return 'feature-usage';
  }
  getPanelText(): string {
    return 'Feature Usage';
  }
  renderControls(): TemplateResult {
    return html`${nothing}`;
  }
  readonly featureSupportBrowsers: BrowsersParameter[] = ['chrome'];
  getDisplayDataChartOptionsInput(): {
    seriesColors: string[];
    vAxisTitle: string;
  } {
    // Compute seriesColors from selected browsers and BROWSER_ID_TO_COLOR
    const seriesColors = this.featureSupportBrowsers.map(browser => {
      const browserKey = browser as keyof typeof BROWSER_ID_TO_COLOR;
      return BROWSER_ID_TO_COLOR[browserKey];
    });

    return {
      seriesColors: seriesColors,
      vAxisTitle: 'Usage (%)',
    };
  }

  // TODO: Setting this true to keep the current look during the migration to WebstatusLineChartPanel.
  // We should allow users of the panel to override the disiplay to show dots without using this.
  override hasMax = true;
}
