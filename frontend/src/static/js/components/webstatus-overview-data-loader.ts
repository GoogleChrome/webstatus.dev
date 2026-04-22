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
import {LitElement, type TemplateResult, html} from 'lit';
import {TaskStatus} from '@lit/task';
import {customElement, property} from 'lit/decorators.js';
import {type components} from 'webstatus.dev-backend';
import {getColumnsSpec, getSortSpec} from '../utils/urls.js';
import {
  ColumnKey,
  DEFAULT_SORT_SPEC,
  parseColumnsSpec,
  renderGroupCells,
  renderHeaderCells,
  renderSavedSearchHeaderCells,
} from './webstatus-overview-cells.js';
import './webstatus-overview-table.js';
import {TaskTracker} from '../utils/task-tracker.js';
import {ApiError} from '../api/errors.js';
import {
  CurrentSavedSearch,
  SavedSearchScope,
} from '../contexts/app-bookmark-info-context.js';
import {type SuccessResponsePageableData} from '../api/client.js';

@customElement('webstatus-overview-data-loader')
export class WebstatusOverviewDataLoader extends LitElement {
  @property({type: Object})
  taskTracker: TaskTracker<
    SuccessResponsePageableData<'/v1/features'>,
    ApiError
  > = {
    status: TaskStatus.INITIAL, // Initial state
    error: undefined,
    data: undefined,
  };

  @property({type: Object})
  location!: {search: string}; // Set by parent.

  @property({type: Object})
  savedSearch: CurrentSavedSearch;

  render(): TemplateResult {
    const columns: ColumnKey[] = parseColumnsSpec(
      getColumnsSpec(this.location),
    );
    const location = this.location;
    if (!location) return html``;
    const sortSpec = getSortSpec(location) || DEFAULT_SORT_SPEC;
    const groupCells = renderGroupCells(location, columns, sortSpec!);
    let headerCells: TemplateResult[] = [];
    const search = this.savedSearch;
    if (
      search?.scope === SavedSearchScope.GlobalSavedSearch &&
      search.value.is_ordered
    ) {
      headerCells = renderSavedSearchHeaderCells(search.value.name, columns);
    } else {
      headerCells = renderHeaderCells(location, columns, sortSpec!);
    }

    const featureTaskTracker: TaskTracker<
      components['schemas']['Feature'][],
      ApiError
    > = {
      status: this.taskTracker.status,
      error: this.taskTracker.error,
      data: this.taskTracker.data?.data,
    };

    return html`<webstatus-overview-table
      .columns=${columns}
      .groupCells=${groupCells}
      .headerCells=${headerCells}
      .location=${this.location}
      .taskTracker=${featureTaskTracker}
    ></webstatus-overview-table>`;
  }
}
