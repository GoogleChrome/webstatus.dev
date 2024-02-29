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
import {getColumnsSpec} from '../utils/urls.js';
import {
  ColumnKey,
  parseColumnsSpec,
  renderFeatureCell,
  renderHeaderCell,
} from './webstatus-overview-cells.js';

const DEFAULT_COLUMNS = [
  ColumnKey.Name,
  ColumnKey.BaselineStatus,
  ColumnKey.WptChrome,
  ColumnKey.WptEdge,
  ColumnKey.WptFirefox,
  ColumnKey.WptSafari,
];

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
    const columns = parseColumnsSpec(
      getColumnsSpec(this.location),
      DEFAULT_COLUMNS
    );
    return html`
      <table class="data-table">
        <thead>
          <tr>
            ${columns.map(col => html` <th>${renderHeaderCell(col)}</th>`)}
          </tr>
        </thead>
        <tbody>
          ${this.features.map(f => this.renderFeatureRow(f, columns))}
        </tbody>
      </table>
    `;
  }

  renderFeatureRow(
    feature: components['schemas']['Feature'],
    columns: ColumnKey[]
  ): TemplateResult {
    return html`
      <tr>
        ${columns.map(
          col => html`
            <td>${renderFeatureCell(feature, this.location, col)}</td>
          `
        )}
      </tr>
    `;
  }
}
