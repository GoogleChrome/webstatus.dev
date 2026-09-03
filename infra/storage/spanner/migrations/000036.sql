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

-- Migration: 000036.sql
-- Description: Add CodeSubscriptionDeliveries table.

CREATE TABLE IF NOT EXISTS CodeSubscriptionDeliveries (
    ID STRING(36) NOT NULL,
    SubscriptionID STRING(36) NOT NULL,
    DeliveryStatus STRING(32) NOT NULL,
    DeliveryChannel STRING(32) NOT NULL,
    LockExpiresAt TIMESTAMP,
    WorkerLockID STRING(64),
    DeliveredAt TIMESTAMP,
    ExternalIssueID STRING(128),
    ExternalIssueURL STRING(1024),
    ErrorMessage STRING(2048),
    CreatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
    UpdatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
    CONSTRAINT FK_Deliveries_Subscription FOREIGN KEY (SubscriptionID) REFERENCES CodeSubscriptions(ID) ON DELETE CASCADE,
) PRIMARY KEY (ID);

CREATE INDEX IF NOT EXISTS IDX_Deliveries_SubscriptionID ON CodeSubscriptionDeliveries(SubscriptionID);
CREATE INDEX IF NOT EXISTS IDX_Deliveries_LockStatus ON CodeSubscriptionDeliveries(DeliveryStatus, LockExpiresAt, CreatedAt);
