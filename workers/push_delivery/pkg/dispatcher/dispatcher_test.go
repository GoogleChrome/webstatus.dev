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

package dispatcher

import (
	"context"
	"errors"
	"testing"
	"time"

	githubissuedeliveryv1 "github.com/GoogleChrome/webstatus.dev/lib/event/githubissuedelivery/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/google/go-cmp/cmp"
)

// --- Mocks ---

type findSubscribersReq struct {
	SearchID  string
	Frequency string
}

type findCodeSubsReq struct {
	TargetQuery string
	Trigger     gcpspanner.SubscriptionTrigger
}

type mockSubscriptionFinder struct {
	findCalledWith     *findSubscribersReq
	findReturnSet      *workertypes.SubscriberSet
	findReturnErr      error
	findCodeSubsCalls  []findCodeSubsReq
	findCodeSubsReturn []gcpspanner.CodeSubscription
	findCodeSubsErr    error
}

func (m *mockSubscriptionFinder) FindSubscribers(_ context.Context, searchID string,
	frequency workertypes.JobFrequency) (*workertypes.SubscriberSet, error) {
	m.findCalledWith = &findSubscribersReq{
		SearchID:  searchID,
		Frequency: string(frequency),
	}

	return m.findReturnSet, m.findReturnErr
}

func (m *mockSubscriptionFinder) FindCodeSubscriptions(_ context.Context, targetQuery string,
	trigger gcpspanner.SubscriptionTrigger) ([]gcpspanner.CodeSubscription, error) {
	m.findCodeSubsCalls = append(m.findCodeSubsCalls, findCodeSubsReq{
		TargetQuery: targetQuery,
		Trigger:     trigger,
	})

	return m.findCodeSubsReturn, m.findCodeSubsErr
}

type mockDeliveryPublisher struct {
	emailJobs         []workertypes.EmailDeliveryJob
	emailJobErr       func(job workertypes.EmailDeliveryJob) error
	webhookJobs       []workertypes.WebhookDeliveryJob
	webhookJobErr     func(job workertypes.WebhookDeliveryJob) error
	githubIssueJobs   []githubissuedeliveryv1.GitHubIssueDeliveryEvent
	githubIssueJobErr func(job githubissuedeliveryv1.GitHubIssueDeliveryEvent) error
}

func (m *mockDeliveryPublisher) PublishEmailJob(_ context.Context, job workertypes.EmailDeliveryJob) error {
	if m.emailJobErr != nil {
		if err := m.emailJobErr(job); err != nil {
			return err
		}
	}
	m.emailJobs = append(m.emailJobs, job)

	return nil
}

func (m *mockDeliveryPublisher) PublishWebhookJob(_ context.Context, job workertypes.WebhookDeliveryJob) error {
	if m.webhookJobErr != nil {
		if err := m.webhookJobErr(job); err != nil {
			return err
		}
	}
	m.webhookJobs = append(m.webhookJobs, job)

	return nil
}

func (m *mockDeliveryPublisher) PublishGitHubIssueJob(
	_ context.Context,
	job githubissuedeliveryv1.GitHubIssueDeliveryEvent,
) error {
	if m.githubIssueJobErr != nil {
		if err := m.githubIssueJobErr(job); err != nil {
			return err
		}
	}
	m.githubIssueJobs = append(m.githubIssueJobs, job)

	return nil
}

// --- Test Helpers ---

// createTestSummary returns a populated EventSummary for testing.
func createTestSummary(hasChanges bool) workertypes.EventSummary {
	categories := workertypes.SummaryCategories{
		QueryChanged:    0,
		Added:           0,
		Deleted:         0,
		Removed:         0,
		Moved:           0,
		Split:           0,
		Updated:         0,
		UpdatedImpl:     0,
		UpdatedRename:   0,
		UpdatedBaseline: 0,
	}

	if hasChanges {
		categories.Added = 1
	}

	summary := workertypes.NewEmptyEventSummary()
	summary.SnapshotOrigin = workertypes.OriginLive
	summary.Text = "Test Summary"
	summary.Categories = categories

	return summary
}

func createTestSummaryWithErrors(errCode workertypes.SummaryQueryErrorCode) workertypes.EventSummary {
	summary := workertypes.NewEmptyEventSummary()
	summary.SnapshotOrigin = workertypes.OriginLive
	summary.Text = "Error occurred"
	summary.SetQueryErrors([]workertypes.SummaryQueryError{{Code: errCode}})

	return summary
}

// mockParserFactory creates a SummaryParser that injects the given summary directly.
func mockParserFactory(summary workertypes.EventSummary, err error) SummaryParser {
	return func(_ []byte, v workertypes.SummaryVisitor) error {
		if err != nil {
			return err
		}

		return v.VisitV1(summary)
	}
}

// --- Tests ---

func emptyFinderReq() findSubscribersReq {
	return findSubscribersReq{
		SearchID:  "",
		Frequency: "",
	}
}

