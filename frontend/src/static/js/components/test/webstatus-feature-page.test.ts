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
import {FeatureMovedError} from '../../api/errors.js';

describe('webstatus-feature-page', () => {
  let el: FeaturePage;
  let renderDescriptionSpy: sinon.SinonSpy;
  let getWPTMetricViewStub: sinon.SinonStub;
  const location = {
    params: {featureId: 'some-feature'},
    search: '',
    pathname: '/features/some-feature',
  };
  beforeEach(async () => {
    el = await fixture<FeaturePage>(
      html`<webstatus-feature-page
        .location=${location}
      ></webstatus-feature-page>`,
    );

    renderDescriptionSpy = sinon.spy(el, 'renderDescription');

    getWPTMetricViewStub = sinon.stub(el, '_getWPTMetricView');
    // Returns nothing by default.
    getWPTMetricViewStub.returns('');

    await el.updateComplete;
  });
  afterEach(() => {
    sinon.restore();
  });
  it('builds the WPT link correctly when there are stable metrics with default metric view', async () => {
    const link = el.buildWPTLink({
      feature_id: 'declarative-shadow-dom',
      wpt: {stable: {}},
    });
    expect(link).to.eq(
      'https://wpt.fyi/results?label=master&label=stable&q=feature%3Adeclarative-shadow-dom+%21is%3Atentative&view=test',
    );
  });

  it('builds the WPT link correctly when there are stable metrics with metric view = subtest_counts', async () => {
    getWPTMetricViewStub.returns('subtest_counts');
    await el.updateComplete;
    const link = el.buildWPTLink({
      feature_id: 'declarative-shadow-dom',
      wpt: {stable: {}},
    });
    expect(link).to.eq(
      'https://wpt.fyi/results?label=master&label=stable&q=feature%3Adeclarative-shadow-dom+%21is%3Atentative&view=subtest',
    );
  });

  it('builds the WPT link correctly when there are stable metrics with metric view = subtest_counts', async () => {
    getWPTMetricViewStub.returns('test_counts');
    await el.updateComplete;
    const link = el.buildWPTLink({
      feature_id: 'declarative-shadow-dom',
      wpt: {stable: {}},
    });
    expect(link).to.eq(
      'https://wpt.fyi/results?label=master&label=stable&q=feature%3Adeclarative-shadow-dom+%21is%3Atentative&view=test',
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
      '#feature-description',
    );
    expect(descriptionSection).to.not.be.null;
    expect(descriptionSection?.textContent).to.contain('AMAZING DESCRIPTION');
  });
  describe('renderDeltaChip', () => {
    let element: FeaturePage;
    let hostElement: HTMLDivElement;

    beforeEach(async () => {
      element = await fixture(
        html`<webstatus-feature-page
          .location=${location}
        ></webstatus-feature-page>`,
      );
      hostElement = document.createElement('div');

      // Create a new Map for featureSupport
      element.featureSupport = new Map<string, Array<WPTRunMetric>>([
        // increase case
        [
          'chrome',
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
          'edge',
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
          'safari',
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
      await element.updateComplete;
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
        html`<webstatus-feature-page
          .location=${location}
        ></webstatus-feature-page>`,
      );
      element.endDate = new Date('2024-01-01');
      hostElement = document.createElement('div');
    });

    it('renders nothing when there is no implementation', async () => {
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
        'Became available on 2024-08-07 in version 123',
      );
    });
  });

  describe('renderDeveloperSignal', () => {
    let element: FeaturePage;
    let hostElement: HTMLDivElement;

    beforeEach(async () => {
      element = await fixture(
        html`<webstatus-feature-page
          .location=${location}
        ></webstatus-feature-page>`,
      );
      hostElement = document.createElement('div');
    });

    it('renders nothing when there is no signal', async () => {
      const signal = undefined;
      const actual = element.renderDeveloperSignal(signal);
      render(actual, hostElement);
      const host = await fixture(hostElement);
      expect(host.textContent?.trim()).to.equal('');
    });

    it('renders nothing when link is missing', async () => {
      const signal = {upvotes: 10};
      const actual = element.renderDeveloperSignal(signal);
      render(actual, hostElement);
      const host = await fixture(hostElement);
      expect(host.textContent?.trim()).to.equal('');
    });

    it('renders nothing when upvotes are missing', async () => {
      const signal = {link: 'http://example.com'};
      const actual = element.renderDeveloperSignal(signal);
      render(actual, hostElement);
      const host = await fixture(hostElement);
      expect(host.textContent?.trim()).to.equal('');
    });

    it('renders the developer signal button', async () => {
      const signal = {upvotes: 10, link: 'http://example.com'};
      const actual = element.renderDeveloperSignal(signal);
      render(actual, hostElement);
      const host = await fixture(hostElement);
      const tooltip = host.querySelector('sl-tooltip');
      const button = host.querySelector('sl-button');
      const icon = host?.querySelector('sl-icon');

      expect(tooltip).to.not.be.null;
      expect(button).to.not.be.null;
      expect(icon).to.not.be.null;

      expect(tooltip?.getAttribute('content')).to.equal(
        '10 developer upvotes. Need this feature across browsers? Click this and upvote it on GitHub.',
      );
      expect(button?.getAttribute('href')).to.equal('http://example.com');
      expect(button?.getAttribute('aria-label')).to.equal(
        '10 developer upvotes',
      );
      expect(button?.textContent?.trim()).to.equal('10');
      expect(icon?.getAttribute('name')).to.equal('hand-thumbs-up');
    });

    it('renders the developer signal when the upvotes are zero', async () => {
      const signal = {upvotes: 0, link: 'http://example.com'};
      const actual = element.renderDeveloperSignal(signal);
      render(actual, hostElement);
      const host = await fixture(hostElement);
      const tooltip = host.querySelector('sl-tooltip');
      const button = host.querySelector('sl-button');

      expect(tooltip).to.not.be.null;
      expect(button).to.not.be.null;

      expect(tooltip?.getAttribute('content')).to.equal(
        '0 developer upvotes. Need this feature across browsers? Click this and upvote it on GitHub.',
      );
      expect(button?.getAttribute('aria-label')).to.equal(
        '0 developer upvotes',
      );
      expect(button?.textContent?.trim()).to.equal('0');
    });

    it('renders the developer signal button with compact number', async () => {
      const signal = {upvotes: 12345, link: 'http://example.com'};
      const actual = element.renderDeveloperSignal(signal);
      render(actual, hostElement);
      const host = await fixture(hostElement);
      const tooltip = host.querySelector('sl-tooltip');
      const button = host.querySelector('sl-button');

      expect(tooltip).to.not.be.null;
      expect(button).to.not.be.null;

      expect(tooltip?.getAttribute('content')).to.equal(
        '12,345 developer upvotes. Need this feature across browsers? Click this and upvote it on GitHub.',
      );
      expect(button?.getAttribute('aria-label')).to.equal(
        '12,345 developer upvotes',
      );
      expect(button?.textContent?.trim()).to.equal('12.3K');
    });
  });
  describe('renderDiscouragedNotice', () => {
    let element: FeaturePage;
    let hostElement: HTMLDivElement;

    beforeEach(async () => {
      element = await fixture(
        html`<webstatus-feature-page
          .location=${location}
        ></webstatus-feature-page>`,
      );
      hostElement = document.createElement('div');
    });

    it('renders nothing when there are no discouraged details', async () => {
      const discouragedDetails = undefined;
      const actual = element.renderDiscouragedNotice(discouragedDetails);
      render(actual, hostElement);
      const host = await fixture(hostElement);
      expect(host.textContent?.trim()).to.equal('');
    });

    it('renders the basic notice when discouraged details are empty', async () => {
      const discouragedDetails = {};
      const actual = element.renderDiscouragedNotice(discouragedDetails);
      render(actual, hostElement);
      const host = await fixture(hostElement);
      const alert = host.querySelector('sl-alert');
      expect(alert).to.not.be.null;
      expect(alert?.textContent).to.contain('Discouraged');
      expect(host.querySelector('ul')).to.be.null;
    });

    it('renders the notice with "according to" links', async () => {
      const discouragedDetails = {
        according_to: [{link: 'http://example.com/rationale'}],
      };
      const actual = element.renderDiscouragedNotice(discouragedDetails);
      render(actual, hostElement);
      const host = await fixture(hostElement);
      expect(host.textContent).to.contain('For the rationale, see:');
      const link = host.querySelector('a');
      expect(link).to.not.be.null;
      expect(link?.href).to.equal('http://example.com/rationale');
      expect(link?.textContent).to.equal('http://example.com/rationale');
    });

    it('renders the notice with "alternatives" links', async () => {
      const discouragedDetails = {
        alternatives: [{id: 'other-feature'}],
      };
      const actual = element.renderDiscouragedNotice(discouragedDetails);
      render(actual, hostElement);
      const host = await fixture(hostElement);
      expect(host.textContent).to.contain(
        'Consider using the following features instead:',
      );
      const link = host.querySelector('a');
      expect(link).to.not.be.null;
      expect(link?.getAttribute('href')).to.equal('/features/other-feature');
      expect(link?.textContent).to.equal('other-feature');
    });

    it('renders the notice with both "according to" and "alternatives" links', async () => {
      const discouragedDetails = {
        according_to: [{link: 'http://example.com/rationale'}],
        alternatives: [{id: 'other-feature'}],
      };
      const actual = element.renderDiscouragedNotice(discouragedDetails);
      render(actual, hostElement);
      const host = await fixture(hostElement);
      expect(host.textContent).to.contain('For the rationale, see:');
      expect(host.textContent).to.contain(
        'Consider using the following features instead:',
      );
      const links = host.querySelectorAll('a');
      expect(links.length).to.equal(2);
      expect(links[0].href).to.equal('http://example.com/rationale');
      expect(links[1].getAttribute('href')).to.equal('/features/other-feature');
    });
  });

  describe('redirects', () => {
    it('shows a redirect notice if the redirected_from URL parameter is present', async () => {
      const redirectedEl = await fixture<FeaturePage>(
        html`<webstatus-feature-page
          .location=${{
            params: {featureId: 'new-feature'},
            search: '?redirected_from=old-feature',
            pathname: '/features/new-feature',
          }}
        ></webstatus-feature-page>`,
      );
      await redirectedEl.updateComplete;

      const alert = redirectedEl.shadowRoot?.querySelector('sl-alert');
      expect(alert).to.not.be.null;
      // Normalize whitespace to avoid issues with formatting in the template literal.
      const text = alert?.textContent?.replace(/\s+/g, ' ').trim();
      expect(text).to.contain(
        'You have been redirected from an old feature ID (old-feature)',
      );
    });

    it('does not show a redirect notice if the URL parameter is not present', async () => {
      const alert = el.shadowRoot?.querySelector('sl-alert');
      expect(alert).to.be.null;
    });

    it('handleMovedFeature updates the history and component state', async () => {
      const pushStateSpy = sinon.spy(history, 'pushState');
      const newFeature = {
        feature_id: 'new-feature',
        name: 'New Feature',
        description: 'A new feature',
        browser_implementations: {},
        wpt: {},
      };
      const fakeError = new FeatureMovedError('foo', 'new-feature', newFeature);

      el.handleMovedFeature('old-feature', fakeError);

      expect(el.featureId).to.equal('new-feature');
      expect(el.oldFeatureId).to.equal('old-feature');
      expect(el.feature).to.deep.equal(newFeature);

      expect(pushStateSpy).to.have.been.calledWith(
        null,
        '',
        '/features/new-feature?redirected_from=old-feature',
      );

      const canonical = document.head.querySelector('link[rel="canonical"]');
      expect(canonical).to.not.be.null;
      expect(canonical?.getAttribute('href')).to.equal('/features/new-feature');
      expect(document.title).to.equal('New Feature');
    });
  });
});
