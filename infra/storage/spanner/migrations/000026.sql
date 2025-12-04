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

-- SavedSearchState
-- This table tracks the "Last Known State" for the diff engine and handles worker locking.
CREATE TABLE IF NOT EXISTS SavedSearchState (
    SavedSearchId STRING(36) NOT NULL,
    SnapshotType STRING(MAX) NOT NULL, -- Enum: IMMEDIATE, WEEKLY, MONTHLY
    LastKnownStateBlobPath STRING(MAX),
    WorkerLockId STRING(MAX),
    WorkerLockExpiresAt TIMESTAMP,

    CONSTRAINT FK_SavedSearchState_SavedSearch
        FOREIGN KEY (SavedSearchId) REFERENCES SavedSearches(ID) ON DELETE CASCADE

) PRIMARY KEY (SavedSearchId, SnapshotType);
