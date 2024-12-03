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
import {TaskTracker} from '../utils/task-tracker.js';
import {ApiError, BadRequestError} from '../api/errors.js';
import {
  GITHUB_REPO_ISSUE_LINK,
  SEARCH_QUERY_README_LINK,
  Bookmark,
} from '../utils/constants.js';
import {threadId} from 'worker_threads';

@customElement('webstatus-overview-table')
export class WebstatusOverviewTable extends LitElement {
  @state()
  taskTracker: TaskTracker<components['schemas']['FeaturePage'], ApiError> = {
    status: TaskStatus.INITIAL, // Initial state
    error: null,
    data: null,
  };

  @state()
  location!: {search: string}; // Set by parent.

  @state()
  bookmark: Bookmark | undefined; // Set by parent.

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
        .baseline-date-block {
          padding-top: var(--content-padding-quarter);
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

  sortDataOrder() {
    if (!this.bookmark || !this.bookmark.is_ordered) {
      return;
    }
    const atoms: string[] = this.bookmark.query.trim().split(' ');
    const results = [];
    for (const atom in atoms) {
      let found = null;
      const terms = atom.split(':');
      for (const d in this.taskTracker.data) {
        if (terms[0] === 'id') {
          if (d.feature_id == terms[1]) {
            found = d;
          }
        } else if (terms[0] == 'name') {
          if (d.feature_id.startsWith(terms[1])) {
            found = d;
          }
        }
        if (found) {
          results.push(found);
          break;
        }
      }
    }
  }

  render(): TemplateResult {
    const columns: ColumnKey[] = parseColumnsSpec(
      getColumnsSpec(this.location),
    );
    const sortSpec: string =
      getSortSpec(this.location) || (DEFAULT_SORT_SPEC as string);

    return html`
      <table class="data-table">
        <thead>
          <tr>
            ${columns.map(
              col => html`${renderHeaderCell(this.location, col, sortSpec)}`,
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
    switch (this.taskTracker.status) {
      case TaskStatus.COMPLETE:
        return this.taskTracker.data?.data?.length === 0
          ? this.renderBodyWhenNoResults(columns)
          : this.renderBodyWhenComplete(columns);
      case TaskStatus.ERROR:
        return this.renderBodyWhenError(columns);
      case TaskStatus.INITIAL:
        // Do the same thing as pending.
        return this.renderBodyWhenPending(columns);
      case TaskStatus.PENDING:
        return this.renderBodyWhenPending(columns);
    }
  }

  renderBodyWhenComplete(columns: ColumnKey[]): TemplateResult {
    return html`
      ${this.taskTracker.data?.data?.map(f =>
        this.renderFeatureRow(f, columns),
      )}
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
      this.taskTracker.data?.data?.length || DEFAULT_SKELETON_ROWS;
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
            <td>${renderFeatureCell(feature, this.location, col)}</td>
          `,
        )}
      </tr>
    `;
  }
}
