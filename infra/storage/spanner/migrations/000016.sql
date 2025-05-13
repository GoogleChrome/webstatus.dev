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

-- WebFeatureBrowserCompatFeatures stores the compat_features list (e.g. "html.elements.address")
-- for each WebFeature. Multiple compat features may exist per feature.
CREATE TABLE IF NOT EXISTS WebFeatureBrowserCompatFeatures (
    ID STRING(36) NOT NULL, -- same name and type as parent
    CompatFeature STRING(255) NOT NULL,
    FOREIGN KEY (ID) REFERENCES WebFeatures(ID)
) PRIMARY KEY (ID, CompatFeature)
,   INTERLEAVE IN PARENT WebFeatures ON DELETE CASCADE;

-- Index to accelerate searches by CompatFeature
CREATE INDEX IDX_CompatFeature ON WebFeatureBrowserCompatFeatures(CompatFeature);
