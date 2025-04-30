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
import {
  FetchFunctionConfig,
  WebstatusLineChartPanel,
} from './webstatus-line-chart-panel.js';
import {Task} from '@lit/task';
import {TemplateResult, html, nothing} from 'lit';
import {
  BROWSER_ID_TO_COLOR,
  BROWSER_ID_TO_LABEL,
  BrowsersParameter,
  ChromeUsageStat,
} from '../api/client.js';

@customElement('webstatus-feature-usage-chart-panel')
export class WebstatusFeatureUsageChartPanel extends WebstatusLineChartPanel {
  @property({type: String})
  featureId!: string;

  browsers: BrowsersParameter[] = ['chrome'];

  private roundUsagePercentage(usage: number | undefined): number {
    if (usage === undefined) {
      return 0.0;
    }
    const percentage = usage * 100;
    if (percentage >= 100) {
      return 100;
    }
    // Round to one decimal place.
    return Math.round(percentage * 10) / 10;
  }

  private formatPercentageForDisplay(percentage: number): string {
    if (percentage === 100) {
      return '100';
    }
    return percentage.toFixed(1);
  }

  private _createFetchFunctionConfigs(
    featureId: string,
    startDate: Date,
    endDate: Date,
  ): FetchFunctionConfig<ChromeUsageStat>[] {
    return this.browsers.map(browser => ({
      label: BROWSER_ID_TO_LABEL[browser],
      fetchFunction: () =>
        this.apiClient.getChromeDailyUsageStats(featureId, startDate, endDate),
      timestampExtractor: (dataPoint: ChromeUsageStat): Date =>
        new Date(dataPoint.timestamp),
      valueExtractor: (dataPoint: ChromeUsageStat): number =>
        this.roundUsagePercentage(dataPoint.usage),
      tooltipExtractor: (dataPoint: ChromeUsageStat): string => {
        const percentage = this.roundUsagePercentage(dataPoint.usage);
        return `${BROWSER_ID_TO_LABEL[browser]}: ${this.formatPercentageForDisplay(percentage)}%`;
      },
    }));
  }

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
        this._populateDataForChart<ChromeUsageStat>(
          this._createFetchFunctionConfigs(featureId, startDate, endDate),
        );
      },
    });
  }
  getPanelID(): string {
    return 'feature-usage';
  }

  getPanelText(): string {
    return 'Feature Usage';
  }

  getPanelDescription(): TemplateResult {
    return html`This chart displays the percentage of page loads that include
    this feature in participating Chrome installs.`;
  }

  renderControls(): TemplateResult {
    return html`${nothing}`;
  }

  getDisplayDataChartOptionsInput(): {
    seriesColors: string[];
    vAxisTitle: string;
  } {
    // Compute seriesColors from selected browsers and BROWSER_ID_TO_COLOR
    const seriesColors = this.browsers.map(browser => {
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
