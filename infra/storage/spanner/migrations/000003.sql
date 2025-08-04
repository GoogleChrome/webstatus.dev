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

-- FeatureSpecs contains spec information for a web feature.
CREATE TABLE IF NOT EXISTS FeatureSpecs (
    WebFeatureID STRING(36) NOT NULL,
    Links ARRAY<STRING(128)>,
    CONSTRAINT FK_FeatureSpecsWebFeatureID FOREIGN KEY (WebFeatureID) REFERENCES WebFeatures(ID) ON DELETE CASCADE,
) PRIMARY KEY (WebFeatureID);