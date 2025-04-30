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

import {TemplateResult, html} from 'lit';
import {
  AdditionalSeriesConfig,
  FetchFunctionConfig,
  WebstatusLineChartPanel,
} from './webstatus-line-chart-panel.js';
import {state} from 'lit/decorators.js';
import {BrowsersParameter} from '../api/client.js';
import {WebStatusDataObj} from './webstatus-gchart.js';

/**
 * Abstract base class for creating line chart panels to display web status data.
 * This class handles data processing, chart rendering using `webstatus-gchart`,
 * and provides a framework for custom controls and panel-specific logic.
 * Subclasses must implement abstract methods to define data loading,
 * panel identification, text display, and chart options.
 */
export abstract class WebstatusLineChartTabbedPanel extends WebstatusLineChartPanel {
  /**
   * The processed data objects for each view of the chart, structured for `webstatus-gchart`.
   * @state
   * @type {Array<WebStatusDataObj | undefined>}
   */
  @state()
  dataByView?: Array<WebStatusDataObj>;

  /**
   * Names of the tabs to be displayed in the panel.
   * @state
   * @type {Array<string>}
   */
  @state()
  tabViews: Array<string> = [];

  /**
   * Indicates which tabbed view is active.
   * @state
   * @type {number}
   */
  @state()
  currentView: number = 0;

  /**
   * The list of supported browsers for each view of the chart.
   * @state
   * @type {ArrayArray<<BrowsersParameter>>}
   */
  @state()
  browsersByView: Array<Array<BrowsersParameter>> = [];

  /**
   *
   * @param _
   * @param index
   */
  _handleTabClick(_: Event, index: number) {
    this.resetPointSelectedTask();
    this.currentView = index;
  }

  /**
   * Returns the view tabs to render for the panel.
   * @returns {TemplateResult} The mobile toggle element if enabled.
   */
  getTabs(): TemplateResult {
    if (this.tabViews.length === 0) {
      return html``;
    }
    return html`<sl-tab-group
      >${this.tabViews.map(
        (tab, index) =>
          html`<sl-tab
            slot="nav"
            panel="${tab.toLowerCase()}"
            @click=${(e: Event) => this._handleTabClick(e, index)}
            >${tab}</sl-tab
          >`,
      )}</sl-tab-group
    >`;
  }

  /**
   * Renders the chart when data loading is complete.
   * @returns {TemplateResult} The chart template, including the `webstatus-gchart` component.
   */
  renderChartWhenComplete(): TemplateResult {
    return html`
      <div
        id="${this.getPanelID()}-complete"
        class="complete-chart-panel chart-panel"
      >
        <webstatus-gchart
          id="${this.getPanelID()}-chart"
          @point-selected=${this.handlePointSelected}
          @point-deselected=${this.handlePointDeselected}
          .updatePoint=${this.updatePoint}
          .hasMax=${this.hasMax}
          .containerId="${this.getPanelID()}-chart-container"
          .currentSelection=${this.chartSelection}
          .chartType="${'LineChart'}"
          .dataObj="${this.dataByView![this.currentView]}"
          .options="${this.generateDisplayDataChartOptions()}"
        >
          Loading chart...
        </webstatus-gchart>
      </div>
    `;
  }

  /**
   * Populate the chart data based on a set of fetch function configurations.
   *
   * @param fetchFunctionConfigs An array of fetch function configurations.
   * @param dataIndex The index of the data array to update.
   * @param additionalSeriesConfigs An optional array of additional series configurations.
   * @event CustomEvent data-fetch-starting - Dispatched when data fetching starts.
   * @event DataFetchedEvent data-fetch-complete - Dispatched when data fetching is complete.
   *    The `detail` property contains a map of
   *    `{ [label: string]: { data: T[] } }`.
   */
  async _populateDataForChartByView<T>(
    fetchFunctionConfigs: FetchFunctionConfig<T>[],
    dataIndex: number,
    additionalSeriesConfigs?: AdditionalSeriesConfig<T>[],
  ) {
    const metricDataArray = await this._getAggregatedData(
      fetchFunctionConfigs,
      additionalSeriesConfigs,
    );
    const dataObj = this.processDisplayDataFromMap(metricDataArray);

    if (this.dataByView === undefined) {
      this.dataByView = new Array(this.tabViews.length);
    }
    this.dataByView[dataIndex] = dataObj;
  }

  render(): TemplateResult {
    return html`
      <sl-card id="${this.getPanelID()}">
        <div class="hbox">
          <div slot="header">${this.getPanelText()}</div>
          <div class="spacer"></div>
        </div>
        <div class="chart-description">${this.getPanelDescription()}</div>
        <div>${this.getTabs()}</div>
        <div>${this.renderChart()}</div>
        ${this.renderPointSelectedDetails()}
      </sl-card>
    `;
  }
}
