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

import * as crypto from 'crypto';
import {APIRequestContext} from '@playwright/test';

export const DEV_WEBHOOK_SECRET = 'dev-webhook-secret-for-testing-only-12345';

export type JSONPrimitive = string | number | boolean | null;
export type JSONValue = JSONPrimitive | JSONObject | JSONArray;
export type JSONObject = {[key: string]: JSONValue};
export type JSONArray = readonly JSONValue[] | JSONValue[];

export function canonicalizeJSON(obj: JSONValue): string {
  if (obj === null || typeof obj !== 'object') {
    return JSON.stringify(obj);
  }
  if (Array.isArray(obj)) {
    return '[' + obj.map(canonicalizeJSON).join(',') + ']';
  }
  const keys = Object.keys(obj).sort();
  const pairs = keys.map(
    k => JSON.stringify(k) + ':' + canonicalizeJSON(obj[k]),
  );
  return '{' + pairs.join(',') + '}';
}

export function signGitHubPayload(secret: string, body: string): string {
  const hmac = crypto.createHmac('sha256', secret);
  hmac.update(body, 'utf8');
  return `sha256=${hmac.digest('hex')}`;
}

export async function dispatchMockPushWebhook(
  request: APIRequestContext,
  baseURL: string,
  options: {
    secret?: string;
    deliveryId?: string;
    event?: string;
    payload?: JSONObject;
  } = {},
) {
  const secret = options.secret ?? DEV_WEBHOOK_SECRET;
  const deliveryId = options.deliveryId ?? crypto.randomUUID();
  const event = options.event ?? 'push';
  const payload = options.payload ?? {
    ref: 'refs/heads/main',
    before: '0000000000000000000000000000000000000000',
    after: 'abcdef1234567890abcdef1234567890abcdef12',
    repository: {
      id: 67890,
      name: 'webstatus.dev',
      full_name: 'GoogleChrome/webstatus.dev',
      owner: {
        login: 'GoogleChrome',
      },
    },
    installation: {
      id: 12345,
    },
    commits: [
      {
        id: 'abcdef1234567890abcdef1234567890abcdef12',
        message: 'Add css grid feature',
      },
    ],
  };

  const bodyStr = canonicalizeJSON(payload);
  const signature = signGitHubPayload(secret, bodyStr);

  return await request.post(`${baseURL}/v1/webhooks/github`, {
    data: bodyStr,
    headers: {
      'Content-Type': 'application/json',
      'X-GitHub-Event': event,
      'X-GitHub-Delivery': deliveryId,
      'X-Hub-Signature-256': signature,
    },
  });
}
