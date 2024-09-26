-- Copyright 2024 Google LLC
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

CREATE TABLE IF NOT EXISTS SavedSearches (
    ID STRING(36) NOT NULL DEFAULT (GENERATE_UUID()),
    Name STRING(MAX) NOT NULL,
    Query STRING(MAX) NOT NULL,
    CreatedAt TIMESTAMP OPTIONS (allow_commit_timestamp=true),
    UpdatedAt TIMESTAMP OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (ID);

CREATE TABLE IF NOT EXISTS SavedSearchRoles (
    SavedSearchID STRING(36) NOT NULL,
    UserID STRING(MAX) NOT NULL,
    UserRole STRING(MAX) NOT NULL,
    FOREIGN KEY (SavedSearchID) REFERENCES SavedSearches(ID)  ON DELETE CASCADE
) PRIMARY KEY (SavedSearchID, UserID);

CREATE TABLE IF NOT EXISTS SavedSearchSubscriptions (
    SavedSearchID STRING(36) NOT NULL,
    UserID STRING(MAX) NOT NULL,
    FOREIGN KEY (SavedSearchID) REFERENCES SavedSearches(ID)  ON DELETE CASCADE
) PRIMARY KEY (SavedSearchID, UserID);