func TestProcessEvent_Success(t *testing.T) {
	ctx := context.Background()
	eventID := "evt-123"
	searchID := "search-abc"
	frequency := workertypes.FrequencyImmediate
	generatedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	summaryBytes := []byte("{}")

	metadata := workertypes.DispatchEventMetadata{
		EventID:     eventID,
		SearchID:    searchID,
		SearchName:  "Test Search",
		Query:       "q=test",
		Frequency:   frequency,
		GeneratedAt: generatedAt,
	}

	// Two subscribers: one matching trigger, one not.
	subSet := &workertypes.SubscriberSet{
		Emails: []workertypes.EmailSubscriber{
			{
				SubscriptionID: "sub-1",
				UserID:         "user-1",
				Triggers:       []workertypes.JobTrigger{workertypes.FeaturePromotedToNewly}, // Matches
				EmailAddress:   "user1@example.com",
				ChannelID:      "chan-1",
			},
			{
				SubscriptionID: "sub-2",
				UserID:         "user-2",
				// Does not match (summary is Newly)
				Triggers:     []workertypes.JobTrigger{workertypes.FeaturePromotedToWidely},
				EmailAddress: "user2@example.com",
				ChannelID:    "chan-2",
			},
		},
		Webhooks: []workertypes.WebhookSubscriber{},
	}

	finder := &mockSubscriptionFinder{
		findReturnSet:      subSet,
		findReturnErr:      nil,
		findCalledWith:     nil,
		findCodeSubsCalls:  nil,
		findCodeSubsReturn: nil,
		findCodeSubsErr:    nil,
	}
	publisher := new(mockDeliveryPublisher)

	// Create a summary that HAS changes so notification logic proceeds.
	summary := createTestSummary(true)
	summary.Categories.UpdatedBaseline = 1
	summary.Categories.Updated = 1
	summary.AddHighlight(workertypes.SummaryHighlight{
		Type:        workertypes.SummaryHighlightTypeChanged,
		FeatureID:   "test-feature-id",
		FeatureName: "Test Feature",
		Docs:        nil,
		NameChange:  nil,
		BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
			From: newBaselineValue(workertypes.BaselineStatusLimited),
			To:   newBaselineValue(workertypes.BaselineStatusNewly),
		},
		BrowserChanges: nil,
		Moved:          nil,
		Split:          nil,
	})
	parser := mockParserFactory(summary, nil)

	d := NewDispatcher(finder, publisher)
	d.parser = parser

	if err := d.ProcessEvent(ctx, metadata, summaryBytes); err != nil {
		t.Fatalf("ProcessEvent unexpected error: %v", err)
	}

	// Assertions
	expectedFinderReq := findSubscribersReq{
		SearchID:  searchID,
		Frequency: string(frequency),
	}
	assertFindSubscribersCalledWith(t, finder, &expectedFinderReq)

	if len(publisher.emailJobs) != 1 {
		t.Fatalf("Expected 1 email job, got %d", len(publisher.emailJobs))
	}

	job := publisher.emailJobs[0]
	expectedJob := workertypes.EmailDeliveryJob{
		SubscriptionID: "sub-1",
		RecipientEmail: "user1@example.com",
		SummaryRaw:     summaryBytes,
		Triggers:       []workertypes.JobTrigger{workertypes.FeaturePromotedToNewly},
		Metadata: workertypes.DeliveryMetadata{
			EventID:     eventID,
			SearchID:    searchID,
			SearchName:  "Test Search",
			Query:       "q=test",
			Frequency:   frequency,
			GeneratedAt: generatedAt,
		},
		ChannelID: "chan-1",
	}

	if diff := cmp.Diff(expectedJob, job); diff != "" {
		t.Errorf("Job mismatch (-want +got):\n%s", diff)
	}
}

func TestProcessEvent_Webhook_Success(t *testing.T) {
	ctx := context.Background()
	eventID := "evt-123"
	searchID := "search-abc"
	frequency := workertypes.FrequencyImmediate
	generatedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	summaryBytes := []byte("{}")

	metadata := workertypes.DispatchEventMetadata{
		EventID:     eventID,
		SearchID:    searchID,
		SearchName:  "",
		Query:       "q=test",
		Frequency:   frequency,
		GeneratedAt: generatedAt,
	}

	// Two subscribers: one matching trigger, one not.
	subSet := &workertypes.SubscriberSet{
		Emails: []workertypes.EmailSubscriber{},
		Webhooks: []workertypes.WebhookSubscriber{
			{
				SubscriptionID: "sub-1",
				UserID:         "user-1",
				Triggers:       []workertypes.JobTrigger{workertypes.FeaturePromotedToNewly}, // Matches
				WebhookURL:     "https://hooks.slack.com/services/123",
				WebhookType:    workertypes.WebhookTypeSlack,
				ChannelID:      "chan-1",
			},
			{
				SubscriptionID: "sub-2",
				UserID:         "user-2",
				// Does not match (summary is Newly)
				Triggers:    []workertypes.JobTrigger{workertypes.FeaturePromotedToWidely},
				WebhookURL:  "https://hooks.slack.com/services/456",
				WebhookType: workertypes.WebhookTypeSlack,
				ChannelID:   "chan-2",
			},
		},
	}

	finder := &mockSubscriptionFinder{
		findReturnSet:      subSet,
		findReturnErr:      nil,
		findCalledWith:     nil,
		findCodeSubsCalls:  nil,
		findCodeSubsReturn: nil,
		findCodeSubsErr:    nil,
	}
	publisher := new(mockDeliveryPublisher)

	// Create a summary that HAS changes so notification logic proceeds.
	summary := createTestSummary(true)
	summary.Categories.UpdatedBaseline = 1
	summary.Categories.Updated = 1
	summary.AddHighlight(workertypes.SummaryHighlight{
		Type:        workertypes.SummaryHighlightTypeChanged,
		FeatureID:   "test-feature-id",
		FeatureName: "Test Feature",
		Docs:        nil,
		NameChange:  nil,
		BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
			From: newBaselineValue(workertypes.BaselineStatusLimited),
			To:   newBaselineValue(workertypes.BaselineStatusNewly),
		},
		BrowserChanges: nil,
		Moved:          nil,
		Split:          nil,
	})
	parser := mockParserFactory(summary, nil)

	d := NewDispatcher(finder, publisher)
	d.parser = parser

	if err := d.ProcessEvent(ctx, metadata, summaryBytes); err != nil {
		t.Fatalf("ProcessEvent unexpected error: %v", err)
	}

	// Assertions
	expectedFinderReq := findSubscribersReq{
		SearchID:  searchID,
		Frequency: string(frequency),
	}
	assertFindSubscribersCalledWith(t, finder, &expectedFinderReq)

	if len(publisher.webhookJobs) != 1 {
		t.Fatalf("Expected 1 webhook job, got %d", len(publisher.webhookJobs))
	}

	job := publisher.webhookJobs[0]
	expectedJob := workertypes.WebhookDeliveryJob{
		SubscriptionID: "sub-1",
		WebhookURL:     "https://hooks.slack.com/services/123",
		WebhookType:    workertypes.WebhookTypeSlack,
		SummaryRaw:     summaryBytes,
		Triggers:       []workertypes.JobTrigger{workertypes.FeaturePromotedToNewly},
		Metadata: workertypes.DeliveryMetadata{
			EventID:     eventID,
			SearchID:    searchID,
			SearchName:  "",
			Query:       "q=test",
			Frequency:   frequency,
			GeneratedAt: generatedAt,
		},
		ChannelID: "chan-1",
	}

	if diff := cmp.Diff(expectedJob, job); diff != "" {
		t.Errorf("Job mismatch (-want +got):\n%s", diff)
	}
}

