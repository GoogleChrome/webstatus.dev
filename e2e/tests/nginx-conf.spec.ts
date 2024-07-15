/**
 * Copyright 2024 Google LLC
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

import {test, expect, Response} from '@playwright/test';
import {gotoOverviewPageUrl} from './utils';

function checkReponseForGZIPCompression(response: Response) {
  expect(
    response.headers()['content-encoding'],
    `GZIP assertion failed for ${response.url()}: Asset not returning GZIP compression. Headers: ${JSON.stringify(
      response.headers()
    )}`
  ).toContain('gzip');
}

test('All public assets should be served with GZIP compression', async ({
  page,
}) => {
  let publicAssetFound = false;
  let homepageFound = false;

  page.on('request', async request => {
    await page.route('**/*', async route => {
      const headers = route.request().headers();
      // Add the "Via" header to simulate Cloud Run environment
      // https://cloud.google.com/load-balancing/docs/https/troubleshooting-ext-https-lbs#compression-not-working
      headers['Via'] = 'test google'; // Simulate a Via header from Google Cloud Load Balancer;
      await route.continue({headers});
    });
  });

  page.on('response', response => {
    // Intercept network requests to check for GZIP compression
    if (response.url().startsWith('http://localhost:5555/public')) {
      checkReponseForGZIPCompression(response);
      publicAssetFound = true;
    } else if (response.url() === 'http://localhost:5555/') {
      checkReponseForGZIPCompression(response);
      homepageFound = true;
    }
  });

  await gotoOverviewPageUrl(page, 'http://localhost:5555');
  expect(
    publicAssetFound,
    'At least one public asset must be loaded'
  ).toBeTruthy();
  expect(homepageFound, 'index.html must be loaded').toBeTruthy();
});
