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

-- WebDXGroups contains basic metadata about groups from the webdx web features repository.
-- Named WebDXGroups because Groups is a reserved word.
CREATE TABLE IF NOT EXISTS WebDXGroups (
    ID STRING(36) NOT NULL DEFAULT (GENERATE_UUID()),
    GroupKey STRING(64) NOT NULL,
    Name STRING(64) NOT NULL,

    -- Additional lowercase columns for case-insensitive search
    GroupKey_Lowercase STRING(64) AS (LOWER(GroupKey)) STORED,
    Name_Lowercase STRING(64) AS (LOWER(Name)) STORED
) PRIMARY KEY (ID);

-- Maps web features to groups.
CREATE TABLE IF NOT EXISTS WebFeatureGroups (
    WebFeatureID STRING(36) NOT NULL,
    -- Stores an array of Group IDs associated with this web feature.
    -- Each feature does not belong to many groups. For now, keep them here.
    GroupIDs ARRAY<STRING(36)>,
    FOREIGN KEY (WebFeatureID) REFERENCES WebFeatures(ID)  ON DELETE CASCADE
) PRIMARY KEY (WebFeatureID);

-- Stores descendant group IDs for each group.
-- This separate table allows efficient retrieval of features associated with a group and its descendants.
-- It also simplifies updates to the group hierarchy compared to storing descendant IDs directly within the WebDXGroups
-- table.
CREATE TABLE IF NOT EXISTS WebDXGroupDescendants (
    GroupID STRING(36) NOT NULL,
    -- Stores IDs of all descendant groups to optimize
    -- queries for features associated with a group and its children.
    -- TODO: Consider a separate GroupChildren table if the group hierarchy
    -- becomes deeply nested.
    DescendantGroupIDs ARRAY<STRING(36)>,
    FOREIGN KEY (GroupID) REFERENCES WebDXGroups(ID)  ON DELETE CASCADE
) PRIMARY KEY (GroupID);