func assertFindSubscribersCalledWith(t *testing.T, finder *mockSubscriptionFinder, expected *findSubscribersReq) {
	t.Helper()
	if diff := cmp.Diff(expected, finder.findCalledWith); diff != "" {
		t.Errorf("FindSubscribers called with mismatch (-want +got):\n%s", diff)
	}
}

func TestProcessEvent_NoChanges_FiltersAll(t *testing.T) {
	ctx := context.Background()
	metadata := workertypes.DispatchEventMetadata{
		EventID:     "evt-1",
		SearchID:    "search-1",
		SearchName:  "Test Search",
		Frequency:   workertypes.FrequencyImmediate,
		Query:       "",
		GeneratedAt: time.Time{},
	}

	subSet := &workertypes.SubscriberSet{
		Emails: []workertypes.EmailSubscriber{
			{
				SubscriptionID: "sub-1",
				UserID:         "user-1",
				Triggers:       []workertypes.JobTrigger{"any_change"},
				EmailAddress:   "user1@example.com",
				ChannelID:      "chan-1",
			},
		},
		Webhooks: []workertypes.WebhookSubscriber{},
	}

	finder := &mockSubscriptionFinder{
		findReturnSet:      subSet,
		findReturnErr:      nil,
		findCalledWith:     nil,
		findCodeSubsCalls:  nil,
		findCodeSubsReturn: nil,
		findCodeSubsErr:    nil,
	}
	publisher := new(mockDeliveryPublisher)

	// Summary with NO changes
	summary := createTestSummary(false)
	parser := mockParserFactory(summary, nil)

	d := NewDispatcher(finder, publisher)
	d.parser = parser

	if err := d.ProcessEvent(ctx, metadata, []byte("{}")); err != nil {
		t.Fatalf("ProcessEvent unexpected error: %v", err)
	}

	if len(publisher.emailJobs) != 0 {
		t.Errorf("Expected 0 jobs due to no changes, got %d", len(publisher.emailJobs))
	}
}

func TestProcessEvent_ParserError(t *testing.T) {
	d := NewDispatcher(nil, nil)
	var summary workertypes.EventSummary
	d.parser = mockParserFactory(summary, errors.New("parse error"))

	metadata := workertypes.DispatchEventMetadata{
		EventID:     "",
		SearchID:    "",
		SearchName:  "",
		Query:       "",
		Frequency:   workertypes.FrequencyImmediate,
		GeneratedAt: time.Time{},
	}

	err := d.ProcessEvent(context.Background(), metadata, []byte("{}"))
	if err == nil {
		t.Error("Expected error from parser, got nil")
	}
}

func TestProcessEvent_FinderError(t *testing.T) {
	finder := &mockSubscriptionFinder{
		findReturnSet:      nil,
		findReturnErr:      errors.New("db error"),
		findCalledWith:     nil,
		findCodeSubsCalls:  nil,
		findCodeSubsReturn: nil,
		findCodeSubsErr:    nil,
	}

	d := NewDispatcher(finder, nil)
	// Provide a valid summary struct so parser succeeds
	var summary workertypes.EventSummary
	d.parser = mockParserFactory(summary, nil)

	metadata := workertypes.DispatchEventMetadata{
		EventID:     "",
		SearchID:    "",
		SearchName:  "",
		Query:       "",
		Frequency:   "",
		GeneratedAt: time.Time{},
	}

	err := d.ProcessEvent(context.Background(), metadata, []byte("{}"))
	if err == nil {
		t.Error("Expected error from finder, got nil")
	}
	assertFindSubscribersCalledWith(t, finder, new(emptyFinderReq()))
}

