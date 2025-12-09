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

-- 1. DiffBlobPath: Points to the GCS object containing the structured FeatureDiff.
--    Replaces the idea of storing the raw JSON in the DB.
--    Example: "gs://bucket/events/{event_id}/diff.json"
--    This is similar to the existing BlobPath column which stores the summary blob path.
--    The blob is self documenting and contains the vesion and kind information within it.
ALTER TABLE SavedSearchNotificationEvents ADD COLUMN DiffBlobPath STRING(MAX);

-- 2. Drop columns that are redundantly handled by the Blob Envelope (metadata inside the blob)
ALTER TABLE SavedSearchNotificationEvents DROP COLUMN DiffKind;
ALTER TABLE SavedSearchNotificationEvents DROP COLUMN DataVersion;

-- 3. Change 'Reason' column to 'Reasons' ARRAY to support multiple reasons per event.
-- The original 'Reason' column was a single string, which couldn't handle the
-- "Both" scenario (Data Updated + Query Edited).
-- We replace it with an ARRAY so we can tag an event with multiple reasons.
ALTER TABLE SavedSearchNotificationEvents DROP COLUMN Reason;
ALTER TABLE SavedSearchNotificationEvents ADD COLUMN Reasons ARRAY<STRING(MAX)>;