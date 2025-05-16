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

import {Task, TaskStatus} from '@lit/task';
import {TemplateResult, html, nothing, css} from 'lit';
import {type components} from 'webstatus.dev-backend';
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
import {customElement, property} from 'lit/decorators.js';
import {formatOverviewPageUrl} from '../utils/urls.js';
import {
  ColumnKey,
  parseColumnsSpec,
  renderSavedSearchHeaderCells,
} from './webstatus-overview-cells.js';
import {MISSING_ONE_TABLE_COLUMNS} from '../utils/constants.js';
import {TaskTracker} from '../utils/task-tracker.js';
import {ApiError, UnknownError} from '../api/errors.js';
import {toast} from '../utils/toast.js';

@customElement('webstatus-stats-missing-one-impl-chart-panel')
export class WebstatusStatsMissingOneImplChartPanel extends WebstatusLineChartPanel<BrowsersParameter> {
  readonly series: BrowsersParameter[] = ['chrome', 'firefox', 'safari'];

  @property({type: Object})
  taskTracker: TaskTracker<components['schemas']['Feature'][], ApiError> = {
    status: TaskStatus.INITIAL, // Initial state
    error: undefined,
    data: undefined,
  };

  supportedBrowsers: BrowsersParameter[] = ['chrome', 'firefox', 'safari'];

  @property({type: Boolean})
  isLoadingFeatures = false;

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
        .table-description {
          font-size: 14px;
          font-style: italic;
          margin: 8px 0;
        }
      `,
    ];
  }

  getOtherBrowsersFromTargetBrowser(
    browser: BrowsersParameter,
  ): BrowsersParameter[] {
    return this.series.filter(value => browser !== value);
  }

  private _createFetchFunctionConfigs(
    startDate: Date,
    endDate: Date,
  ): FetchFunctionConfig<BrowserReleaseFeatureMetric>[] {
    return this.series.map(browser => {
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
        const fetchFunctionConfigs = this._createFetchFunctionConfigs(
          startDate,
          endDate,
        );
        await this._populateDataForChart(fetchFunctionConfigs);
      },
    });
  }

  getDisplayDataChartOptionsInput<BrowsersParameter>(
    browsers: BrowsersParameter[],
  ): {
    seriesColors: string[];
    vAxisTitle: string;
  } {
    // Compute seriesColors from selected browsers and BROWSER_ID_TO_COLOR
    const selectedBrowsers = browsers;
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

  async getAllFeatureData(
    features: MissingOneImplFeaturesList,
  ): Promise<components['schemas']['Feature'][]> {
    if (features.length === 0) {
      return [];
    }
    const query = `id:${features.map(f => f.feature_id).join(' OR id:')}`;
    return await this.apiClient.getAllFeatures(query, 'name_asc');
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
    const label =
      event.detail.label === 'Chrome/Edge' ? 'Chrome' : event.detail.label;
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
        return await this.getAllFeatureData(features);
      },
      args: () => [targetDate, targetBrowser],
      onComplete: features => {
        this.taskTracker = {
          status: TaskStatus.COMPLETE,
          error: undefined,
          data: features,
        };
      },
      onError: async (error: unknown) => {
        if (error instanceof ApiError) {
          this.taskTracker = {
            status: TaskStatus.ERROR,
            error: error,
            data: undefined,
          };
          await toast(`${error.message}`, 'danger', 'exclamation-triangle');
        } else {
          // Should never reach here but let's handle it.
          this.taskTracker = {
            status: TaskStatus.ERROR,
            error: new UnknownError('unknown error fetching features'),
            data: undefined,
          };
        }
      },
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
        <div class="table-description">
          * This table represents values for Availability and Usage
          <strong>as of today</strong>, and not at the selected timestamp.
        </div>
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
    const columns: ColumnKey[] = parseColumnsSpec(MISSING_ONE_TABLE_COLUMNS);
    let headerCells: TemplateResult[] = [];
    headerCells = renderSavedSearchHeaderCells(this.getPanelText(), columns);

    return html`<webstatus-overview-table
      .columns=${columns}
      .headerCells=${headerCells}
      .taskTracker=${this.taskTracker}
    ></webstatus-overview-table>`;
  }
}
