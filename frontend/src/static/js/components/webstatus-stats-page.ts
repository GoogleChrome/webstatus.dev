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

//import {consume} from '@lit/context';
//import {Task} from '@lit/task';
import {LitElement, type TemplateResult, html, CSSResultGroup, css} from 'lit';
import {customElement} from 'lit/decorators.js';
import {SHARED_STYLES} from '../css/shared-css.js';
//import {type components} from 'webstatus.dev-backend';

//import {type APIClient} from '../api/client.js';
//import {apiClientContext} from '../contexts/api-client-context.js';

@customElement('webstatus-stats-page')
export class StatsPage extends LitElement {

  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        .hbox,
        .vbox {
          gap: var(--content-padding-large);
        }

        #titleAndControls {
          align-items: center;
        }

        .under-construction {
          min-height: 12em;
        }
      `,
    ];
  }

  render(): TemplateResult | undefined {
    return this.renderWhenComplete();
  }

  renderTitleAndControls(): TemplateResult {
    return html`
      <div id="titleAndControls" class="hbox">
        <h1 class="hgrow">Statistics</h1>
        <sl-checkbox>Show browser versions</sl-checkbox>
        <sl-button href="#TODO">
          <sl-icon slot="prefix" name="calendar-blank" library="phosphor"></sl-icon>
          Select range
        </sl-button>
        <sl-radio-group>
          <sl-radio-button value="WPT">WPT</sl-radio-button>
          <sl-radio-button value="BCD">BCD</sl-radio-button>
        <sl-radio-group>
      </div>
    `;
  }

  renderGlobalFeatureSupport(): TemplateResult {
    return html`
      <sl-card id="global-featur-support">
        <div slot="header">Global feature support</div>
        <p class="under-construction">Chart goes here...</p>
      </sl-card>
    `;
  }

  renderFeaturesLagging(): TemplateResult {
    return html`
      <sl-card id="features-lagging">
        <div slot="header">Features missing in only 1 browser</div>
        <p class="under-construction">Chart goes here...</p>
      </sl-card>
    `;
  }

  renderBaselineFeatures(): TemplateResult {
    return html`
      <sl-card class="hgrow" id="baseline-features">
        <div slot="header">Baseline features</div>
        <p class="under-construction">Small chart goes here...</p>
      </sl-card>
    `;
  }

  renderTimeToAvailability(): TemplateResult {
    return html`
      <sl-card class="hgrow" id="time-to-availibility">
        <div slot="header">Time to availablity</div>
        <p class="under-construction">Small chart goes here...</p>
      </sl-card>
    `;
  }

  renderWhenComplete(): TemplateResult {
    return html`
      <div class="vbox">
        ${this.renderTitleAndControls()}
        ${this.renderGlobalFeatureSupport()}
        ${this.renderFeaturesLagging()}
        <div class="hbox">
          ${this.renderBaselineFeatures()} ${this.renderTimeToAvailability()}
        </div>
      </div>
    `;
  }
}
