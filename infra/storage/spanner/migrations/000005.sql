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

-- LatestWPTRunFeatureMetrics contains latest snapshot of WPT metric information for a web feature.
CREATE TABLE IF NOT EXISTS LatestWPTRunFeatureMetrics (
    ID STRING(36) NOT NULL,
    WebFeatureID STRING(36) NOT NULL,
    BrowserName STRING(64) NOT NULL,
    Channel STRING(32) NOT NULL,
    FOREIGN KEY (ID, WebFeatureID) REFERENCES WPTRunFeatureMetrics(ID, WebFeatureID) ON DELETE CASCADE
) PRIMARY KEY (WebFeatureID, BrowserName, Channel);