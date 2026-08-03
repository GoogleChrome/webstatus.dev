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

import {Page, expect, type Locator} from '@playwright/test';

const DEFAULT_FAKE_NOW = 'Dec 1 2020 12:34:56';

export const BASE_URL = 'http://localhost:5555';
export const WIREMOCK_URL = process.env.WIREMOCK_URL || 'http://localhost:8087';

export async function forceTheme(page: Page, theme: 'light' | 'dark') {
  await page.addInitScript(theme => {
    localStorage.setItem('webstatus-theme', theme);
  }, theme);
  await page.emulateMedia({colorScheme: theme});
}

/**
 * Captures screenshots for both light and dark themes.
 * Resets the theme to light after capturing.
 */
export async function expectDualThemeScreenshot(
  page: Page,
  locator: Locator | Page,
  name: string,
  options?: Parameters<Locator['screenshot']>[0],
) {
  // 1. Ensure light theme and capture
  await forceTheme(page, 'light');
  await expect(locator).toHaveScreenshot(`${name}.png`, options);

  // 2. Change to dark theme and capture
  await forceTheme(page, 'dark');
  await expect(locator).toHaveScreenshot(`${name}-dark.png`, options);

  // 3. Reset to light for subsequent tests
  await forceTheme(page, 'light');
}

export async function waitForChartCompletion(page: Page, containerId: string) {
  const chartContainer = page.locator(`#${containerId}`);
  // First wait for the chart component to be attached to the DOM
  await expect(chartContainer).toBeAttached({timeout: 10000});
  // Then wait for the internal \`webstatus-gchart\` to drop its \`.loading\` spinner
  const loadingOverlay = chartContainer.locator('webstatus-gchart .loading');
  await expect(loadingOverlay).toHaveCount(0, {timeout: 10000});
}

export async function waitForTabbedChartCompletion(
  page: Page,
  panelBaseId: string,
  tabIndex: number,
) {
  // webstatus-line-chart-tabbed-panel generates IDs like 'feature-wpt-implementation-progress-0-complete'
  const containerId = `${panelBaseId}-${tabIndex}-complete`;
  await waitForChartCompletion(page, containerId);
}

export async function setupFakeNow(
  page: Page,
  fakeNowDateString = DEFAULT_FAKE_NOW,
): Promise<void> {
  // Get fakeNow from UTC to extract the timeZone offset used in the test
  const fakeNowFromUTC = new Date(fakeNowDateString);
  const offset = fakeNowFromUTC.getTimezoneOffset();
  const offsetSign = offset < 0 ? '-' : '+';
  const offsetHours = `${Math.abs(Math.floor(offset / 60))}`.padStart(2, '0');
  const offsetMinutes = `${Math.abs(offset % 60)}`.padStart(2, '0');
  const offsetText = `${offsetSign}${offsetHours}:${offsetMinutes}`;

  // Get fakeNow from the test timeZone
  const fakeNow = new Date(`${fakeNowDateString}Z${offsetText}`).valueOf();

  // Update the Date accordingly in your test pages
  await page.addInitScript(`{
    // Extend Date constructor to default to fakeNow
    Date = class extends Date {
      constructor(...args) {
        if (args.length === 0) {
          super(${fakeNow});
        } else {
          super(...args);
        }
      }
    }
    // Override Date.now() to start from fakeNow
    const __DateNowOffset = ${fakeNow} - Date.now();
    const __DateNow = Date.now;
    Date.now = () => __DateNow() + __DateNowOffset;
  }`);
}

export async function waitForOverviewPageLoad(page: Page) {
  // Wait for the loading indicator to disappear and be replaced (with timeout):
  await page
    .locator('webstatus-overview-content >> text=Loading features...')
    .waitFor({state: 'hidden', timeout: 15000});
}

export async function waitForSidebarLoaded(page: Page) {
  const sidebar = page.locator('webstatus-sidebar');
  // Wait for sidebar to be attached
  await expect(sidebar).toBeAttached({timeout: 10000});
  // Wait for the absence of sl-skeleton inside the sidebar
  const skeleton = sidebar.locator('sl-skeleton');
  await expect(skeleton).toHaveCount(0, {timeout: 10000});
}

export async function gotoOverviewPageUrl(page: Page, url: string) {
  await page.goto(url);

  await waitForOverviewPageLoad(page);
  await waitForSidebarLoaded(page);
}

export async function getOverviewPageFeatureCount(page: Page): Promise<number> {
  await waitForOverviewPageLoad(page);
  const regex = /(\d+) features/;
  const statsSummary = page.getByText(regex);
  expect(statsSummary).toBeVisible();
  const text = await statsSummary.innerText();
  return parseInt(text.match(regex)![1]);
}

