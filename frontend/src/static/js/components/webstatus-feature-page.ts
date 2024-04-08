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

import {consume} from '@lit/context';
import {Task} from '@lit/task';
import {
  LitElement,
  type TemplateResult,
  html,
  CSSResultGroup,
  css,
  nothing,
} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {SHARED_STYLES} from '../css/shared-css.js';
import {type components} from 'webstatus.dev-backend';

import {type APIClient} from '../api/client.js';
import {formatFeaturePageUrl, formatOverviewPageUrl} from '../utils/urls.js';
import {apiClientContext} from '../contexts/api-client-context.js';
import {
  BASELINE_CHIP_CONFIGS,
  renderWPTScore,
} from './webstatus-overview-cells.js';

import {
  makeRandomDataForAllBrowserChannelCombos,
  BrowsersParameter,
  ALL_BROWSERS,
  browserChannelDataMap,
  browserChannelDataMapKey,
  ChannelsParameter,
} from './random-data.js';
import {SlMenu, SlMenuItem} from '@shoelace-style/shoelace';

@customElement('webstatus-feature-page')
export class FeaturePage extends LitElement {
  _loadingTask: Task;

  @consume({context: apiClientContext})
  @state()
  apiClient!: APIClient;

  @state()
  feature?: components['schemas']['Feature'] | undefined;

  @state()
  featureId!: string;

  @state()
  implementationProgressBrowsers: BrowsersParameter[] = ALL_BROWSERS;

  // @state()
  // implementationProgressChannels: ChannelsParameter[] = ALL_FEATURES;

  @state()
  implementationProgressChartOptions: google.visualization.LineChartOptions = {
    title: 'Implementation Progress',
    hAxis: {title: 'Date'},
    vAxis: {title: 'Percent'},
    series: {
      0: {color: '#3367d6'},
      1: {color: '#d63367'},
    },
  };

  @state()
  startDate: Date = new Date(2021, 1, 1);

  @state()
  endDate: Date = new Date(2024, 4, 1);

  @state()
  implementationProgress: Array<components['schemas']['WPTRunMetric']> = [];

  location!: {params: {featureId: string}; search: string}; // Set by router.

  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        .crumbs {
          color: #aaa;
        }
        .crumbs a {
          text-decoration: none;
        }

