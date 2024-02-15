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

test('Bad URL redirection to 404 page', async ({page}) => {
  const badUrls = [
    // Test for bad public asset
    'http://localhost:5555/public/junk',
    // Test for bad URL goes to the not found component
    'http://localhost:5555/bad_url',
    // TODO. Test for bad app urls (e.g. bad feature id)
  ];

  for (const badUrl of badUrls) {
    await test.step(`Testing redirection for: ${badUrl}`, async () => {
      await page.goto(badUrl);
      // Should keep the same URL
      await expect(page).toHaveURL(badUrl);

      const response = await page.context().request.fetch(page.url());

      // Assert that the response status code is 404
      expect(response.status()).toBe(404);
    });
  }
});

test('matches the screenshot', async ({page}) => {
  await page.goto('http://localhost:5555/bad_url');
  const pageContainer = page.locator('.page-container');
  await expect(pageContainer).toHaveScreenshot('not-found-error-page.png', {
    // Temporarily allow a higher diff ratio as these tests become more stable
    maxDiffPixelRatio: 0.02,
  });
});
