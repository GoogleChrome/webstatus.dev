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

-- Indices to optimize missing-in-one-browser feature queries.
CREATE INDEX BrowserFeatureSupportEvents_FeatureStatusPerBrowserRelease ON
    BrowserFeatureSupportEvents (
        TargetBrowserName,
        WebFeatureID,
        SupportStatus,
        EventReleaseDate
    );

CREATE INDEX BrowserFeatureSupportEvents_OtherBrowserSupported ON
    BrowserFeatureSupportEvents (
        TargetBrowserName,
        SupportStatus,
        WebFeatureID,
        EventReleaseDate
    );

CREATE INDEX BrowserReleases_BrowserNamesByRelease ON
    BrowserReleases (
        BrowserName,
        ReleaseDate
    );
