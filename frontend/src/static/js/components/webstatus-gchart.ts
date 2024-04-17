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

import {LitElement, type TemplateResult, html} from 'lit';
import {customElement, property, state} from 'lit/decorators.js';

@customElement('webstatus-gchart')
export class WebstatusGChart extends LitElement {
  // Properties for chartwrapper spec fields.
  @property({type: String, attribute: 'containerId'})
  containerId: string | undefined;

  @property({type: String, attribute: 'chartType'})
  chartType = 'LineChart';

  @property({type: Object, attribute: 'options'})
  options: google.visualization.LineChartOptions | undefined;

  @property({type: Object, attribute: 'dataTable'})
  dataTable:
    | google.visualization.DataTable
    | google.visualization.DataView
    | undefined;

  @state()
  chartWrapper: google.visualization.ChartWrapper | undefined;

  constructor() {
    super();
    this.chartWrapper = new google.visualization.ChartWrapper({
      containerId: this.containerId,
      chartType: this.chartType,
      options: this.options,
      dataTable: this.dataTable,
    });

    // Since ChartWrapper wants to look up the container element by id,
    // which would fail to find it in the shadowDom, we have to replace the
    // chartWrapper.getContainer method with a function that returns the div
    // corresponding to this.containerId
    (
      this.chartWrapper as unknown as {getContainer: () => Element}
    ).getContainer = () => {
      return this.shadowRoot!.getElementById(this.containerId!)!;
    };
  }

  render(): TemplateResult {
    return html`
      <div
        id="${this.containerId}"
        class="chart_div"
        style="padding: 0; margin: 0; border: 0"
      ></div>
    `;
  }

  updated() {
    if (
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
