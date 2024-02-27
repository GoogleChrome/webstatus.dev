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

-- WebFeatures contains basic metadata about web features
CREATE TABLE IF NOT EXISTS WebFeatures (
    ID STRING(36) NOT NULL DEFAULT (GENERATE_UUID()),
    FeatureID STRING(64) NOT NULL, -- From web features repo.
    Name STRING(64) NOT NULL,
) PRIMARY KEY (ID);

-- Used to enforce that only one FeatureID from web features can exist.
CREATE UNIQUE NULL_FILTERED INDEX WebFeaturesByFeatureID ON WebFeatures (FeatureID);

-- WPTRuns contains metadata from wpt.fyi runs.
-- More information: https://github.com/web-platform-tests/wpt.fyi/blob/main/api/README.md#apiruns
CREATE TABLE IF NOT EXISTS WPTRuns (
    ID STRING(36) NOT NULL DEFAULT (GENERATE_UUID()),
    ExternalRunID INT64 NOT NULL, -- ID from WPT
    TimeStart TIMESTAMP NOT NULL,
    TimeEnd TIMESTAMP NOT NULL,
    BrowserName STRING(64),
    BrowserVersion STRING(32), -- From wpt.fyi. Contains major.minor.patch and more.
    Channel STRING(32),
    OSName STRING(64),
    OSVersion STRING(32),
    FullRevisionHash STRING(40),
) PRIMARY KEY (ID);

-- Used to enforce that only one ExternalRunID from wpt.fyi can exist.
CREATE UNIQUE NULL_FILTERED INDEX RunsByExternalRunID ON WPTRuns (ExternalRunID);

-- Useful index for the runs for feature search query.
CREATE INDEX RunsForFeatureSearch ON WPTRuns (BrowserName, TimeStart DESC);


-- WPTRunFeatureMetrics contains metrics for individual features for a given run.
CREATE TABLE IF NOT EXISTS WPTRunFeatureMetrics (
    ID STRING(36) NOT NULL,
    ExternalRunID INT64 NOT NULL, -- ID from WPT
    FeatureID STRING(64) NOT NULL,
    TotalTests INT64,
    TestPass INT64,
    FOREIGN KEY (FeatureID) REFERENCES WebFeatures(FeatureID),
    FOREIGN KEY (ID) REFERENCES WPTRuns(ID)
) PRIMARY KEY (ID, FeatureID)
,    INTERLEAVE IN PARENT WPTRuns ON DELETE CASCADE;

-- Used to enforce that only one combination of ExternalRunID and FeatureID can exist.
CREATE UNIQUE NULL_FILTERED INDEX MetricsByExternalRunIDAndFeature ON WPTRunFeatureMetrics (ExternalRunID, FeatureID);

-- BrowserReleases contains information regarding browser releases.
-- Information from https://github.com/mdn/browser-compat-data/tree/main/browsers
CREATE TABLE IF NOT EXISTS BrowserReleases (
    BrowserName STRING(64) NOT NULL, -- From BCD not wpt.fyi.
    BrowserVersion STRING(8) NOT NULL, -- From BCD not wpt.fyi. Only contains major number.
    ReleaseDate TIMESTAMP NOT NULL,
) PRIMARY KEY (BrowserName, BrowserVersion);


-- BrowserFeatureAvailabilities contains information when a browser is available for a feature.
-- Information from https://github.com/mdn/browser-compat-data/tree/main/browsers
CREATE TABLE IF NOT EXISTS BrowserFeatureAvailabilities (
    BrowserName STRING(64), -- From BCD not wpt.fyi.
    BrowserVersion STRING(8) NOT NULL, -- From BCD not wpt.fyi. Only contains major number.
    FeatureID STRING(64) NOT NULL, -- From web features repo.
    FOREIGN KEY (FeatureID) REFERENCES WebFeatures(FeatureID),
    FOREIGN KEY (BrowserName, BrowserVersion) REFERENCES BrowserReleases(BrowserName, BrowserVersion),
) PRIMARY KEY (FeatureID, BrowserName);

-- Used to enforce that only one combination of FeatureID and BrowserName can exist.
CREATE UNIQUE INDEX UniqueFeatureBrowser ON BrowserFeatureAvailabilities (FeatureID, BrowserName);


-- FeatureBaselineStatus contains information about the current baseline status of a feature.
CREATE TABLE IF NOT EXISTS FeatureBaselineStatus (
    FeatureID STRING(64) NOT NULL, -- From web features repo.
    Status STRING(16) NOT NULL,
    LowDate TIMESTAMP,
    HighDate TIMESTAMP,
    FOREIGN KEY (FeatureID) REFERENCES WebFeatures(FeatureID),
    -- Options come from https://github.com/web-platform-dx/web-features/blob/3d4d066c47c9f07514bf743b3955572a6073ff1e/packages/web-features/README.md?plain=1#L17-L24
    CHECK (Status IN ('undefined', 'limited', 'low', 'high'))
) PRIMARY KEY (FeatureID);
