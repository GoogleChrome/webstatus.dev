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

import {type TemplateResult, html, CSSResultGroup, css, nothing} from 'lit';
import {customElement} from 'lit/decorators.js';
import {getFeaturesLaggingFlag} from '../utils/urls.js';
import {BaseChartsPage} from './webstatus-base-charts-page.js';

import './webstatus-stats-global-feature-count-chart-panel.js';
import './webstatus-stats-missing-one-impl-chart-panel.js';

@customElement('webstatus-stats-page')
export class StatsPage extends BaseChartsPage {
  // Change the default start date to Jan 1, 2020.
  override startDate: Date = new Date(2020, 0, 1);
  static get styles(): CSSResultGroup {
    return [
      super.styles!,
      css`
        /*  Make the dropdown menu button icon rotate when the menu is open,
            so it looks like sl-select. */
        sl-dropdown > sl-button > sl-icon {
          rotate: 0deg;
          transition: var(--sl-transition-medium) rotate ease;
        }
        sl-dropdown[open] > sl-button > sl-icon {
          rotate: -180deg;
          transition: var(--sl-transition-medium) rotate ease;
        }
      `,
    ];
  }
  constructor() {
    super();
  }

  renderTitleAndControls(): TemplateResult {
    return html`
      <div id="titleAndControls" class="hbox">
        <h1>Statistics</h1>
        <div class="spacer"></div>
        ${this.renderDateRangePicker()}
      </div>
    `;
  }

  renderGlobalFeatureSupport(): TemplateResult {
    return html`
      <webstatus-stats-global-feature-chart-panel
        .startDate=${this.startDate}
        .endDate=${this.endDate}
      ></webstatus-stats-global-feature-chart-panel>
    `;
  }

  renderFeaturesLagging(): TemplateResult {
    return html`
      <webstatus-stats-missing-one-impl-chart-panel
        .startDate=${this.startDate}
        .endDate=${this.endDate}
      >
      </webstatus-stats-missing-one-impl-chart-panel>
    `;
  }

  render(): TemplateResult {
    return html`
      <div class="vbox">
        ${this.renderTitleAndControls()} ${this.renderGlobalFeatureSupport()}
        ${getFeaturesLaggingFlag(this.location)
          ? this.renderFeaturesLagging()
          : nothing}
      </div>
    `;
  }
}
