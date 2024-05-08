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

import {expect, fixture} from '@open-wc/testing';
import {FeaturePage} from '../webstatus-feature-page.js';
import '../webstatus-feature-page.js';

describe('webstatus-feature-page', () => {
  let el: FeaturePage;
  beforeEach(async () => {
    el = await fixture<FeaturePage>(
      '<webstatus-feature-page></webstatus-feature-page>'
    );

    await el.updateComplete;
  });
  it('builds the WPT link correctly', async () => {
    const link = el.buildWPTLink('declarative-shadow-dom');
    expect(link).to.eq(
      'https://wpt.fyi/results?label=master&label=stable&aligned=&q=feature%3Adeclarative-shadow-dom%21is%3Atentative'
    );
  });
});
