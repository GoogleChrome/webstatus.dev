/**
 * Copyright 2026 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import {type components} from 'webstatus.dev-backend';

import globalSavedSearches from './global-saved-searches.json';
import userSavedSearches from './user-saved-searches.json';
import userNotificationChannels from './user-notification-channels.json';
import featuresPage1 from './features-page-1.json';
import featuresPage2 from './features-page-2.json';
import featureDetailAnchorPositioning from './feature-detail-anchor-positioning.json';
import featureDetailDiscouraged from './feature-detail-discouraged.json';
import featureWptMetrics from './feature-wpt-metrics.json';
import featureUmaMetrics from './feature-uma-metrics.json';
import statsLowDateCounts from './stats-low-date-counts.json';
import statsMissingOneImpl from './stats-missing-one-impl.json';

import {
  featureWptMetricsChromePage1,
  featureWptMetricsChromePage2,
  featureWptMetricsFirefoxPage1,
  featureWptMetricsFirefoxPage2,
  featureWptMetricsSafariPage1,
  featureWptMetricsSafariPage2,
  featureWptMetricsEdgePage1,
  featureWptMetricsEdgePage2,
  featureUmaMetricsPage1,
  featureUmaMetricsPage2,
  statsFeatureCountsChromePage1,
  statsFeatureCountsChromePage2,
  statsFeatureCountsFirefoxPage1,
  statsFeatureCountsFirefoxPage2,
  statsFeatureCountsSafariPage1,
  statsFeatureCountsSafariPage2,
  statsLowDateCountsPage1,
  statsLowDateCountsPage2,
  statsMissingOneChromePage1,
  statsMissingOneChromePage2,
  statsMissingOneFirefoxPage1,
  statsMissingOneFirefoxPage2,
  statsMissingOneSafariPage1,
  statsMissingOneSafariPage2,
  statsMissingFeaturesListPage1,
  statsMissingFeaturesListPage2,
} from './mock-data';

// Compile-time assertions: Validates that fixture shapes strictly conform to OpenAPI models
export const _typeCheckGlobalSavedSearches =
  globalSavedSearches satisfies components['schemas']['GlobalSavedSearchPage'];

export const _typeCheckUserSavedSearches =
  userSavedSearches satisfies components['schemas']['UserSavedSearchPage'];

export const _typeCheckUserNotificationChannels =
  userNotificationChannels satisfies components['schemas']['NotificationChannelPage'];

export const _typeCheckFeaturesPage1 =
  featuresPage1 satisfies components['schemas']['FeaturePage'];

export const _typeCheckFeaturesPage2 =
  featuresPage2 satisfies components['schemas']['FeaturePage'];

export const _typeCheckFeatureDetail =
  featureDetailAnchorPositioning satisfies components['schemas']['Feature'];

export const _typeCheckFeatureDiscouraged =
  featureDetailDiscouraged satisfies components['schemas']['Feature'];

export const _typeCheckWptMetricsChrome1 =
  featureWptMetricsChromePage1 satisfies components['schemas']['WPTRunMetricsPage'];
export const _typeCheckWptMetricsChrome2 =
  featureWptMetricsChromePage2 satisfies components['schemas']['WPTRunMetricsPage'];
export const _typeCheckWptMetricsFirefox1 =
  featureWptMetricsFirefoxPage1 satisfies components['schemas']['WPTRunMetricsPage'];
export const _typeCheckWptMetricsFirefox2 =
  featureWptMetricsFirefoxPage2 satisfies components['schemas']['WPTRunMetricsPage'];
export const _typeCheckWptMetricsSafari1 =
  featureWptMetricsSafariPage1 satisfies components['schemas']['WPTRunMetricsPage'];
export const _typeCheckWptMetricsSafari2 =
  featureWptMetricsSafariPage2 satisfies components['schemas']['WPTRunMetricsPage'];
export const _typeCheckWptMetricsEdge1 =
  featureWptMetricsEdgePage1 satisfies components['schemas']['WPTRunMetricsPage'];
export const _typeCheckWptMetricsEdge2 =
  featureWptMetricsEdgePage2 satisfies components['schemas']['WPTRunMetricsPage'];

export const _typeCheckUmaMetrics1 =
  featureUmaMetricsPage1 satisfies components['schemas']['ChromeDailyStatsPage'];
export const _typeCheckUmaMetrics2 =
  featureUmaMetricsPage2 satisfies components['schemas']['ChromeDailyStatsPage'];

export const _typeCheckStatsFeatureCountsChrome1 =
  statsFeatureCountsChromePage1 satisfies components['schemas']['BrowserReleaseFeatureMetricsPage'];
export const _typeCheckStatsFeatureCountsChrome2 =
  statsFeatureCountsChromePage2 satisfies components['schemas']['BrowserReleaseFeatureMetricsPage'];
export const _typeCheckStatsFeatureCountsFirefox1 =
  statsFeatureCountsFirefoxPage1 satisfies components['schemas']['BrowserReleaseFeatureMetricsPage'];
export const _typeCheckStatsFeatureCountsFirefox2 =
  statsFeatureCountsFirefoxPage2 satisfies components['schemas']['BrowserReleaseFeatureMetricsPage'];
export const _typeCheckStatsFeatureCountsSafari1 =
  statsFeatureCountsSafariPage1 satisfies components['schemas']['BrowserReleaseFeatureMetricsPage'];
export const _typeCheckStatsFeatureCountsSafari2 =
  statsFeatureCountsSafariPage2 satisfies components['schemas']['BrowserReleaseFeatureMetricsPage'];

export const _typeCheckStatsLowDate1 =
  statsLowDateCountsPage1 satisfies components['schemas']['BaselineStatusMetricsPage'];
export const _typeCheckStatsLowDate2 =
  statsLowDateCountsPage2 satisfies components['schemas']['BaselineStatusMetricsPage'];

export const _typeCheckStatsMissingOneChrome1 =
  statsMissingOneChromePage1 satisfies components['schemas']['BrowserReleaseFeatureMetricsPage'];
export const _typeCheckStatsMissingOneChrome2 =
  statsMissingOneChromePage2 satisfies components['schemas']['BrowserReleaseFeatureMetricsPage'];
export const _typeCheckStatsMissingOneFirefox1 =
  statsMissingOneFirefoxPage1 satisfies components['schemas']['BrowserReleaseFeatureMetricsPage'];
export const _typeCheckStatsMissingOneFirefox2 =
  statsMissingOneFirefoxPage2 satisfies components['schemas']['BrowserReleaseFeatureMetricsPage'];
export const _typeCheckStatsMissingOneSafari1 =
  statsMissingOneSafariPage1 satisfies components['schemas']['BrowserReleaseFeatureMetricsPage'];
export const _typeCheckStatsMissingOneSafari2 =
  statsMissingOneSafariPage2 satisfies components['schemas']['BrowserReleaseFeatureMetricsPage'];

export const _typeCheckStatsMissingFeaturesList1 =
  statsMissingFeaturesListPage1 satisfies components['schemas']['MissingOneImplFeaturesPage'];
export const _typeCheckStatsMissingFeaturesList2 =
  statsMissingFeaturesListPage2 satisfies components['schemas']['MissingOneImplFeaturesPage'];
