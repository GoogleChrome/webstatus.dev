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

import {Task} from '@lit/task';
import {TemplateResult, html} from 'lit';
import {
  LineChartMetricData,
  WebstatusLineChartPanel,
} from './webstatus-line-chart-panel.js';
import {
  BrowserReleaseFeatureMetric,
  type APIClient,
  ALL_BROWSERS,
  BrowsersParameter,
  BROWSER_ID_TO_COLOR,
} from '../api/client.js';
import {SlMenu, SlMenuItem} from '@shoelace-style/shoelace';
import {customElement, state} from 'lit/decorators.js';

@customElement('webstatus-stats-missing-one-impl-chart-panel')
export class WebstatusStatsMissingOneImplChartPanel extends WebstatusLineChartPanel {
  @state()
  supportedBrowsers: BrowsersParameter[] = ALL_BROWSERS;

  createLoadingTask(): Task {
    return new Task(this, {
      args: () =>
        [this.apiClient, this.startDate, this.endDate] as [
          APIClient,
          Date,
          Date,
        ],
      task: async ([apiClient, startDate, endDate]: [
        APIClient,
        Date,
        Date,
      ]) => {
        await this._fetchMissingOneImplemenationCounts(
          apiClient,
          startDate,
          endDate,
        );
        return;
      },
    });
  }

  getDisplayDataChartOptionsInput(): {
    seriesColors: string[];
    vAxisTitle: string;
  } {
    // Compute seriesColors from selected browsers and BROWSER_ID_TO_COLOR
    const selectedBrowsers = this.supportedBrowsers;
    const seriesColors = [...selectedBrowsers].map(browser => {
      const browserKey = browser as keyof typeof BROWSER_ID_TO_COLOR;
      return BROWSER_ID_TO_COLOR[browserKey];
    });

    return {
      seriesColors: seriesColors,
      vAxisTitle: 'Number of features missing',
    };
  }

  async _fetchMissingOneImplemenationCounts(
    apiClient: APIClient,
    startDate: Date,
    endDate: Date,
  ) {
    if (typeof apiClient !== 'object') return;
    const browserMetricData: Array<
      LineChartMetricData<BrowserReleaseFeatureMetric>
    > = ALL_BROWSERS.map(browser => ({
      label: browser,
      data: [],
      getTimestamp: (dataPoint: BrowserReleaseFeatureMetric) =>
        new Date(dataPoint.timestamp),
      getValue: (dataPoint: BrowserReleaseFeatureMetric) => dataPoint.count,
    }));
    const promises = ALL_BROWSERS.map(async browser => {
      const browserData = browserMetricData.find(
        data => data.label === browser,
      );
      if (!browserData) return;

      const otherBrowsers = ALL_BROWSERS.filter(value => browser !== value);
      for await (const page of apiClient.getMissingOneImplementationCountsForBrowser(
        browser,
        otherBrowsers,
        startDate,
        endDate,
      )) {
        browserData.data.push(...page);
      }
    });
    await Promise.all(promises); // Wait for all browsers to finish

    this.setDisplayDataFromMap(browserMetricData);
  }
  getPanelID(): string {
    return 'missing-one-implementation';
  }
  getPanelText(): string {
    return 'Features missing in only 1 browser';
  }
  renderControls(): TemplateResult {
    return html`
      <sl-dropdown
        id="${this.getPanelID()}-browser-selector"
        multiple
        stay-open-on-select
        .value="${this.supportedBrowsers.join(' ')}"
      >
        <sl-button slot="trigger">
          <sl-icon slot="suffix" name="chevron-down"></sl-icon>
          Browsers
        </sl-button>
        <sl-menu @sl-select=${this.handleBrowserSelection}>
          <sl-menu-item type="checkbox" value="chrome">Chrome</sl-menu-item>
          <sl-menu-item type="checkbox" value="edge">Edge</sl-menu-item>
          <sl-menu-item type="checkbox" value="firefox">Firefox</sl-menu-item>
          <sl-menu-item type="checkbox" value="safari">Safari</sl-menu-item>
        </sl-menu>
      </sl-dropdown>
    `;
  }

  // TODO(#1104) - Consolidate this into a new common browser specific panel for charts only used on the stats page.
  // Do not add it to the main base abstract class because that panel will be used for the feature detail page charts
  // too. And those do not have browser dropdowns.
  handleBrowserSelection(event: Event) {
    const menu = event.target as SlMenu;
    const menuItemsArray: Array<SlMenuItem> = Array.from(menu.children).filter(
      child => child instanceof SlMenuItem,
    ) as Array<SlMenuItem>;

    // Build the list of values of checked menu-items.
    this.supportedBrowsers = menuItemsArray
      .filter(menuItem => menuItem.checked)
      .map(menuItem => menuItem.value) as BrowsersParameter[];
    // Regenerate data and redraw.  We should instead just filter it.
  }
}
