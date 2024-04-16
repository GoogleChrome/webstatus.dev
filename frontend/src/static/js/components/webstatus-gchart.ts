// See https://github.com/DefinitelyTyped/DefinitelyTyped/blob/master/types/google.visualization/index.d.ts
/// <reference types="@types/google.visualization" />

import {LitElement, type TemplateResult, html} from 'lit';
import {customElement, property, state} from 'lit/decorators.js';

@customElement('webstatus-gchart')
export class WebstatusGChart extends LitElement {

  // Properties for chartwrapper spec fields. containerId, chartType, options, dataTable.

  @property({ type: String, attribute: 'containerId' })
  containerId: string | undefined;

  @property({ type: String, attribute: 'chartType' })
  chartType = "ComboChart";

  @property({ type: Object, attribute: 'options' })
  options: Object | undefined;

  @property({ type: Object, attribute: 'dataTable' })
  dataTable: google.visualization.DataTable | google.visualization.DataView | undefined;

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
  }

  render(): TemplateResult {
    return html`
      <div class="chart_div" style="padding: 0; margin: 0; border: 0"></div>
    `;
  }
}
