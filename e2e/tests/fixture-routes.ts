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

import {type Page, type Route} from '@playwright/test';

import {
  globalSavedSearchesFixture,
  userSavedSearchesFixture,
  userNotificationChannelsFixture,
  featuresPage1Fixture,
  featuresPage2Fixture,
  featuresSortFixture,
  featuresUpvotesFixture,
  featureDetailAnchorPositioningFixture,
  featureDetailDiscouragedFixture,
  featureWptMetricsChromePage1,
  featureWptMetricsChromePage2,
  featureWptMetricsFirefoxPage1,
  featureWptMetricsFirefoxPage2,
  featureWptMetricsSafariPage1,
  featureWptMetricsSafariPage2,
  featureWptMetricsEdgePage1,
  featureWptMetricsEdgePage2,
  featureUmaMetricsPage1,
  featureUmaMetricsPage2,
  statsFeatureCountsChromePage1,
  statsFeatureCountsChromePage2,
  statsFeatureCountsFirefoxPage1,
  statsFeatureCountsFirefoxPage2,
  statsFeatureCountsSafariPage1,
  statsFeatureCountsSafariPage2,
  statsLowDateCountsPage1,
  statsLowDateCountsPage2,
  statsMissingOneChromePage1,
  statsMissingOneChromePage2,
  statsMissingOneFirefoxPage1,
  statsMissingOneFirefoxPage2,
  statsMissingOneSafariPage1,
  statsMissingOneSafariPage2,
  statsMissingFeaturesListPage1,
  statsMissingFeaturesListPage2,
} from '../fixtures/mock-data';

export interface VisualFixtureOptions {
  userRole?: 'unauthenticated' | 'authenticated';
  pingState?: 'idle' | 'syncing' | 'error';
}

/**
 * Attaches static route fixtures to intercept backend network requests
 * during visual regression screenshot tests.
 */
