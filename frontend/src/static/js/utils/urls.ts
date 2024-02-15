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

/* Given the router location object, return a query string with
   parameters that maintain the user's navigational state.
   E.g., if I start searching for 'mouse', then as I navigate
   around, I should still be searching for 'mouse'. */
function getContextualQueryStringParams(
  location: {search: string} | undefined): string {
  if (location === undefined) {
    return '';
  }
  const parts: string[] = [];
  if (getSearchQuery(location)) {
    parts.push('q=' + getSearchQuery(location));
  }
  // TODO(jrobbins): Pagination, sorting, columns, etc.
  if (parts.length > 0) {
    return '?' + parts.join('&');
  }
  return '';
}

/* Return a URL for the overview (feature list) page. */
export function formatOverviewPageUrl(location?: {search: string}): string {
  const qs = getContextualQueryStringParams(location);
  return `/${qs}`;
}

/* Return a URL to the given feature. */
export function formatFeaturePageUrl(
  feature: components['schemas']['Feature'],
  location?: {search: string}
): string {
  const qs = getContextualQueryStringParams(location);
  return `/features/${feature.feature_id}${qs}`;
}
