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

-- Migration: 000035.sql
-- Description: Add CodeSubscriptions table.

CREATE TABLE CodeSubscriptions (
    ID STRING(36) NOT NULL,
    VCSProvider STRING(32) NOT NULL,
    VCSInstallationID STRING(128) NOT NULL,
    VCSRepositoryID STRING(128) NOT NULL,
    RepositoryFullName STRING(256) NOT NULL,
    TargetQuery STRING(2048) NOT NULL,
    Triggers ARRAY<STRING(64)> NOT NULL,
    Status STRING(32) NOT NULL,
    Occurrences JSON NOT NULL,
    CreatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
    UpdatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ID);

CREATE UNIQUE INDEX IDX_CodeSubscriptions_Provider_Repo_Target ON CodeSubscriptions(VCSProvider, VCSRepositoryID, TargetQuery);
CREATE INDEX IDX_CodeSubscriptions_Provider_Repo ON CodeSubscriptions(VCSProvider, VCSRepositoryID);
CREATE INDEX IDX_CodeSubscriptions_TargetQuery ON CodeSubscriptions(TargetQuery);
