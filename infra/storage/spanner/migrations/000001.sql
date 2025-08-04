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
    FeatureKey STRING(64) NOT NULL, -- From web features repo.
    Name STRING(64) NOT NULL,
    -- Additional lowercase columns for case-insensitive search
    FeatureKey_Lowercase STRING(64) AS (LOWER(FeatureKey)) STORED,
    Name_Lowercase STRING(64) AS (LOWER(Name)) STORED,
) PRIMARY KEY (ID);

-- Used to enforce that only one FeatureKey from web features can exist.
CREATE UNIQUE NULL_FILTERED INDEX WebFeaturesByFeatureID ON WebFeatures (FeatureKey);

-- Index on FeatureKey and Name for case-insensitive search
CREATE INDEX IDX_FEATUREID_LOWER ON WebFeatures(FeatureKey_Lowercase);
CREATE INDEX IDX_NAME_LOWER ON WebFeatures(Name_Lowercase);

-- WPTRuns contains metadata from wpt.fyi runs.
-- More information: https://github.com/web-platform-tests/wpt.fyi/blob/main/api/README.md#apiruns
CREATE TABLE IF NOT EXISTS WPTRuns (
    ID STRING(36) NOT NULL DEFAULT (GENERATE_UUID()),
    ExternalRunID INT64 NOT NULL, -- ID from WPT
    TimeStart TIMESTAMP NOT NULL,
    TimeEnd TIMESTAMP NOT NULL,
    BrowserName STRING(64) NOT NULL,
    BrowserVersion STRING(32) NOT NULL, -- From wpt.fyi. Contains major.minor.patch and more.
    Channel STRING(32) NOT NULL,
    OSName STRING(64),
    OSVersion STRING(32),
    FullRevisionHash STRING(40),
) PRIMARY KEY (ID);


-- Used to enforce that only one ExternalRunID from wpt.fyi can exist.
CREATE UNIQUE NULL_FILTERED INDEX RunsByExternalRunID ON WPTRuns (ExternalRunID);

-- Useful index for the runs for feature search query.
CREATE INDEX RunsForFeatureSearchWithChannel ON WPTRuns(ExternalRunID, Channel, TimeStart DESC, BrowserName);

-- Useful index for feature search. Used to get the latest runs beforehand.
CREATE INDEX LatestRunsByBrowserChannel ON WPTRuns (BrowserName, Channel, TimeStart DESC);


-- WPTRunFeatureMetrics contains metrics for individual features for a given run.
CREATE TABLE IF NOT EXISTS WPTRunFeatureMetrics (
    ID STRING(36) NOT NULL,
    WebFeatureID STRING(36) NOT NULL,
    TotalTests INT64,
    TestPass INT64,
    TestPassRate NUMERIC,
    TotalSubtests INT64,
    SubtestPass INT64,
    SubtestPassRate NUMERIC,
    -- Denormalized data from WPTRuns. This helps with aggregations over time.
    Channel STRING(32) NOT NULL,
    BrowserName STRING(64) NOT NULL,
    TimeStart TIMESTAMP NOT NULL,
    -- End denormalized data.
    CONSTRAINT FK_WPTRunFeatureMetricsWebFeatureID FOREIGN KEY (WebFeatureID) REFERENCES WebFeatures(ID) ON DELETE CASCADE,
    FOREIGN KEY (ID) REFERENCES WPTRuns(ID)
) PRIMARY KEY (ID, WebFeatureID)
,    INTERLEAVE IN PARENT WPTRuns ON DELETE CASCADE;

-- Used to enforce that only one combination of ID and WebFeatureID can exist.
CREATE UNIQUE NULL_FILTERED INDEX MetricsByRunIDAndFeature ON WPTRunFeatureMetrics (ID, WebFeatureID);

-- Used to help with metrics aggregation calculations.
CREATE INDEX MetricsFeatureChannelBrowserTime ON
  WPTRunFeatureMetrics(WebFeatureID, Channel, BrowserName, TimeStart DESC);

CREATE INDEX MetricsFeatureChannelBrowserTimeTestPassRate ON WPTRunFeatureMetrics(WebFeatureID, Channel, BrowserName, TimeStart DESC, TestPassRate);
CREATE INDEX MetricsFeatureChannelBrowserTimeSubtestPassRate ON WPTRunFeatureMetrics(WebFeatureID, Channel, BrowserName, TimeStart DESC, SubtestPassRate);


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
    BrowserName STRING(64) NOT NULL, -- From BCD not wpt.fyi.
    BrowserVersion STRING(8) NOT NULL, -- From BCD not wpt.fyi. Only contains major number.
    WebFeatureID STRING(36) NOT NULL, -- From web features table.
    CONSTRAINT FK_BrowserFeatureAvailabilitiesWebFeatureID FOREIGN KEY (WebFeatureID) REFERENCES WebFeatures(ID) ON DELETE CASCADE,
    FOREIGN KEY (BrowserName, BrowserVersion) REFERENCES BrowserReleases(BrowserName, BrowserVersion),
) PRIMARY KEY (WebFeatureID, BrowserName);

-- Used to enforce that only one combination of WebFeatureID and BrowserName can exist.
CREATE UNIQUE INDEX UniqueFeatureBrowser ON BrowserFeatureAvailabilities (WebFeatureID, BrowserName);


-- FeatureBaselineStatus contains information about the current baseline status of a feature.
CREATE TABLE IF NOT EXISTS FeatureBaselineStatus (
    WebFeatureID STRING(36) NOT NULL, -- From web features table.
    Status STRING(16),
    LowDate TIMESTAMP,
    HighDate TIMESTAMP,
    CONSTRAINT FK_FeatureBaselineStatusWebFeatureID FOREIGN KEY (WebFeatureID) REFERENCES WebFeatures(ID) ON DELETE CASCADE,
    -- Options come from https://github.com/web-platform-dx/web-features/blob/3d4d066c47c9f07514bf743b3955572a6073ff1e/packages/web-features/README.md?plain=1#L17-L24
    CHECK (Status IN ('none', 'low', 'high'))
) PRIMARY KEY (WebFeatureID);

-- Index to accelerate lookups and joins in FeatureBaselineStatus based on WebFeatureID.
-- Primarily supports queries involving the WebFeatures table.
CREATE INDEX IDX_FBS_FEATUREID ON FeatureBaselineStatus(WebFeatureID);
