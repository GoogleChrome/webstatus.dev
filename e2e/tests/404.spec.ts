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

import {test, expect} from '@playwright/test';

const homePageUrl = 'http://localhost:5555/';

// TODO. Redirect to the 404 page once we have it.
test('Bad URL redirection to home page', async ({page}) => {
  const badUrls = [
    // Test for bad public asset
    'http://localhost:5555/public/junk',
    // TODO. Test for bad app urls (e.g. bad feature id)
  ];

  for (const badUrl of badUrls) {
    await test.step(`Testing redirection for: ${badUrl}`, async () => {
      await page.goto(badUrl);
      await expect(page).toHaveURL(homePageUrl);
    });
  }
});
