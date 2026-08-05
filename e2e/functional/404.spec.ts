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
import {BASE_URL, expect404PageButtons, goTo404Page} from '../utils/utils.js';

test('Bad URL redirection to 404 page', async ({page}) => {
  const badUrls = [`${BASE_URL}/public/junk`, `${BASE_URL}/bad_url`];

  for (const badUrl of badUrls) {
    await test.step(`Testing redirection for: ${badUrl}`, async () => {
      await page.goto(badUrl);
      await expect(page).toHaveURL(badUrl);

      const response = await page.context().request.fetch(page.url());
      expect(response.status()).toBe(404);

      const errorMessage = page.locator('#error-detailed-message');
      await expect(errorMessage).toBeVisible();
      await expect(errorMessage).toContainText(
        "We couldn't find the page you're looking for.",
      );

      await expect(page.locator('#error-action-home-btn')).toBeVisible();
      await expect(page.locator('#error-action-report')).toBeVisible();
    });
  }
});

test('shows only home and report buttons when no similar features found', async ({
  page,
}) => {
  const query = 'nonexistent-feature-xyz';
  await goTo404Page(page, query);

  await expect(page.locator('.similar-features-container')).toHaveCount(0);
  await expect404PageButtons(page, {hasSearch: false});

  await expect(page.locator('#error-detailed-message')).toContainText(
    `We could not find Feature ID: ${query}`,
  );

  await expect(page.locator('.error-message')).toContainText(
    'No similar features found.',
  );
});

test('should allow navigation from 404 page', async ({page}) => {
  const badUrl = `${BASE_URL}/feature/doesNotExist123`;
  await page.goto(badUrl);
  await expect(page).toHaveURL(badUrl);

  const homeButton = page.locator('#error-action-home-btn');
  await expect(homeButton).toBeVisible();
  await homeButton.click();
  await expect(page).toHaveURL(BASE_URL);

  await page.goBack();

  const reportButton = page.locator('#error-action-report');
  await expect(reportButton).toBeVisible();
  await expect(reportButton).toHaveAttribute(
    'href',
    'https://github.com/GoogleChrome/webstatus.dev/issues/new/choose',
  );
});
