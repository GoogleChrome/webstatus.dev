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

-- SystemGlobalSavedSearches controls the display of global saved searches that
-- are visible to all users.
-- This is intended for use by system-defined searches such as Baselines and
-- Top Interop issues, which are critical for users to easily discover and track.
-- By centralizing the management of these global searches in a dedicated table,
-- we can ensure they are consistently displayed across the application and
-- prevent accidental modifications or deletions by end users.
CREATE TABLE SystemGlobalSavedSearches (
    SavedSearchID STRING(36) NOT NULL,
    -- DisplayOrder dictates the UI ordering of global searches (e.g. Baseline 2026, Top CSS).
    -- It is queried using ORDER BY DESC (Highest is Top) so it can scale infinitely forward
    -- as new years are added without descending into zero or negative integers.
    DisplayOrder INT64 NOT NULL,
    -- Status indicates whether the global saved search is currently active and should be displayed to users.
    -- This allows for easy deprecation of old searches without deleting them from the database, preserving historical data and preventing broken references in user-generated content.
    -- The "all" saved search will also be marked as unlisted to prevent it from showing up in the UI, but it will still be accessible via direct link and can be used as a catch-all query for API requests.
    Status STRING(32) NOT NULL,
    CONSTRAINT FK_SystemGlobal_SavedSearches FOREIGN KEY (SavedSearchID) REFERENCES SavedSearches (ID) ON DELETE CASCADE
) PRIMARY KEY (SavedSearchID);

-- SavedSearchFeatureSortOrder specifies an explicit UI sort order for features
-- within a global saved search
CREATE TABLE SavedSearchFeatureSortOrder (
    SavedSearchID STRING(36) NOT NULL,
    FeatureKey STRING(64) NOT NULL,
    -- PositionIndex dictates the UI ordering of features within a specific curated search.
    -- It is queried using ORDER BY ASC (Lowest is Top). Start at 10 and explicitly increment
    -- by 10 (10, 20, 30) to allow injecting new features comfortably between them over time.
    PositionIndex INT64 NOT NULL,
    CONSTRAINT FK_SavedSearchSort_SavedSearch FOREIGN KEY (SavedSearchID) REFERENCES SavedSearches (ID) ON DELETE CASCADE
) PRIMARY KEY (SavedSearchID, FeatureKey);
