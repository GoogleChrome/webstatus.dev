-- Copyright 2025 Google LLC
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

CREATE TABLE IF NOT EXISTS MovedWebFeatures (
    ID STRING(36) NOT NULL DEFAULT (GENERATE_UUID()),
    OriginalFeatureKey STRING(64) NOT NULL, -- From web features repo.
    TargetWebFeatureID STRING(36) NOT NULL, -- From web features repo.
    -- Additional lowercase columns for case-insensitive search
    OriginalFeatureKey_Lowercase STRING(64) AS (LOWER(OriginalFeatureKey)) STORED,
    CONSTRAINT FK_MovedWebFeaturesTargetWebFeatureID FOREIGN KEY (TargetWebFeatureID) REFERENCES WebFeatures(ID) ON DELETE CASCADE,
) PRIMARY KEY (ID);

CREATE TABLE IF NOT EXISTS SplitWebFeatures (
    ID STRING(36) NOT NULL DEFAULT (GENERATE_UUID()),
    OriginalFeatureKey STRING(64) NOT NULL, -- From web features repo.
    TargetWebFeatureID STRING(36) NOT NULL, -- From web features repo.
    -- Additional lowercase columns for case-insensitive search
    OriginalFeatureKey_Lowercase STRING(64) AS (LOWER(OriginalFeatureKey)) STORED,
    CONSTRAINT FK_SplitWebFeatures_TargetWebFeatureID FOREIGN KEY (TargetWebFeatureID) REFERENCES WebFeatures(ID) ON DELETE CASCADE,
) PRIMARY KEY (ID);