func TestProcessEvent_PublisherPartialFailure(t *testing.T) {
	ctx := context.Background()
	// Two subscribers
	subSet := &workertypes.SubscriberSet{
		Emails: []workertypes.EmailSubscriber{
			{SubscriptionID: "sub-1", Triggers: []workertypes.JobTrigger{workertypes.FeaturePromotedToNewly},
				UserID: "u1", EmailAddress: "e1", ChannelID: "chan-1"},
			{SubscriptionID: "sub-2", Triggers: []workertypes.JobTrigger{workertypes.FeaturePromotedToNewly},
				UserID: "u2", EmailAddress: "e2", ChannelID: "chan-2"},
		},
		Webhooks: []workertypes.WebhookSubscriber{},
	}

	finder := &mockSubscriptionFinder{
		findReturnSet:      subSet,
		findReturnErr:      nil,
		findCalledWith:     nil,
		findCodeSubsCalls:  nil,
		findCodeSubsReturn: nil,
		findCodeSubsErr:    nil,
	}

	// Publisher returns error for first job, success for second
	publisher := &mockDeliveryPublisher{
		emailJobs:         nil,
		webhookJobs:       nil,
		githubIssueJobs:   nil,
		githubIssueJobErr: nil,
		emailJobErr: func(job workertypes.EmailDeliveryJob) error {
			if job.SubscriptionID == "sub-1" {
				return errors.New("queue full")
			}

			return nil
		},
		webhookJobErr: nil,
	}

	summaryWithNewly := withBaselineHighlight(createTestSummary(false),
		workertypes.BaselineStatusLimited, workertypes.BaselineStatusNewly)
	d := NewDispatcher(finder, publisher)
	d.parser = mockParserFactory(summaryWithNewly, nil)

	metadata := workertypes.DispatchEventMetadata{
		EventID:     "",
		SearchID:    "",
		SearchName:  "",
		Query:       "",
		Frequency:   "",
		GeneratedAt: time.Time{},
	}

	err := d.ProcessEvent(ctx, metadata, []byte("{}"))
	if err == nil {
		t.Error("Expected error due to partial publish failure")
	}

	if len(publisher.emailJobs) != 1 {
		t.Errorf("Expected 1 successful job recorded, got %d", len(publisher.emailJobs))
	}
	if publisher.emailJobs[0].SubscriptionID != "sub-2" {
		t.Errorf("Expected sub-2 to succeed, got %s", publisher.emailJobs[0].SubscriptionID)
	}
	if publisher.emailJobs[0].ChannelID != "chan-2" {
		t.Errorf("Expected chan-2 to succeed, got %s", publisher.emailJobs[0].ChannelID)
	}
	assertFindSubscribersCalledWith(t, finder, new(emptyFinderReq()))
}

func TestProcessEvent_JobCount(t *testing.T) {
	// Verify that if no jobs are generated (e.g. no matching triggers), ProcessEvent returns early/cleanly.
	subSet := &workertypes.SubscriberSet{
		Emails: []workertypes.EmailSubscriber{
			{SubscriptionID: "sub-1", Triggers: []workertypes.JobTrigger{}, EmailAddress: "e1", UserID: "u1",
				ChannelID: "chan-1"}, // No match
		},
		Webhooks: []workertypes.WebhookSubscriber{},
	}
	finder := &mockSubscriptionFinder{
		findReturnSet:      subSet,
		findReturnErr:      nil,
		findCalledWith:     nil,
		findCodeSubsCalls:  nil,
		findCodeSubsReturn: nil,
		findCodeSubsErr:    nil,
	}
	publisher := new(mockDeliveryPublisher)
	d := NewDispatcher(finder, publisher)
	d.parser = mockParserFactory(createTestSummary(true), nil)

	metadata := workertypes.DispatchEventMetadata{
		EventID:     "",
		SearchID:    "",
		SearchName:  "",
		Query:       "",
		Frequency:   "",
		GeneratedAt: time.Time{},
	}

	if err := d.ProcessEvent(context.Background(), metadata, []byte("{}")); err != nil {
		t.Errorf("Expected no error for 0 jobs, got %v", err)
	}
	if len(publisher.emailJobs) != 0 {
		t.Error("Expected 0 jobs")
	}
	assertFindSubscribersCalledWith(t, finder, new(emptyFinderReq()))
}

// --- shouldNotifyV1 Test Helpers ---

func newBaselineValue(status workertypes.BaselineStatus) workertypes.BaselineValue {
	t := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	return workertypes.BaselineValue{
		Status:   status,
		LowDate:  &t,
		HighDate: nil,
	}
}

func newBrowserValue(status workertypes.BrowserStatus) workertypes.BrowserValue {
	version := "100"
	testDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	return workertypes.BrowserValue{
		Status:  status,
		Version: &version,
		Date:    &testDate,
	}
}

func withBaselineHighlight(
	s workertypes.EventSummary, from, to workertypes.BaselineStatus) workertypes.EventSummary {
	s.AddHighlight(workertypes.SummaryHighlight{
		Type:        workertypes.SummaryHighlightTypeChanged,
		FeatureID:   "test-feature-id",
		FeatureName: "Test Feature",
		Docs:        nil,
		BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
			From: newBaselineValue(from),
			To:   newBaselineValue(to),
		},
		BrowserChanges: nil,
		NameChange:     nil,
		Moved:          nil,
		Split:          nil,
	})
	s.Categories.Updated = 1
	s.Categories.UpdatedBaseline = 1

	return s
}