        #nameAndOffsiteLinks {
          align-items: center;
        }

        .hbox,
        .vbox {
          gap: var(--content-padding-large);
        }

        .wptScore > div + div {
          margin-top: var(--content-padding-half);
        }
        .wptScore .icon {
          float: right;
        }
        .wptScore .score {
          font-size: 150%;
        }
        .wptScore .avail {
          color: var(--unimportant-text-color);
        }
        .chip.increased {
          background: var(--chip-background-increased);
          color: var(--chip-color-increased);
        }
        .chip.unchanged {
          background: var(--chip-background-unchanged);
          color: var(--chip-color-unchanged);
        }
        .chip.decreased {
          background: var(--chip-background-decreased);
          color: var(--chip-color-decreased);
        }

        #current-bugs li {
          list-style: none;
          margin-bottom: var(--content-padding);
        }

        #general-information .vbox {
          gap: var(--content-padding);
        }

        .info-section-header {
          font-weight: bold;
          width: 16em;
        }

        dt {
          font-weight: bold;
        }

        dd {
          margin-bottom: var(--content-padding-large);
        }

        .under-construction {
          min-height: 12em;
        }

        /* Make the dropdown menu button icon rotate when the menu is open,
          so it looks like sl-select. */
        sl-dropdown > sl-button > sl-icon {
          rotate: 0deg;
          transition: var(--sl-transition-medium) rotate ease;
        }
        sl-dropdown[open] > sl-button > sl-icon {
          rotate: -180deg;
          transition: var(--sl-transition-medium) rotate ease;
        }

        #implementation-progress-chart {
          min-height: 20em;
        }
      `,
    ];
  }

  constructor() {
    super();
    this._loadingTask = new Task(this, {
      args: () => [this.apiClient, this.featureId],
      task: async ([apiClient, featureId]) => {
        if (typeof apiClient === 'object' && typeof featureId === 'string') {
          this.feature = await apiClient.getFeature(featureId);
        }
        return this.feature;
      },
    });
  }

  setupImplementationProgressBrowsersHandler() {
    // Get the implementation progress data browser selector.
    const browserSelectorMenu = this.shadowRoot!.querySelector(
      '#implementation-progress-browser-selector sl-menu'
    ) as SlMenu;
    // Add a listener to the browserSelectorMenu to update the list of
    // browsers in implementationProgressBrowsers.
    browserSelectorMenu.addEventListener('sl-select', event => {
      const menu = event.target as SlMenu;
      const menuItemsArray: Array<SlMenuItem> = Array.from(
        menu.children
      ).filter(child => child instanceof SlMenuItem) as Array<SlMenuItem>;

      // Build the list of values of checked menu-items.
      this.implementationProgressBrowsers = menuItemsArray
        .filter(menuItem => menuItem.checked)
        .map(menuItem => menuItem.value) as BrowsersParameter[];
      // Regenerate data and redraw. We should instead just filter it.
      this.setupImplementationProgressChart();
    });
  }

  setupDateRangeHandler() {
    const startDateInput = this.shadowRoot!.querySelector(
      '#start-date'
    ) as HTMLInputElement;
    startDateInput.addEventListener('sl-blur', event => {
      const currentStartDate = this.startDate;
      this.startDate = new Date((event.target as HTMLInputElement).value);
      if (this.startDate.getTime() === currentStartDate.getTime()) return;
      // Regenerate data and redraw. We should instead just filter it.
      this.setupImplementationProgressChart();
    });
    const endDateInput = this.shadowRoot!.querySelector(
      '#end-date'
    ) as HTMLInputElement;
    endDateInput.addEventListener('sl-blur', event => {
      const currentEndDate = this.endDate;
      this.endDate = new Date((event.target as HTMLInputElement).value);
      if (this.endDate.getTime() === currentEndDate.getTime()) return;
      // Regenerate data and redraw. We should instead just filter it.
      this.setupImplementationProgressChart();
    });
  }

  setupImplementationProgressChart() {
    makeRandomDataForAllBrowserChannelCombos(this.startDate, this.endDate);

    google.charts.load('current', {
      packages: ['corechart'],
    });
    google.charts.setOnLoadCallback(() => {
      // Let's render a chart...
      this.createImplementationProgressChart();
    });

    // Add window resize event handler to redraw the chart.
    window.addEventListener('resize', () => {
      this.createImplementationProgressChart();
    });
  }

  async firstUpdated(): Promise<void> {
    // TODO(jrobbins): Use routerContext instead of this.location so that
    // nested components could also access the router.
    this.featureId = this.location.params.featureId;
  }

  // Make a DataTable from the data in browserChannelDataMap
  createImplementationProgressDataTableFromMap(
    browsers: BrowsersParameter[],
    channel: ChannelsParameter
  ): google.visualization.DataTable {
    const dataTable = new google.visualization.DataTable();
    dataTable.addColumn('date', 'Date');
    for (const browser of browsers) {
      dataTable.addColumn('number', browser);
    }
    dataTable.addColumn('number', 'Total');

    // Map from date to array of counts for each browser
    const dateToBrowserDataMap = new Map<number, Array<number>>();
    // Map from date to array of total_tests_count, the same for all browsers.
    const dateToTotalTestsCountMap = new Map<number, number>();

    // Merge data across all browsers into one array of rows.
    for (const browser of browsers) {
      const data = browserChannelDataMap.get(
        browserChannelDataMapKey(browser, channel)
      );
      if (!data) continue;
      for (const row of data) {
        if (!row) continue;
        const dateSeconds = new Date(row.run_timestamp).getTime();
        const testPassCount = row.test_pass_count!;
        if (!dateToBrowserDataMap.has(dateSeconds)) {
          dateToBrowserDataMap.set(dateSeconds, [testPassCount]);
          dateToTotalTestsCountMap.set(dateSeconds, row.total_tests_count!);
        } else {
          dateToBrowserDataMap.get(dateSeconds)!.push(testPassCount);
        }
      }
    }

    // Sort the dateToBrowserDataMap by dateSeconds
    const data = Array.from(dateToBrowserDataMap.entries()).sort(
      ([d1], [d2]) => d1 - d2
    );

    // For each date, add a row to the dataTable
    for (const row of data) {
      const dateSeconds = row[0];
      const date = new Date(dateSeconds);
      const browserCounts = row[1];
      const total = dateToTotalTestsCountMap.get(dateSeconds)!;
      dataTable.addRow([date, ...browserCounts, total]);
    }
    return dataTable;
  }

  createImplementationProgressChart(): void {
    const data = this.createImplementationProgressDataTableFromMap(
      this.implementationProgressBrowsers,
      'stable'
    );

    // Add 2 weeks to this.endDate.
    const endDate = new Date(this.endDate.getTime() + 1000 * 60 * 60 * 24 * 14);
    const options = {
      hAxis: {
        title: '',
        titleTextStyle: {color: '#333'},
        viewWindow: {min: this.startDate, max: endDate},
      },
      vAxis: {minValue: 0},
      legend: {position: 'top'},
      chartArea: {left: 60, right: 16, top: 40, bottom: 40},
    } as google.visualization.LineChartOptions;

    const chart = new google.visualization.LineChart(
      this.shadowRoot!.getElementById('implementation-progress-chart')!
    );
    chart.draw(data, options);
  }

  afterRenderWhenComplete() {
    this.setupImplementationProgressBrowsersHandler();
    this.setupDateRangeHandler();
    this.setupImplementationProgressChart();
  }

  render(): TemplateResult | undefined {
    return this._loadingTask.render({
      complete: () => this.renderWhenComplete(),
      error: () => this.renderWhenError(),
      initial: () => this.renderWhenInitial(),
      pending: () => this.renderWhenPending(),
    });
  }

  renderCrumbs(): TemplateResult {
    const overviewUrl = formatOverviewPageUrl(this.location);
    const canonicalFeatureUrl = formatFeaturePageUrl(this.feature!);
    return html`
      <div class="crumbs">
        <a href=${overviewUrl}>Feature overview</a>
        &rsaquo;
        <a href=${canonicalFeatureUrl}>${this.feature!.name}</a>
      </div>
    `;
  }

  renderNameAndOffsiteLinks(): TemplateResult {
    const mdnLink = '#TODO';
    const canIUseLink = '#TODO';
    return html`
      <div id="nameAndOffsiteLinks" class="hbox">
        <h1 class="halign-stretch">${this.feature!.name}</h1>
        <sl-button variant="default" href=${mdnLink}>
          <sl-icon slot="suffix" name="box-arrow-up-right"></sl-icon>
          MDN
        </sl-button>
        <sl-button variant="default" href=${canIUseLink}>
          <sl-icon slot="suffix" name="box-arrow-up-right"></sl-icon>
          CanIUse
        </sl-button>
      </div>
    `;
  }

  renderOneWPTCard(
    browser: components['parameters']['browserPathParam'],
    icon: string
  ): TemplateResult {
    const scorePart = this.feature
      ? renderWPTScore(this.feature, {search: ''}, {browser: browser})
      : nothing;

    return html`
      <sl-card class="halign-stretch wptScore">
        <img height="32" src="/public/img/${icon}" class="icon" />
        <div>${browser[0].toUpperCase() + browser.slice(1)}</div>
        <div class="score">
          ${scorePart}
          <span class="chip small increased">+1.2%</span>
        </div>
        <div class="avail">Available since ...</div>
      </sl-card>
    `;
  }

  renderBaselineCard(): TemplateResult {
    if (!this.feature) return html``;

    const chipConfig = BASELINE_CHIP_CONFIGS[this.feature.baseline_status];

    return html`
      <sl-card class="halign-stretch wptScore">
        <img height="28" src="/public/img/${chipConfig.icon}" class="icon" />
        <div>Baseline</div>
        <div class="score">${chipConfig.word}</div>
        <div class="avail">Baseline since ...</div>
      </sl-card>
    `;
  }

  renderWPTScores(): TemplateResult {
    return html`
      <section id="wpt-scores">
        <h3>Web platform test scores</h3>
        <div class="hbox" style="margin:0">
          ${this.renderOneWPTCard('chrome', 'chrome_32x32.png')}
          ${this.renderOneWPTCard('edge', 'edge_32x32.png')}
          ${this.renderOneWPTCard('firefox', 'firefox_32x32.png')}
          ${this.renderOneWPTCard('safari', 'safari_32x32.png')}
          ${this.renderBaselineCard()}
        </div>
      </section>
    `;
  }

  renderImplentationProgress(): TemplateResult {
    return html`
      <sl-card id="implementation-progress">
        <div slot="header" class="hbox">
          Implementation progress
          <div class="spacer"></div>
          <sl-select>
            <sl-option> All features </sl-option>
            <sl-option> how to select ? </sl-option>
          </sl-select>
          <sl-dropdown
            id="implementation-progress-browser-selector"
            multiple
            stay-open-on-select
            .value="${this.implementationProgressBrowsers.join(' ')}"
          >
            <sl-button slot="trigger">
              <sl-icon slot="suffix" name="chevron-down"> </sl-icon>
              Browsers
            </sl-button>
            <sl-menu>
              <sl-menu-item type="checkbox" value="Chrome">
                Chrome
              </sl-menu-item>
              <sl-menu-item type="checkbox" value="Edge"> Edge </sl-menu-item>
              <sl-menu-item type="checkbox" value="Firefox">
                Firefox
              </sl-menu-item>
              <sl-menu-item type="checkbox" value="Safari">
                Safari
              </sl-menu-item>
            </sl-menu>
          </sl-dropdown>
        </div>
        <webstatus-chart
          id="implementation-progress-chart"
          .data="${this.createImplementationProgressDataTableFromMap(
            this.implementationProgressBrowsers,
            'stable'
          )}"
          .options="${this.implementationProgressChartOptions}"
        >
          Loading chart...
        </webstatus-chart>
      </sl-card>
    `;
  }

  renderBug(bugId: number): TemplateResult {
    return html`
      <li>
        <a href="#TODO" target="_blank">
          <img height="16" src="/public/img/chrome_24x24.png" />
          ${bugId}: Title of issue
        </a>
      </li>
    `;
  }

  renderCurrentBugs(): TemplateResult {
    return html`
      <sl-details id="current-bugs">
        <div slot="summary">Current bugs</div>
        <ul class="under-construction">
          ${[21830, 123412, 12983712, 1283, 987123, 12982, 1287].map(bugId =>
            this.renderBug(bugId)
          )}
        </ul>
      </sl-details>
    `;
  }

  renderWhenComplete(): TemplateResult {
    return html`
      <div class="vbox">
        ${this.renderCrumbs()} ${this.renderNameAndOffsiteLinks()}
        ${this.renderWPTScores()} ${this.renderImplentationProgress()}
      </div>
    `;

    // TODO: Fetch and display current bugs.
    //   ${this.renderCurrentBugs()}
  }

  renderWhenError(): TemplateResult {
    return html`Error when loading feature ${this.featureId}.`;
  }

  renderWhenInitial(): TemplateResult {
    return html`Preparing request for ${this.featureId}.`;
  }

  renderWhenPending(): TemplateResult {
    return html`Loading ${this.featureId}.`;
  }
}
