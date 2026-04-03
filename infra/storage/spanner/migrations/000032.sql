-- Copyright 2026 Google LLC
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--     http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

-- Note: System Global Saved Searches use descriptive string IDs (e.g. 'baseline-2026')
-- instead of random UUIDs to prevent accidental overwrites of user-generated searches
-- and to ensure these system records are easily identifiable in the database.

-- BASELINE SEARCHES
-- These searches rely on the default feature search sorting logic (Baseline Status / Low Date).
-- Baseline 2026
INSERT INTO SavedSearches (ID, Name, Description, Query, Scope, AuthorID, CreatedAt, UpdatedAt) VALUES (
    'baseline-2026',
    'Baseline 2026',
    'All features that reached Baseline during 2026',
    'baseline_date:2026-01-01..2026-12-31',
    'SYSTEM_GLOBAL',
    'system',
    CURRENT_TIMESTAMP(),
    CURRENT_TIMESTAMP()
);
INSERT INTO SystemGlobalSavedSearches (SavedSearchID, DisplayOrder, Status) VALUES ('baseline-2026', 10000, 'LISTED');
-- Baseline 2025
INSERT INTO SavedSearches (ID, Name, Description, Query, Scope, AuthorID, CreatedAt, UpdatedAt) VALUES (
    'baseline-2025',
    'Baseline 2025',
    'All features that reached Baseline during 2025',
    'baseline_date:2025-01-01..2025-12-31',
    'SYSTEM_GLOBAL',
    'system',
    CURRENT_TIMESTAMP(),
    CURRENT_TIMESTAMP()
);
INSERT INTO SystemGlobalSavedSearches (SavedSearchID, DisplayOrder, Status) VALUES ('baseline-2025', 9000, 'LISTED');
-- Baseline 2024
INSERT INTO SavedSearches (ID, Name, Description, Query, Scope, AuthorID, CreatedAt, UpdatedAt) VALUES (
    'baseline-2024',
    'Baseline 2024',
    'All features that reached Baseline during 2024.',
    'baseline_date:2024-01-01..2024-12-31',
    'SYSTEM_GLOBAL',
    'system',
    CURRENT_TIMESTAMP(),
    CURRENT_TIMESTAMP()
);
INSERT INTO SystemGlobalSavedSearches (SavedSearchID, DisplayOrder, Status) VALUES ('baseline-2024', 8000, 'LISTED');
-- Baseline 2023
INSERT INTO SavedSearches (ID, Name, Description, Query, Scope, AuthorID, CreatedAt, UpdatedAt) VALUES (
    'baseline-2023',
    'Baseline 2023',
    'All features that reached Baseline during 2023.',
    'baseline_date:2023-01-01..2023-12-31',
    'SYSTEM_GLOBAL',
    'system',
    CURRENT_TIMESTAMP(),
    CURRENT_TIMESTAMP()
);
INSERT INTO SystemGlobalSavedSearches (SavedSearchID, DisplayOrder, Status) VALUES ('baseline-2023', 7000, 'LISTED');
-- Baseline 2022
INSERT INTO SavedSearches (ID, Name, Description, Query, Scope, AuthorID, CreatedAt, UpdatedAt) VALUES (
    'baseline-2022',
    'Baseline 2022',
    'All features that reached Baseline during 2022.',
    'baseline_date:2022-01-01..2022-12-31',
    'SYSTEM_GLOBAL',
    'system',
    CURRENT_TIMESTAMP(),
    CURRENT_TIMESTAMP()
);
INSERT INTO SystemGlobalSavedSearches (SavedSearchID, DisplayOrder, Status) VALUES ('baseline-2022', 6000, 'LISTED');
-- Baseline 2021
INSERT INTO SavedSearches (ID, Name, Description, Query, Scope, AuthorID, CreatedAt, UpdatedAt) VALUES (
    'baseline-2021',
    'Baseline 2021',
    'All features that reached Baseline during 2021.',
    'baseline_date:2021-01-01..2021-12-31',
    'SYSTEM_GLOBAL',
    'system',
    CURRENT_TIMESTAMP(),
    CURRENT_TIMESTAMP()
);
INSERT INTO SystemGlobalSavedSearches (SavedSearchID, DisplayOrder, Status) VALUES ('baseline-2021', 5000, 'LISTED');
-- Baseline 2020
INSERT INTO SavedSearches (ID, Name, Description, Query, Scope, AuthorID, CreatedAt, UpdatedAt) VALUES (
    'baseline-2020',
    'Baseline 2020',
    'All features that reached Baseline during 2020.',
    'baseline_date:2020-01-01..2020-12-31',
    'SYSTEM_GLOBAL',
    'system',
    CURRENT_TIMESTAMP(),
    CURRENT_TIMESTAMP()
);
INSERT INTO SystemGlobalSavedSearches (SavedSearchID, DisplayOrder, Status) VALUES ('baseline-2020', 4000, 'LISTED');

