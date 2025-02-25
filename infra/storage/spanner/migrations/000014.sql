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

-- FeatureDiscouragedDetails contains information about why a feature is discouraged
CREATE TABLE IF NOT EXISTS FeatureDiscouragedDetails (
    WebFeatureID STRING(MAX) NOT NULL,
    AccordingTo ARRAY<STRING(MAX)>,
    Alternatives ARRAY<STRING(MAX)>,
    CONSTRAINT FK_DiscouragedDetailsWebFeatureID FOREIGN KEY (WebFeatureID) REFERENCES WebFeatures(ID) ON DELETE CASCADE,
) PRIMARY KEY (WebFeatureID);