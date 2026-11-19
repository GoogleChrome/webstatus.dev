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

-- Migration: 000039.sql
-- Description: Add a nullable RulesVersion column to CodeSubscriptionScanLogs. The scanner records
-- the version of its scan rules (lib/codescan.RulesVersion) on every row, so vcs-sync can tell
-- whether a repository's latest scan is still valid after the rules change.
-- Existing rows are left NULL and treated as outdated; no backfill is needed.

ALTER TABLE CodeSubscriptionScanLogs ADD COLUMN RulesVersion INT64;
