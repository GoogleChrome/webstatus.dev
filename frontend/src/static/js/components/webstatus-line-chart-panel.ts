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

import {CSSResultGroup, LitElement, TemplateResult, css, html} from 'lit';
import {property, state} from 'lit/decorators.js';
import {WebStatusDataObj} from './webstatus-gchart.js';
import {Task} from '@lit/task';
import {APIClient, apiClientContext} from '../contexts/api-client-context.js';
import {consume} from '@lit/context';
import {SHARED_STYLES} from '../css/shared-css.js';

export interface LineChartMetricData<T> {
  label: string;
  data: Array<T>;
  getTimestamp: (item: T) => Date; // Function to extract timestamp
  getData: (item: T) => number | undefined; // Function to extract data
}

export abstract class WebstatusLineChartPanel extends LitElement {
  @property({type: Object})
  startDate!: Date;

  @property({type: Object})
  endDate!: Date;

  @state()
  data?: WebStatusDataObj;

  @consume({context: apiClientContext})
  apiClient!: APIClient;

  abstract createLoadingTask(): Task;

  abstract getPanelID(): string;

  abstract getPanelText(): string;

  abstract renderControls(): TemplateResult;

  abstract getDisplayDataChartOptionsInput(): {
    seriesColors: Array<string>;
    vAxisTitle: string;
  };

  _task?: Task;

  constructor() {
    super();
    this._task = this.createLoadingTask();
  }

  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        .hbox,
        .vbox {
          gap: var(--content-padding-large);
        }
        /*  Make the dropdown menu button icon rotate when the menu is open,
            so it looks like sl-select. */
        sl-dropdown > sl-button > sl-icon {
          rotate: 0deg;
          transition: var(--sl-transition-medium) rotate ease;
        }
        sl-dropdown[open] > sl-button > sl-icon {
          rotate: -180deg;
          transition: var(--sl-transition-medium) rotate ease;
        }
        sl-card {
          display: block;
          width: auto;
        }
      `,
    ];
  }

  setDisplayDataFromMap<T>(metricDataArray: Array<LineChartMetricData<T>>) {
    const dataObj: WebStatusDataObj = {cols: [], rows: []};
    dataObj.cols.push({type: 'date', label: 'Date', role: 'domain'});

    for (const metricData of metricDataArray) {
      dataObj.cols.push({
        type: 'number',
        label: metricData.label,
        role: 'data',
      });
    }

    const dateToDataMap = new Map<number, {[key: string]: number | null}>();

    for (const metricData of metricDataArray) {
      if (!Array.isArray(metricData.data)) continue;
      for (const item of metricData.data) {
        const timestamp = metricData.getTimestamp(item);
        const dateSeconds = timestamp.getTime();
        const dataValue = metricData.getData(item);

        if (!dateToDataMap.has(dateSeconds)) {
          dateToDataMap.set(dateSeconds, {});
        }
        const dateData = dateToDataMap.get(dateSeconds)!;
        dateData[metricData.label] = dataValue || null;
      }
    }

    const data = Array.from(dateToDataMap.entries()).sort(
      ([d1], [d2]) => d1 - d2,
    );

    for (const [dateSeconds, dateData] of data) {
      const date = new Date(dateSeconds);
      const row: [Date, ...Array<number | string | null>] = [date];

      for (const metricData of metricDataArray) {
        row.push(
          dateData[metricData.label] ? dateData[metricData.label] : null,
        );
      }
      dataObj.rows.push(row);
    }

    this.data = dataObj;
  }

  renderChartWhenError(): TemplateResult {
    return html`Error when loading stats.`;
  }

  renderChartWhenInitial(): TemplateResult {
    return html`Preparing request for stats.`;
  }

  renderChartWhenPending(): TemplateResult {
    return html`Loading stats.`;
  }

  renderChart(): TemplateResult | undefined {
    if (!this._task) return html``;
    return this._task?.render({
      complete: () => this.renderChartWhenComplete(),
      error: () => this.renderChartWhenError(),
      initial: () => this.renderChartWhenInitial(),
      pending: () => this.renderChartWhenPending(),
    });
  }

  renderChartWhenComplete(): TemplateResult {
    return html`
      <webstatus-gchart
        id="${this.getPanelID()}-chart"
        .hasMax=${false}
        .containerId="${'global-feature-support-chart-container'}"
        .chartType="${'LineChart'}"
        .dataObj="${this.data}"
        .options="${this.generateDisplayDataChartOptions()}"
      >
        Loading chart...
      </webstatus-gchart>
    `;
  }

  generateDisplayDataChartOptions(): google.visualization.LineChartOptions {
    const {seriesColors, vAxisTitle} = this.getDisplayDataChartOptionsInput();
    // Add one day to this.endDate.
    const endDate = new Date(this.endDate.getTime() + 1000 * 60 * 60 * 24);
    const options = {
      height: 300, // This is necessary to avoid shrinking to 0 or 18px.
      hAxis: {
        title: '',
        titleTextStyle: {color: '#333'},
        viewWindow: {min: this.startDate, max: endDate},
      },
      vAxis: {
        minValue: 0,
        title: vAxisTitle,
        format: '#,###',
      },
      legend: {position: 'top'},
      colors: seriesColors,
      chartArea: {left: 100, right: 16, top: 40, bottom: 40},

      interpolateNulls: true,

      // Multiple selection of points will be summarized in one tooltip.
      tooltip: {trigger: 'selection'},
      selectionMode: 'multiple',
      aggregationTarget: 'category',

      // Enable explorer mode
      explorer: {
        actions: ['dragToZoom', 'rightClickToReset'],
        axis: 'horizontal',
        keepInBounds: true,
        maxZoomIn: 4,
        maxZoomOut: 4,
        zoomDelta: 0.01,
      },
    } as google.visualization.LineChartOptions;

    return options;
  }

  render(): TemplateResult {
    return html`
      <sl-card id="${this.getPanelID()}">
        <div slot="header" class="hbox">
          ${this.getPanelText()}
          <div class="spacer"></div>
          ${this.renderControls()}
        </div>
        <div>${this.renderChart()}</div>
      </sl-card>
    `;
  }
}
