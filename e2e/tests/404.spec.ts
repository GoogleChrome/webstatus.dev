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
import {BASE_URL, expect404PageButtons, goTo404Page} from './utils';

test('Bad URL redirection to 404 page', async ({page}) => {
  const badUrls = [
    // Test for bad public asset
    `${BASE_URL}/public/junk`,
    // Test for bad URL goes to the not found component
    `${BASE_URL}/bad_url`,
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

      // Check page content
      const errorMessage = page.locator('#error-detailed-message');
      await expect(errorMessage).toBeVisible();
      await expect(errorMessage).toContainText(
        "We couldn't find the page you're looking for.",
      );

      // Check buttons
      await expect(page.locator('#error-action-home-btn')).toBeVisible();
      await expect(page.locator('#error-action-report')).toBeVisible();
    });
  }
});

test('shows similar features and all buttons when results exist', async ({
  page,
}) => {
  const query = 'g';
  await goTo404Page(page, query);

  await expect(page.locator('.similar-features-container')).toBeVisible();
  await expect404PageButtons(page, {hasSearch: true});

  const similarContainerButton = page.locator('#error-action-search-btn');
  const pageContainer = page.locator('.page-container');

  // Snapshot
  await expect(pageContainer).toHaveScreenshot(
    'not-found-error-page-similar-results.png',
  );

  // Clicking the search button should redirect to homepage with search
  await Promise.all([page.waitForNavigation(), similarContainerButton.click()]);
  await expect(page).toHaveURL(`${BASE_URL}?q=${query}`);
});

test('shows only home and report buttons when no similar features found', async ({
  page,
}) => {
  const query = 'nonexistent-feature';
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

  // Home button navigation
  const homeButton = page.locator('#error-action-home-btn');
  await expect(homeButton).toBeVisible();
  await homeButton.click();
  await expect(page).toHaveURL(BASE_URL);

  await page.goBack();

  // Report an issue button should be present
  const reportButton = page.locator('#error-action-report');
  await expect(reportButton).toBeVisible();
  await expect(reportButton).toHaveAttribute(
    'href',
    'https://github.com/GoogleChrome/webstatus.dev/issues/new/choose',
  );
});

test('matches the screenshot 404 not found page', async ({page}) => {
  await page.goto(`${BASE_URL}/bad_url`);
  const pageContainer = page.locator('.page-container');
  await expect(pageContainer).toHaveScreenshot('not-found-error-page.png');
});
