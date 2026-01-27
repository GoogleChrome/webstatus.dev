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

-- Adds a table to store the system generated saved search for a feature.
-- This is a one-to-one mapping.
CREATE TABLE SystemManagedSavedSearches (
    FeatureID STRING(36) NOT NULL,
    SavedSearchID STRING(36) NOT NULL,
    CreatedAt TIMESTAMP NOT NULL OPTIONS(allow_commit_timestamp = true),
    UpdatedAt TIMESTAMP NOT NULL OPTIONS(allow_commit_timestamp = true),
    CONSTRAINT FK_SystemManagedSavedSearches_WebFeatures FOREIGN KEY (FeatureID) REFERENCES WebFeatures (ID) ON DELETE CASCADE,
    CONSTRAINT FK_SystemManagedSavedSearches_SavedSearches FOREIGN KEY (SavedSearchID) REFERENCES SavedSearches (ID) ON DELETE CASCADE
) PRIMARY KEY (FeatureID);

CREATE UNIQUE INDEX IX_SystemManagedSavedSearches_SavedSearchId ON SystemManagedSavedSearches(SavedSearchID);