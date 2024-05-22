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

import {expect, fixture, html} from '@open-wc/testing';
import {FeaturePage} from '../webstatus-feature-page.js';
import '../webstatus-feature-page.js';
import sinon from 'sinon';

describe('webstatus-feature-page', () => {
  let el: FeaturePage;
  let renderDescriptionSpy: sinon.SinonSpy;
  beforeEach(async () => {
    const location = {params: {featureId: 'some-feature'}, search: ''};
    el = await fixture<FeaturePage>(
      html`<webstatus-feature-page
        .location=${location}
      ></webstatus-feature-page>`
    );

    renderDescriptionSpy = sinon.spy(el, 'renderDescription');

    await el.updateComplete;
  });
  it('builds the WPT link correctly', async () => {
    const link = el.buildWPTLink('declarative-shadow-dom');
    expect(link).to.eq(
      'https://wpt.fyi/results?label=master&label=stable&aligned=&q=feature%3Adeclarative-shadow-dom+%21is%3Atentative'
    );
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

  it('renders nothing when no featureMetadata or description is provided', async () => {
    const callbacks = {
      complete: sinon.fake(),
      error: sinon.fake(),
      initial: sinon.fake(),
      pending: sinon.fake(),
    };
    el._loadingTask?.render(callbacks);
    el._loadingTask?.run();
    callbacks.complete();

    await el.updateComplete;

    expect(renderDescriptionSpy.callCount).to.be.greaterThan(0);
    const descriptionSection = el.querySelector('#feature-description');
    expect(descriptionSection).to.be.null;
  });

  it('renders a description after task completion', async () => {
    el.featureMetadata = {description: 'AMAZING DESCRIPTION'};
    await el.updateComplete;

    const callbacks = {
      complete: sinon.fake(),
      error: sinon.fake(),
      initial: sinon.fake(),
      pending: sinon.fake(),
    };
    el._loadingTask?.render(callbacks);
    el._loadingTask?.run();
    callbacks.complete();

    await el.updateComplete;

    expect(renderDescriptionSpy.callCount).to.be.greaterThan(0);
    const descriptionSection = el.shadowRoot?.querySelector(
      '#feature-description'
    );
    expect(descriptionSection).to.not.be.null;
    expect(descriptionSection?.textContent).to.contain('AMAZING DESCRIPTION');
  });
});
