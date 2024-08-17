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

-- Snapshots contains basic metadata about snapshots from the WebDX web features repository.
CREATE TABLE IF NOT EXISTS WebDXSnapshots (
    ID STRING(36) NOT NULL DEFAULT (GENERATE_UUID()),
    SnapshotKey STRING(64) NOT NULL,
    Name STRING(64) NOT NULL,
    -- Additional lowercase columns for case-insensitive search
    SnapshotKey_Lowercase STRING(64) AS (LOWER(SnapshotKey)) STORED,
    Name_Lowercase STRING(64) AS (LOWER(Name)) STORED,
) PRIMARY KEY (ID);

-- Maps web features to snapshots.
CREATE TABLE IF NOT EXISTS WebFeatureSnapshots (
    WebFeatureID STRING(36) NOT NULL,
    -- Stores an array of Snapshot IDs associated with this web feature.
    -- Each feature does not belong to many snapshots. For now, keep them here.
    SnapshotIDs ARRAY<STRING(36)>,
    FOREIGN KEY (WebFeatureID) REFERENCES WebFeatures(ID)  ON DELETE CASCADE
) PRIMARY KEY (WebFeatureID);