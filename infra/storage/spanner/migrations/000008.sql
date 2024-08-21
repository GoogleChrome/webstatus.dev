
-- ChromiumHistogramEnums contains metadata about histograms found in
-- https://chromium.googlesource.com/chromium/src/+/main/tools/metrics/histograms/enums.xml
CREATE TABLE IF NOT EXISTS ChromiumHistogramEnums (
    ID STRING(36) NOT NULL DEFAULT (GENERATE_UUID()),
    HistogramName STRING(MAX) NOT NULL,
    BucketID INT64 NOT NULL,
    Label STRING(MAX) NOT NULL,
    Name STRING(64) NOT NULL,
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