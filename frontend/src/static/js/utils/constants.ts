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

export const GITHUB_REPO_ISSUE_LINK =
  'https://github.com/GoogleChrome/webstatus.dev/issues/new/choose';
export const SEARCH_QUERY_README_LINK =
  'https://github.com/GoogleChrome/webstatus.dev/blob/main/antlr/FeatureSearch.md';
export const ABOUT_PAGE_LINK =
  'https://github.com/GoogleChrome/webstatus.dev/wiki/About-Web-Platform-Status';

export interface Bookmark {
  // Bookmark display name
  name: string;
  // Query for filtering
  query: string;
  // Overview page description
  description?: string;
  // Should display query results in query's order.
  is_ordered?: boolean;
  // Override the num parameter value, if provided.
  override_num_param?: number;
  // Server side id for bookmark
  id?: string;
}

export const DEFAULT_BOOKMARKS: Bookmark[] = [
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
    query:
      'id:anchor-positioning OR id:container-queries OR id:has OR id:nesting OR id:view-transitions OR id:subgrid OR id:grid OR name:scrollbar OR id:scroll-driven-animations OR id:scope',
    description:
      "This list reflects the top 10 interoperability pain points identified by developers in the State of CSS 2024 survey. We have also included their implementation status across Baseline browsers. You will notice that in some cases the items are already Baseline features, but may not have have been Baseline for long enough for developers to use with their target audience's browser support requirements. Since some voted-on pain points involve multiple web features, the list extends beyond 10 individual items for clarity and comprehensive coverage.",
    is_ordered: true,
    override_num_param: 25,
  },
  {
    name: 'Top HTML Interop issues',
    query:
      'id:popover OR id:anchor-positioning OR id:cross-document-view-transitions OR id:dialog OR id:datalist OR id:customized-built-in-elements OR id:file-system-access OR id:scroll-driven-animations OR id:notifications OR id:web-bluetooth',
    description:
      "This list reflects the top 10 interoperability pain points identified by developers in the State of HTML 2024 survey. We have also included their implementation status across Baseline browsers. You will notice that in some cases the items are already Baseline features, but may not have have been Baseline for long enough for developers to use with their target audience's browser support requirements.",
    is_ordered: true,
    override_num_param: 25,
  },
];
