/**
 * Copyright 2024 Google LLC
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

import {type components} from 'webstatus.dev-backend';

function getQueryParam(qs: string, paramName: string): string {
  const params = new URLSearchParams(qs);
  return params.get(paramName) || '';
}

export function getSearchQuery(location: {search: string}): string {
  return getQueryParam(location.search, 'q');
}

export function getColumnsSpec(location: {search: string}): string {
  return getQueryParam(location.search, 'columns');
}

export function getSortSpec(location: {search: string}): string {
  return getQueryParam(location.search, 'sort');
}

export function getPaginationStart(location: {search: string}): number {
  return Number(getQueryParam(location.search, 'start'));
}

export function getWPTMetricView(location: {search: string}): string {
  return getQueryParam(location.search, 'wpt_metric_view');
}

export interface DateRange {
  start?: Date;
  end?: Date;
}

// getDate is used to get the date range specified in the URL.
export function getDateRange(location: {search: string}): DateRange {
  const start = getQueryParam(location.search, 'startDate');
  const end = getQueryParam(location.search, 'endDate');

  return {
    start: start ? new Date(start) : undefined,
    end: end ? new Date(end) : undefined,
  };
}

export const DEFAULT_ITEMS_PER_PAGE = 25;
export function getPageSize(location: {search: string}): number {
  const num = Number(
    getQueryParam(location.search, 'num') || DEFAULT_ITEMS_PER_PAGE
  );
  return Math.min(100, Math.max(num, 1));
}

type QueryStringOverrides = {
  q?: string;
  start?: number;
  num?: number;
  sort?: string;
  columns?: string[];
  wpt_metric_view?: string;
  dateRange?: DateRange;
};

/* Given the router location object, return a query string with
   parameters that maintain the user's navigational state.
   E.g., if I start searching for 'mouse', then as I navigate
   around, I should still be searching for 'mouse'. */
function getContextualQueryStringParams(
  location: {search: string} | undefined,
  overrides: QueryStringOverrides = {}
): string {
  if (location === undefined) {
    return '';
  }
  const searchParams = new URLSearchParams();
  const searchQuery = 'q' in overrides ? overrides.q : getSearchQuery(location);
  if (searchQuery) {
    searchParams.set('q', searchQuery);
  }
  const colSpec =
    'columns' in overrides
      ? overrides.columns!.join(',')
      : getColumnsSpec(location);
  if (colSpec) {
    searchParams.set('columns', colSpec);
  }
  const sortSpec = 'sort' in overrides ? overrides.sort : getSortSpec(location);
  if (sortSpec) {
    searchParams.set('sort', sortSpec);
  }
  const start =
    'start' in overrides ? overrides.start : getPaginationStart(location);
  if (start) {
    searchParams.set('start', '' + start);
  }

  const num = 'num' in overrides ? overrides.num : getPageSize(location);
  if (num !== DEFAULT_ITEMS_PER_PAGE) {
    searchParams.set('num', '' + num);
  }

  const wptMetricView =
    'wpt_metric_view' in overrides
      ? overrides.wpt_metric_view
      : getWPTMetricView(location);
  if (wptMetricView) {
    searchParams.set('wpt_metric_view', wptMetricView);
  }

  const dateRange =
    'dateRange' in overrides ? overrides.dateRange : getDateRange(location);
  if (dateRange?.start) {
    // format startDate as yyyy-mm-dd
    const startDate = dateRange.start.toISOString().split('T')[0];
    searchParams.set('startDate', startDate);
  }
  if (dateRange?.end) {
    // format endDate as yyyy-mm-dd
    const endDate = dateRange.end.toISOString().split('T')[0];
    searchParams.set('endDate', endDate);
  }

  return searchParams.toString() ? '?' + searchParams.toString() : '';
}

/* Return a URL for the overview (feature list) page. */
export function formatOverviewPageUrl(
  location?: {search: string},
  overrides: QueryStringOverrides = {}
): string {
  const qs = getContextualQueryStringParams(location, overrides);
  return `/${qs}`;
}

/* Return a URL to the given feature. */
export function formatFeaturePageUrl(
  feature: components['schemas']['Feature'],
  location?: {search: string},
  overrides: QueryStringOverrides = {}
): string {
  const qs = getContextualQueryStringParams(location, overrides);
  return `/features/${feature.feature_id}${qs}`;
}

/* Update URL for the feature page */
export function updateFeaturePageUrl(
  feature: components['schemas']['Feature'],
  location: {search: string},
  overrides: QueryStringOverrides = {}
): void {
  const url = formatFeaturePageUrl(feature, location, overrides);
  window.history.replaceState({}, '', url);
}
