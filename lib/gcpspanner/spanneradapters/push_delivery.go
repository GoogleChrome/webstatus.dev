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

package spanneradapters

import (
	"context"
	"log/slog"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

type PushDeliverySpannerClient interface {
	FindAllActivePushSubscriptions(
		ctx context.Context,
		savedSearchID string,
		frequency gcpspanner.SavedSearchSnapshotType,
	) ([]gcpspanner.SubscriberDestination, error)
}

type PushDeliverySubscriberFinder struct {
	client PushDeliverySpannerClient
}

func NewPushDeliverySubscriberFinder(client PushDeliverySpannerClient) *PushDeliverySubscriberFinder {
	return &PushDeliverySubscriberFinder{client: client}
}

func (f *PushDeliverySubscriberFinder) FindSubscribers(ctx context.Context, searchID string,
	frequency workertypes.JobFrequency) (*workertypes.SubscriberSet, error) {
	spannerFrequency := convertFrequencyToSnapshotType(frequency)

	dests, err := f.client.FindAllActivePushSubscriptions(ctx, searchID, spannerFrequency)
	if err != nil {
		return nil, err
	}

	set := &workertypes.SubscriberSet{
		Emails: make([]workertypes.EmailSubscriber, 0),
	}

	for _, dest := range dests {
		// If EmailConfig is set, it's an email subscriber.
		if dest.EmailConfig != nil {
			set.Emails = append(set.Emails, workertypes.EmailSubscriber{
				SubscriptionID: dest.SubscriptionID,
				UserID:         dest.UserID,
				Triggers:       convertSpannerTriggersToJobTriggers(dest.Triggers),
				EmailAddress:   dest.EmailConfig.Address,
			})
		}
	}

	return set, nil
}

func convertSpannerTriggersToJobTriggers(triggers []gcpspanner.SubscriptionTrigger) []workertypes.JobTrigger {
	if triggers == nil {
		return nil
	}
	jobTriggers := make([]workertypes.JobTrigger, 0, len(triggers))
	for _, t := range triggers {
		jobTriggers = append(jobTriggers, convertSpannerTriggerToJobTrigger(t))
	}

	return jobTriggers
}

func convertSpannerTriggerToJobTrigger(trigger gcpspanner.SubscriptionTrigger) workertypes.JobTrigger {
	switch trigger {
	case gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToNewly:
		return workertypes.FeaturePromotedToNewly
	case gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToWidely:
		return workertypes.FeaturePromotedToWidely
	case gcpspanner.SubscriptionTriggerFeatureBaselineRegressionToLimited:
		return workertypes.FeatureRegressedToLimited
	case gcpspanner.SubscriptionTriggerBrowserImplementationAnyComplete:
		return workertypes.BrowserImplementationAnyComplete
	case gcpspanner.SubscriptionTriggerUnknown:
		break
	}
	// Should not reach here.
	slog.WarnContext(context.TODO(), "unknown subscription trigger encountered in push deliveryspanner adapter",
		"trigger", trigger)

	return ""
}
