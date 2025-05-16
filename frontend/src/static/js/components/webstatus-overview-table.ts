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
import {TaskStatus} from '@lit/task';
import {range} from 'lit/directives/range.js';
import {map} from 'lit/directives/map.js';
import {customElement, property} from 'lit/decorators.js';
import {SHARED_STYLES} from '../css/shared-css.js';
import {type components} from 'webstatus.dev-backend';
import {
  ColumnKey,
  CELL_DEFS,
  renderFeatureCell,
  renderColgroups,
} from './webstatus-overview-cells.js';
import {ApiError, BadRequestError} from '../api/errors.js';
import {
  GITHUB_REPO_ISSUE_LINK,
  SEARCH_QUERY_README_LINK,
} from '../utils/constants.js';
import {CurrentSavedSearch} from '../contexts/app-bookmark-info-context.js';
import {TaskTracker} from '../utils/task-tracker.js';

@customElement('webstatus-overview-table')
export class WebstatusOverviewTable extends LitElement {
  @property({type: Object})
  taskTracker: TaskTracker<components['schemas']['Feature'][], ApiError> = {
    status: TaskStatus.INITIAL, // Initial state
    error: undefined,
    data: undefined,
  };

  @property({attribute: false})
  columns: ColumnKey[] = [];

  @property({attribute: false})
  groupCells: TemplateResult[] = [];

  @property({attribute: false})
  headerCells: TemplateResult[] = [];

  @property({type: Object})
  location!: {search: string}; // Set by parent.

  @property({type: Object})
  savedSearch: CurrentSavedSearch;

  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        .data-table {
          margin: var(--content-padding) 0;
        }
        .data-table th,
        .data-table th {
          padding: var(--content-padding-quarter) var(--content-padding);
        }
        .header-row th {
          border-left: var(--default-border);
          border-right: var(--default-border);
          height: 28px;
        }
        .header-row th.sortable {
          min-width: 4em;
        }
        .header-row th.sortable:hover {
          background: var(--table-header-hover-background);
        }
        th span.hover-only {
          display: none;
        }
        th:hover span.hover-only {
          display: inline;
        }
        .baseline-date-block {
          padding-top: var(--content-padding-quarter);
        }
        .browser-impl-unavailable img {
          filter: grayscale(1);
          opacity: 50%;
        }
        .percent {
          display: inline-block;
          width: 6ex;
          text-align: right;
        }
        .missing.percent,
        tr .centered {
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

        .feature-name-cell {
          display: inline-flex;
          gap: 4px;
        }

        img.platform {
          margin-left: calc(-0.3 * var(--platform-logo-size));
          height: var(--platform-logo-size);
          width: var(--platform-logo-size);
        }
        .browser-cell {
          font-size: 22px;
          font-weight: 300;
        }
        .availability-icon {
          margin-right: calc(-0.3 * var(--platform-logo-size));
        }
      `,
    ];
  }

  render(): TemplateResult {
    return html`
      <table class="data-table">
        ${renderColgroups(this.columns)}
        <thead>
          ${this.groupCells}
          <tr class="header-row">
            ${this.headerCells}
          </tr>
        </thead>
        <tbody>
          ${this.renderTableBody(this.columns)}
        </tbody>
      </table>
    `;
  }

  renderTableBody(columns: ColumnKey[]): TemplateResult {
    switch (this.taskTracker.status) {
      case TaskStatus.INITIAL:
        return this.renderBodyWhenPending(columns);
      case TaskStatus.PENDING:
        return this.renderBodyWhenPending(columns);
      case TaskStatus.COMPLETE:
        return this.taskTracker.data?.length === 0
          ? this.renderBodyWhenNoResults(columns)
          : this.renderBodyWhenComplete(columns);
      case TaskStatus.ERROR:
        return this.renderBodyWhenError(columns);
    }
  }

  renderBodyWhenComplete(columns: ColumnKey[]): TemplateResult {
    return html`
      ${this.taskTracker.data?.map(f => this.renderFeatureRow(f, columns))}
    `;
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

  renderBodyWhenError(columns: ColumnKey[]): TemplateResult {
    if (this.taskTracker.error instanceof BadRequestError) {
      return html`
        <tr>
          <td class="message" colspan=${columns.length}>
            <div>Invalid query...</div>
            <div>
              Please review the
              <a href="${SEARCH_QUERY_README_LINK}" target="_blank"
                >search syntax</a
              >
              or
              <a href="${GITHUB_REPO_ISSUE_LINK}" target="_blank"
                >report an error</a
              >.
            </div>
          </td>
        </tr>
      `;
    }
    return html`
      <tr>
        <td class="message" colspan=${columns.length}>
          <div>Something went wrong...</div>
          <div>We had some trouble loading this data.</div>
          <div>
            Please refresh the page or
            <a href="${GITHUB_REPO_ISSUE_LINK}" target="_blank"
              >report an error</a
            >.
          </div>
        </td>
      </tr>
    `;
  }

  renderBodyWhenPending(columns: ColumnKey[]): TemplateResult {
    const DEFAULT_SKELETON_ROWS = 10;
    const skeleton_rows =
      this.taskTracker.data?.length || DEFAULT_SKELETON_ROWS;
    return html`
      ${map(
        range(skeleton_rows),
        () => html`
          <tr>
            ${columns.map(col => html` <td>${this.renderShimmer(col)}</td> `)}
          </tr>
        `,
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
    columns: ColumnKey[],
  ): TemplateResult {
    return html`
      <tr>
        ${columns.map(
          col => html`
            <td class=${CELL_DEFS[col].cellClass ?? ''}>
              ${renderFeatureCell(feature, this.location, col)}
            </td>
          `,
        )}
      </tr>
    `;
  }
}
