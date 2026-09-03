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

-- Migration: 000034.sql
-- Description: Add VCSInstallations table for GitHub App and multi-VCS connections.

CREATE TABLE IF NOT EXISTS VCSInstallations (
    ID STRING(36) NOT NULL,
    VCSProvider STRING(32) NOT NULL,
    VCSInstallationID STRING(128) NOT NULL,
    AccountLogin STRING(256) NOT NULL,
    AccountType STRING(32) NOT NULL,
    RepositorySelection STRING(32) NOT NULL,
    Permissions JSON NOT NULL,
    CreatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
    UpdatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ID);

CREATE UNIQUE INDEX IF NOT EXISTS IDX_VCSInstallations_Provider_InstallationID ON VCSInstallations(VCSProvider, VCSInstallationID);
CREATE INDEX IF NOT EXISTS IDX_VCSInstallations_Provider_AccountLogin ON VCSInstallations(VCSProvider, AccountLogin);
