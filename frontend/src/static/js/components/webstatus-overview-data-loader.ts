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
  renderHeaderCell,
  renderSavedSearchHeaderCells,
} from './webstatus-overview-cells.js';
import './webstatus-overview-table.js';
import {TaskTracker} from '../utils/task-tracker.js';
import {ApiError} from '../api/errors.js';
import {Toast} from '../utils/toast.js';
import {
  CurrentSavedSearch,
  SavedSearchScope,
} from '../contexts/app-bookmark-info-context.js';

@customElement('webstatus-overview-data-loader')
export class WebstatusOverviewDataLoader extends LitElement {
  @property({type: Object})
  taskTracker: TaskTracker<components['schemas']['FeaturePage'], ApiError> = {
    status: TaskStatus.INITIAL, // Initial state
    error: undefined,
    data: undefined,
  };

  @property({type: Object})
  location!: {search: string}; // Set by parent.

  @property({type: Object})
  savedSearch: CurrentSavedSearch;

  findFeaturesFromAtom(
    searchKey: string,
    searchValue: string,
  ): components['schemas']['Feature'][] {
    if (!this.taskTracker.data?.data) {
      return [];
    }

    const features: components['schemas']['Feature'][] = [];
    for (const feature of this.taskTracker.data.data) {
      if (searchKey === 'id' && feature?.feature_id === searchValue) {
        features.push(feature);
        break;
      } else if (
        searchKey === 'name' &&
        (feature?.feature_id.includes(searchValue) ||
          feature?.name.includes(searchValue))
      ) {
        features.push(feature);
      }
    }
    return features;
  }

  reorderByQueryTerms(): components['schemas']['Feature'][] | undefined {
    if (
      !this.savedSearch ||
      this.savedSearch.scope !== SavedSearchScope.GlobalSavedSearch ||
      !this.savedSearch.value.is_ordered
    ) {
      return undefined;
    }

    const atoms: string[] = this.savedSearch.value.query.trim().split('OR');
    const features = [];
    for (const atom of atoms) {
      const terms = atom.trim().split(':');
      const foundFeatures = this.findFeaturesFromAtom(terms[0], terms[1]);
      if (foundFeatures) {
        features.push(...foundFeatures);
      }
    }

    if (features.length !== this.taskTracker.data?.data?.length) {
      void new Toast().toast(
        `Unable to apply custom sorting to saved search "${this.savedSearch.value.name}". Defaulting to normal sorting.`,
        'warning',
        'exclamation-triangle',
      );
      return undefined;
    }
    return features;
  }

  render(): TemplateResult {
    const columns: ColumnKey[] = parseColumnsSpec(
      getColumnsSpec(this.location),
    );
    const sortSpec: string =
      getSortSpec(this.location) || (DEFAULT_SORT_SPEC as string);

    let headerCells: TemplateResult[] = [];
    if (
      this.savedSearch?.scope === SavedSearchScope.GlobalSavedSearch &&
      this.savedSearch.value?.is_ordered
    ) {
      headerCells = renderSavedSearchHeaderCells(
        this.savedSearch.value.name,
        columns,
      );
    } else {
      headerCells = columns.map(
        col => html`${renderHeaderCell(this.location, col, sortSpec)}`,
      );
    }

    if (
      this.taskTracker.status === TaskStatus.COMPLETE &&
      this.taskTracker.data
    ) {
      this.taskTracker.data.data =
        this.reorderByQueryTerms() || this.taskTracker.data?.data;
    }

    return html`<webstatus-overview-table
      .columns=${columns}
      .headerCells=${headerCells}
      .location=${this.location}
      .isLoading=${this.taskTracker.status === TaskStatus.PENDING ||
      this.taskTracker.status === TaskStatus.INITIAL}
      .dataError=${this.taskTracker.error}
      .data=${this.taskTracker?.data?.data}
    ></webstatus-overview-table>`;
  }
}
