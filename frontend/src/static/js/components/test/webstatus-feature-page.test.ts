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
import {WPTRunMetric} from '../../api/client.js';
import {render} from 'lit';

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
  describe('renderDeltaChip', () => {
    let element: FeaturePage;
    let hostElement: HTMLDivElement;

    beforeEach(async () => {
      element = await fixture(
        html`<webstatus-feature-page></webstatus-feature-page>`
      );
      hostElement = document.createElement('div');

      // Create a new Map for featureSupport
      element.featureSupport = new Map<string, Array<WPTRunMetric>>([
        // increase case
        [
          'chrome-stable',
          [
            {
              test_pass_count: 90,
              total_tests_count: 100,
              run_timestamp: '2023-12-27T01:28:25.177Z',
            },
            {
              test_pass_count: 85,
              total_tests_count: 100,
              run_timestamp: '2023-12-26T01:28:07.225Z',
            },
          ],
        ],
        // decrease case
        [
          'edge-stable',
          [
            {
              test_pass_count: 70,
              total_tests_count: 100,
              run_timestamp: '2023-12-27T01:28:25.177Z',
            },
            {
              test_pass_count: 75,
              total_tests_count: 100,
              run_timestamp: '2023-12-26T01:28:07.225Z',
            },
          ],
        ],
        // no changes case
        [
          'safari-stable',
          [
            {
              test_pass_count: 80,
              total_tests_count: 100,
              run_timestamp: '2023-12-27T01:28:25.177Z',
            },
            {
              test_pass_count: 80,
              total_tests_count: 100,
              run_timestamp: '2023-12-26T01:28:07.225Z',
            },
          ],
        ],
        // firefox will be the no runs case
      ]);
    });

    it('renders unchanged chip when there are no runs', async () => {
      const chipTemplate = element.renderDeltaChip('firefox');
      render(chipTemplate, hostElement);
      const host = await fixture(hostElement);
      const chip = host.querySelector('span');
      expect(chip?.classList.contains('unchanged')).to.be.true;
      expect(chip?.textContent).to.equal('');
    });

    it('renders chip with positive delta and increased class', async () => {
      const chipTemplate = element.renderDeltaChip('chrome');
      render(chipTemplate, hostElement);
      const host = await fixture(hostElement);
      const chip = host.querySelector('span');
      expect(chip?.classList.contains('increased')).to.be.true;
      expect(chip?.textContent).to.equal('+5.0%');
    });

    it('renders chip with negative delta and decreased class', async () => {
      const chipTemplate = element.renderDeltaChip('edge');
      render(chipTemplate, hostElement);
      const host = await fixture(hostElement);
      const chip = host.querySelector('span');
      expect(chip?.classList.contains('decreased')).to.be.true;
      expect(chip?.textContent).to.equal('-5.0%');
    });

    it('renders chip with no delta and unchanged class', async () => {
      const chipTemplate = element.renderDeltaChip('safari');
      render(chipTemplate, hostElement);
      const host = await fixture(hostElement);
      const chip = host.querySelector('span');
      expect(chip?.classList.contains('unchanged')).to.be.true;
      expect(chip?.textContent).to.equal('0.0%');
    });
  });
  describe('renderBrowserImpl', () => {
    let element: FeaturePage;
    let hostElement: HTMLDivElement;

    beforeEach(async () => {
      element = await fixture(
        html`<webstatus-feature-page></webstatus-feature-page>`
      );
      element.endDate = new Date('2024-01-01');
      hostElement = document.createElement('div');
    });

    it('renders nothing when there is no implemenation', async () => {
      const browserImpl = undefined;
      const actual = element.renderBrowserImpl(browserImpl);
      render(actual, hostElement);
      const host = await fixture(hostElement);
      expect(host.textContent?.trim()).to.equal('');
    });

    it('renders the "since" phrase', async () => {
      const browserImpl = {date: '2024-08-07'};
      const actual = element.renderBrowserImpl(browserImpl);
      render(actual, hostElement);
      const host = await fixture(hostElement);
      expect(host?.textContent).to.contain('Became available on 2024-08-07');
    });

    it('renders the "version" phrase', async () => {
      const browserImpl = {date: '2024-08-07', version: '123'};
      const actual = element.renderBrowserImpl(browserImpl);
      render(actual, hostElement);
      const host = await fixture(hostElement);
      expect(host?.textContent).to.contain(
        'Became available on 2024-08-07 in version 123'
      );
    });
  });
});
