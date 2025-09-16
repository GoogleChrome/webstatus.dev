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
import '../webstatus-feature-gone-split-page.js';
import {WebstatusFeatureGoneSplitPage} from '../webstatus-feature-gone-split-page.js';
import {Task} from '@lit/task';
import {APIClient} from '../../contexts/api-client-context.js';
import {GITHUB_REPO_ISSUE_LINK} from '../../utils/constants.js';

type NewFeature = {name: string; url: string};

describe('webstatus-feature-gone-split-page', () => {
  const newFeatureIds = 'feature1,feature2';
  const mockNewFeatures: NewFeature[] = [
    {name: 'Feature One', url: '/features/feature1'},
    {name: 'Feature Two', url: '/features/feature2'},
  ];

  it('renders the correct error message', async () => {
    const component = await fixture<WebstatusFeatureGoneSplitPage>(
      html`<webstatus-feature-gone-split-page
        .location=${{search: ''}}
      ></webstatus-feature-gone-split-page>`,
    );

    expect(
      component.shadowRoot
        ?.querySelector('#error-status-code')
        ?.textContent?.trim(),
    ).to.equal('410');

    expect(
      component.shadowRoot
        ?.querySelector('#error-headline')
        ?.textContent?.trim(),
    ).to.equal('Feature Gone');

    expect(
      component.shadowRoot
        ?.querySelector('#error-detailed-message .error-message')
        ?.textContent?.trim(),
    ).to.equal('This feature has been split into multiple new features.');
  });

  it('displays "Loading new features..." when the API request is pending', async () => {
    const component = await createComponentWithMockedNewFeatures(
      newFeatureIds,
      [],
      {stayPending: true},
    );

    const loadingMessage =
      component.shadowRoot?.querySelector('.loading-message');
    expect(loadingMessage).to.exist;
    expect(loadingMessage?.textContent?.trim()).to.equal(
      'Loading new features...',
    );
  });

  it('renders new features when API returns results', async () => {
    const component = await createComponentWithMockedNewFeatures(
      newFeatureIds,
      mockNewFeatures,
    );

    const featureList =
      component.shadowRoot?.querySelectorAll('.feature-list li');
    expect(featureList?.length).to.equal(2);
    expect(featureList?.[0]?.textContent?.trim()).to.equal('Feature One');
    expect(featureList?.[1]?.textContent?.trim()).to.equal('Feature Two');
  });

  it('renders action buttons', async () => {
    const component = await fixture<WebstatusFeatureGoneSplitPage>(html`
      <webstatus-feature-gone-split-page
        .location=${{search: ''}}
      ></webstatus-feature-gone-split-page>
    `);

    expect(component.shadowRoot?.querySelector('#error-action-home-btn')).to
      .exist;
    expect(component.shadowRoot?.querySelector('#error-action-report')).to
      .exist;
  });

  it('report issue button links to GitHub', async () => {
    const component = await fixture<WebstatusFeatureGoneSplitPage>(html`
      <webstatus-feature-gone-split-page
        .location=${{search: ''}}
      ></webstatus-feature-gone-split-page>
    `);

    const reportButton = component.shadowRoot?.querySelector(
      '#error-action-report',
    );
    expect(reportButton?.getAttribute('href')).to.equal(GITHUB_REPO_ISSUE_LINK);
  });

  async function createComponentWithMockedNewFeatures(
    newFeatureIds: string,
    mockData: NewFeature[],
    options: {stayPending?: boolean} = {},
  ): Promise<WebstatusFeatureGoneSplitPage> {
    const component = await fixture<WebstatusFeatureGoneSplitPage>(html`
      <webstatus-feature-gone-split-page
        .location=${{search: `?new_features=${newFeatureIds}`}}
      ></webstatus-feature-gone-split-page>
    `);

    component._newFeatures = new Task<[APIClient, string], NewFeature[]>(
      component,
      {
        args: () => [undefined as unknown as APIClient, newFeatureIds],
        task: async () => {
          if (options.stayPending) return new Promise(() => {});
          return mockData;
        },
      },
    );

    component._newFeatures.run();
    await component.updateComplete;
    return component;
  }
});
