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
import {LitElement, type TemplateResult, html, PropertyValues} from 'lit';
import {customElement, property, state} from 'lit/decorators.js';
import {gchartsContext} from '../contexts/gcharts-context.js';

export type WebStatusDataObj = {
  cols: Array<{ type: string, label: string; }>,
  rows: Array<[Date, ...Array<number | null>]>;
};


@customElement('webstatus-gchart')
export class WebstatusGChart extends LitElement {

  @consume({ context: gchartsContext, subscribe: true })
  @state()
  gchartsLibraryLoaded = false;

  // Properties for chartwrapper spec fields.
  @property({ type: String, attribute: 'containerId' })
  containerId: string | undefined;

  @property({ type: String, attribute: 'chartType' })
  chartType = 'LineChart';

  @property({ type: Object, attribute: 'options' })
  options: google.visualization.LineChartOptions | undefined;

  @property({
    type: Object, attribute: 'dataObj'
   })
  dataObj: WebStatusDataObj | undefined;

  @property( {state: true})
  dataTable:
    | google.visualization.DataTable
    | google.visualization.DataView
    | undefined;

  @state()
  chartWrapper: google.visualization.ChartWrapper | undefined;

  // Convert the WebStatusDataObj to a DataTable.
  convertWebStatusDataObjToDataTable(dataObj: WebStatusDataObj):
    google.visualization.DataTable {
    const dataTable = new google.visualization.DataTable();
    dataObj.cols.forEach((col) => {
      dataTable.addColumn(col.type, col.label);
    });
    dataObj.rows.forEach((row) => {
      dataTable.addRow(row);
    });
    return dataTable;
  }

  render(): TemplateResult {

    return html`
      <div
        id="${this.containerId!}"
        class="chart_div"
        style="padding: 0; margin: 0; border: 0"
      >
      Loading chart library.
    </div>
    `;
  }

  willUpdate(changedProperties: PropertyValues<this>) {
    if (this.gchartsLibraryLoaded) {

      // If dataObj is provided, and it is changed, then (re)generate the dataTable.
      if (this.dataObj && changedProperties.has('dataObj')) {
        this.dataTable = this.convertWebStatusDataObjToDataTable(this.dataObj);
      }

      if (!this.chartWrapper) {
        this.chartWrapper = new google.visualization.ChartWrapper();

        const extendedChartWrapper =
          this.chartWrapper as unknown as { getContainer: () => Element; }

        // Since ChartWrapper wants to look up the container element by id,
        // but it would fail to find it in the shadowDom, we have to replace the
        // chartWrapper.getContainer method with a function that returns the div
        // corresponding to this.containerId, which we know how to find.
        extendedChartWrapper.getContainer = () => {
          return this.shadowRoot!.getElementById(this.containerId!)!;
        };
      }

    }
  }

  updated() {
    if (
      this.gchartsLibraryLoaded &&
      this.chartWrapper &&
      this.containerId &&
      this.chartType &&
      this.options &&
      this.dataTable
    ) {
      this.chartWrapper.setChartType(this.chartType);
      this.chartWrapper.setOptions(this.options);

      this.chartWrapper.setDataTable(
        this.dataTable as google.visualization.DataTable
      );
      this.chartWrapper.draw();
    }
  }
}
