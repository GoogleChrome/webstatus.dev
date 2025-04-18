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
import '../webstatus-notfound-error-page.js';
import {WebstatusNotFoundErrorPage} from '../webstatus-notfound-error-page.js';
import {Task} from '@lit/task';
import {APIClient} from '../../contexts/api-client-context.js';
import {GITHUB_REPO_ISSUE_LINK} from '../../utils/constants.js';

type SimilarFeature = {name: string; url: string};

describe('webstatus-notfound-error-page', () => {
  const featureIdWithMockResults = 'g';
  const mockSimilarFeatures: SimilarFeature[] = [
    {name: 'Feature One', url: '/features/dignissimos44'},
    {name: 'Feature Two', url: '/features/fugiat37'},
  ];

  it('renders the correct error message when featureId is missing', async () => {
    const component = await fixture<WebstatusNotFoundErrorPage>(
      html`<webstatus-notfound-error-page
        .location=${{search: ''}}
      ></webstatus-notfound-error-page>`,
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

  it('renders correct message when featureId is provided', async () => {
    const component = await fixture<WebstatusNotFoundErrorPage>(html`
      <webstatus-notfound-error-page
        .location=${{search: '?q=test-feature'}}
      ></webstatus-notfound-error-page>
    `);

    expect(
      component.shadowRoot?.querySelector('#error-detailed-message')
        ?.textContent,
    ).to.include('We could not find Feature ID: test-feature');
  });

  it('displays "Loading similar features..." when the API request is pending', async () => {
    const component = await createComponentWithMockedSimilarFeatures(
      'test-feature',
      [],
      {stayPending: true},
    );

    const loadingMessage =
      component.shadowRoot?.querySelector('.loading-message');
    expect(loadingMessage).to.exist;
    expect(loadingMessage?.textContent?.trim()).to.equal(
      'Loading similar features...',
    );
  });

  it('renders similar features when API returns results', async () => {
    const component = await createComponentWithMockedSimilarFeatures(
      featureIdWithMockResults,
      mockSimilarFeatures,
    );

    const featureList =
      component.shadowRoot?.querySelectorAll('.feature-list li');
    expect(featureList?.length).to.equal(2);
    expect(featureList?.[0]?.textContent?.trim()).to.equal('Feature One');
    expect(featureList?.[1]?.textContent?.trim()).to.equal('Feature Two');
  });

  it('renders only two buttons when featureId does not exist', async () => {
    const component = await fixture<WebstatusNotFoundErrorPage>(html`
      <webstatus-notfound-error-page
        .location=${{search: ''}}
      ></webstatus-notfound-error-page>
    `);

    expect(component.shadowRoot?.querySelector('#error-action-search-btn')).to
      .not.exist;
    expect(component.shadowRoot?.querySelector('#error-action-home-btn')).to
      .exist;
    expect(component.shadowRoot?.querySelector('#error-action-report')).to
      .exist;
  });

  it('renders all three buttons when featureId and similar results exist', async () => {
    const component = await createComponentWithMockedSimilarFeatures(
      featureIdWithMockResults,
      mockSimilarFeatures,
    );

    expect(component.shadowRoot?.querySelector('#error-action-search-btn')).to
      .exist;
    expect(component.shadowRoot?.querySelector('#error-action-home-btn')).to
      .exist;
    expect(component.shadowRoot?.querySelector('#error-action-report')).to
      .exist;
  });

  it('search button contains the correct query parameter when similar results exist', async () => {
    const component = await createComponentWithMockedSimilarFeatures(
      featureIdWithMockResults,
      mockSimilarFeatures,
    );

    const searchButton = component.shadowRoot?.querySelector(
      '#error-action-search-btn',
    );
    expect(searchButton?.getAttribute('href')).to.equal(
      `/?q=${featureIdWithMockResults}`,
    );
  });

  it('report issue button links to GitHub', async () => {
    const component = await fixture<WebstatusNotFoundErrorPage>(html`
      <webstatus-notfound-error-page
        .location=${{search: ''}}
      ></webstatus-notfound-error-page>
    `);

    const reportButton = component.shadowRoot?.querySelector(
      '#error-action-report',
    );
    expect(reportButton?.getAttribute('href')).to.equal(GITHUB_REPO_ISSUE_LINK);
  });

  async function createComponentWithMockedSimilarFeatures(
    featureId: string,
    mockData: SimilarFeature[],
    options: {stayPending?: boolean} = {},
  ): Promise<WebstatusNotFoundErrorPage> {
    const component = await fixture<WebstatusNotFoundErrorPage>(html`
      <webstatus-notfound-error-page
        .location=${{search: `?q=${featureId}`}}
      ></webstatus-notfound-error-page>
    `);

    component._similarResults = new Task<[APIClient, string], SimilarFeature[]>(
      component,
      {
        args: () => [undefined as unknown as APIClient, featureId],
        task: async () => {
          if (options.stayPending) return new Promise(() => {});
          return mockData;
        },
      },
    );

    component._similarResults.run();
    await component.updateComplete;
    return component;
  }
});
