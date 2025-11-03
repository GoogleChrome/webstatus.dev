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

-- NotificationChannelDeliveryAttempts stores a log of delivery attempts for a channel.
CREATE TABLE IF NOT EXISTS NotificationChannelDeliveryAttempts (
    ID STRING(36) NOT NULL DEFAULT (GENERATE_UUID()),
    ChannelID STRING(36) NOT NULL,
    AttemptTimestamp TIMESTAMP NOT NULL,
    Status STRING(MAX) NOT NULL, -- SUCCESS or FAILURE
    Details JSON,
    CONSTRAINT FK_NotificationChannelDeliveryAttempt_NotificationChannel FOREIGN KEY (ChannelID) REFERENCES NotificationChannels(ID) ON DELETE CASCADE
) PRIMARY KEY (ID, ChannelID);

-- Index to get the latest attempts for a channel
CREATE INDEX IX_NotificationChannelDeliveryAttempt_AttemptTimestamp
ON NotificationChannelDeliveryAttempts (ChannelID, AttemptTimestamp DESC);