-- TOP CSS INTEROP ISSUES
-- The display order of the features within this search is explicitly overridden below to ensure the most critical items surface first.
INSERT INTO SavedSearches (ID, Name, Description, Query, Scope, AuthorID, CreatedAt, UpdatedAt) VALUES (
    'top-css-interop',
    'Top CSS Interop issues',
    "This list reflects the top 10 interoperability pain points identified by developers in the State of CSS 2025 survey." ||
        " We have also included their implementation status across Baseline browsers." ||
        " You will notice that in some cases the items are already Baseline features, but may not have have been Baseline for long enough for developers to use with their target audience's browser support requirements." ||
        " Since some voted-on pain points involve multiple web features, the list extends beyond 10 individual items for clarity and comprehensive coverage.",
    'id:anchor-positioning OR id:scroll-driven-animations OR id:view-transitions OR id:cross-document-view-transitions OR id:container-style-queries OR id:nesting OR id:has OR id:container-queries OR id:scope OR id:if OR id:grid',
    'SYSTEM_GLOBAL',
    'system',
    CURRENT_TIMESTAMP(),
    CURRENT_TIMESTAMP()
);
INSERT INTO SystemGlobalSavedSearches (SavedSearchID, DisplayOrder, Status) VALUES ('top-css-interop', 3000, 'LISTED');
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-css-interop', 'anchor-positioning', 10);
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-css-interop', 'scroll-driven-animations', 20);
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-css-interop', 'view-transitions', 30);
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-css-interop', 'cross-document-view-transitions', 40);
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-css-interop', 'container-style-queries', 50);
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-css-interop', 'nesting', 60);
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-css-interop', 'has', 70);
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-css-interop', 'container-queries', 80);
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-css-interop', 'scope', 90);
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-css-interop', 'if', 100);
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-css-interop', 'grid', 110);

-- TOP HTML INTEROP ISSUES
-- The display order of the features within this search is explicitly overridden below to ensure the most critical items surface first.
INSERT INTO SavedSearches (ID, Name, Description, Query, Scope, AuthorID, CreatedAt, UpdatedAt) VALUES (
    'top-html-interop',
    'Top HTML Interop issues',
    "This list reflects the top 10 interoperability pain points identified by developers in the State of HTML 2025 survey." ||
        " We have also included their implementation status across Baseline browsers." ||
        " You will notice that in some cases the items are already Baseline features, but may not have have been Baseline for long enough for developers to use with their target audience's browser support requirements." ||
        " Since some voted-on pain points involve multiple web features, the list extends beyond 10 individual items for clarity and comprehensive coverage.",
    'id:customizable-select OR id:popover OR id:anchor-positioning OR id:customized-built-in-elements OR id:shadow-dom OR id:dialog OR id:view-transitions OR id:cross-document-view-transitions OR id:file-system-access OR id:input-date-time OR id:invoker-commands OR id:webusb',
    'SYSTEM_GLOBAL',
    'system',
    CURRENT_TIMESTAMP(),
    CURRENT_TIMESTAMP()
);
INSERT INTO SystemGlobalSavedSearches (SavedSearchID, DisplayOrder, Status) VALUES ('top-html-interop', 2000, 'LISTED');
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-html-interop', 'customizable-select', 10);
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-html-interop', 'popover', 20);
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-html-interop', 'anchor-positioning', 30);
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-html-interop', 'customized-built-in-elements', 40);
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-html-interop', 'shadow-dom', 50);
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-html-interop', 'dialog', 60);
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-html-interop', 'view-transitions', 70);
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-html-interop', 'cross-document-view-transitions', 80);
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-html-interop', 'file-system-access', 90);
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-html-interop', 'input-date-time', 100);
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-html-interop', 'invoker-commands', 110);
INSERT INTO SavedSearchFeatureSortOrder (SavedSearchID, FeatureKey, PositionIndex) VALUES ('top-html-interop', 'webusb', 120);

-- ALL FEATURES
-- Default ordering implicitly applied. Shown for completeness.
INSERT INTO SavedSearches (ID, Name, Description, Query, Scope, AuthorID, CreatedAt, UpdatedAt) VALUES (
    'all',
    'All Features',
    'A system defined search containing all features.',
    '',
    'SYSTEM_GLOBAL',
    'system',
    CURRENT_TIMESTAMP(),
    CURRENT_TIMESTAMP()
);
INSERT INTO SystemGlobalSavedSearches (SavedSearchID, DisplayOrder, Status) VALUES ('all', 1000, 'UNLISTED');
