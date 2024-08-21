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


-- ChromiumHistogramEnums contains metadata about histograms found in
-- https://chromium.googlesource.com/chromium/src/+/main/tools/metrics/histograms/enums.xml
CREATE TABLE IF NOT EXISTS ChromiumHistogramEnums (
    ID STRING(36) NOT NULL DEFAULT (GENERATE_UUID()),
    HistogramName STRING(MAX) NOT NULL,
    BucketID INT64 NOT NULL,
    Label STRING(MAX) NOT NULL,
    -- Additional lowercase columns for case-insensitive search
    HistogramName_Lowercase STRING(64) AS (LOWER(HistogramName)) STORED,
    Label_Lowercase STRING(64) AS (LOWER(Label)) STORED,
) PRIMARY KEY (ID);

-- Used to enforce that only one combination of HistogramName and BucketID can exist.
CREATE UNIQUE NULL_FILTERED INDEX ChromiumHistogramEnumsByHistogramNameAndBucketID ON ChromiumHistogramEnums (HistogramName, BucketID);

-- DailyChromiumHistogramBucketMetrics contains the daily metrics per bucket
CREATE TABLE IF NOT EXISTS DailyChromiumHistogramBucketMetrics (
    ChromiumHistogramEnumID STRING(36) NOT NULL,
    Day DATE NOT NULL,
    Percentage FLOAT64 NOT NULL,
    FOREIGN KEY (ChromiumHistogramEnumID) REFERENCES ChromiumHistogramEnums(ID)  ON DELETE CASCADE
) PRIMARY KEY (ChromiumHistogramEnumID, Day);

-- Maps web features to ChromiumHistogramEnums.
CREATE TABLE IF NOT EXISTS WebFeatureChromiumHistogramEnums (
    WebFeatureID STRING(36) NOT NULL,
    ChromiumHistogramEnumID STRING(36),
    FOREIGN KEY (WebFeatureID) REFERENCES WebFeatures(ID)  ON DELETE CASCADE,
    FOREIGN KEY (ChromiumHistogramEnumID) REFERENCES ChromiumHistogramEnums(ID)  ON DELETE CASCADE
) PRIMARY KEY (WebFeatureID);