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

-- These fields will always be there according to:
-- https://github.com/web-platform-dx/web-features/blob/6a8a139eaf30a64422a94d825b87ab9549cb6b89/schemas/data.schema.json#L257-L259

-- Add description to web features.
ALTER TABLE WebFeatures ADD COLUMN Description STRING(MAX) NOT NULL DEFAULT ('');
ALTER TABLE WebFeatures ADD COLUMN DescriptionHtml STRING(MAX) NOT NULL DEFAULT ('');
-- Lowercase only the description column for basic searching for now.
ALTER TABLE WebFeatures ADD COLUMN Description_Lowercase STRING(MAX) AS (LOWER(Description)) STORED;
