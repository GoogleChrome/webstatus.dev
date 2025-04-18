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

import {expect, fixture, html, assert} from '@open-wc/testing';
import {TaskTracker} from '../../utils/task-tracker.js';
import {type components} from 'webstatus.dev-backend';
import {ApiError} from '../../api/errors.js';
import {TaskStatus} from '@lit/task';
import {WebstatusOverviewTable} from '../webstatus-overview-table.js';
import '../webstatus-overview-table.js';
import {
  CurrentSavedSearch,
  SavedSearchScope,
} from '../../contexts/app-bookmark-info-context.js';

describe('webstatus-overview-table', () => {
  const orderedSavedSearch: CurrentSavedSearch = {
    scope: SavedSearchScope.GlobalSavedSearch,
    value: {
      name: 'Ordered Bookmark 1',
      query: 'name:test3 OR id:test1 OR id:test2',
      description: 'test description1',
      is_ordered: true,
    },
  };
  const defaultOrderSavedSearch: CurrentSavedSearch = {
    scope: SavedSearchScope.GlobalSavedSearch,
    value: {
      name: 'No order Bookmark 2',
      query: 'id:nothing',
      description: 'test description1',
      is_ordered: false,
    },
  };
  const page = {
    data: [
      {
        feature_id: 'test1',
        name: 'test1_feature',
      },
      {
        feature_id: 'test2',
        name: 'test2_feature',
      },
      {
        feature_id: 'test3',
        name: 'test3_featureA',
      },
      {
        feature_id: 'test4',
        name: 'test3_featureB',
      },
    ],
    metadata: {
      total: 4,
    },
  };
  const taskTracker: TaskTracker<
    components['schemas']['FeaturePage'],
    ApiError
  > = {
    status: TaskStatus.COMPLETE,
    error: undefined,
    data: page,
  };
  it('renders with no data', async () => {
    const location = {search: ''};
    const component = await fixture<WebstatusOverviewTable>(
      html`<webstatus-overview-table
        .location=${location}
      ></webstatus-overview-table>`,
    );
    assert.exists(component);
  });

  it('reorderByQueryTerms() sorts correctly', async () => {
    const location = {search: '?q=name:test3 OR id:test1 OR id:test2'};
    const component: WebstatusOverviewTable =
      await fixture<WebstatusOverviewTable>(
        html`<webstatus-overview-table
          .location=${location}
          .savedSearch=${orderedSavedSearch}
          .taskTracker=${taskTracker}
        ></webstatus-overview-table>`,
      );
    await component.updateComplete;
    assert.instanceOf(component, WebstatusOverviewTable);
    assert.exists(component);

    const sortedFeatures = component.reorderByQueryTerms();

    assert.exists(sortedFeatures);
    expect(sortedFeatures.length).to.equal(4);
    expect(sortedFeatures[0].feature_id).to.equal('test3');
    expect(sortedFeatures[1].feature_id).to.equal('test4');
    expect(sortedFeatures[2].feature_id).to.equal('test1');
    expect(sortedFeatures[3].feature_id).to.equal('test2');
  });

  it('reorderByQueryTerms() sorts correctly with DEFAULT_GLOBAL_SAVED_SEARCHES', async () => {
    const cssQuery =
      'id:anchor-positioning OR id:container-queries OR id:has OR id:nesting OR id:view-transitions OR id:subgrid OR id:grid OR name:scrollbar OR id:scroll-driven-animations OR id:scope';
    const cssSavedSearch: CurrentSavedSearch = {
      scope: SavedSearchScope.GlobalSavedSearch,
      value: {
        name: 'css',
        query: cssQuery,
        description: 'test description1',
        is_ordered: true,
      },
    };
    const cssPage = {
      data: [
        {
          feature_id: 'has',
          name: ':has()',
        },
        {
          feature_id: 'nesting',
          name: 'Nesting',
        },
        {
          feature_id: 'subgrid',
          name: 'Subgrid',
        },
        {
          feature_id: 'container-queries',
          name: 'Container queries',
        },
        {
          feature_id: 'grid',
          name: 'Grid',
        },
        {
          feature_id: 'anchor-positioning',
          name: 'Anchor positioning',
        },
        {
          feature_id: 'scope',
          name: '@scope',
        },
        {
          feature_id: 'scroll-driven-animations',
          name: 'Scroll-driven animations',
        },
        {
          feature_id: 'scrollbar-color',
          name: 'scrollbar-color',
        },
        {
          feature_id: 'scrollbar-gutter',
          name: 'scrollbar-gutter',
        },
        {
          feature_id: 'scrollbar-width',
          name: 'scrollbar-width',
        },
        {
          feature_id: 'view-transitions',
          name: 'View transitions',
        },
      ],
      metadata: {
        total: 12,
      },
    };
    const cssTaskTracker: TaskTracker<
      components['schemas']['FeaturePage'],
      ApiError
    > = {
      status: TaskStatus.COMPLETE,
      error: undefined,
      data: cssPage,
    };
    const location = {search: `?q=${cssQuery}`};
    const component: WebstatusOverviewTable =
      await fixture<WebstatusOverviewTable>(
        html`<webstatus-overview-table
          .location=${location}
          .savedSearch=${cssSavedSearch}
          .taskTracker=${cssTaskTracker}
        ></webstatus-overview-table>`,
      );
    await component.updateComplete;
    assert.instanceOf(component, WebstatusOverviewTable);
    assert.exists(component);

    const sortedFeatures = component.reorderByQueryTerms();

    assert.exists(sortedFeatures);
    expect(sortedFeatures.length).to.equal(12);
    expect(sortedFeatures[0].feature_id).to.equal('anchor-positioning');
    expect(sortedFeatures[1].feature_id).to.equal('container-queries');
    expect(sortedFeatures[2].feature_id).to.equal('has');
    expect(sortedFeatures[3].feature_id).to.equal('nesting');
    expect(sortedFeatures[4].feature_id).to.equal('view-transitions');
    expect(sortedFeatures[5].feature_id).to.equal('subgrid');
    expect(sortedFeatures[6].feature_id).to.equal('grid');
    expect(sortedFeatures[7].feature_id).to.equal('scrollbar-color');
    expect(sortedFeatures[8].feature_id).to.equal('scrollbar-gutter');
    expect(sortedFeatures[9].feature_id).to.equal('scrollbar-width');
    expect(sortedFeatures[10].feature_id).to.equal('scroll-driven-animations');
    expect(sortedFeatures[11].feature_id).to.equal('scope');
  });

  it('reorderByQueryTerms() return undefined when query is not ordered', async () => {
    const location = {search: 'id:nothing'};
    const component = await fixture<WebstatusOverviewTable>(
      html`<webstatus-overview-table
        .location=${location}
        .savedSearch=${defaultOrderSavedSearch}
        .taskTracker=${taskTracker}
      ></webstatus-overview-table>`,
    );
    await component.updateComplete;
    assert.instanceOf(component, WebstatusOverviewTable);
    assert.exists(component);

    const sortedFeatures = component.reorderByQueryTerms();

    assert.notExists(sortedFeatures);
  });
});
