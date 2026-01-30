/**
 * Copyright 2025 Google LLC
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

import {test, expect, Page} from '@playwright/test';
import {gotoOverviewPageUrl, loginAsUser, resetUserData} from './utils';

const USER1 = {
  username: 'test user 1',
  userID: 'abcdedf1234567890',
};
const USER2 = {
  username: 'test user 2',
  userID: 'abcdedf1234567891',
};

const USER1_SEARCH1 = {
  id: '74bdb85f-59d3-43b0-8061-20d5818e8c97',
  name: 'my first project query',
  query: 'baseline_status:newly',
};
const USER1_SEARCH2 = {
  id: 'a09386fe-65f1-4640-b28d-3cf2f2de69c9',
  name: 'I like queries',
  query: 'baseline_status:limited OR available_on:chrome',
};
const USER2_SEARCH1 = {
  id: 'bb85baf7-aa1e-42bf-ada0-cf9d2811dd42',
  name: "test user 2's query",
  query: 'baseline_status:limited',
};

// --- Locators ---
const controlsLocator = (page: Page) =>
  page.locator('webstatus-saved-search-controls');
const editorLocator = (page: Page) =>
  page.locator('webstatus-saved-search-editor');
const mainQueryInputLocator = (page: Page) =>
  page.getByTestId('overview-query-input').getByRole('textbox');
const mainQuerySubmitButtonLocator = (page: Page) =>
  page.getByTestId('overview-query-input').getByRole('button');
const mainDescriptionLocator = (page: Page) =>
  page.locator('#overview-description');
const mainTitleLocator = (page: Page) => page.locator('#overview-title');
const saveButtonLocator = (page: Page) =>
  controlsLocator(page)
    .getByTestId('saved-search-save-button')
    .getByRole('button', {name: 'Save'});
const shareButtonLocator = (page: Page) =>
  controlsLocator(page).getByLabel('Copy', {exact: true});
const bookmarkEmptyIconLocator = (page: Page) =>
  page.getByRole('button', {name: 'Bookmark', exact: true});
const bookmarkFilledIconLocator = (page: Page) =>
  page.getByRole('button', {name: 'Unbookmark', exact: true});
const editIconLocator = (page: Page) =>
  controlsLocator(page).getByRole('button', {name: 'Edit'});
const deleteIconLocator = (page: Page) =>
  controlsLocator(page).getByRole('button', {name: 'Delete'});
const editorDialogLocator = (page: Page) =>
  editorLocator(page).locator('sl-dialog');
const editorNameInputLocator = (page: Page) =>
  editorLocator(page).getByRole('textbox', {name: 'Name *'});
const editorDescriptionInputLocator = (page: Page) =>
  editorLocator(page).getByRole('textbox', {name: 'Description'});
const editorQueryInputLocator = (page: Page) =>
  editorLocator(page)
    .getByTestId('saved-search-editor-query-input')
    .getByRole('textbox', {name: 'Query'});
const editorSubmitButtonLocator = (page: Page) =>
  editorDialogLocator(page)
    .locator('#editor-form')
    .getByRole('button', {name: 'Save'});
const editorDeleteButtonLocator = (page: Page) =>
  editorDialogLocator(page)
    .locator('#editor-form')
    .getByRole('button', {name: 'Delete'});
const editorCancelButtonLocator = (page: Page) =>
  editorDialogLocator(page).getByRole('button', {name: 'Cancel'});
const editorAlertLocator = (page: Page) =>
  editorLocator(page).locator('sl-alert#editor-alert');

test.describe('Saved Searches on Overview Page', () => {
  test.beforeAll(async () => {
    await resetUserData();
  });
  test.afterAll(async () => {
    await resetUserData();
  });

  test.beforeEach(async ({page}) => {
    page.on('dialog', dialog => dialog.accept());
    // Navigate and potentially clear state if needed
    await gotoOverviewPageUrl(page, 'http://localhost:5555');
    // Ensure no search_id is present initially for some tests
    await page.waitForURL(url => !url.searchParams.has('search_id'));
  });

  test('User 1 can save a new search', async ({page}) => {
    await loginAsUser(page, USER1.username); // Use your login helper
    await gotoOverviewPageUrl(page, 'http://localhost:5555'); // Go again after login if needed

    const newQuery = 'available_on:chrome';
    const newName = 'My Chrome Search';
    const newDescription = 'Testing Chrome features';

    // 1. Enter query in main typeahead
    await mainQueryInputLocator(page).fill(newQuery);
    await mainQuerySubmitButtonLocator(page).click();
    await page.waitForURL(`**/*q=${encodeURIComponent(newQuery)}*`); // Wait for query param

    // 2. Click save icon
    await saveButtonLocator(page).click();
    await expect(editorDialogLocator(page)).toBeVisible();
    await expect(editorDialogLocator(page)).toHaveAttribute(
      'label',
      'Save New Search',
    );

    // 3. Fill editor form
    await expect(editorQueryInputLocator(page)).toHaveValue(newQuery); // Verify query carry-over
    await editorNameInputLocator(page).fill(newName);
    await editorDescriptionInputLocator(page).fill(newDescription);

    // 4. Submit
    await editorSubmitButtonLocator(page).click();

    // 5. Verify dialog closes and URL updates with search_id
    await expect(editorDialogLocator(page)).not.toBeVisible();
    await page.waitForURL(url => url.searchParams.has('search_id'));
    const searchId = new URL(page.url()).searchParams.get('search_id');
    expect(searchId).toBeTruthy();

    // 6. Verify overview page shows description and name
    await expect(mainDescriptionLocator(page)).toHaveText(newDescription);
    await expect(mainTitleLocator(page)).toHaveText(newName);
  });

  test('User 1 (Owner) sees correct controls for their search', async ({
    page,
  }) => {
    await loginAsUser(page, USER1.username);
    await gotoOverviewPageUrl(
      page,
      `http://localhost:5555?search_id=${USER1_SEARCH1.id}`,
    );
    await page.waitForURL(`**/*search_id=${USER1_SEARCH1.id}*`);

    // Verify controls are present
    await expect(saveButtonLocator(page)).toBeVisible();
    await expect(shareButtonLocator(page)).toBeVisible();
    await expect(bookmarkFilledIconLocator(page)).toBeVisible();
    await expect(bookmarkFilledIconLocator(page)).toBeDisabled(); // Owner bookmark is filled and disabled
    await expect(editIconLocator(page)).toBeVisible();
    await expect(deleteIconLocator(page)).toBeVisible();

    // Verify empty bookmark icon is not present
    await expect(bookmarkEmptyIconLocator(page)).not.toBeVisible();
  });

  test('User 1 (Viewer) sees correct controls for bookmarked search', async ({
    page,
  }) => {
    await loginAsUser(page, USER1.username);
    await gotoOverviewPageUrl(
      page,
      `http://localhost:5555?search_id=${USER2_SEARCH1.id}`,
    ); // User 1 views User 2's search
    await page.waitForURL(`**/*search_id=${USER2_SEARCH1.id}*`);

    // Verify controls are present
    await expect(saveButtonLocator(page)).toBeVisible();
    await expect(shareButtonLocator(page)).toBeVisible();
    await expect(bookmarkFilledIconLocator(page)).toBeVisible(); // Is bookmarked
    await expect(bookmarkFilledIconLocator(page)).toBeEnabled(); // Can unbookmark

    // Verify owner/other controls are NOT present
    await expect(bookmarkEmptyIconLocator(page)).not.toBeVisible();
    await expect(editIconLocator(page)).not.toBeVisible();
    await expect(deleteIconLocator(page)).not.toBeVisible();
  });

  test('User 2 (Viewer) sees correct controls for non-bookmarked search', async ({
    page,
  }) => {
    await loginAsUser(page, USER2.username);
    await gotoOverviewPageUrl(
      page,
      `http://localhost:5555?search_id=${USER1_SEARCH1.id}`,
    ); // User 2 views User 1's search
    await page.waitForURL(`**/*search_id=${USER1_SEARCH1.id}*`);

    // Verify controls are present
    await expect(saveButtonLocator(page)).toBeVisible();
    await expect(shareButtonLocator(page)).toBeVisible();
    await expect(bookmarkEmptyIconLocator(page)).toBeVisible(); // Not bookmarked
    await expect(bookmarkEmptyIconLocator(page)).toBeEnabled(); // Can bookmark

    // Verify owner/other controls are NOT present
    await expect(bookmarkFilledIconLocator(page)).not.toBeVisible();
    await expect(editIconLocator(page)).not.toBeVisible();
    await expect(deleteIconLocator(page)).not.toBeVisible();
  });

  test('User 1 can edit their own search', async ({page}) => {
    await loginAsUser(page, USER1.username);
    await gotoOverviewPageUrl(
      page,
      `http://localhost:5555?search_id=${USER1_SEARCH1.id}`,
    );
    await page.waitForURL(`**/*search_id=${USER1_SEARCH1.id}*`);

    const updatedName = 'My Updated Query Name';
    const updatedDescription = 'Description is now updated.';

    // 1. Click edit icon
    await editIconLocator(page).click();
    await expect(editorDialogLocator(page)).toBeVisible();
    await expect(editorDialogLocator(page)).toHaveAttribute(
      'label',
      'Edit Saved Search',
    );

    // 2. Verify pre-filled data
    await expect(editorNameInputLocator(page)).toHaveValue(USER1_SEARCH1.name);
    await expect(editorQueryInputLocator(page)).toHaveValue(
      USER1_SEARCH1.query,
    );

    // 3. Change data
    await editorNameInputLocator(page).fill(updatedName);
    await editorDescriptionInputLocator(page).fill(updatedDescription);

    // 4. Submit
    await editorSubmitButtonLocator(page).click();

    // 5. Verify dialog closes
    await expect(editorDialogLocator(page)).not.toBeVisible();

    // 6. Verify page is updated with new data
    await expect(mainTitleLocator(page)).toContainText(updatedName);
    await expect(mainDescriptionLocator(page)).toContainText(
      updatedDescription,
    );
    // 7. Navigate away and back to verify persistence
    await gotoOverviewPageUrl(page, 'http://localhost:5555');
    await gotoOverviewPageUrl(
      page,
      `http://localhost:5555?search_id=${USER1_SEARCH1.id}`,
    );
    await expect(mainTitleLocator(page)).toContainText(updatedName);
    await expect(mainDescriptionLocator(page)).toContainText(
      updatedDescription,
    );
  });

  test('User 1 can delete their own search', async ({page}) => {
    await loginAsUser(page, USER1.username);
    // Use the second search to avoid conflicts if other tests rely on the first one
    await gotoOverviewPageUrl(
      page,
      `http://localhost:5555?search_id=${USER1_SEARCH2.id}`,
    );
    await page.waitForURL(`**/*search_id=${USER1_SEARCH2.id}*`);

    // 1. Click delete icon
    await deleteIconLocator(page).click();
    await expect(editorDialogLocator(page)).toBeVisible();
    await expect(editorDialogLocator(page)).toHaveAttribute(
      'label',
      'Delete Saved Search',
    );
    await expect(editorDialogLocator(page)).toContainText(
      'Are you sure you want to delete this search?',
    );

    // 2. Confirm deletion
    await editorDeleteButtonLocator(page).click();

    // 3. Verify dialog closes and URL updates (removes search_id)
    await expect(editorDialogLocator(page)).not.toBeVisible();
    await page.waitForURL(url => !url.searchParams.has('search_id'));

    // 4. Verify controls no longer show the deleted search
    await expect(editIconLocator(page)).not.toBeVisible();
    await expect(deleteIconLocator(page)).not.toBeVisible();
  });

  test('User 2 can bookmark a search', async ({page}) => {
    await loginAsUser(page, USER2.username);
    await gotoOverviewPageUrl(
      page,
      `http://localhost:5555?search_id=${USER1_SEARCH1.id}`,
    ); // User 2 views User 1's search
    await page.waitForURL(`**/*search_id=${USER1_SEARCH1.id}*`);

    // 1. Verify empty star is present
    await expect(bookmarkEmptyIconLocator(page)).toBeVisible();
    await expect(bookmarkFilledIconLocator(page)).not.toBeVisible();

    // 2. Click empty star
    await bookmarkEmptyIconLocator(page).click();

    // 3. Verify star becomes filled (wait for potential async update)
    await expect(bookmarkFilledIconLocator(page)).toBeVisible();
    await expect(bookmarkEmptyIconLocator(page)).not.toBeVisible();

    // 4. Refresh and verify persistence
    await page.reload();
    await page.waitForURL(`**/*search_id=${USER1_SEARCH1.id}*`);
    await expect(bookmarkFilledIconLocator(page)).toBeVisible();
  });

  test('User 1 can unbookmark a search', async ({page}) => {
    await loginAsUser(page, USER1.username);
    await gotoOverviewPageUrl(
      page,
      `http://localhost:5555?search_id=${USER2_SEARCH1.id}`,
    );
    // User 1 views User 2's search (which they bookmarked)
    await page.waitForURL(`**/*search_id=${USER2_SEARCH1.id}*`);

    // 1. Verify filled star is present
    await expect(bookmarkFilledIconLocator(page)).toBeVisible();
    await expect(bookmarkEmptyIconLocator(page)).not.toBeVisible();

    // 2. Click filled star
    await bookmarkFilledIconLocator(page).click();

    // 3. Verify it goes back to the home page and you see no controls
    await expect(bookmarkEmptyIconLocator(page)).not.toBeVisible();
    await expect(bookmarkFilledIconLocator(page)).not.toBeVisible();
  });

  test('Share button copies correct URL to clipboard', async ({
    page,
    browserName,
  }) => {
    // Skip if it is WebKit
    // https://github.com/microsoft/playwright/issues/13037
    if (browserName === 'webkit') {
      test.skip();
    }

    await loginAsUser(page, USER1.username);
    await gotoOverviewPageUrl(
      page,
      `http://localhost:5555?search_id=${USER1_SEARCH1.id}`,
    );
    await page.waitForURL(`**/*search_id=${USER1_SEARCH1.id}*`);

    // Click the share icon
    await shareButtonLocator(page).click();

    // Verify clipboard content
    const expectedUrl = `${page.url()}`; // The current URL should be the shareable one
    const clipboardText = await page.evaluate(() =>
      navigator.clipboard.readText(),
    );
    expect(clipboardText).toBe(expectedUrl);
  });

  test('Edit dialog opens automatically with edit_saved_search=true URL parameter', async ({
    page,
  }) => {
    await loginAsUser(page, USER1.username);
    await gotoOverviewPageUrl(
      page,
      `http://localhost:5555?search_id=${USER1_SEARCH1.id}&edit_saved_search=true`,
    );

    // Verify dialog opens automatically
    await expect(editorDialogLocator(page)).toBeVisible();
    await expect(editorDialogLocator(page)).toHaveAttribute(
      'label',
      'Edit Saved Search',
    );

    // Verify URL parameter is removed after dialog opens
    await page.waitForURL(url => !url.searchParams.has('edit_saved_search'));
    expect(page.url()).not.toContain('edit_saved_search=true');

    // Close the dialog
    await editorCancelButtonLocator(page).click();
    await expect(editorDialogLocator(page)).not.toBeVisible();
  });

  test('Save dialog shows validation alert on invalid submission', async ({
    page,
  }) => {
    await loginAsUser(page, USER1.username);
    await gotoOverviewPageUrl(page, 'http://localhost:5555');

    const newQuery = 'available_on:firefox';

    // 1. Enter query and click save
    await mainQueryInputLocator(page).fill(newQuery);
    await mainQuerySubmitButtonLocator(page).click();
    await saveButtonLocator(page).click();
    await expect(editorDialogLocator(page)).toBeVisible();

    // 2. Attempt to submit without filling the name (which is required)
    // Ensure query is filled but name is empty
    await expect(editorQueryInputLocator(page)).toHaveValue(newQuery);
    await expect(editorNameInputLocator(page)).toHaveValue('');

    await editorSubmitButtonLocator(page).click();

    // 3. Verify alert is shown and dialog remains open
    await expect(editorAlertLocator(page)).toBeVisible();
    await expect(editorAlertLocator(page)).toContainText(
      'Please check that you provided at least a name and query',
    );
    await expect(editorDialogLocator(page)).toBeVisible(); // Should not close

    // 4. Fill name and submit successfully
    await editorNameInputLocator(page).fill('My Firefox Search');
    await editorSubmitButtonLocator(page).click();
    await expect(editorDialogLocator(page)).not.toBeVisible(); // Should close now
    await page.waitForURL(url => url.searchParams.has('search_id'));
  });

  test.describe('Subscriptions', () => {
    test('User 1 can edit an existing subscription', async ({page}) => {
      await loginAsUser(page, USER1.username);
      await gotoOverviewPageUrl(
        page,
        `http://localhost:5555?search_id=${USER1_SEARCH1.id}`,
      );
      await page.getByRole('button', {name: 'Subscribe'}).click();
      const dialog = page.locator('webstatus-manage-subscriptions-dialog');
      await expect(
        dialog.getByRole('heading', {name: 'Manage notifications'}),
      ).toBeVisible();

      // Select the already subscribed channel.
      await dialog.getByText('test.user.1@example.com').click();

      // The button should say "Save preferences".
      const saveButton = dialog.getByRole('button', {name: 'Save preferences'});
      await expect(saveButton).toBeVisible();

      // Change the frequency.
      await dialog.locator('sl-radio', {hasText: 'Monthly digest'}).click();
      await saveButton.click();

      // Assert that the success toast appears.
      await expect(
        page.locator('sl-alert', {hasText: 'Subscription saved!'}),
      ).toBeVisible();
    });

    test('User 1 can add a subscription to a new channel', async ({page}) => {
      await loginAsUser(page, USER1.username);
      await gotoOverviewPageUrl(
        page,
        `http://localhost:5555?search_id=${USER1_SEARCH1.id}`,
      );
      await page.getByRole('button', {name: 'Subscribe'}).click();
      const dialog = page.locator('webstatus-manage-subscriptions-dialog');
      await expect(
        dialog.getByRole('heading', {name: 'Manage notifications'}),
      ).toBeVisible();

      // Select the un-subscribed channel.
      await dialog.getByText('third@mock-github.local').click();

      // Select a trigger to enable the create button.
      await dialog
        .locator('sl-checkbox')
        .filter({hasText: '...becomes widely available'})
        .locator('label')
        .click();

      // The button should say "Create Subscription".
      const createButton = dialog.getByRole('button', {
        name: 'Create Subscription',
      });
      await expect(createButton).toBeVisible();
      await createButton.click();

      // Assert that the success toast appears.
      await expect(
        page.locator('sl-alert', {hasText: 'Subscription saved!'}),
      ).toBeVisible();
    });

    test('User 1 can delete a subscription', async ({page}) => {
      await loginAsUser(page, USER1.username);
      await gotoOverviewPageUrl(
        page,
        `http://localhost:5555?search_id=${USER1_SEARCH1.id}`,
      );
      await page.getByRole('button', {name: 'Subscribe'}).click();
      const dialog = page.locator('webstatus-manage-subscriptions-dialog');
      await expect(
        dialog.getByRole('heading', {name: 'Manage notifications'}),
      ).toBeVisible();

      // Select the already subscribed channel.
      await dialog.getByText('test.user.1@example.com').click();

      // Click the delete button.
      await dialog.getByRole('button', {name: 'Delete Subscription'}).click();

      // Assert that the success toast appears.
      await expect(
        page.locator('sl-alert', {hasText: 'Subscription deleted!'}),
      ).toBeVisible();
    });

    test('User 2 can subscribe to a saved search owned by User 1', async ({
      page,
    }) => {
      await loginAsUser(page, USER2.username);
      await gotoOverviewPageUrl(
        page,
        `http://localhost:5555?search_id=${USER1_SEARCH1.id}`,
      );
      await page.getByRole('button', {name: 'Subscribe'}).click();
      const dialog = page.locator('webstatus-manage-subscriptions-dialog');
      await expect(
        dialog.getByRole('heading', {name: 'Manage notifications'}),
      ).toBeVisible();

      // Select User 2's email channel.
      await dialog.getByText('test.user.2@example.com').click();

      // Select a trigger to enable the create button.
      await dialog
        .locator('sl-checkbox')
        .filter({hasText: '...becomes widely available'})
        .locator('label')
        .click();

      // The button should say "Create Subscription".
      const createButton = dialog.getByRole('button', {
        name: 'Create Subscription',
      });
      await expect(createButton).toBeVisible();
      await createButton.click();

      // Assert that the success toast appears.
      await expect(
        page.locator('sl-alert', {hasText: 'Subscription saved!'}),
      ).toBeVisible();
    });
  });
});
