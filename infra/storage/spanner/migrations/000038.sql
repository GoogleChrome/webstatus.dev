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

-- Migration: 000038.sql
-- Description: Add a nullable ErrorClass column to CodeSubscriptionScanLogs so failed scans
-- record a machine-readable failure category (e.g. TIMEOUT, STALL, PERMANENT_CLIENT).
-- Existing rows are left NULL; no backfill is needed.

ALTER TABLE CodeSubscriptionScanLogs ADD COLUMN ErrorClass STRING(32);
