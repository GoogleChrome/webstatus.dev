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

-- ExcludedFeatureKeys contains a list of feature keys that we should exclude
-- from queries. These feature keys can be current feature keys or future feature keys.
-- We use the FeatureKey instead of ID from WebFeatures so that admins
-- can easily add the features into the table in GCP without looking up the UUID.
CREATE TABLE IF NOT EXISTS ExcludedFeatureKeys (
    FeatureKey STRING(64) NOT NULL, -- From web features repo.
) PRIMARY KEY (FeatureKey);