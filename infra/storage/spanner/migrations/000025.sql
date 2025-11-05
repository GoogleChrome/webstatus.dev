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

-- SavedSearchSubscriptions links a User to a Saved Search via a Channel.
CREATE TABLE IF NOT EXISTS SavedSearchSubscriptions (
    ID STRING(36) NOT NULL DEFAULT (GENERATE_UUID()),
    ChannelID STRING(36) NOT NULL,
    SavedSearchID STRING(36) NOT NULL,
    Triggers ARRAY<STRING(MAX)>,
    Frequency STRING(MAX),
    CreatedAt TIMESTAMP NOT NULL OPTIONS(allow_commit_timestamp = true),
    UpdatedAt TIMESTAMP NOT NULL OPTIONS(allow_commit_timestamp = true),
    CONSTRAINT FK_SavedSearchSubscription_NotificationChannel FOREIGN KEY (ChannelID) REFERENCES NotificationChannels (ID) ON DELETE CASCADE,
    CONSTRAINT FK_SavedSearchSubscription_SavedSearches FOREIGN KEY (SavedSearchID) REFERENCES SavedSearches (ID) ON DELETE CASCADE
) PRIMARY KEY (ID);
