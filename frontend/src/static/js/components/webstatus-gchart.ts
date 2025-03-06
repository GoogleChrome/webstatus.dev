/**
 * Copyright 2023 Google LLC
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

// See https://github.com/DefinitelyTyped/DefinitelyTyped/blob/master/types/google.visualization/index.d.ts
/// <reference types="@types/google.visualization" />

import {consume} from '@lit/context';
import {
  LitElement,
  type TemplateResult,
  type CSSResultGroup,
  css,
  html,
  PropertyValues,
  nothing,
} from 'lit';
import {customElement, property, state} from 'lit/decorators.js';
import {gchartsContext} from '../contexts/gcharts-context.js';
import {TaskStatus} from '@lit/task';
import {classMap} from 'lit/directives/class-map.js';

export interface ChartClickEventDetail {
  label: string;
  timestamp: Date;
  value: number;
}

export type ChartSelectPointEvent = CustomEvent<ChartClickEventDetail>;

export type ChartDeselectPointEvent = CustomEvent<undefined>;

// The dataObj is a subset of the possible data that can be used to
// generate a google.visualization.DataTable.
// It assumes the rows are sorted by the 'datetime' in the first column.
// The subsequent columns are all numbers, string, or nulls.
export type WebStatusDataObj = {
  cols: Array<{type: string; label: string; role: string}>;
  rows: Array<[Date, ...Array<number | string | null>]>;
};

/**
 * A web component wrapper around a Google Chart.
 *
 * @event ChartSelectPointEvent point-selected - Dispatched when a data point on the chart is clicked.
 *  The `detail` property contains an object with the `label`, `timestamp`,
 *  and `value` of the clicked data point.
 * @event ChartDeselectPointEvent point-deselected - Dispatched when the user deselects a data point on the chart.
 *  The `detail` property is undefined.
 *  Note: Since we only support one selected point at a time, we don't need to return the deselected point itself.
 */
@customElement('webstatus-gchart')
export class WebstatusGChart extends LitElement {
  @consume({context: gchartsContext, subscribe: true})
  @property({attribute: false})
  gchartsLibraryLoaded?: boolean;

  private _pendingDataObj: WebStatusDataObj | undefined;

  // Properties for chartwrapper spec fields.
  @property({type: String, attribute: 'containerId'})
  containerId: string | undefined;

  @property({type: String, attribute: 'chartType'})
  chartType = 'ComboChart';

  @property({type: Object, attribute: 'options'})
  options: google.visualization.ComboChartOptions | undefined;

  @property({
    type: Object,
    attribute: 'dataObj',
  })
  dataObj: WebStatusDataObj | undefined;

  @property({
    type: Boolean,
    attribute: 'hasMax',
  })
  hasMax = true;

  // Selected data points on the chart.
  // If the chart ever re-draws due to resize or the encompassing component
  // re-drawing, we need to manually set the current selection.
  currentSelection: google.visualization.ChartSelection[] | undefined;

  @property({state: true, type: Object})
  dataTable:
    | google.visualization.DataTable
    | google.visualization.DataView
    | undefined;

  @state()
  chartWrapper: google.visualization.ChartWrapper | undefined;

  @state()
  dataLoadingStatus: TaskStatus = TaskStatus.INITIAL;

  private _chartClickListenerAdded = false;

  static get styles(): CSSResultGroup {
    return [
      css`
        .chart_container {
          padding: 0;
          margin: 0;
          border: 0;
        }

        /* Disable chart interaction while loading */
        .chart_container.loading .google-visualization-charteditor-svg {
          pointer-events: none;
        }

        /* override the fixed width of the chart */
        .chart_container > div > div > div {
          width: auto !important;
        }
      `,
    ];
  }

  private _resizeObserver: ResizeObserver | undefined;

  draw() {
    if (this.chartWrapper) {
      this.chartWrapper.draw();
      this.chartWrapper?.getChart()?.setSelection(this.currentSelection);
    }
  }

  firstUpdated() {
    // 1. Create the ResizeObserver
    this._resizeObserver = new ResizeObserver(() => {
      // 2. Redraw the chart when a resize occurs
      this.draw();
    });

    // 3. Start observing the chart container element
    this._resizeObserver.observe(
      this.shadowRoot!.getElementById(this.containerId!)!,
    );
  }

  disconnectedCallback() {
    // 4. Clean up the ResizeObserver
    if (this._resizeObserver) {
      this._resizeObserver.disconnect();
    }
    super.disconnectedCallback();
  }

  // Convert the WebStatusDataObj to a DataTable.
  convertWebStatusDataObjToDataTable(
    dataObj: WebStatusDataObj,
  ): google.visualization.DataTable {
    const dataTable = new google.visualization.DataTable();
    dataObj.cols.forEach(col => {
      dataTable.addColumn({type: col.type, label: col.label, role: col.role});
    });
    dataObj.rows.forEach(row => {
      dataTable.addRow(row);
    });
    return dataTable;
  }

