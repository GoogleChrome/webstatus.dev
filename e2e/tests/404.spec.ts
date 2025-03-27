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

const BASE_URL = 'http://localhost:5555';

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

test('should fetch similar features from API and show results', async ({
  page,
}) => {
  // Test URLs for different feature IDs
  const badUrls = [
    {badUrl: `${BASE_URL}/features/badID_1234`, query: 'badID_1234'},
    {badUrl: `${BASE_URL}/features/g`, query: 'g'}
  ];
  const API_BASE_URL =
    'http://localhost:8080/v1/features?q={query}&page_size=5';

  for (const {badUrl, query} of badUrls) {
    await test.step(`Testing API response for: ${badUrl}`, async () => {
      await page.goto(badUrl);
      await expect(page).toHaveURL(
        'http://localhost:5555/errors-404/feature-not-found?q=' + query,
      );

      const featurePageResponse = await page
        .context()
        .request.fetch(page.url());

      // Assert that the response status code is 404
      expect(featurePageResponse.status()).toBe(404);

      // Mock API response for similar features
      const apiUrl = API_BASE_URL.replace('{query}', query);
      const response = await page.context().request.get(apiUrl);
      expect(response.status()).toBe(200);

      const data = await response.json();
      const hasResults = Array.isArray(data?.data) && data.data.length > 0;

      if (hasResults) {
        // Show similar features container
        const similarContainer = page.locator('.similar-features-container');
        await expect(similarContainer).toBeVisible();
        await expect(page.locator('.feature-list li')).toHaveCount(
          data.data.length,
        );

        // ✅ Click first similar feature
        const firstFeature = data.data[0];
        const firstFeatureLink = page.locator('.feature-list li a').first();
        await expect(firstFeatureLink).toHaveText(firstFeature.name);

        // Click and wait for navigation
        await Promise.all([page.waitForNavigation(), firstFeatureLink.click()]);

        await expect(page).toHaveURL(
          `${BASE_URL}/features/${firstFeature.feature_id}`,
        );

        // Go back to error page to test second part
        await page.goBack();

        // ✅ Click "Search for more similar features" button
        const searchButton = page.locator('#error-action-search-btn');
        await expect(searchButton).toBeVisible();

        await Promise.all([page.waitForNavigation(), searchButton.click()]);

        await expect(page).toHaveURL(`${BASE_URL}?q=${query}`);
      } else {
        // No similar features found
        await expect(
          page.locator('.similar-features-container'),
        ).not.toBeVisible();
        await expect(page.locator('.error-message')).toContainText(
          'No similar features found.',
        );
      }
    });
  }
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

test('matches the screenshot', async ({page}) => {
  await page.goto(`${BASE_URL}/bad_url`);
  const pageContainer = page.locator('.page-container');
  await expect(pageContainer).toHaveScreenshot('not-found-error-page.png');
});

test('matches the screenshot with similar results', async ({page}) => {
  await page.goto(`${BASE_URL}/features/g`);

  // ✅ Wait for similar results to appear
  const similarContainer = page.locator('.similar-features-container');
  await expect(similarContainer).toBeVisible({timeout: 5000});

  const pageContainer = page.locator('.page-container');
  await expect(pageContainer).toHaveScreenshot(
    'not-found-error-page-similar-results.png',
  );
});
