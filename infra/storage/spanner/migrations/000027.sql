-- Copyright 2025 Google LLC
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

-- SavedSearchNotificationEvents
-- The append-only log of all generated events ("The Fact Table").
CREATE TABLE IF NOT EXISTS SavedSearchNotificationEvents (
    EventId STRING(36) NOT NULL DEFAULT (GENERATE_UUID()),
    SavedSearchId STRING(36) NOT NULL,
    SnapshotType STRING(MAX) NOT NULL,
    Timestamp TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
    EventType STRING(MAX) NOT NULL, -- Enum: IMMEDIATE_DIFF, WEEKLY_DIGEST, etc.
    Reason STRING(MAX) NOT NULL,    -- Enum: DATA_UPDATED, QUERY_EDITED, etc.
    BlobPath STRING(MAX) NOT NULL,
    Summary JSON,
    DiffKind STRING(MAX),           -- e.g., "SavedSearchDiffList"
    DataVersion STRING(MAX),        -- e.g., "v1"

    CONSTRAINT FK_Events_State
        FOREIGN KEY (SavedSearchId, SnapshotType)
        REFERENCES SavedSearchState (SavedSearchId, SnapshotType) ON DELETE CASCADE

) PRIMARY KEY (EventId);


-- This index prevents full table scans during RSS polling.
CREATE INDEX IF NOT EXISTS SavedSearchNotificationEvents_BySearchAndType
ON SavedSearchNotificationEvents (SavedSearchId, EventType, Timestamp DESC);