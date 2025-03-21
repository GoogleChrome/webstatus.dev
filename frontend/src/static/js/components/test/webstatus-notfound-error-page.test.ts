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

import {expect, fixture, html} from '@open-wc/testing';
import '../webstatus-not-found-error-page.js';
import {WebstatusNotFoundErrorPage} from '../webstatus-notfound-error-page.js';

const GITHUB_REPO_ISSUE_LINK = 'https://github.com/example/repo/issues';

describe('webstatus-not-found-error-page', () => {
  it('renders the correct error message when featureId is missing', async () => {
    const component = await fixture<WebstatusNotFoundErrorPage>(
      html`<webstatus-not-found-error-page></webstatus-not-found-error-page>`,
    );

    expect(
      component.shadowRoot
        ?.querySelector('#error-status-code')
        ?.textContent?.trim(),
    ).to.equal('404');

    expect(
      component.shadowRoot
        ?.querySelector('#error-headline')
        ?.textContent?.trim(),
    ).to.equal('Page not found');

    expect(
      component.shadowRoot
        ?.querySelector('#error-detailed-message .error-message')
        ?.textContent?.trim(),
    ).to.equal("We couldn't find the page you're looking for.");
  });

  it('renders the correct error message when featureId is provided', async () => {
    const component = await fixture<WebstatusNotFoundErrorPage>(html`
      <webstatus-not-found-error-page
        .location=${{search: '?q=test-feature'}}
      ></webstatus-not-found-error-page>
    `);

    expect(
      component.shadowRoot?.querySelector('#error-detailed-message')
        ?.textContent,
    ).to.include('We could not find Feature ID: test-feature');
  });

  it('displays "Loading similar features..." when the API request is pending', async () => {
    const component = await fixture<WebstatusNotFoundErrorPage>(html`
      <webstatus-not-found-error-page
        .location=${{search: '?q=test-feature'}}
      ></webstatus-not-found-error-page>
    `);

    component._loadingSimilarResults = {status: 'pending'} as any;
    await component.updateComplete;

    const loadingMessage =
      component.shadowRoot?.querySelector('.loading-message');
    expect(loadingMessage).to.exist;
    expect(loadingMessage?.textContent?.trim()).to.equal(
      'Loading similar features...',
    );
  });

  it('renders similar features when API returns results', async () => {
    const component = await fixture<WebstatusNotFoundErrorPage>(html`
      <webstatus-not-found-error-page
        .location=${{search: '?q=test-feature'}}
      ></webstatus-not-found-error-page>
    `);

    component.similarFeatures = [
      {name: 'Feature One', url: '/features/one'},
      {name: 'Feature Two', url: '/features/two'},
    ];
    await component.updateComplete;

    const featureList =
      component.shadowRoot?.querySelectorAll('.feature-list li');
    expect(featureList?.length).to.equal(2);
    expect(featureList?.[0]?.textContent?.trim()).to.equal('Feature One');
    expect(featureList?.[1]?.textContent?.trim()).to.equal('Feature Two');
  });

  it('renders "No similar features found." when API returns no results', async () => {
    const component = await fixture<WebstatusNotFoundErrorPage>(html`
      <webstatus-not-found-error-page
        .location=${{search: '?q=test-feature'}}
      ></webstatus-not-found-error-page>
    `);

    component.similarFeatures = [];
    await component.updateComplete;

    const noResultsMessage = component.shadowRoot?.querySelector(
      '.similar-features-container p',
    );
    expect(noResultsMessage).to.exist;
    expect(noResultsMessage?.textContent?.trim()).to.equal(
      'No similar features found.',
    );
  });

  it('renders all three buttons when featureId exists', async () => {
    const component = await fixture<WebstatusNotFoundErrorPage>(html`
      <webstatus-not-found-error-page
        .location=${{search: '?q=test-feature'}}
      ></webstatus-not-found-error-page>
    `);

    expect(component.shadowRoot?.querySelector('#error-action-search-btn')).to
      .exist;
    expect(component.shadowRoot?.querySelector('#error-action-home-btn')).to
      .exist;
    expect(component.shadowRoot?.querySelector('#error-action-report')).to
      .exist;
  });

  it('renders only two buttons when featureId does not exist', async () => {
    const component = await fixture<WebstatusNotFoundErrorPage>(html`
      <webstatus-not-found-error-page
        .location=${{search: ''}}
      ></webstatus-not-found-error-page>
    `);

    expect(component.shadowRoot?.querySelector('#error-action-search-btn')).to
      .not.exist;
    expect(component.shadowRoot?.querySelector('#error-action-home-btn')).to
      .exist;
    expect(component.shadowRoot?.querySelector('#error-action-report')).to
      .exist;
  });

  it('search button contains the correct query parameter', async () => {
    const component = await fixture<WebstatusNotFoundErrorPage>(html`
      <webstatus-not-found-error-page
        .location=${{search: '?q=correct-query'}}
      ></webstatus-not-found-error-page>
    `);

    const searchButton = component.shadowRoot?.querySelector(
      '#error-action-search-btn',
    );
    expect(searchButton?.getAttribute('href')).to.equal('/?q=correct-query');
  });

  it('report issue button links to GitHub', async () => {
    const component = await fixture<WebstatusNotFoundErrorPage>(html`
      <webstatus-not-found-error-page></webstatus-not-found-error-page>
    `);

    const reportButton = component.shadowRoot?.querySelector(
      '#error-action-report',
    );
    expect(reportButton?.getAttribute('href')).to.equal(GITHUB_REPO_ISSUE_LINK);
  });

  it('applies correct gap spacing in error-actions when featureId is present', async () => {
    const component = await fixture<WebstatusNotFoundErrorPage>(html`
      <webstatus-not-found-error-page
        .location=${{search: '?q=test-feature'}}
      ></webstatus-not-found-error-page>
    `);

    const errorActions = component.shadowRoot?.querySelector('#error-actions');
    expect(errorActions?.getAttribute('style')).to.include('gap: 32px');
  });

  it('applies correct gap spacing in error-actions when featureId is absent', async () => {
    const component = await fixture<WebstatusNotFoundErrorPage>(html`
      <webstatus-not-found-error-page
        .location=${{search: ''}}
      ></webstatus-not-found-error-page>
    `);

    const errorActions = component.shadowRoot?.querySelector('#error-actions');
    expect(errorActions?.getAttribute('style')).to.include('gap: 16px');
  });
});
