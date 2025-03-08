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
import {TemplateResult, html, nothing, css} from 'lit';
import {
  FetchFunctionConfig,
  WebstatusLineChartPanel,
} from './webstatus-line-chart-panel.js';
import {
  BrowserReleaseFeatureMetric,
  BrowsersParameter,
  BROWSER_ID_TO_COLOR,
  BROWSER_ID_TO_LABEL,
  MissingOneImplFeaturesList,
} from '../api/client.js';
import {ChartSelectPointEvent} from './webstatus-gchart.js';
import {customElement, state} from 'lit/decorators.js';

@customElement('webstatus-stats-missing-one-impl-chart-panel')
export class WebstatusStatsMissingOneImplChartPanel extends WebstatusLineChartPanel {
  @state()
  supportedBrowsers: BrowsersParameter[] = ['chrome', 'firefox', 'safari'];

  missingFeaturesList: MissingOneImplFeaturesList = [];
  selectedBrowser: string = '';
  selectedDate: string = '';

  static get styles() {
    return [
      super.styles,
      css`
        #missing-one-implementation-datapoint-details-complete {
          display: block;
        }
        .missing-features-table {
          width: 100%;
          overflow-x: auto;
          white-space: nowrap;
        }
      `,
    ];
  }

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
      args: () =>
        [this.dataFetchStartDate, this.dataFetchEndDate] as [Date, Date],
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

  /**
   * Creates a task and a renderer for handling point-selected events.
   * Overrides createPointSelectedTask() in the parent class when an point is
   * selected on the chart.
   *
   * @param {ChartSelectPointEvent} _ The point-selected event.
   * @returns {{ task: Task | undefined; renderSuccess?: () => TemplateResult; }}
   */
  createPointSelectedTask(_: ChartSelectPointEvent): {
    task: Task | undefined;
    renderSuccess?: () => TemplateResult;
  } {
    const task = new Task(this, {
      task: async () => {
        // TODO(https://github.com/GoogleChrome/webstatus.dev/issues/1181):
        // implement the adapter logic to retrieve feature IDs.
        const pageData = {
          data: [
            {
              feature_id: 'css',
            },
            {
              feature_id: 'html',
            },
            {
              feature_id: 'javascript',
            },
            {
              feature_id: 'bluetooth',
            },
          ],
          metadata: {
            total: 4,
          },
        };
        for (let i = 0; i < 80; i++) {
          pageData.data.push({
            feature_id: 'item' + i,
          });
        }
        this.missingFeaturesList = pageData.data;
        // TODO:(kyleju) return these data from the API.
        this.selectedDate = '2024-08-20';
        this.selectedBrowser = 'chrome';
        return this.missingFeaturesList;
      },
      args: () => [],
    });
    return {task: task, renderSuccess: this.pointSelectedTaskRenderOnSuccess};
  }

  /**
   * Renders the success state of the createPointSelectedTask above.
   * This method implements the _renderCustomPointSelectedSuccess
   * in the parent class.
   *
   * @returns {TemplateResult} The rendered content for the success state.
   */
  pointSelectedTaskRenderOnSuccess(): TemplateResult {
    return html`
      <div slot="header" id="${this.getPanelID()}-list-header">
        The missing feature IDs on ${this.selectedDate} for
        ${this.selectedBrowser}:
      </div>
      ${this.renderMissingFeaturesTable()}
    `;
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

  renderMissingFeaturesTable(): TemplateResult {
    const numCols = Math.ceil(this.missingFeaturesList.length / 10);

    // Create table body with `numCols` columns and 10 rows each.
    const bodyRows = [];
    for (let i = 0; i < 10; i++) {
      const cells = [];
      for (let j = 0; j < numCols; j++) {
        const featureIndex = j * 10 + i;
        if (featureIndex < this.missingFeaturesList.length) {
          const feature_id = this.missingFeaturesList[featureIndex].feature_id;
          cells.push(
            html` <td>
              <a href="/features/${feature_id}">${feature_id}</a>
            </td>`,
          );
        } else {
          // Empty cell.
          cells.push(html`<td></td>`);
        }
      }

      bodyRows.push(
        html`<tr>
          ${cells}
        </tr>`,
      );
    }

    return html`
      <table class="missing-features-table">
        <tbody>
          ${bodyRows}
        </tbody>
      </table>
    `;
  }
}
