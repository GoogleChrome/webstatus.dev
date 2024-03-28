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

import {assert, fixture, html} from '@open-wc/testing';

import {
  openColumnsDialog,
  type WebstatusColumnsDialog,
} from '../webstatus-columns-dialog.js';

describe('webstatus-columns-dialog', () => {
  it('can be added to the page', async () => {
    const component = await fixture<WebstatusColumnsDialog>(
      html` <webstatus-columns-dialog></webstatus-columns-dialog>`
    );
    assert.exists(component);
  });
});

describe('openColumnsDialog', () => {
  it('can add the dialog to the page and open it', async () => {
    const loc = {search: ''};
    const component = await openColumnsDialog(loc);
    assert.exists(component);
  });
});
