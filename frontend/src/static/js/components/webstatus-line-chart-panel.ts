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

/**
 * Interface defining the structure of metric data for the line chart.
 * @template T The type of the data points.
 */
export interface LineChartMetricData<T> {
  /**
   * The label for the metric (displayed on the chart legend).
   * @type {string}
   */
  label: string;

  /**
   * The array of data points for the metric.
   * @type {Array<T>}
   */
  data: Array<T>;

  /**
   * Function to extract the timestamp from a data point.
   * @param {T} dataPoint The data point.
   * @returns {Date} The timestamp of the data point.
   */
  getTimestamp: (dataPoint: T) => Date;

  /**
   * Function to extract the data value from a data point.
   * @param {T} dataPoint The data point.
   * @returns {number | undefined} The data value of the data point.
   */
  getValue: (dataPoint: T) => number | undefined;
}

// Type for the data fetched event (using type alias)
// type DataFetchedEvent<T> = CustomEvent<{label: string; data: T[]}>;

// Type for series calculator functions (
export type SeriesCalculator<T> = (
  dataPoint: T,
  metricData: LineChartMetricData<T>,
  cacheMap: Map<string, T>,
) => void;

// Type for extracting timestamp from a data point
type TimestampExtractor<T> = (dataPoint: T) => Date;

// Type for extracting value from a data point
type ValueExtractor<T> = (dataPoint: T) => number;

// Interface for additional series configuration
export interface AdditionalSeriesConfig<T> {
  label: string;
  calculator: SeriesCalculator<T>;
  timestampExtractor: TimestampExtractor<T>;
  valueExtractor: ValueExtractor<T>;
  cacheMap: Map<string, T>;
}

// Interface for fetch function configuration
export interface FetchFunctionConfig<T> {
  label: string;
  fetchFunction: () => AsyncIterable<T[]>;
  timestampExtractor: TimestampExtractor<T>;
  valueExtractor: ValueExtractor<T>;
}

/**
 * Abstract base class for creating line chart panels to display web status data.
 * This class handles data processing, chart rendering using `webstatus-gchart`,
 * and provides a framework for custom controls and panel-specific logic.
 * Subclasses must implement abstract methods to define data loading,
 * panel identification, text display, and chart options.
 */
export abstract class WebstatusLineChartPanel extends LitElement {
  /**
   * The start date for the data to be displayed in the chart.
   * @property
   * @type {Date}
   */
  @property({type: Object})
  startDate!: Date;

  /**
   * The end date for the data to be displayed in the chart.
   * @property
   * @type {Date}
   */
  @property({type: Object})
  endDate!: Date;

  /**
   * The processed data object for the chart, structured for `webstatus-gchart`.
   * @state
   * @type {WebStatusDataObj | undefined}
   */
  @state()
  data?: WebStatusDataObj;

  /**
   * The API client for fetching web status data. Injected via context.
   * @consume
   * @type {APIClient}
   */
  @consume({context: apiClientContext})
  apiClient!: APIClient;

  /**
   * The Lit task for managing the asynchronous data loading process.
   * Subclasses must implement this method to define how data is fetched.
   * @abstract
   * @returns {Task} A new Task instance.
   */
  abstract createLoadingTask(): Task;

  /**
   * Returns a unique identifier for the panel.
   * @abstract
   * @returns {string} The panel ID.
   */
  abstract getPanelID(): string;

  /**
   * Returns the text to display in the panel header.
   * @abstract
   * @returns {string} The panel text.
   */
  abstract getPanelText(): string;

  /**
   * Renders the controls for the panel (e.g., dropdowns, buttons).
   * @abstract
   * @returns {TemplateResult} The controls template.
   */
  abstract renderControls(): TemplateResult;

