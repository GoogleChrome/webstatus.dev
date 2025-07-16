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

-- A denormalized lookup table mapping features to direct/inherited groups.
-- It is interleaved in WebDXGroups to optimize search query JOINs.
CREATE TABLE IF NOT EXISTS FeatureGroupIDsLookup (
    -- This column's name, 'ID', must match the primary key
    -- column's name in the interleaved parent table, 'WebDXGroups'.
    ID STRING(36) NOT NULL,
    WebFeatureID STRING(36) NOT NULL,
    -- The hierarchy level of the association. A value of 0 means a direct
    -- link. A value of 1 means this group is the direct parent of the
    -- feature's group, 2 means it's a grandparent, and so on.
    -- TODO. Future queries may use this column
    Depth INT64 NOT NULL,
    CONSTRAINT FK_FeatureGroupIDsLookup_WebFeatures FOREIGN KEY (WebFeatureID) REFERENCES WebFeatures(ID) ON DELETE CASCADE,
    CONSTRAINT FK_FeatureGroupIDsLookup_WebDXGroups FOREIGN KEY (ID) REFERENCES WebDXGroups(ID)
) PRIMARY KEY (ID, WebFeatureID)
,   INTERLEAVE IN PARENT WebDXGroups ON DELETE CASCADE;