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

-- Migration: 000037.sql
-- Description: Add CodeSubscriptionScanLogs and VCSWebhookDeliveries tables with native row deletion policies.

CREATE TABLE IF NOT EXISTS CodeSubscriptionScanLogs (
    ID STRING(36) NOT NULL,
    VCSProvider STRING(32) NOT NULL,
    VCSRepositoryID STRING(128) NOT NULL,
    CommitSHA STRING(64) NOT NULL,
    Branch STRING(256) NOT NULL,
    ScanStatus STRING(32) NOT NULL,
    FilesScanned INT64 NOT NULL,
    DirectivesFound INT64 NOT NULL,
    ErrorMessage STRING(2048),
    ScannedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ID),
ROW DELETION POLICY (OLDER_THAN(ScannedAt, INTERVAL 30 DAY));

CREATE INDEX IF NOT EXISTS IDX_CodeSubscriptionScanLogs_Repo_ScannedAt ON CodeSubscriptionScanLogs(VCSProvider, VCSRepositoryID, ScannedAt DESC);

CREATE TABLE IF NOT EXISTS VCSWebhookDeliveries (
    VCSProvider STRING(32) NOT NULL,
    DeliveryGUID STRING(128) NOT NULL,
    EventType STRING(64) NOT NULL,
    VCSRepositoryID STRING(128) NOT NULL,
    ReceivedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (VCSProvider, DeliveryGUID),
ROW DELETION POLICY (OLDER_THAN(ReceivedAt, INTERVAL 7 DAY));

CREATE INDEX IF NOT EXISTS IDX_VCSWebhookDeliveries_ReceivedAt ON VCSWebhookDeliveries(ReceivedAt);
