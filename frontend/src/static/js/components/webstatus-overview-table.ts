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
import {LitElement, type TemplateResult, html, CSSResultGroup, css} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {SHARED_STYLES} from '../css/shared-css.js';
import {type components} from 'webstatus.dev-backend';
import {formatFeaturePageUrl} from '../utils/urls.js';

const BASELINE_CHIP_CLASSES = {
  none: 'limited',
  low: 'newly',
  high: 'widely',
};

const BASELINE_CHIP_ICONS = {
  none: 'cross.svg',
  low: 'cross.svg', // TODO(jrobbins): need dotted check
  high: 'check.svg',
};

const BASELINE_CHIP_WORDS = {
  none: 'Limited',
  low: 'New',
  high: 'Widely available',
};

@customElement('webstatus-overview-table')
export class WebstatusOverviewTable extends LitElement {
  @state()
  features: Array<components['schemas']['Feature']> = [];

  location!: {search: string}; // Set by parent.

  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        .data-table {
          margin: var(--content-padding) 0;
        }
        .limited {
          background: var(--chip-background-limited);
          color: var(--chip-color-limited);
        }
        .newly {
          background: var(--chip-background-newly);
          color: var(--chip-color-newly);
        }
        .widely {
          background: var(--chip-background-widely);
          color: var(--chip-color-widely);
        }
      `,
    ];
  }

  render(): TemplateResult {
    return html`
      <table class="data-table">
        <thead>
          <tr>
            <th>Feature Name</th>
            <th>Baseline</th>
            <th><img src="/public/img/chrome-dev_24x24.png" /></th>
            <th><img src="/public/img/firefox-nightly_24x24.png" /></th>
            <th><img src="/public/img/safari-preview_24x24.png" /></th>
          </tr>
        </thead>
        <tbody>
          ${this.features.map(f => this.renderFeatureRow(f))}
        </tbody>
      </table>
    `;
  }

  renderBaselineChip(baselineStatus: string): TemplateResult {
    let baselineKey: keyof typeof BASELINE_CHIP_CLASSES = 'none';
    if (baselineStatus === 'low') {
      baselineKey = 'low';
    }
    if (baselineStatus === 'high') {
      baselineKey = 'high';
    }
    const chipClass = BASELINE_CHIP_CLASSES[baselineKey];
    const chipIcon = BASELINE_CHIP_ICONS[baselineKey];
    const chipWords = BASELINE_CHIP_WORDS[baselineKey];
    return html`
      <span class="chip ${chipClass}">
        <img height="24" src="/public/img/${chipIcon}" />
        ${chipWords}
      </span>
    `;
  }

  renderFeatureRow(feature: components['schemas']['Feature']): TemplateResult {
    const featureUrl = formatFeaturePageUrl(feature, this.location);
    return html`
      <tr>
        <td><a href=${featureUrl}>${feature.name}</a></td>
        <td>${this.renderBaselineChip(feature.baseline_status)}</td>
        <td>100%</td>
        <td>100%</td>
        <td>100%</td>
      </tr>
    `;
  }
}
