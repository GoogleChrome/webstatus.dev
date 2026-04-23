// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httpserver

import (
	"testing"
)

// TestServerOption defines a function type to override Server fields in tests.
type TestServerOption func(*Server)

// setupTestServer creates a Server instance initialized with safe defaults for testing.
func setupTestServer(t *testing.T, options ...TestServerOption) *Server {
	t.Helper()

	// Default mock implementation
	mockStorer := &MockWPTMetricsStorer{
		t:                                                 t,
		featureCfg:                                        nil,
		aggregateCfg:                                      nil,
		listChromeDailyUsageStatsCfg:                      nil,
		featuresSearchCfg:                                 nil,
		getFeatureByIDConfig:                              nil,
		listBrowserFeatureCountMetricCfg:                  nil,
		getIDFromFeatureKeyConfig:                         nil,
		listMissingOneImplCountCfg:                        nil,
		listMissingOneImplFeaturesCfg:                     nil,
		listBaselineStatusCountsCfg:                       nil,
		createUserSavedSearchCfg:                          nil,
		deleteUserSavedSearchCfg:                          nil,
		getSavedSearchCfg:                                 nil,
		listUserSavedSearchesCfg:                          nil,
		getSavedSearchPublicCfg:                           nil,
		updateUserSavedSearchCfg:                          nil,
		putUserSavedSearchBookmarkCfg:                     nil,
		removeUserSavedSearchBookmarkCfg:                  nil,
		syncUserProfileInfoCfg:                            nil,
		getNotificationChannelCfg:                         nil,
		listNotificationChannelsCfg:                       nil,
		deleteNotificationChannelCfg:                      nil,
		createNotificationChannelCfg:                      nil,
		updateNotificationChannelCfg:                      nil,
		createSavedSearchSubscriptionCfg:                  nil,
		deleteSavedSearchSubscriptionCfg:                  nil,
		getSavedSearchSubscriptionCfg:                     nil,
		getSavedSearchSubscriptionPublicCfg:               nil,
		listSavedSearchSubscriptionsCfg:                   nil,
		listSavedSearchNotificationEventsCfg:              nil,
		updateSavedSearchSubscriptionCfg:                  nil,
		validateQueryReferencesCfg:                        nil,
		listGlobalSavedSearchesCfg:                        nil,
		callCountListMetricsForFeatureIDBrowserAndChannel: 0,
		callCountListMetricsOverTimeWithAggregatedTotals:  0,
		callCountListChromeDailyUsageStats:                0,
		callCountFeaturesSearch:                           0,
		callCountGetFeature:                               0,
		callCountListBrowserFeatureCountMetric:            0,
		callCountListMissingOneImplCounts:                 0,
		callCountListMissingOneImplFeatures:               0,
		callCountListBaselineStatusCounts:                 0,
		callCountCreateUserSavedSearch:                    0,
		callCountDeleteUserSavedSearch:                    0,
		callCountGetSavedSearch:                           0,
		callCountListUserSavedSearches:                    0,
		callCountGetSavedSearchPublic:                     0,
		callCountUpdateUserSavedSearch:                    0,
		callCountPutUserSavedSearchBookmark:               0,
		callCountRemoveUserSavedSearchBookmark:            0,
		callCountSyncUserProfileInfo:                      0,
		callCountGetNotificationChannel:                   0,
		callCountListNotificationChannels:                 0,
		callCountDeleteNotificationChannel:                0,
		callCountCreateNotificationChannel:                0,
		callCountUpdateNotificationChannel:                0,
		callCountCreateSavedSearchSubscription:            0,
		callCountDeleteSavedSearchSubscription:            0,
		callCountGetSavedSearchSubscription:               0,
		callCountListSavedSearchSubscriptions:             0,
		callCountUpdateSavedSearchSubscription:            0,
		callCountListSavedSearchNotificationEvents:        0,
		callCountValidateQueryReferences:                  0,
		callCountListGlobalSavedSearches:                  0,
		callCountGetGlobalSavedSearch:                     0,
		callCountGetSavedSearchSubscriptionPublic:         0,
	}

	srv := &Server{
		metadataStorer:          nil,
		wptMetricsStorer:        mockStorer,
		operationResponseCaches: nil,
		baseURL:                 getTestBaseURL(t),
		userGitHubClientFactory: nil,
		eventPublisher:          nil,
		rssRenderer:             NewRSSRenderer(),
	}

	// Apply Functional Options to override defaults
	for _, option := range options {
		option(srv)
	}

	return srv
}

// Helper options to set specialized mocks if needed in tests

func withCustomStorer(s WPTMetricsStorer) TestServerOption {
	return func(srv *Server) {
		srv.wptMetricsStorer = s
	}
}

func withCustomMetadataStorer(m WebFeatureMetadataStorer) TestServerOption {
	return func(srv *Server) {
		srv.metadataStorer = m
	}
}

func withCustomEventPublisher(p EventPublisher) TestServerOption {
	return func(srv *Server) {
		srv.eventPublisher = p
	}
}

func withCustomCaches(c *operationResponseCaches) TestServerOption {
	return func(srv *Server) {
		srv.operationResponseCaches = c
	}
}

func withCustomGitHubClientFactory(f UserGitHubClientFactory) TestServerOption {
	return func(srv *Server) {
		srv.userGitHubClientFactory = f
	}
}