func withBrowserChangeHighlight(
	s workertypes.EventSummary, from, to workertypes.BrowserStatus) workertypes.EventSummary {
	s.AddHighlight(workertypes.SummaryHighlight{
		Type:           workertypes.SummaryHighlightTypeChanged,
		FeatureID:      "test-feature-id",
		FeatureName:    "Test Feature",
		Docs:           nil,
		BaselineChange: nil,
		BrowserChanges: map[workertypes.BrowserName]*workertypes.Change[workertypes.BrowserValue]{
			workertypes.BrowserChrome: {
				From: newBrowserValue(from),
				To:   newBrowserValue(to),
			},
			workertypes.BrowserChromeAndroid:  nil,
			workertypes.BrowserFirefox:        nil,
			workertypes.BrowserFirefoxAndroid: nil,
			workertypes.BrowserEdge:           nil,
			workertypes.BrowserSafari:         nil,
			workertypes.BrowserSafariIos:      nil,
		},
		NameChange: nil,
		Moved:      nil,
		Split:      nil,
	})
	s.Categories.Updated = 1
	s.Categories.UpdatedImpl = 1

	return s
}

func TestShouldNotifyV1(t *testing.T) {
	summaryWithNewly := withBaselineHighlight(createTestSummary(false),
		workertypes.BaselineStatusLimited, workertypes.BaselineStatusNewly)
	summaryWithWidely := withBaselineHighlight(createTestSummary(false),
		workertypes.BaselineStatusNewly, workertypes.BaselineStatusWidely)
	summaryWithLimited := withBaselineHighlight(createTestSummary(false),
		workertypes.BaselineStatusWidely, workertypes.BaselineStatusLimited)
	summaryWithBrowserAvailable := withBrowserChangeHighlight(createTestSummary(false),
		workertypes.BrowserStatusUnknown, workertypes.BrowserStatusAvailable)
	summaryWithBrowserInDev := withBrowserChangeHighlight(createTestSummary(false),
		workertypes.BrowserStatusUnknown, workertypes.BrowserStatusUnknown)
	summaryQueryChanged := createTestSummary(false)
	summaryQueryChanged.Categories.QueryChanged = 1

	testCases := []struct {
		name     string
		triggers []workertypes.JobTrigger
		summary  workertypes.EventSummary
		want     bool
	}{
		{
			name:     "no changes should return false",
			triggers: []workertypes.JobTrigger{workertypes.FeaturePromotedToNewly},
			summary:  createTestSummary(false),
			want:     false,
		},
		{
			name:     "query errors present should return true immediately regardless of triggers",
			triggers: []workertypes.JobTrigger{workertypes.FeaturePromotedToWidely},
			summary:  createTestSummaryWithErrors(workertypes.SummaryQueryErrorCodeQueryGrammar),
			want:     true,
		},
		{
			name:     "changes but no triggers should return false",
			triggers: []workertypes.JobTrigger{},
			summary:  createTestSummary(true),
			want:     false,
		},
		{
			name:     "changes and triggers but no highlights should return false",
			triggers: []workertypes.JobTrigger{workertypes.FeaturePromotedToNewly},
			summary:  createTestSummary(true),
			want:     false,
		},
		{
			name:     "changes, triggers, highlights, but no match should return false",
			triggers: []workertypes.JobTrigger{workertypes.FeaturePromotedToWidely},
			summary:  summaryWithNewly,
			want:     false,
		},
		{
			name:     "match on FeaturePromotedToNewly should return true",
			triggers: []workertypes.JobTrigger{workertypes.FeaturePromotedToNewly},
			summary:  summaryWithNewly,
			want:     true,
		},
		{
			name:     "match on FeaturePromotedToWidely should return true",
			triggers: []workertypes.JobTrigger{workertypes.FeaturePromotedToWidely},
			summary:  summaryWithWidely,
			want:     true,
		},
		{
			name:     "match on FeatureRegressedToLimited should return true",
			triggers: []workertypes.JobTrigger{workertypes.FeatureRegressedToLimited},
			summary:  summaryWithLimited,
			want:     true,
		},
		{
			name:     "match on BrowserImplementationAnyComplete should return true",
			triggers: []workertypes.JobTrigger{workertypes.BrowserImplementationAnyComplete},
			summary:  summaryWithBrowserAvailable,
			want:     true,
		},
		{
			name:     "no match on BrowserImplementation when status is not Available",
			triggers: []workertypes.JobTrigger{workertypes.BrowserImplementationAnyComplete},
			summary:  summaryWithBrowserInDev,
			want:     false,
		},
		{
			name: "multiple triggers with one match should return true",
			triggers: []workertypes.JobTrigger{
				workertypes.FeaturePromotedToWidely, workertypes.FeaturePromotedToNewly},
			summary: summaryWithNewly,
			want:    true,
		},
		{
			name:     "multiple highlights with one match should return true",
			triggers: []workertypes.JobTrigger{workertypes.FeaturePromotedToWidely},
			summary: withBaselineHighlight(summaryWithNewly,
				workertypes.BaselineStatusNewly, workertypes.BaselineStatusWidely),
			want: true,
		},
		{
			name: "QueryChanged is considered a change and matches with highlight",
			triggers: []workertypes.JobTrigger{
				workertypes.FeaturePromotedToNewly,
			},
			summary: withBaselineHighlight(summaryQueryChanged,
				workertypes.BaselineStatusLimited, workertypes.BaselineStatusNewly),
			want: true,
		},
		{
			name:     "no match when baseline highlight has wrong status",
			triggers: []workertypes.JobTrigger{workertypes.FeaturePromotedToNewly},
			summary:  summaryWithWidely,
			want:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := shouldNotifyV1(tc.triggers, &tc.summary)
			if err != nil {
				t.Fatalf("shouldNotifyV1 unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("shouldNotifyV1() = %v, want %v", got, tc.want)
			}
		})
	}
}

