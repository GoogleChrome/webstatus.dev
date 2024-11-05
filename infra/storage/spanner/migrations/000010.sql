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

-- This table provides an exhaustive view of feature support events for each browser,
-- considering the releases of ALL browsers.
-- It differs from BrowserFeatureAvailabilities, which only lists features explicitly
-- mentioned as supported in a specific release.
--
-- TargetBrowserName: The browser for which we're tracking feature support.
-- EventBrowserName: The browser whose release potentially affects the support status.
-- EventReleaseDate: The release date of the EventBrowserName.
--
-- This structure is necessary due to the lack of full support for window functions
-- in Spanner. It allows for efficient querying and analysis of "missing one
-- implementation" counts and other feature-related metrics, considering the
-- impact of releases from any browser, without relying on complex or
-- potentially inefficient queries.
CREATE TABLE IF NOT EXISTS BrowserFeatureSupportEvents (
    TargetBrowserName STRING(64) NOT NULL,  -- The browser for which we're tracking support
    EventBrowserName  STRING(64) NOT NULL,  -- The browser whose release triggered the event
    EventReleaseDate TIMESTAMP NOT NULL,  -- The release date of the EventBrowserName
    WebFeatureID      STRING(36) NOT NULL,
    SupportStatus     STRING(32) NOT NULL, -- Check lib/gcpspanner/browser_feature_support_event.go for possible values.
    CONSTRAINT FK_WebFeatureID FOREIGN KEY (WebFeatureID) REFERENCES WebFeatures(ID) ON DELETE CASCADE,
    CONSTRAINT FK_EventBrowserRelease FOREIGN KEY (EventBrowserName, EventReleaseDate) REFERENCES BrowserReleases(BrowserName, ReleaseDate) ON DELETE CASCADE
) PRIMARY KEY (TargetBrowserName, EventBrowserName, EventReleaseDate, WebFeatureID);
