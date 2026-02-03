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

package chimeadapters

import (
	"context"
	"errors"

	"github.com/GoogleChrome/webstatus.dev/lib/email/chime"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

type EmailSender interface {
	Send(ctx context.Context, id string, to string, subject string, htmlBody string) error
}

type EmailWorkerChimeAdapter struct {
	chimeSender EmailSender
}

// NewEmailWorkerChimeAdapter creates a new adapter for the email worker to use Chime.
func NewEmailWorkerChimeAdapter(chimeSender EmailSender) *EmailWorkerChimeAdapter {
	return &EmailWorkerChimeAdapter{
		chimeSender: chimeSender,
	}
}

// Send implements the EmailSender interface for the email worker.
func (a *EmailWorkerChimeAdapter) Send(ctx context.Context, id string, to string,
	subject string, htmlBody string) error {
	err := a.chimeSender.Send(ctx, id, to, subject, htmlBody)
	if err != nil {
		if errors.Is(err, chime.ErrPermanentUser) {
			return errors.Join(workertypes.ErrUnrecoverableUserFailureEmailSending, err)
		} else if errors.Is(err, chime.ErrPermanentSystem) {
			return errors.Join(workertypes.ErrUnrecoverableSystemFailureEmailSending, err)
		} else if errors.Is(err, chime.ErrDuplicate) {
			return errors.Join(workertypes.ErrUnrecoverableSystemFailureEmailSending, err)
		}

		// Will be recorded as a transient error
		return err
	}

	return nil
}