export async function setupVisualFixtures(
  page: Page,
  options: VisualFixtureOptions = {},
): Promise<void> {
  const {userRole = 'unauthenticated', pingState = 'idle'} = options;

  // Intercept Global Saved Searches (Sidebar & Bookmarks)
  await page.route('**/v1/global-saved-searches*', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(globalSavedSearchesFixture),
    });
  });

  // Intercept User Saved Searches
  await page.route('**/v1/users/me/saved-searches*', async (route: Route) => {
    if (userRole === 'unauthenticated') {
      await route.fulfill({
        status: 401,
        body: JSON.stringify({code: 401, message: 'Unauthorized'}),
      });
    } else {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(userSavedSearchesFixture),
      });
    }
  });

  // Intercept User Notification Channels
  await page.route(
    '**/v1/users/me/notification-channels*',
    async (route: Route) => {
      if (userRole === 'unauthenticated') {
        await route.fulfill({
          status: 401,
          body: JSON.stringify({code: 401, message: 'Unauthorized'}),
        });
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(userNotificationChannelsFixture),
        });
      }
    },
  );

  // Intercept User Ping / Profile Sync
  await page.route('**/v1/users/me/ping*', async (route: Route) => {
    if (pingState === 'syncing') {
      // Delay response to capture syncing spinner
      await new Promise(resolve => setTimeout(resolve, 3000));
      await route.fulfill({status: 204});
    } else if (pingState === 'error') {
      await route.fulfill({
        status: 500,
        body: JSON.stringify({code: 500, message: 'Sync failed'}),
      });
    } else {
      await route.fulfill({status: 204});
    }
  });

  // Intercept Stats Page Metrics Data (Per-Browser & Paginated)
  await page.route('**/v1/stats/**', async (route: Route) => {
    const url = new URL(route.request().url());
    const pathname = url.pathname;
    const isPage2 = url.searchParams.get('page_token') === 'page_2';

    // 1. Point selection list: /v1/stats/features/browsers/{browser}/missing_one_implementation_counts/{date}/features
    if (
      pathname.includes('missing_one_implementation_counts') &&
      pathname.endsWith('/features')
    ) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          isPage2
            ? statsMissingFeaturesListPage2
            : statsMissingFeaturesListPage1,
        ),
      });
      return;
    }

    // 2. Missing in one implementation browser counts: /v1/stats/features/browsers/{browser}/missing_one_implementation_counts
    if (pathname.includes('missing_one_implementation_counts')) {
      let payload = isPage2
        ? statsMissingOneChromePage2
        : statsMissingOneChromePage1;
      if (pathname.includes('/browsers/firefox/')) {
        payload = isPage2
          ? statsMissingOneFirefoxPage2
          : statsMissingOneFirefoxPage1;
      } else if (pathname.includes('/browsers/safari/')) {
        payload = isPage2
          ? statsMissingOneSafariPage2
          : statsMissingOneSafariPage1;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(payload),
      });
      return;
    }

    // 3. Global feature support browser counts: /v1/stats/features/browsers/{browser}/feature_counts
    if (pathname.includes('/feature_counts')) {
      let payload = isPage2
        ? statsFeatureCountsChromePage2
        : statsFeatureCountsChromePage1;
      if (pathname.includes('/browsers/firefox/')) {
        payload = isPage2
          ? statsFeatureCountsFirefoxPage2
          : statsFeatureCountsFirefoxPage1;
      } else if (pathname.includes('/browsers/safari/')) {
        payload = isPage2
          ? statsFeatureCountsSafariPage2
          : statsFeatureCountsSafariPage1;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(payload),
      });
      return;
    }

    // 4. Baseline total counts: /v1/stats/baseline_status/low_date_feature_counts
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(
        isPage2 ? statsLowDateCountsPage2 : statsLowDateCountsPage1,
      ),
    });
  });

  // Unified Features Interceptor (List, Details, WPT metrics, UMA metrics, 404s, Redirects, Splits)
  await page.route('**/v1/features/**', async (route: Route) => {
    const url = new URL(route.request().url());
    const pathname = url.pathname;

    // Feature WPT metrics chart data (Per-Browser & Paginated)
    if (pathname.includes('/stats/wpt/')) {
      const isPage2 = url.searchParams.get('page_token') === 'page_2';
      let payload = isPage2
        ? featureWptMetricsChromePage2
        : featureWptMetricsChromePage1;

      if (pathname.includes('/browsers/chrome/')) {
        payload = isPage2
          ? featureWptMetricsChromePage2
          : featureWptMetricsChromePage1;
      } else if (pathname.includes('/browsers/firefox/')) {
        payload = isPage2
          ? featureWptMetricsFirefoxPage2
          : featureWptMetricsFirefoxPage1;
      } else if (pathname.includes('/browsers/safari/')) {
        payload = isPage2
          ? featureWptMetricsSafariPage2
          : featureWptMetricsSafariPage1;
      } else if (pathname.includes('/browsers/edge/')) {
        payload = isPage2
          ? featureWptMetricsEdgePage2
          : featureWptMetricsEdgePage1;
      }

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(payload),
      });
      return;
    }

    // Feature UMA metrics chart data (Paginated)
    if (pathname.includes('/stats/usage/')) {
      const isPage2 = url.searchParams.get('page_token') === 'page_2';
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          isPage2 ? featureUmaMetricsPage2 : featureUmaMetricsPage1,
        ),
      });
      return;
    }

    // Single feature detail endpoint: /v1/features/{id}
    const pathParts = pathname.split('/').filter(Boolean);
    if (
      pathParts.length === 3 &&
      pathParts[0] === 'v1' &&
      pathParts[1] === 'features'
    ) {
      const featureId = pathParts[2];
      if (featureId === 'anchor-positioning') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(featureDetailAnchorPositioningFixture),
        });
        return;
      }
      if (featureId === 'discouraged') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(featureDetailDiscouragedFixture),
        });
        return;
      }
      if (featureId === 'old-feature') {
        await route.fulfill({
          status: 301,
          headers: {Location: 'http://localhost:5555/v1/features/new-feature'},
          body: JSON.stringify({code: 301, message: 'Moved Permanently'}),
        });
        return;
      }
      if (featureId === 'new-feature') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            ...featureDetailAnchorPositioningFixture,
            feature_id: 'new-feature',
            name: 'New Feature',
          }),
        });
        return;
      }
      if (featureId === 'before-split-feature') {
        await route.fulfill({
          status: 410,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 410,
            message: 'Feature has split into multiple features',
            new_features: [
              {id: 'after-split-feature-1'},
              {id: 'after-split-feature-2'},
            ],
          }),
        });
        return;
      }
      // 404 for unknown feature
      await route.fulfill({
        status: 404,
        contentType: 'application/json',
        body: JSON.stringify({code: 404, message: 'Feature not found'}),
      });
      return;
    }

    // Fallback: pass through or 404
    await route.continue();
  });

  // Feature List / Query / Pagination endpoint: /v1/features
  await page.route('**/v1/features', async (route: Route) => {
    const url = new URL(route.request().url());
    const pageToken = url.searchParams.get('page_token');
    const sort = url.searchParams.get('sort');
    const columns = url.searchParams.get('columns');
    const query = url.searchParams.get('q');

    // Invalid query simulation
    if (query && query.includes('available_on:chrom')) {
      await route.fulfill({
        status: 400,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 400,
          message: 'Invalid query syntax',
          errors: {q: 'Unknown keyword: chrom'},
        }),
      });
      return;
    }

    // Similar features query for 404 page
    if (query === 'g') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          metadata: {total: 2},
          data: [featuresPage1Fixture.data[0], featuresPage1Fixture.data[1]],
        }),
      });
      return;
    }

    // Nonexistent feature query (returns 0 results)
    if (query && query.includes('nonexistent-feature')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({metadata: {}, data: []}),
      });
      return;
    }

    // Sort simulation
    if (sort && sort.includes('availability')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(featuresSortFixture),
      });
      return;
    }

    // Upvotes column simulation
    if (columns && columns.includes('developer_upvotes')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(featuresUpvotesFixture),
      });
      return;
    }

    // Pagination: Page 2
    if (pageToken === 'page_2_token') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(featuresPage2Fixture),
      });
      return;
    }

    // Default: Page 1
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(featuresPage1Fixture),
    });
  });
}