  /**
   * Returns the input for generating chart options.
   * @abstract
   * @returns {{seriesColors: Array<string>; vAxisTitle: string;}} Chart options input.
   */
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
        sl-card {
          display: block;
          width: auto;
        }
      `,
    ];
  }

  /**
   * Processes the input metric data and formats it into a `WebStatusDataObj`
   * suitable for the `webstatus-gchart` component.
   * @param {Array<LineChartMetricData<T>>} metricDataArray Array of metric data objects.
   * @template T The data type of the metric data.
   *
   * TODO(kyleju): refactor this method acorss feature detail page
   * and stats page, https://github.com/GoogleChrome/webstatus.dev/issues/964.
   */
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
      for (const dataPoint of metricData.data) {
        const timestamp = metricData.getTimestamp(dataPoint);
        const dateSeconds = timestamp.getTime();
        const dataValue = metricData.getValue(dataPoint);

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

  /**
   * Renders an error message when an error occurs during data loading.
   * @returns {TemplateResult} The error message template.
   */
  renderChartWhenError(): TemplateResult {
    return html`<div id="${this.getPanelID()}-error">
      Error when loading stats.
    </div>`;
  }

  /**
   * Renders a message before data loading starts.
   * @returns {TemplateResult} The initial message template.
   */
  renderChartWhenInitial(): TemplateResult {
    return html`<div id="${this.getPanelID()}-initial">
      Preparing request for stats.
    </div>`;
  }

  /**
   * Renders a message while data is being loaded.
   * @returns {TemplateResult} The loading message template.
   */
  renderChartWhenPending(): TemplateResult {
    return html`<div id="${this.getPanelID()}-pending">Loading stats.</div>`;
  }

  /**
   * Renders the chart based on the current state of the data loading task.
   * @returns {TemplateResult} The chart template or undefined if no task.
   */
  renderChart(): TemplateResult {
    if (!this._task) return html``;
    return this._task?.render({
      complete: () => this.renderChartWhenComplete(),
      error: () => this.renderChartWhenError(),
      initial: () => this.renderChartWhenInitial(),
      pending: () => this.renderChartWhenPending(),
    });
  }

  /**
   * Renders the chart when data loading is complete.
   * @returns {TemplateResult} The chart template, including the `webstatus-gchart` component.
   */
  renderChartWhenComplete(): TemplateResult {
    return html`
      <div id="${this.getPanelID()}-complete">
        <webstatus-gchart
          id="${this.getPanelID()}-chart"
          .hasMax=${false}
          .containerId="${this.getPanelID()}-chart-container"
          .chartType="${'LineChart'}"
          .dataObj="${this.data}"
          .options="${this.generateDisplayDataChartOptions()}"
        >
          Loading chart...
        </webstatus-gchart>
      </div>
    `;
  }

  generateDisplayDataChartOptions(): google.visualization.LineChartOptions {
    const {seriesColors, vAxisTitle} = this.getDisplayDataChartOptionsInput();
    // Add one day to this.endDate.
    const endDate = new Date(this.endDate.getTime() + 1000 * 60 * 60 * 24);
    const options: google.visualization.LineChartOptions = {
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

      // Enable explorer mode
      explorer: {
        actions: ['dragToZoom', 'rightClickToReset'],
        axis: 'horizontal',
        keepInBounds: true,
        maxZoomIn: 4,
        maxZoomOut: 4,
        zoomDelta: 0.01,
      },
    };

    return options;
  }

  /**
   * Fetches and aggregates data for the chart.
   * This method takes an array of fetch function configurations and an optional
   * array of additional series configurations. It fetches data for each fetch
   * function configuration concurrently, then applies the additional series
   * calculators to the fetched data. The processed data is then formatted into
   * a `LineChartMetricData` array and passed to `setDisplayDataFromMap` for
   * rendering.
   *
   * @param fetchFunctionConfigs An array of fetch function configurations.
   * @param additionalSeriesConfigs An optional array of additional series configurations.
   */
  async _fetchAndAggregateData<T>(
    fetchFunctionConfigs: FetchFunctionConfig<T>[],
    additionalSeriesConfigs?: AdditionalSeriesConfig<T>[],
  ) {
    // Create an array of metric data objects for each fetch function
    const metricDataArray: Array<LineChartMetricData<T>> =
      fetchFunctionConfigs.map(
        ({label, timestampExtractor, valueExtractor}) => ({
          label,
          data: [],
          getTimestamp: timestampExtractor,
          getValue: valueExtractor,
        }),
      );

    // Dispatch an event to signal the start of data fetching
    const event = new CustomEvent('data-fetch-starting');
    this.dispatchEvent(event);

    // Fetch data for each configuration concurrently
    const promises = fetchFunctionConfigs.map(
      async ({fetchFunction, label}) => {
        for await (const page of fetchFunction()) {
          // Find the corresponding metric data object
          const metricData = metricDataArray.find(data => data.label === label);
          if (metricData) {
            metricData.data.push(...page);
          }
        }
      },
    );

    await Promise.all(promises);

    // Apply additionalSeriesConfigs if provided
    if (additionalSeriesConfigs) {
      fetchFunctionConfigs.forEach(({label}) => {
        const metricData = metricDataArray.find(data => data.label === label);

        if (metricData) {
          // Create a new cacheMap for each fetch function config
          const cacheMap = new Map<
            string,
            T
            // {timestamp: string; count: number}
          >();

          metricData.data.forEach((dataPoint: T) => {
            additionalSeriesConfigs.forEach(({calculator}) => {
              calculator(dataPoint, metricData, cacheMap);
            });
          });

          // Convert cacheMap to array and create new LineChartMetricData entries
          additionalSeriesConfigs.forEach(
            ({label, cacheMap, valueExtractor, timestampExtractor}) => {
              const newMetricData: LineChartMetricData<T> = {
                label: label,
                data: Array.from(cacheMap.values()),
                getTimestamp: timestampExtractor,
                getValue: valueExtractor,
              };
              metricDataArray.push(newMetricData);
            },
          );
        }
      });
    }

    this.setDisplayDataFromMap(metricDataArray);
  }

  /**
   * Creates a `DataFetchedEvent` containing the fetched data for each series.
   *
   * @param {FetchFunctionConfig<T>} fetchFunctionConfigs The array of fetch function configurations.
   * @returns {DataFetchedEvent<T>} The custom event containing the fetched data.
   * @template T The type of the fetched data.
   */
  // private _createDataFetchedEvent<T>(
  //   fetchFunctionConfigs: Array<FetchFunctionConfig<T>>,
  //   metricDataArray: Array<LineChartMetricData<T>>,
  // ): DataFetchedEvent<T> {
  //   return new CustomEvent('data-fetched', {
  //     detail: fetchFunctionConfigs.map(({label}) => ({
  //       label,
  //       // Get the data from the corresponding metricData object
  //       data: metricDataArray.find(data => data.label === label)?.data || [],
  //     })),
  //     bubbles: true,
  //   });
  // }

  /**
   * SeriesCalculator to calculate the maximum value for each timestamp.
   * This method takes a data point, the metric data for the series,
   * and a cache map to store and retrieve intermediate calculations.
   * It calculates the maximum value for each timestamp by comparing the
   * values of data points with the same timestamp.
   *
   * @param dataPoint The data point to process.
   * @param metricData The metric data for the series.
   * @param cacheMap The cache map to store and retrieve intermediate calculations.
   */
  calculateMax<T>(
    dataPoint: T,
    metricData: LineChartMetricData<T>,
    cacheMap: Map<string, T>,
  ) {
    const value = metricData.getValue(dataPoint) || 0;
    const timestamp = metricData.getTimestamp(dataPoint);
    const dateString = timestamp.toISOString();

    const existingDataPoint = cacheMap.get(dateString);

    if (existingDataPoint !== undefined) {
      if (value > (metricData.getValue(existingDataPoint) ?? 0)) {
        cacheMap.set(dateString, dataPoint);
      }
    } else {
      cacheMap.set(dateString, dataPoint);
    }
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
