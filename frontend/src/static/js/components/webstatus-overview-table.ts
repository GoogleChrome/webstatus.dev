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
import {Task} from '@lit/task';
import {range} from 'lit/directives/range.js';
import {map} from 'lit/directives/map.js';
import {customElement, state} from 'lit/decorators.js';
import {SHARED_STYLES} from '../css/shared-css.js';
import {type components} from 'webstatus.dev-backend';
import {getColumnsSpec, getSortSpec} from '../utils/urls.js';
import {
  ColumnKey,
  DEFAULT_SORT_SPEC,
  parseColumnsSpec,
  renderFeatureCell,
  renderHeaderCell,
} from './webstatus-overview-cells.js';

@customElement('webstatus-overview-table')
export class WebstatusOverviewTable extends LitElement {
  @state()
  features: Array<components['schemas']['Feature']> = [];

  loadingTask!: Task; // Set by parent.

  @state()
  location!: {search: string}; // Set by parent.

  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        .data-table {
          margin: var(--content-padding) 0;
        }
        .data-table th:hover {
          background: var(--table-header-hover-background);
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
        .browser-impl-unavailable {
          color: var(--icon-color-avail-unavailable);
        }
        .percent {
          display: inline-block;
          width: 6ex;
          text-align: right;
        }
        .missing.percent {
          text-align: center;
        }

        td.message {
          height: 8em;
          text-align: center;
        }
        td.message div:first-child {
          font-size: 110%;
          font-weight: bold;
          padding-bottom: var(--content-padding-half);
        }

        sl-skeleton {
          width: 4em;
        }
        sl-skeleton.col-name {
          width: 6em;
        }
        tr:nth-of-type(even) sl-skeleton.col-name {
          width: 9em;
        }
        sl-skeleton.col-baseline_status {
          width: 5em;
        }
        tr:nth-of-type(even) sl-skeleton.col-baseline_status {
          width: 7em;
        }
      `,
    ];
  }

  render(): TemplateResult {
    const columns: ColumnKey[] = parseColumnsSpec(
      getColumnsSpec(this.location)
    );
    const sortSpec: string =
      getSortSpec(this.location) || (DEFAULT_SORT_SPEC as string);

    return html`
      <table class="data-table">
        <thead>
          <tr>
            ${columns.map(
              col => html`${renderHeaderCell(this.location, col, sortSpec)}`
            )}
          </tr>
        </thead>
        <tbody>
          ${this.renderTableBody(columns)}
        </tbody>
      </table>
    `;
  }

  renderTableBody(columns: ColumnKey[]): TemplateResult {
    return this.loadingTask.render({
      complete: () => {
        return this.features.length === 0
          ? this.renderBodyWhenNoResults(columns)
          : this.renderBodyWhenComplete(columns);
      },
      error: () => this.renderBodyWhenError(columns),
      initial: () => this.renderBodyWhenInitial(columns),
      pending: () => this.renderBodyWhenPending(columns),
    });
  }

  renderBodyWhenComplete(columns: ColumnKey[]): TemplateResult {
    return html` ${this.features.map(f => this.renderFeatureRow(f, columns))} `;
  }

  renderBodyWhenNoResults(columns: ColumnKey[]): TemplateResult {
    return html`
      <tr>
        <td class="message" colspan=${columns.length}>
          <div>This query produced zero results.</div>
          <div>Try removing some query terms.</div>
        </td>
      </tr>
    `;
  }

  // TODO(jrobbins): This never gets called, even when request fails.
  renderBodyWhenError(columns: ColumnKey[]): TemplateResult {
    return html`
      <tr>
        <td class="message" colspan=${columns.length}>
          <div>Something went wrong...</div>
          <div>We had some trouble loading this data.</div>
          <div>
            Please refresh the page or
            <a href="#TODO" target="_blank">report an error</a>.
          </div>
        </td>
      </tr>
    `;
  }

  renderBodyWhenInitial(columns: ColumnKey[]): TemplateResult {
    return html`
      <tr>
        <td class="message" colspan=${columns.length}>
          <div>Requesting data...</div>
        </td>
      </tr>
    `;
  }

  renderBodyWhenPending(columns: ColumnKey[]): TemplateResult {
    const DEFAULT_SKELETON_ROWS = 10;
    const skeleton_rows = this.features?.length || DEFAULT_SKELETON_ROWS;
    return html`
      ${map(
        range(skeleton_rows),
        () => html`
          <tr>
            ${columns.map(col => html` <td>${this.renderShimmer(col)}</td> `)}
          </tr>
        `
      )}
    `;
  }

  renderShimmer(column: ColumnKey) {
    return html`
      <sl-skeleton effect="sheen" class="col-${column}"></sl-skeleton>
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
