/**
 * Copyright 2023 Google LLC
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

import {test, expect} from '@playwright/test';

test('matches the screenshot', async ({page}) => {
  await page.goto('http://localhost:5555/');

  // Wait for the loading indicator to disappear and be replaced (with timeout):
  await page
    .locator('webstatus-overview-content >> text=Loading features...')
    .waitFor({state: 'hidden', timeout: 30000});

  await expect(page).toHaveScreenshot();
});

test('shows an error that their query is invalid', async ({page}) => {
  await page.goto('http://localhost:5555/?q=available_on%3Achrom');

  await page.getByText('Invalid query...');
});

test('shows an unknown error when there is an internal error', async ({
  page,
}) => {
  await page.route('**/v1/features', route =>
    route.fulfill({
      status: 500,
      contentType: 'application/json',
      json: {
        code: 500,
        message: 'uh-oh',
      },
    })
  );
  await page.goto('http://localhost:5555/');

  await page.getByText('Something went wrong...');
});
