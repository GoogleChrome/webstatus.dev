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
 * Extension of the WebstatusLineChartTabbedPanel abstract base class to add
 * tabbed functionality for multiple views.
 */
export abstract class WebstatusLineChartTabbedPanel<
  S,
> extends WebstatusLineChartPanel<S> {
  /**
   * The processed data objects for each view of the chart, structured for `webstatus-gchart`.
   * @state
   * @type {Array<WebStatusDataObj | undefined>}
   */
  @state()
  dataByView?: Array<WebStatusDataObj>;

  /**
   * Names of the tabs to be displayed in the panel.
   * @abstract
   * @type {Array<string>}
   */
  abstract tabViews: Array<string>;

  /**
   * The list of supported browsers for each view of the chart.
   * @abstract
   * @type {ArrayArray<<BrowsersParameter>>}
   */
  abstract browsersByView: Array<Array<BrowsersParameter>>;

  _handleTabClick() {
    this.resetPointSelectedTask();
  }

  getTabTooltip(tab: string): TemplateResult {
    if (tab === 'Mobile') {
      return html`${tab}
        <sl-tooltip content="${'Mobile results are a work in progress.'}">
          <sl-icon
            class="icon-button__icon"
            aria-hidden="true"
            name="exclamation-triangle"
            library="default"
          ></sl-icon
        ></sl-tooltip>`;
    }
    return html`${tab}`;
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
      >${this.tabViews.map((tab, index) => {
        return html`<sl-tab
            slot="nav"
            id="${this.getPanelID()}-tab-${tab.toLowerCase()}"
            panel="${tab.toLowerCase()}"
            @click=${this._handleTabClick}
          >
            ${this.getTabTooltip(tab)}
          </sl-tab>
          <sl-tab-panel name="${tab.toLowerCase()}">
            <div>${this.renderChartByView(index)}</div>
            ${this.renderPointSelectedDetails()}
          </sl-tab-panel>`;
      })}
    </sl-tab-group>`;
  }

  /**
   * Renders the chart for the current view based on the current state of the
   * data loading task.
   * @param {number} index - The index of the current view.
   * @returns {TemplateResult} The chart template.
   */
  renderChartByView(view: number): TemplateResult {
    if (!this._task) return html``;
    return this._task?.render({
      complete: () => this.renderChartWhenCompleteByView(view),
      error: error => this.renderChartWhenError(error),
      initial: () => this.renderChartWhenInitial(),
      pending: () => this.renderChartWhenPending(),
    });
  }

  /**
   * Renders the chart when data loading is complete.
   * @returns {TemplateResult} The chart template, including the `webstatus-gchart` component.
   */
  renderChartWhenCompleteByView(view: number): TemplateResult {
    return html`
      <div
        id="${this.getPanelID()}-${view}-complete"
        class="complete-chart-panel chart-panel"
      >
        <webstatus-gchart
          id="${this.getPanelID()}-chart"
          @point-selected=${this.handlePointSelected}
          @point-deselected=${this.handlePointDeselected}
          .hasMax=${this.hasMax}
          .containerId="${this.getPanelID()}-${view}-chart-container"
          .chartType="${'LineChart'}"
          .dataObj="${this.dataByView ? this.dataByView[view] : undefined}"
          .options="${this.generateDisplayDataChartOptionsByView(view)}"
        >
          Loading chart...
        </webstatus-gchart>
      </div>
    `;
  }

  generateDisplayDataChartOptionsByView(
    view: number,
  ): google.visualization.LineChartOptions {
    const {seriesColors, vAxisTitle} = this.getDisplayDataChartOptionsInput(
      this.browsersByView[view],
    );
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
    const metricDataArray = await this._fetchAndAggregateData(
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
      </sl-card>
    `;
  }
}
