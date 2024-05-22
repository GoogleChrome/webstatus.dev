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
  it('builds the WPT link correctly when there are stable metrics', async () => {
    const link = el.buildWPTLink({
      feature_id: 'declarative-shadow-dom',
      wpt: {stable: {}},
    });
    expect(link).to.eq(
      'https://wpt.fyi/results?label=master&label=stable&aligned=&q=feature%3Adeclarative-shadow-dom+%21is%3Atentative'
    );
  });

  it('builds a null WPT link correctly when there are no stable metrics', async () => {
    const noStableMetricsLink = el.buildWPTLink({
      feature_id: 'declarative-shadow-dom',
      wpt: {experimental: {}},
    });
    expect(noStableMetricsLink).to.eq(null);

    const missingWPTSectionLink = el.buildWPTLink({
      feature_id: 'declarative-shadow-dom',
    });
    expect(missingWPTSectionLink).to.eq(null);

    const missingFeatureLink = el.buildWPTLink();
    expect(missingFeatureLink).to.eq(null);
  });

  it('optionally builds a caniuse link', async () => {
    // Single item renders a link
    const singleItem = {
      items: [{id: 'flexbox'}],
    };
    const singleItemLink = el.findCanIUseLink(singleItem);
    expect(singleItemLink).to.eq('https://caniuse.com/flexbox');

    // Multiple item renders no link
    const multipleItems = {
      items: [{id: 'flexbox'}, {id: 'grid'}],
    };
    const multipleItemsLink = el.findCanIUseLink(multipleItems);
    expect(multipleItemsLink).to.eq(null);

    // No item renders no link
    const emptyItems = {
      items: [],
    };
    const emptyItemsLink = el.findCanIUseLink(emptyItems);
    expect(emptyItemsLink).to.eq(null);

    // Undefined items renders no link
    const undefinedItems = {
      items: undefined,
    };
    const undefinedItemsLink = el.findCanIUseLink(undefinedItems);
    expect(undefinedItemsLink).to.eq(null);

    // Undefined object renders no link
    const undefinedObjItems = undefined;
    const undefinedObjItemsLink = el.findCanIUseLink(undefinedObjItems);
    expect(undefinedObjItemsLink).to.eq(null);
  });
});
