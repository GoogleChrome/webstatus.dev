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

-- SavedSearches contains the most current revisions of saved searches.
--
-- A future table will contain the historical revisions of saved searches
-- allowing for auditing and reverting to older versions of the saved search.
CREATE TABLE IF NOT EXISTS SavedSearches (
    ID STRING(36) NOT NULL DEFAULT (GENERATE_UUID()),
    -- Name is the name of the saved search.
    Name STRING(MAX) NOT NULL,
    -- Query is the query string of the saved search.
    Query STRING(MAX) NOT NULL,
    -- Scope is the scope of the saved search.
    -- Refer to the SavedSearchScope enum in lib/gcpspanner/saved_searches.go for possible values.
    Scope STRING(MAX) NOT NULL,
    -- AuthorID is only for auditing purposes. The author may not always be the owner.
    -- Instead, we should always rely on SavedSearchUserRoles for current roles.
    -- This ID is the unique ID for each user managed by Google Cloud Identity Platform.
    AuthorID STRING(MAX) NOT NULL,
    -- CreatedAt is the timestamp of the first saved search revision.
    CreatedAt TIMESTAMP OPTIONS (allow_commit_timestamp=true),
    -- UpdatedAt is the timestamp of the most recent revision.
    UpdatedAt TIMESTAMP OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (ID);

-- SavedSearchUserRoles keeps track of the user's role for a given saved search.
CREATE TABLE IF NOT EXISTS SavedSearchUserRoles (
    SavedSearchID STRING(36) NOT NULL,
    UserID STRING(MAX) NOT NULL,
    -- Refer to the SavedSearchRole enum in lib/gcpspanner/saved_search_user_roles.go for possible values.
    UserRole STRING(MAX) NOT NULL,
    FOREIGN KEY (SavedSearchID) REFERENCES SavedSearches(ID)  ON DELETE CASCADE
) PRIMARY KEY (UserID, SavedSearchID);

-- UserSavedSearchBookmarks keeps track of the user's bookmarks for user-created saved searches.
CREATE TABLE IF NOT EXISTS UserSavedSearchBookmarks (
    SavedSearchID STRING(36) NOT NULL,
    UserID STRING(MAX) NOT NULL,
    FOREIGN KEY (SavedSearchID) REFERENCES SavedSearches(ID)  ON DELETE CASCADE
) PRIMARY KEY (UserID, SavedSearchID);