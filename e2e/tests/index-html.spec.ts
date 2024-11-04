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
import {gotoOverviewPageUrl} from './utils';

test('preconnect tag present', async ({page}) => {
  interface PreconnectLink {
    origin: string;
    crossorigin?: 'use-credentials';
  }
  await gotoOverviewPageUrl(page, 'http://localhost:5555/');

  const expectedPreconnectLinks: PreconnectLink[] = [
    {origin: 'http://localhost:8080', crossorigin: 'use-credentials'},
  ];

  // 1. Verify Expected Preconnects Exist
  for (const link of expectedPreconnectLinks) {
    const selector = `link[rel="preconnect"][href="${link.origin}"]`;
    const preconnectLink = await page.$(selector);
    expect(
      preconnectLink,
      `Preconnect link not found for ${link.origin}`,
    ).toBeTruthy();

    if (link.crossorigin) {
      const actualCrossorigin =
        await preconnectLink?.getAttribute('crossorigin');
      expect(actualCrossorigin).toBe(link.crossorigin);
    }
  }

  // 2. Verify No Extra Preconnects
  const allPreconnectLinks = await page.$$('link[rel="preconnect"]');
  const actualOrigins = await Promise.all(
    allPreconnectLinks.map(async link => await link.getAttribute('href')),
  );

  for (const actualOrigin of actualOrigins) {
    const isExpected = expectedPreconnectLinks.some(
      link => link.origin === actualOrigin,
    );
    expect(
      isExpected,
      `Unexpected preconnect link found for ${actualOrigin}`,
    ).toBeTruthy();
  }
});
