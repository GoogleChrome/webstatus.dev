-- Copyright 2024 Google LLC
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

CREATE TABLE IF NOT EXISTS WPTRuns (
    ID STRING(36) NOT NULL DEFAULT (GENERATE_UUID()),
    ExternalRunID INT64 NOT NULL, -- ID from WPT
    TimeStart TIMESTAMP NOT NULL,
    TimeEnd TIMESTAMP NOT NULL,
    BrowserName STRING(64),
    BrowserVersion STRING(32),
    Channel STRING(32),
    OSName STRING(64),
    OSVersion STRING(32),
    FullRevisionHash STRING(40),
) PRIMARY KEY (ID);

CREATE UNIQUE NULL_FILTERED INDEX RunsByExternalRunID ON WPTRuns (ExternalRunID);
CREATE UNIQUE NULL_FILTERED INDEX RunsByExternalRunIDAndID ON WPTRuns (ExternalRunID, ID);
CREATE UNIQUE NULL_FILTERED INDEX RunsByExternalRunIDAndTimeDesc ON WPTRuns (ExternalRunID, TimeStart DESC);

CREATE TABLE IF NOT EXISTS WPTRunFeatureMetrics (
    ID STRING(36) NOT NULL,
    ExternalRunID INT64 NOT NULL, -- ID from WPT
    FeatureID STRING(64) NOT NULL,
    TotalTests INT64,
    TestPass INT64,
) PRIMARY KEY (ID, FeatureID)
,    INTERLEAVE IN PARENT WPTRuns ON DELETE CASCADE;

-- CREATE NULL_FILTERED INDEX MetricsByExternalRunID ON WPTRunFeatureMetrics (ExternalRunID);
CREATE UNIQUE NULL_FILTERED INDEX MetricsByExternalRunIDAndFeature ON WPTRunFeatureMetrics (ExternalRunID, FeatureID);
