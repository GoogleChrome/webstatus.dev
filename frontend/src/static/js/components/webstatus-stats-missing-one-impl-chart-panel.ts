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
  BROWSER_LABEL_TO_ID,
  MissingOneImplFeaturesList,
  BROWSER_ID_TO_LABEL,
} from '../api/client.js';
import {ChartSelectPointEvent} from './webstatus-gchart.js';
import {customElement, state} from 'lit/decorators.js';
import {formatOverviewPageUrl} from '../utils/urls.js';
import {
  getTopCssIdentifierTemplate,
  getTopHtmlIdentifierTemplate,
} from './utils.js';

@customElement('webstatus-stats-missing-one-impl-chart-panel')
export class WebstatusStatsMissingOneImplChartPanel extends WebstatusLineChartPanel {
  @state()
  browsers: BrowsersParameter[] = ['chrome', 'firefox', 'safari'];

  missingFeaturesList: MissingOneImplFeaturesList = [];
  selectedBrowser: string = '';
  selectedDate: string = '';
  featureListHref: string = '';

  static get styles() {
    return [
      super.styles,
      css`
        #missing-one-implementation-datapoint-details-complete {
          display: block;
        }
        #missing-one-implementation-list-header {
          margin-bottom: 1em;
          font-size: large;
        }
        .missing-features-table {
          width: 100%;
          overflow-x: auto;
          white-space: nowrap;
        }
        .missing-feature-id {
          padding: 0.5em 1em 0 0;
        }
        .survey-result,
        .survey-result:hover,
        .survey-result a {
          font-size: 10px;
          text-decoration: none;
          cursor: help;
        }
      `,
    ];
  }

  getOtherBrowsersFromTargetBrowser(
    browser: BrowsersParameter,
  ): BrowsersParameter[] {
    return this.browsers.filter(value => browser !== value);
  }

  private _createFetchFunctionConfigs(
    browser: BrowsersParameter,
    startDate: Date,
    endDate: Date,
  ): FetchFunctionConfig<BrowserReleaseFeatureMetric>[] {
    return browsers.map(browser => {
      const label =
        browser === 'chrome' ? 'Chrome/Edge' : BROWSER_ID_TO_LABEL[browser];
      return {
        label,
        fetchFunction: () => {
          const otherBrowsers = this.getOtherBrowsersFromTargetBrowser(browser);
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
      };
    });
  }

  createLoadingTask(): Task {
    return new Task(this, {
      args: () =>
        [this.dataFetchStartDate, this.dataFetchEndDate] as [Date, Date],
      task: async ([startDate, endDate]: [Date, Date]) => {
        const fetchFunctionConfigs = this.browsers.map(browser =>
          this._createFetchFunctionConfigs(browser, startDate, endDate),
        );
        return this._populateDataForChart(fetchFunctionConfigs);
      },
    });
  }

  getDisplayDataChartOptionsInput(): {
    seriesColors: string[];
    vAxisTitle: string;
  } {
    // Compute seriesColors from selected browsers and BROWSER_ID_TO_COLOR
    const selectedBrowsers = this.browsers;
    const seriesColors = [...selectedBrowsers].map(browser => {
      const browserKey = browser as keyof typeof BROWSER_ID_TO_COLOR;
      return BROWSER_ID_TO_COLOR[browserKey];
    });

    return {
      seriesColors: seriesColors,
      vAxisTitle: 'Number of features missing',
    };
  }

  updateFeatureListHref(featureList: MissingOneImplFeaturesList) {
    if (featureList.length === 0) {
      this.featureListHref = '';
      return;
    }
    let query = 'id:' + featureList[0].feature_id;
    for (let i = 1; i < featureList.length; i++) {
      query += ' OR id:' + featureList[i].feature_id;
    }

    this.featureListHref = formatOverviewPageUrl({search: ''}, {q: query});
  }

  /**
   * Creates a task and a renderer for handling point-selected events.
   * Overrides createPointSelectedTask() in the parent class when an point is
   * selected on the chart.
   *
   * @param {ChartSelectPointEvent} _ The point-selected event.
   * @returns {{ task: Task | undefined; renderSuccess?: () => TemplateResult; }}
   */
  createPointSelectedTask(event: ChartSelectPointEvent): {
    task: Task | undefined;
    renderSuccess?: () => TemplateResult;
  } {
    const targetDate = event.detail.timestamp;
    const label = event.detail.label;
    const targetBrowser = BROWSER_LABEL_TO_ID[label];
    const otherBrowsers = this.getOtherBrowsersFromTargetBrowser(targetBrowser);
    const task = new Task(this, {
      task: async ([date, browser]) => {
        const features =
          await this.apiClient.getMissingOneImplementationFeatures(
            browser,
            otherBrowsers,
            date,
          );
        this.missingFeaturesList = features;
        this.selectedDate = targetDate.toISOString().substring(0, 10);
        this.selectedBrowser = label;
        this.updateFeatureListHref(features);
        return features;
      },
      args: () => [targetDate, targetBrowser],
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
    if (this.missingFeaturesList.length === 0) {
      return html`<div slot="header" id="${this.getPanelID()}-list-header">
        No missing features for on ${this.selectedDate} for
        ${this.selectedBrowser}
      </div> `;
    }

    return html`
      <div slot="header" id="${this.getPanelID()}-list-header">
        Missing features on ${this.selectedDate} for ${this.selectedBrowser}:
        <a href="${this.featureListHref}"
          >${this.missingFeaturesList.length} features</a
        >
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

  getPanelDescription(): TemplateResult {
    return html`This chart displays the number of features that are missing in
      exactly one major browser, for each browser. The counted features are
      present in all browsers except that browser. Each of the features would be
      considered Baseline once the feature is supported in the corresponding
      browser. <br />Click on a line in the chart to see the exact number at any
      given time and the list of specific features missing in that browser. `;
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
          const featureId = this.missingFeaturesList[featureIndex].feature_id;
          const extraIdentifiers: TemplateResult[] = [];
          const cssIdentifier = getTopCssIdentifierTemplate(featureId);
          if (cssIdentifier) {
            extraIdentifiers.push(cssIdentifier);
          }
          const htmlIdentifier = getTopHtmlIdentifierTemplate(featureId);
          if (htmlIdentifier) {
            extraIdentifiers.push(htmlIdentifier);
          }
          cells.push(
            html` <td>
              <div class="missing-feature-id">
                <a href="/features/${featureId}">${featureId}</a>
                ${extraIdentifiers}
              </div>
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