  // Augment the options with options that apply for all charts.
  augmentOptions(
    options: google.visualization.ComboChartOptions,
  ): google.visualization.ComboChartOptions {
    if (!this.hasMax) {
      options = {
        ...options,
        tooltip: {trigger: 'selection'},
      };
      return options;
    }

    const numColumns = this.dataTable!.getNumberOfColumns();
    // The number of series is the number of columns with role 'data'.
    let numSeries = 0;
    for (let i = 0; i < numColumns; i++) {
      const role = this.dataTable!.getColumnProperty(i, 'role');
      if (role === 'data') {
        numSeries++;
      }
    }

    // Make the 'total' series, which is the last series, be drawn
    // with type 'area' so that it fills the area under the lines.
    const totalSeriesIndex = numSeries - 1;

    // Compute the size of points on the total line to be inversely proportional
    // to the number of data points, the more points, the smaller they are.
    const pointSize = Math.min(
      2,
      100 / (this.dataTable!.getNumberOfRows() || 1),
    );

    // Get the current series option, if any, and augment with the total series.
    const seriesOptions = options.series || {};
    seriesOptions[totalSeriesIndex] = {
      type: 'area',
      areaOpacity: 0.08,
      opacity: 0.25,
      lineWidth: 0.2,
      pointSize: pointSize,
    };

    return {
      ...options,
      series: seriesOptions,
    };
  }

  willUpdate(changedProperties: PropertyValues<this>) {
    if (this.gchartsLibraryLoaded) {
      // If dataObj is provided, and it is changed, then (re)generate the dataTable.
      if (this.dataObj && changedProperties.has('dataObj')) {
        this.dataTable = this.convertWebStatusDataObjToDataTable(this.dataObj);
      }

      if (!this.chartWrapper) {
        this.chartWrapper = new google.visualization.ChartWrapper();

        const extendedChartWrapper = this.chartWrapper as unknown as {
          getContainer: () => Element;
        };

        // Since ChartWrapper wants to look up the container element by id,
        // but it would fail to find it in the shadowDom, we have to replace the
        // chartWrapper.getContainer method with a function that returns the div
        // corresponding to this.containerId, which we know how to find.
        extendedChartWrapper.getContainer = () => {
          return this.shadowRoot!.getElementById(this.containerId!)!;
        };
      }
    } else {
      // If the library is not loaded, store the updated dataObj
      if (this.dataObj && changedProperties.has('dataObj')) {
        this._pendingDataObj = this.dataObj;
      }
    }
  }

  render(): TemplateResult {
    const chartContainerClasses = classMap({
      chart_container: true,
      loading: this.dataLoadingStatus !== TaskStatus.COMPLETE,
    });

    return html`
      <div class="${chartContainerClasses}">
        ${this.dataLoadingStatus === TaskStatus.ERROR
          ? html`<div class="error-message">Error loading chart data.</div>`
          : nothing}
        <div id="${this.containerId!}"></div>
      </div>
    `;
  }

  updated(changedProperties: PropertyValues<this>) {
    // If the library just became loaded, process pending dataObj
    if (
      changedProperties.has('gchartsLibraryLoaded') &&
      this.gchartsLibraryLoaded &&
      this._pendingDataObj
    ) {
      this.dataTable = this.convertWebStatusDataObjToDataTable(
        this._pendingDataObj,
      );
      this._pendingDataObj = undefined; // Clear the pending data
    }
    if (
      this.gchartsLibraryLoaded &&
      this.chartWrapper &&
      this.containerId &&
      this.chartType &&
      this.options &&
      this.dataTable
    ) {
      this.chartWrapper.setContainerId(this.containerId); // Still required?
      this.chartWrapper.setChartType(this.chartType);
      this.chartWrapper.setOptions(this.augmentOptions(this.options));
      this.chartWrapper.setDataTable(
        this.dataTable as google.visualization.DataTable,
      );
      if (!this._chartClickListenerAdded) {
        // Check the flag
        google.visualization.events.addListener(
          this.chartWrapper,
          'select',
          () => {
            this._handleChartClick();
          },
        );
        this._chartClickListenerAdded = true; // Set the flag after adding the listener
      }
      this.draw();
    }
  }
  private _handleChartClick() {
    const selection = this.chartWrapper?.getChart()?.getSelection();
    if (selection === undefined) {
      this.currentSelection = undefined;
      return;
    }
    if (selection.length > 0) {
      this.currentSelection = selection;
      // TODO: For now only look at the first selection since we only configure for one selection at a time.
      const item = selection[0];
      const row = item.row;
      const column = item.column;
      // row and column both have the type: number|null|undefined
      if (
        row !== null &&
        column !== null &&
        row !== undefined &&
        column !== undefined
      ) {
        const label = this.dataTable!.getColumnLabel(column);
        // Assuming timestamp is in the first column
        const timestamp = this.dataTable!.getValue(row, 0);
        const value = this.dataTable!.getValue(row, column);

        // Dispatch the chart click event
        const chartClickEvent: ChartSelectPointEvent = new CustomEvent(
          'point-selected',
          {
            detail: {label, timestamp, value},
            bubbles: true,
          },
        );
        this.dispatchEvent(chartClickEvent);
      }
    } else if (selection.length === 0) {
      this.currentSelection = [];
      const chartDeselectEvent: ChartDeselectPointEvent = new CustomEvent(
        'point-deselected',
        {detail: undefined},
      );
      this.dispatchEvent(chartDeselectEvent);
    }
  }
}