// Based on util/cmd/load_test_users/main.go
export const testUsers = {
  'test user 1': 'test.user.1@example.com',
  'test user 2': 'test.user.2@example.com',
  'test user 3': 'test.user.3@example.com',
  'fresh user': 'fresh.user@example.com',
  'chromium user': 'chromium.user@example.com',
  'firefox user': 'firefox.user@example.com',
  'webkit user': 'webkit.user@example.com',
};

/**
 * Sets the Wiremock scenario state for user emails based on the provided username.
 * This ensures that Wiremock serves the correct email stubs for the logged-in user.
 */
export async function setWiremockScenario(scenarioName: string, state: string) {
  try {
    const response = await fetch(
      `${WIREMOCK_URL}/__admin/scenarios/${scenarioName}/state`,
      {
        method: 'PUT',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({state}),
      },
    );
    if (!response.ok) {
      console.warn(
        `Failed to set Wiremock scenario ${scenarioName} to ${state}: ${response.statusText}`,
      );
    }
  } catch (error) {
    console.warn(`Could not connect to Wiremock at ${WIREMOCK_URL}:`, error);
  }
}

export async function setUserWiremockScenarioState(
  _page: Page,
  username: keyof typeof testUsers,
) {
  const state = username === 'test user 2' ? 'user2_logged_in' : 'Started';
  await Promise.all([
    setWiremockScenario('user_profile', state),
    setWiremockScenario('user_emails', state),
  ]);
}

export async function resetWiremockScenarioState() {
  await Promise.all([
    setWiremockScenario('user_profile', 'Started'),
    setWiremockScenario('user_emails', 'Started'),
  ]);
}

export async function loginAsUser(
  page: Page,
  username: keyof typeof testUsers,
  options: {waitFor: 'idle' | 'syncing' | 'error'} = {waitFor: 'idle'},
) {
  // Set Wiremock scenario state based on the user.
  await setUserWiremockScenarioState(page, username);

  // Clicking the log in button will create a popup that we need to capture.
  const popupPromise = page.waitForEvent('popup');
  await page.goto('http://localhost:5555/');
  await waitForSidebarLoaded(page);
  await page.getByRole('banner').getByText('Log in').click();
  const popup = await popupPromise;

  await popup.waitForLoadState();
  await popup.getByText(username).waitFor({timeout: 5000});
  await popup.getByText(username).hover(); // Needed for Firefox for some reason.
  await popup.getByText(username).click();
  await popup.waitForEvent('close');

  const email = testUsers[username];
  const loginButton = page.getByRole('button', {name: email});

  // Wait for the button to become visible on the main page.
  await expect(loginButton).toBeVisible();

  switch (options.waitFor) {
    case 'syncing':
      // Wait for the loading spinner on the user button to appear.
      await expect(loginButton.locator('sl-spinner')).toBeVisible();
      break;
    case 'error':
      // Wait for the error icon to be present in the DOM.
      await expect(
        page
          .getByTestId('error-while-syncing-button')
          .getByTestId('error-icon'),
      ).toBeVisible();
      break;
    case 'idle':
    default:
      // Wait for the loading spinner on the user button to disappear.
      await expect(loginButton.locator('sl-spinner')).toBeHidden();
      break;
  }
}

export async function dismissToast(page: Page) {
  const toast = page.locator('sl-alert[variant="danger"][open]');
  await toast.locator('sl-icon-button[name="x-lg"]').click();
  await expect(toast).not.toBeVisible();
}

export async function freezeAnimations(page: Page) {
  await page.addStyleTag({
    content: `
      *,
      *::before,
      *::after {
        animation-play-state: paused !important;
        animation-duration: 0.01s !important;
        animation-iteration-count: 1 !important;
        caret-color: transparent !important;
        transition: none !important;
      }
    `,
  });
}

export async function goTo404Page(page: Page, query: string): Promise<void> {
  const responsePromise = page.waitForResponse(
    response =>
      response.url().includes(`/v1/features/${query}`) &&
      response.request().method() === 'GET',
  );
  await page.goto(`${BASE_URL}/features/${query}`);
  await responsePromise;

  await expect(page).toHaveURL(
    `${BASE_URL}/errors-404/feature-not-found?q=${query}`,
  );

  const response = await page.context().request.fetch(page.url());
  expect(response.status()).toBe(404);
}

export async function expect404PageButtons(
  page: Page,
  {hasSearch}: {hasSearch: boolean},
) {
  await expect(page.locator('#error-action-home-btn')).toBeVisible();
  await expect(page.locator('#error-action-report')).toBeVisible();

  if (hasSearch) {
    await expect(page.locator('#error-action-search-btn')).toBeVisible();
  } else {
    await expect(page.locator('#error-action-search-btn')).toHaveCount(0);
  }
}

export async function resetUserData() {
  await resetWiremockScenarioState();
}