func createRecoveredSummary() workertypes.EventSummary {
	summary := workertypes.NewEmptyEventSummary()
	summary.SnapshotOrigin = workertypes.OriginLive
	summary.Text = "Search query recovered and tracking 2 features normally."
	summary.SetResolvedQueryErrors([]workertypes.SummaryQueryError{{Code: workertypes.SummaryQueryErrorCodeQueryGrammar}})

	return summary
}

func TestShouldNotifyV1_ResolvedQueryErrors(t *testing.T) {
	summaryRecovered := createRecoveredSummary()

	testCases := []struct {
		name     string
		triggers []workertypes.JobTrigger
		summary  workertypes.EventSummary
		want     bool
	}{
		{
			name:     "resolved query error should return true immediately with no triggers and 0 highlights",
			triggers: []workertypes.JobTrigger{},
			summary:  summaryRecovered,
			want:     true,
		},
		{
			name:     "resolved query error should return true immediately across email channel triggers",
			triggers: []workertypes.JobTrigger{workertypes.FeaturePromotedToWidely},
			summary:  summaryRecovered,
			want:     true,
		},
		{
			name:     "resolved query error should return true immediately across webhook channel triggers",
			triggers: []workertypes.JobTrigger{workertypes.BrowserImplementationAnyComplete},
			summary:  summaryRecovered,
			want:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := shouldNotifyV1(tc.triggers, &tc.summary)
			if err != nil {
				t.Fatalf("shouldNotifyV1 unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("shouldNotifyV1() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestProcessEvent_ResolvedQueryErrors_AllChannels(t *testing.T) {
	emailSub := new(workertypes.EmailSubscriber)
	emailSub.SubscriptionID = "sub-email-recovery"
	emailSub.EmailAddress = "eng@google.com"
	emailSub.ChannelID = "chan-email"
	emailSub.Triggers = []workertypes.JobTrigger{workertypes.FeaturePromotedToNewly}

	webhookSub := new(workertypes.WebhookSubscriber)
	webhookSub.SubscriptionID = "sub-webhook-recovery"
	webhookSub.WebhookURL = "https://hooks.slack.com/services/test"
	webhookSub.WebhookType = workertypes.WebhookTypeSlack
	webhookSub.ChannelID = "chan-slack"
	webhookSub.Triggers = []workertypes.JobTrigger{workertypes.FeaturePromotedToWidely}

	subSet := new(workertypes.SubscriberSet)
	subSet.Emails = []workertypes.EmailSubscriber{*emailSub}
	subSet.Webhooks = []workertypes.WebhookSubscriber{*webhookSub}

	finder := new(mockSubscriptionFinder)
	finder.findReturnSet = subSet

	publisher := new(mockDeliveryPublisher)
	d := NewDispatcher(finder, publisher)

	recoveredSummary := createRecoveredSummary()
	d.parser = mockParserFactory(recoveredSummary, nil)

	metadata := new(workertypes.DispatchEventMetadata)
	metadata.EventID = "event-recovery"
	metadata.SearchID = "search-8520cfc1"
	metadata.SearchName = "My CSS Features"
	metadata.Query = "group:css"
	metadata.Frequency = workertypes.FrequencyImmediate
	metadata.GeneratedAt = time.Now()

	rawPayload := []byte(`{"resolvedQueryErrors":[{"code":"query_grammar_invalid"}]}`)
	if err := d.ProcessEvent(context.Background(), *metadata, rawPayload); err != nil {
		t.Fatalf("ProcessEvent unexpected error: %v", err)
	}

	if len(publisher.emailJobs) != 1 {
		t.Fatalf("expected exactly 1 email delivery job dispatched upon recovery, got %d", len(publisher.emailJobs))
	}
	if publisher.emailJobs[0].SubscriptionID != "sub-email-recovery" {
		t.Errorf("expected subscription ID sub-email-recovery, got %s", publisher.emailJobs[0].SubscriptionID)
	}

	if len(publisher.webhookJobs) != 1 {
		t.Fatalf("expected exactly 1 webhook delivery job dispatched upon recovery, got %d", len(publisher.webhookJobs))
	}
	if publisher.webhookJobs[0].SubscriptionID != "sub-webhook-recovery" {
		t.Errorf("expected subscription ID sub-webhook-recovery, got %s", publisher.webhookJobs[0].SubscriptionID)
	}
}

func TestShouldNotifyV1_NilSummary(t *testing.T) {
	// Verify that if the summary is nil, shouldNotifyV1 returns false and no error.
	got, err := shouldNotifyV1([]workertypes.JobTrigger{workertypes.FeaturePromotedToNewly}, nil)
	if err != nil {
		t.Fatalf("shouldNotifyV1 unexpected error: %v", err)
	}
	if got {
		t.Errorf("shouldNotifyV1(triggers, nil) = %v, want false", got)
	}
}

func TestProcessEvent_BaselinePromoteToWidely_DispatchesGitHubIssueJob(t *testing.T) {
	finder := new(mockSubscriptionFinder)
	finder.findCodeSubsReturn = []gcpspanner.CodeSubscription{
		{
			ID:                 "sub-popover",
			VCSProvider:        "github",
			VCSInstallationID:  "inst-123",
			VCSRepositoryID:    "repo-456",
			RepositoryFullName: "test-owner/test-repo",
			TargetQuery:        "id:popover",
			Triggers: []gcpspanner.SubscriptionTrigger{
				gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToWidely,
			},
			Status: gcpspanner.SubscriptionActive,
			Occurrences: []gcpspanner.SubscriptionOccurrence{
				{
					FilePath:       "src/app.ts",
					LineNumber:     10,
					CommentSnippet: "// TODO(baseline/popover)",
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	publisher := new(mockDeliveryPublisher)
	d := NewDispatcher(finder, publisher)

	summary := workertypes.NewEmptyEventSummary()
	summary.Highlights = []workertypes.SummaryHighlight{
		{
			Type:        workertypes.SummaryHighlightTypeChanged,
			FeatureID:   "popover",
			FeatureName: "Popover API",
			BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
				From: workertypes.BaselineValue{Status: workertypes.BaselineStatusNewly, LowDate: nil, HighDate: nil},
				To:   workertypes.BaselineValue{Status: workertypes.BaselineStatusWidely, LowDate: nil, HighDate: nil},
			},
			Docs:           nil,
			NameChange:     nil,
			BrowserChanges: nil,
			Moved:          nil,
			Split:          nil,
		},
	}
	d.parser = mockParserFactory(summary, nil)

	metadata := workertypes.DispatchEventMetadata{
		EventID:     "event-1",
		SearchID:    "search-1",
		SearchName:  "Test",
		Query:       "id:popover",
		Frequency:   workertypes.FrequencyImmediate,
		GeneratedAt: time.Now(),
	}

	if err := d.ProcessEvent(context.Background(), metadata, []byte(`{}`)); err != nil {
		t.Fatalf("ProcessEvent unexpected error: %v", err)
	}

	if len(publisher.githubIssueJobs) != 1 {
		t.Fatalf("expected 1 github issue job, got %d", len(publisher.githubIssueJobs))
	}

	job := publisher.githubIssueJobs[0]
	if job.SubscriptionID != "sub-popover" {
		t.Errorf("SubscriptionID = %v, want sub-popover", job.SubscriptionID)
	}
	if job.FeatureID != "popover" {
		t.Errorf("FeatureID = %v, want popover", job.FeatureID)
	}
	if job.FeatureName != "Popover API" {
		t.Errorf("FeatureName = %v, want Popover API", job.FeatureName)
	}
	if job.Trigger != string(gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToWidely) {
		t.Errorf("Trigger = %v, want %v", job.Trigger, gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToWidely)
	}
	if job.RepositoryOwner != "test-owner" || job.RepositoryName != "test-repo" {
		t.Errorf("Owner/Repo = %v/%v, want test-owner/test-repo", job.RepositoryOwner, job.RepositoryName)
	}
	if len(job.Occurrences) != 1 || job.Occurrences[0].LineNumber != 10 {
		t.Errorf("Occurrences = %v, want 1 occurrence at line 10", job.Occurrences)
	}
}

func TestProcessEvent_BaselinePromoteToNewly_DispatchesGitHubIssueJob(t *testing.T) {
	finder := new(mockSubscriptionFinder)
	finder.findCodeSubsReturn = []gcpspanner.CodeSubscription{
		{
			ID:                 "sub-dialog",
			VCSProvider:        "github",
			VCSInstallationID:  "inst-123",
			VCSRepositoryID:    "repo-456",
			RepositoryFullName: "test-owner/test-repo",
			TargetQuery:        "id:dialog",
			Triggers: []gcpspanner.SubscriptionTrigger{
				gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToNewly,
			},
			Status: gcpspanner.SubscriptionActive,
			Occurrences: []gcpspanner.SubscriptionOccurrence{
				{
					FilePath:       "src/dialog.ts",
					LineNumber:     5,
					CommentSnippet: "// TODO(baseline/dialog)",
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	publisher := new(mockDeliveryPublisher)
	d := NewDispatcher(finder, publisher)

	summary := workertypes.NewEmptyEventSummary()
	summary.Highlights = []workertypes.SummaryHighlight{
		{
			Type:        workertypes.SummaryHighlightTypeChanged,
			FeatureID:   "dialog",
			FeatureName: "Dialog element",
			BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
				From: workertypes.BaselineValue{Status: workertypes.BaselineStatusLimited, LowDate: nil, HighDate: nil},
				To:   workertypes.BaselineValue{Status: workertypes.BaselineStatusNewly, LowDate: nil, HighDate: nil},
			},
			Docs:           nil,
			NameChange:     nil,
			BrowserChanges: nil,
			Moved:          nil,
			Split:          nil,
		},
	}
	d.parser = mockParserFactory(summary, nil)

	metadata := workertypes.DispatchEventMetadata{
		EventID:     "event-2",
		SearchID:    "search-2",
		SearchName:  "Test",
		Query:       "id:dialog",
		Frequency:   workertypes.FrequencyImmediate,
		GeneratedAt: time.Now(),
	}

	if err := d.ProcessEvent(context.Background(), metadata, []byte(`{}`)); err != nil {
		t.Fatalf("ProcessEvent unexpected error: %v", err)
	}

	if len(publisher.githubIssueJobs) != 1 {
		t.Fatalf("expected 1 github issue job, got %d", len(publisher.githubIssueJobs))
	}

	job := publisher.githubIssueJobs[0]
	if job.SubscriptionID != "sub-dialog" {
		t.Errorf("SubscriptionID = %v, want sub-dialog", job.SubscriptionID)
	}
	if job.Trigger != string(gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToNewly) {
		t.Errorf("Trigger = %v, want %v", job.Trigger, gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToNewly)
	}
	wantDeliveryID := githubissuedeliveryv1.DeriveDeliveryID(job.SubscriptionID, job.Trigger)
	if job.DeliveryID != wantDeliveryID {
		t.Errorf("DeliveryID = %v, want %v", job.DeliveryID, wantDeliveryID)
	}
}

func TestProcessEvent_NonBaselineChange_NoGitHubIssueJobs(t *testing.T) {
	finder := new(mockSubscriptionFinder)
	publisher := new(mockDeliveryPublisher)
	d := NewDispatcher(finder, publisher)

	summary := workertypes.NewEmptyEventSummary()
	summary.Highlights = []workertypes.SummaryHighlight{
		{
			Type:           workertypes.SummaryHighlightTypeChanged,
			FeatureID:      "css-grid",
			FeatureName:    "CSS Grid",
			BaselineChange: nil,
			Docs:           nil,
			NameChange:     &workertypes.Change[string]{From: "Old Name", To: "CSS Grid"},
			BrowserChanges: nil,
			Moved:          nil,
			Split:          nil,
		},
	}
	d.parser = mockParserFactory(summary, nil)

	metadata := workertypes.DispatchEventMetadata{
		EventID:     "event-3",
		SearchID:    "search-3",
		SearchName:  "Test",
		Query:       "id:css-grid",
		Frequency:   workertypes.FrequencyImmediate,
		GeneratedAt: time.Now(),
	}

	if err := d.ProcessEvent(context.Background(), metadata, []byte(`{}`)); err != nil {
		t.Fatalf("ProcessEvent unexpected error: %v", err)
	}

	if len(publisher.githubIssueJobs) != 0 {
		t.Fatalf("expected 0 github issue jobs for non-baseline change, got %d", len(publisher.githubIssueJobs))
	}
	if len(finder.findCodeSubsCalls) != 0 {
		t.Fatalf("expected 0 calls to FindCodeSubscriptions, got %d", len(finder.findCodeSubsCalls))
	}
}

func TestProcessEvent_PublisherGitHubIssueError(t *testing.T) {
	errPublish := errors.New("pubsub publish error")
	finder := new(mockSubscriptionFinder)
	finder.findCodeSubsReturn = []gcpspanner.CodeSubscription{
		{
			ID:                 "sub-popover",
			VCSProvider:        "github",
			VCSInstallationID:  "inst-123",
			VCSRepositoryID:    "repo-456",
			RepositoryFullName: "test-owner/test-repo",
			TargetQuery:        "id:popover",
			Triggers: []gcpspanner.SubscriptionTrigger{
				gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToWidely,
			},
			Status:      gcpspanner.SubscriptionActive,
			Occurrences: nil,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	publisher := new(mockDeliveryPublisher)
	publisher.githubIssueJobErr = func(_ githubissuedeliveryv1.GitHubIssueDeliveryEvent) error {
		return errPublish
	}
	d := NewDispatcher(finder, publisher)

	summary := workertypes.NewEmptyEventSummary()
	summary.Highlights = []workertypes.SummaryHighlight{
		{
			Type:        workertypes.SummaryHighlightTypeChanged,
			FeatureID:   "popover",
			FeatureName: "Popover API",
			BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
				From: workertypes.BaselineValue{Status: workertypes.BaselineStatusNewly, LowDate: nil, HighDate: nil},
				To:   workertypes.BaselineValue{Status: workertypes.BaselineStatusWidely, LowDate: nil, HighDate: nil},
			},
			Docs:           nil,
			NameChange:     nil,
			BrowserChanges: nil,
			Moved:          nil,
			Split:          nil,
		},
	}
	d.parser = mockParserFactory(summary, nil)

	metadata := workertypes.DispatchEventMetadata{
		EventID:     "event-4",
		SearchID:    "search-4",
		SearchName:  "Test",
		Query:       "id:popover",
		Frequency:   workertypes.FrequencyImmediate,
		GeneratedAt: time.Now(),
	}

	err := d.ProcessEvent(context.Background(), metadata, []byte(`{}`))
	if err == nil {
		t.Fatal("expected error from ProcessEvent, got nil")
	}
}

func TestProcessEvent_BaselineDemoteToNewly_NoGitHubIssueJobs(t *testing.T) {
	finder := new(mockSubscriptionFinder)
	publisher := new(mockDeliveryPublisher)
	d := NewDispatcher(finder, publisher)

	summary := workertypes.NewEmptyEventSummary()
	summary.Highlights = []workertypes.SummaryHighlight{
		{
			Type:        workertypes.SummaryHighlightTypeChanged,
			FeatureID:   "dialog",
			FeatureName: "Dialog element",
			BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
				From: workertypes.BaselineValue{Status: workertypes.BaselineStatusWidely, LowDate: nil, HighDate: nil},
				To:   workertypes.BaselineValue{Status: workertypes.BaselineStatusNewly, LowDate: nil, HighDate: nil},
			},
			Docs:           nil,
			NameChange:     nil,
			BrowserChanges: nil,
			Moved:          nil,
			Split:          nil,
		},
	}
	d.parser = mockParserFactory(summary, nil)

	metadata := workertypes.DispatchEventMetadata{
		EventID:     "event-demote",
		SearchID:    "search-demote",
		SearchName:  "Test",
		Query:       "id:dialog",
		Frequency:   workertypes.FrequencyImmediate,
		GeneratedAt: time.Now(),
	}

	if err := d.ProcessEvent(context.Background(), metadata, []byte(`{}`)); err != nil {
		t.Fatalf("ProcessEvent unexpected error: %v", err)
	}

	if len(publisher.githubIssueJobs) != 0 {
		t.Fatalf("expected 0 github issue jobs for baseline demotion, got %d", len(publisher.githubIssueJobs))
	}
	if len(finder.findCodeSubsCalls) != 0 {
		t.Fatalf("expected 0 calls to FindCodeSubscriptions for demotion, got %d", len(finder.findCodeSubsCalls))
	}
}
