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
export const GITHUB_REPO_ISSUE_LINK =
  'https://github.com/GoogleChrome/webstatus.dev/issues/new/choose';
export const SEARCH_QUERY_README_LINK =
  'https://github.com/GoogleChrome/webstatus.dev/blob/main/antlr/FeatureSearch.md';
export const ABOUT_PAGE_LINK =
  'https://github.com/GoogleChrome/webstatus.dev/wiki/About-Web-Platform-Status';

export type BookmarkPermissions =
  components['schemas']['UserSavedSearchPermissions'];
export const BookmarkOwnerRole: components['schemas']['UserSavedSearchPermissions']['role'] =
  'saved_search_owner';

export type BookmarkStatus = components['schemas']['UserSavedSearchBookmark'];
export const BookmarkStatusActive: components['schemas']['UserSavedSearchBookmark']['status'] =
  'bookmark_active';

export type SavedSearchOperationType = 'save' | 'edit' | 'delete';

export interface UserSavedSearch extends SavedSearch {
  // Make id required
  id: string;
  // Permissions
  permissions?: BookmarkPermissions;
  // Bookmark status
  bookmark_status?: BookmarkStatus;
  // Updated At
  updated_at?: string;
  // Created At
  created_at?: string;
}

export interface GlobalSavedSearch extends SavedSearch {
  // Should display query results in query's order.
  is_ordered?: boolean;
  // Override the num parameter value, if provided.
  override_num_param?: number;
}
export interface SavedSearch {
  // Saved search display name
  name: string;
  // Query for filtering
  query: string;
  // Overview page description
  description?: string;
}

export interface OpenSavedSearchEvent {
  type: SavedSearchOperationType;
  savedSearch?: UserSavedSearch;
  overviewPageQueryInput?: string;
}

export const TOP_CSS_INTEROP_ISSUES: string[] = [
  'anchor-positioning',
  'container-queries',
  'has',
  'nesting',
  'view-transitions',
  'subgrid',
  'grid',
  'scrollbar-gutter',
  'scrollbar-width',
  'scrollbar-color',
  'scroll-driven-animations',
  'scope',
];

export const TOP_HTML_INTEROP_ISSUES: string[] = [
  'popover',
  'anchor-positioning',
  'cross-document-view-transitions',
  'dialog',
  'datalist',
  'customized-built-in-elements',
  'file-system-access',
  'scroll-driven-animations',
  'notifications',
  'web-bluetooth',
];

// This focus area to web feature mapping is defined at
// https://github.com/web-platform-tests/interop/blob/main/web-features.json
export const INTEROP_FEATURES: string[] = [
  'backdrop-filter',
  'largest-contentful-paint',
  'anchor-positioning',
  'details',
  'flexbox',
  'grid',
  'subgrid',
  'json-modules',
  'navigation',
  'pointer-events-api',
  'mouse-events',
  'mutation-events',
  'scope',
  'scrollend',
  'storage-access',
  'text-decoration',
  'urlpattern',
  'view-transitions',
  'view-transitions-class',
  'wasm-string-builtins',
  'appearance',
  'zoom',
  'list-style',
  'webrtc-encoded-transform',
  'writing-mode',
];

export const DEFAULT_GLOBAL_SAVED_SEARCHES: GlobalSavedSearch[] = [
  {
    name: 'Baseline 2025',
    query: 'baseline_date:2025-01-01..2025-12-31',
    description: 'All Baseline 2025 features',
  },
  {
    name: 'Baseline 2024',
    query: 'baseline_date:2024-01-01..2024-12-31',
    description: 'All Baseline 2024 features',
  },
  {
    name: 'Baseline 2023',
    query: 'baseline_date:2023-01-01..2023-12-31',
    description: 'All Baseline 2023 features',
  },
  {
    name: 'Baseline 2022',
    query: 'baseline_date:2022-01-01..2022-12-31',
    description: 'All Baseline 2022 features',
  },
  {
    name: 'Baseline 2021',
    query: 'baseline_date:2021-01-01..2021-12-31',
    description: 'All Baseline 2021 features',
  },
  {
    name: 'Baseline 2020',
    query: 'baseline_date:2020-01-01..2020-12-31',
    description: 'All Baseline 2020 features',
  },
  {
    name: 'Top CSS Interop issues',
    query: `id:${TOP_CSS_INTEROP_ISSUES.join(' OR id:')}`,
    description:
      "This list reflects the top 10 interoperability pain points identified by developers in the State of CSS 2024 survey. We have also included their implementation status across Baseline browsers. You will notice that in some cases the items are already Baseline features, but may not have have been Baseline for long enough for developers to use with their target audience's browser support requirements. Since some voted-on pain points involve multiple web features, the list extends beyond 10 individual items for clarity and comprehensive coverage.",
    is_ordered: true,
    override_num_param: 25,
  },
  {
    name: 'Top HTML Interop issues',
    query: `id:${TOP_HTML_INTEROP_ISSUES.join(' OR id:')}`,
    description:
      "This list reflects the top 10 interoperability pain points identified by developers in the State of HTML 2024 survey. We have also included their implementation status across Baseline browsers. You will notice that in some cases the items are already Baseline features, but may not have have been Baseline for long enough for developers to use with their target audience's browser support requirements.",
    is_ordered: true,
    override_num_param: 25,
  },
];

