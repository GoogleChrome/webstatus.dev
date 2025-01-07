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
  'https://github.com/GoogleChrome/webstatus.dev/issues/new';
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
}

export const DEFAULT_BOOKMARKS: Bookmark[] = [
  {
    name: 'Baseline 2023',
    query: 'baseline_date:2023-01-01..2023-12-31',
    description: 'All Baseline 2023 features',
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
      'id:popover OR id:anchor-positioning OR id:cross-document-view-transitions OR id:dialog OR id:datalist OR id:file-system-access',
    description:
      "This list reflects the top 6 interoperability pain points identified by developers in the State of HTML 2024 survey. We have also included their implementation status across Baseline browsers. You will notice that in some cases the items are already Baseline features, but may not have have been Baseline for long enough for developers to use with their target audience's browser support requirements. Since some voted-on pain points involve multiple web features, the list extends beyond 10 individual items for clarity and comprehensive coverage.",
    is_ordered: true,
    override_num_param: 25,
  },
];
