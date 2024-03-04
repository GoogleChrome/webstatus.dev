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

  // The sidebar menu should be shown by default.

  const sidebar = page.locator('webstatus-sidebar');
  const sidebarBox = await sidebar.boundingBox();
  await expect(sidebar).toHaveScreenshot('sidebar.png', {
    clip: sidebarBox,
    // Temporarily allow a higher diff ratio as these tests become more stable
    maxDiffPixelRatio: 0.2,
  });
});
