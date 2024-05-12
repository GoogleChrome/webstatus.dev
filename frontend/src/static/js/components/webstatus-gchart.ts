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

// The dataObj is a subset of the possible data that can be used to
// generate a google.visualization.DataTable.
// It assumes the rows are sorted by the 'datetime' in the first column.
// The subsequent columns are all numbers or nulls.
export type WebStatusDataObj = {
  cols: Array<{type: string; label: string}>;
  rows: Array<[Date, ...Array<number | null>]>;
};

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
  chartType = 'LineChart';

  @property({type: Object, attribute: 'options'})
  options: google.visualization.LineChartOptions | undefined;

  @property({
    type: Object,
    attribute: 'dataObj',
  })
  dataObj: WebStatusDataObj | undefined;

  @property({state: true, type: Object})
  dataTable:
    | google.visualization.DataTable
    | google.visualization.DataView
    | undefined;

  @state()
  chartWrapper: google.visualization.ChartWrapper | undefined;

  @state()
  dataLoadingStatus: TaskStatus = TaskStatus.INITIAL;

  static get styles(): CSSResultGroup {
    return [
      css`
        .chart_container {
          padding: 0;
          margin: 0;
          border: 0;
        }

        .loading-overlay {
          position: absolute;
          top: 0;
          left: 0;
          pointer-events: none;
          z-index: 10;
          width: calc(100% - 100px);
          height: calc(100% - 80px);
          margin-left: 100px;
          margin-top: 40px;
          display: flex; /* Center the spinner */
          align-items: center;
          justify-content: center;
        }

        .loading-overlay::before {
          /* Semi-transparent background */
          content: '';
          position: absolute;
          top: 0;
          left: 0;
          width: 100%;
          height: 100%;
          background-color: rgba(255, 255, 255, 0.7);
        }

        /* Hide the overlay when not loading */
        .loading-overlay.hidden {
          display: none;
        }

        .loading-overlay > * {
          /* Disable clicks on content inside the overlay */
          pointer-events: none;
        }

        /* Disable chart interaction while loading */
        .chart_container.loading .google-visualization-charteditor-svg {
          pointer-events: none;
        }
      `,
    ];
  }

  // Convert the WebStatusDataObj to a DataTable.
  convertWebStatusDataObjToDataTable(
    dataObj: WebStatusDataObj
  ): google.visualization.DataTable {
    const dataTable = new google.visualization.DataTable();
    dataObj.cols.forEach(col => {
      dataTable.addColumn(col.type, col.label);
    });
    dataObj.rows.forEach(row => {
      dataTable.addRow(row);
    });
    return dataTable;
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
        ${this.dataLoadingStatus !== TaskStatus.COMPLETE &&
        this.dataLoadingStatus !== TaskStatus.ERROR
          ? html`<div class="loading-overlay">
              <sl-spinner></sl-spinner>
            </div>`
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
        this._pendingDataObj
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
      this.chartWrapper.setOptions(this.options);
      this.chartWrapper.setDataTable(
        this.dataTable as google.visualization.DataTable
      );
      this.chartWrapper.draw();
    }
  }
}