export const VOCABULARY = [
  {
    name: 'available_date:chrome:2023-01-01..2024-01-01',
    doc: 'Became available on Chrome between the given dates',
  },
  {
    name: 'available_date:edge:2023-01-01..2024-01-01',
    doc: 'Became available on Edge between the given dates',
  },
  {
    name: 'available_date:firefox:2023-01-01..2024-01-01',
    doc: 'Became available on Firefox between the given dates',
  },
  {
    name: 'available_date:safari:2023-01-01..2024-01-01',
    doc: 'Became available on Safari between the given dates',
  },
  {
    name: 'available_date:chrome_android:2023-01-01..2024-01-01',
    doc: 'Became available on Chrome Android between the given dates',
  },
  {
    name: 'available_date:firefox_android:2023-01-01..2024-01-01',
    doc: 'Became available on Firefox between the given dates',
  },
  {
    name: 'available_date:safari_ios:2023-01-01..2024-01-01',
    doc: 'Became available on Safari iOS between the given dates',
  },
  {
    name: 'available_on:chrome',
    doc: 'Features available on Chrome',
  },
  {
    name: 'available_on:edge',
    doc: 'Features available on Edge',
  },
  {
    name: 'available_on:firefox',
    doc: 'Features available on Firefox',
  },
  {
    name: 'available_on:safari',
    doc: 'Features available on Safari',
  },
  {
    name: 'available_on:chrome_android',
    doc: 'Features available on Chrome Android',
  },
  {
    name: 'available_on:firefox_android',
    doc: 'Features available on Firefox Android',
  },
  {
    name: 'available_on:safari_ios',
    doc: 'Features available on Safari iOS',
  },
  {
    name: 'baseline_date:2023-01-01..2024-01-01',
    doc: 'Reached baseline between the given dates',
  },
  {
    name: 'baseline_status:limited',
    doc: 'Features that are not yet in baseline',
  },
  {
    name: 'baseline_status:newly',
    doc: 'Features newly added to baseline',
  },
  {
    name: 'baseline_status:widely',
    doc: 'Features in baseline and widely available',
  },
  {
    name: 'group:',
    doc: 'Features in a group or its descendants. E.g., group:css',
  },
  {
    name: 'snapshot:',
    doc: 'Features in a snapshot. E.g., snapshot:ecmascript-5',
  },
  {
    name: 'name:',
    doc: 'Find by substring of the name. E.g., name:grid',
  },
  {
    name: 'name:"a substring"',
    doc: 'Find by substring of the name. E.g., name:"CSS Grid"',
  },
  {
    name: 'id:',
    doc: 'Find by its feature identifier . E.g., id:html',
  },
  {
    name: 'OR',
    doc: 'Combine query terms with a logical-OR',
  },
  {
    name: '-',
    doc: 'Negate search term with a leading minus',
  },
];

export const MISSING_ONE_TABLE_COLUMNS: string =
  'name,availability_chrome,availability_firefox,availability_safari,availability_chrome_android,availability_firefox_android,availability_safari_ios,chrome_usage';

interface BadgeConfig {
  name: string;
  url: string;
  description: string;
  variant: 'primary' | 'success' | 'neutral' | 'warning' | 'danger' | 'text';
}

export const BADGE_PARAMS_BY_TYPE = {
  css: {
    name: 'TOP CSS',
    url: 'https://2024.stateofcss.com/',
    description:
      'This feature was listed as a top interoperability pain point in the recent State of CSS survey.',
    variant: 'success',
  },
  html: {
    name: 'TOP HTML',
    url: 'https://2024.stateofhtml.com/',
    description:
      'This feature was listed as a top interoperability pain point in the recent State of HTML survey.',
    variant: 'primary',
  },
  interop: {
    name: 'INTEROP',
    url: 'https://wpt.fyi/interop',
    description:
      'This feature is part of a focus area for the Interop 2025 effort.',
    variant: 'neutral',
  },
} as const satisfies Record<string, BadgeConfig>;

export type BadgeType = keyof typeof BADGE_PARAMS_BY_TYPE;
