// Copyright 2025 Google LLC
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

	"github.com/GoogleChrome/webstatus.dev/lib/generic"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/google/go-cmp/cmp"
)

// --- Mocks ---

type findSubscribersReq struct {
	SearchID  string
	Frequency string
}

type mockSubscriptionFinder struct {
	findCalledWith *findSubscribersReq
	findReturnSet  *workertypes.SubscriberSet
	findReturnErr  error
}

func (m *mockSubscriptionFinder) FindSubscribers(_ context.Context, searchID string,
	frequency workertypes.JobFrequency) (*workertypes.SubscriberSet, error) {
	m.findCalledWith = &findSubscribersReq{
		SearchID:  searchID,
		Frequency: string(frequency),
	}

	return m.findReturnSet, m.findReturnErr
}

type mockDeliveryPublisher struct {
	emailJobs   []workertypes.EmailDeliveryJob
	emailJobErr func(job workertypes.EmailDeliveryJob) error
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

	return workertypes.EventSummary{
		SchemaVersion: "v1",
		Text:          "Test Summary",
		Categories:    categories,
		Truncated:     false,
		Highlights:    nil,
	}
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
	}

	finder := &mockSubscriptionFinder{
		findReturnSet:  subSet,
		findReturnErr:  nil,
		findCalledWith: nil,
	}
	publisher := &mockDeliveryPublisher{
		emailJobs:   nil,
		emailJobErr: nil,
	}

	// Create a summary that HAS changes so notification logic proceeds.
	summary := createTestSummary(true)
	summary.Categories.UpdatedBaseline = 1
	summary.Categories.Updated = 1
	summary.Highlights = []workertypes.SummaryHighlight{
		{
			Type:        workertypes.SummaryHighlightTypeChanged,
			FeatureID:   "test-feature-id",
			FeatureName: "Test Feature",
			DocLinks:    nil,
			NameChange:  nil,
			BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
				From: newBaselineValue(workertypes.BaselineStatusLimited),
				To:   newBaselineValue(workertypes.BaselineStatusNewly),
			},
			BrowserChanges: nil,
			Moved:          nil,
			Split:          nil,
		},
	}
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
		Metadata: workertypes.DeliveryMetadata{
			EventID:     eventID,
			SearchID:    searchID,
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
	}

	finder := &mockSubscriptionFinder{
		findReturnSet:  subSet,
		findReturnErr:  nil,
		findCalledWith: nil,
	}
	publisher := &mockDeliveryPublisher{
		emailJobs:   nil,
		emailJobErr: nil,
	}

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
		findReturnSet:  nil,
		findReturnErr:  errors.New("db error"),
		findCalledWith: nil,
	}

	d := NewDispatcher(finder, nil)
	// Provide a valid summary struct so parser succeeds
	var summary workertypes.EventSummary
	d.parser = mockParserFactory(summary, nil)

	metadata := workertypes.DispatchEventMetadata{
		EventID:     "",
		SearchID:    "",
		Query:       "",
		Frequency:   "",
		GeneratedAt: time.Time{},
	}

	err := d.ProcessEvent(context.Background(), metadata, []byte("{}"))
	if err == nil {
		t.Error("Expected error from finder, got nil")
	}
	assertFindSubscribersCalledWith(t, finder, generic.ValuePtr(emptyFinderReq()))
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
	}

	finder := &mockSubscriptionFinder{
		findReturnSet:  subSet,
		findReturnErr:  nil,
		findCalledWith: nil,
	}

	// Publisher returns error for first job, success for second
	publisher := &mockDeliveryPublisher{
		emailJobs: nil,
		emailJobErr: func(job workertypes.EmailDeliveryJob) error {
			if job.SubscriptionID == "sub-1" {
				return errors.New("queue full")
			}

			return nil
		},
	}

	summaryWithNewly := withBaselineHighlight(createTestSummary(false),
		workertypes.BaselineStatusLimited, workertypes.BaselineStatusNewly)
	d := NewDispatcher(finder, publisher)
	d.parser = mockParserFactory(summaryWithNewly, nil)

	metadata := workertypes.DispatchEventMetadata{
		EventID:     "",
		SearchID:    "",
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
	assertFindSubscribersCalledWith(t, finder, generic.ValuePtr(emptyFinderReq()))
}

func TestProcessEvent_JobCount(t *testing.T) {
	// Verify that if no jobs are generated (e.g. no matching triggers), ProcessEvent returns early/cleanly.
	subSet := &workertypes.SubscriberSet{
		Emails: []workertypes.EmailSubscriber{
			{SubscriptionID: "sub-1", Triggers: []workertypes.JobTrigger{}, EmailAddress: "e1", UserID: "u1",
				ChannelID: "chan-1"}, // No match
		},
	}
	finder := &mockSubscriptionFinder{
		findReturnSet:  subSet,
		findReturnErr:  nil,
		findCalledWith: nil,
	}
	publisher := new(mockDeliveryPublisher)
	d := NewDispatcher(finder, publisher)
	d.parser = mockParserFactory(createTestSummary(true), nil)

	metadata := workertypes.DispatchEventMetadata{
		EventID:     "",
		SearchID:    "",
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
	assertFindSubscribersCalledWith(t, finder, generic.ValuePtr(emptyFinderReq()))
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
	s.Highlights = append(s.Highlights, workertypes.SummaryHighlight{
		Type:        workertypes.SummaryHighlightTypeChanged,
		FeatureID:   "test-feature-id",
		FeatureName: "Test Feature",
		DocLinks:    nil,
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
	s.Highlights = append(s.Highlights, workertypes.SummaryHighlight{
		Type:           workertypes.SummaryHighlightTypeChanged,
		FeatureID:      "test-feature-id",
		FeatureName:    "Test Feature",
		DocLinks:       nil,
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
			got := shouldNotifyV1(tc.triggers, tc.summary)
			if got != tc.want {
				t.Errorf("shouldNotifyV1() = %v, want %v", got, tc.want)
			}
		})
	}
}
