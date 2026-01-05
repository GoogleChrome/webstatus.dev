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

import {execSync} from 'child_process';

const LOAD_FAKE_DATA_CMD =
  "make dev_fake_data LOAD_FAKE_DATA_FLAGS='-trigger-scenario=%s'";

export async function triggerBatchJob(frequency: string) {
  const message = {
    messages: [
      {
        data: Buffer.from(JSON.stringify({frequency})).toString('base64'),
      },
    ],
  };

  await fetch(
    'http://localhost:8060/v1/projects/local/topics/batch-updates-topic-id:publish',
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(message),
    },
  );
}

export async function getLatestEmail(recipient: string): Promise<any | null> {
  const response = await fetch('http://localhost:8025/api/v1/messages');
  const data = await response.json();
  const messages = data.messages || [];
  for (const message of messages) {
    if (message.To.some((r: any) => r.Address === recipient)) {
      return message;
    }
  }
  return null;
}

export function triggerNonMatchingChange() {
  execSync(LOAD_FAKE_DATA_CMD.replace('%s', 'non-matching'));
}

export function triggerMatchingChange() {
  execSync(LOAD_FAKE_DATA_CMD.replace('%s', 'matching'));
}

export function triggerBatchChange() {
  execSync(LOAD_FAKE_DATA_CMD.replace('%s', 'batch-change'));
}
