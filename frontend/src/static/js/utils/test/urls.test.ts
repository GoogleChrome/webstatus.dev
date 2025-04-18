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

import {assert} from '@open-wc/testing';
import {type components} from 'webstatus.dev-backend';

import {
  getSearchQuery,
  formatOverviewPageUrl,
  formatFeaturePageUrl,
  getWPTMetricView,
  getColumnOptions,
  getSearchID,
  getEditSavedSearch,
} from '../urls.js';

describe('getSearchQuery', () => {
  it('returns empty string when there was no q= param', () => {
    const q = getSearchQuery({search: ''});
    assert.equal(q, '');
  });

  it('returns empty string when the q= param has no value', () => {
    const q = getSearchQuery({search: '?q='});
    assert.equal(q, '');
  });

  it('returns the string when the q= param was set', () => {
    const q = getSearchQuery({search: '?q=memory%20leak'});
    assert.equal(q, 'memory leak');
  });
});

describe('getColumnsSpec', () => {
  it('returns empty string when there was no columns= param', () => {
    const cs = getSearchQuery({search: ''});
    assert.equal(cs, '');
  });

  it('returns empty string when the columns= param has no value', () => {
    const cs = getSearchQuery({search: '?columns='});
    assert.equal(cs, '');
  });

  it('returns the string when the columns= param was set', () => {
    const cs = getSearchQuery({search: '?q=name, baseline_stats '});
    assert.equal(cs, 'name, baseline_stats ');
  });
});

describe('getColumnOptions', () => {
  it('returns empty string when there was no column_options= param', () => {
    const cs = getColumnOptions({search: ''});
    assert.equal(cs, '');
  });

  it('returns empty string when the column_options= param has no value', () => {
    const cs = getColumnOptions({search: '?column_options='});
    assert.equal(cs, '');
  });

  it('returns the string when the column_options= param was set', () => {
    const cs = getColumnOptions({
      search: '?column_options=baseline_stats_high_date',
    });
    assert.equal(cs, 'baseline_stats_high_date');
  });
});

describe('getSortSpec', () => {
  it('returns empty string when there was no sort= param', () => {
    const cs = getSearchQuery({search: ''});
    assert.equal(cs, '');
  });

  it('returns empty string when the sort= param has no value', () => {
    const cs = getSearchQuery({search: '?sort='});
    assert.equal(cs, '');
  });

  it('returns the string when the sort= param was set', () => {
    const cs = getSearchQuery({search: '?q=name, baseline_stats '});
    assert.equal(cs, 'name, baseline_stats ');
  });
});

describe('getWPTMetricView', () => {
  it('returns empty string when there was no wpt_metric_view= param', () => {
    const cs = getWPTMetricView({search: ''});
    assert.equal(cs, '');
  });

  it('returns empty string when the wpt_metric_view= param has no value', () => {
    const cs = getWPTMetricView({search: '?wpt_metric_view='});
    assert.equal(cs, '');
  });

  it('returns the string when the wpt_metric_view= param was set', () => {
    const cs = getWPTMetricView({search: '?wpt_metric_view=subtest_counts'});
    assert.equal(cs, 'subtest_counts');
  });
});

describe('getSearchID', () => {
  it('returns empty string when there was no search_id= param', () => {
    const cs = getSearchID({search: ''});
    assert.equal(cs, '');
  });

  it('returns empty string when the search_id= param has no value', () => {
    const cs = getSearchID({search: '?search_id='});
    assert.equal(cs, '');
  });

  it('returns the string when the search_id= param was set', () => {
    const cs = getSearchID({search: '?search_id=fake_uuid'});
    assert.equal(cs, 'fake_uuid');
  });
});

describe('getEditSavedSearch', () => {
  it('returns false when there was no edit_saved_search= param', () => {
    const cs = getEditSavedSearch({search: ''});
    assert.equal(cs, false);
  });

  it('returns false when the edit_saved_search= param has no value', () => {
    const cs = getEditSavedSearch({search: '?edit_saved_search='});
    assert.equal(cs, false);
  });

  it('returns the string when the edit_saved_search= param was set', () => {
    const cs = getEditSavedSearch({search: '?edit_saved_search=true'});
    assert.equal(cs, true);
  });
});

describe('formatOverviewPageUrl', () => {
  it('returns a plain URL when no location is passed', () => {
    const url = formatOverviewPageUrl();
    assert.equal(url, '/');
  });

  it('returns a plain URL when no navigational params were set', () => {
    const url = formatOverviewPageUrl({search: ''});
    assert.equal(url, '/');
  });

  it('returns a URL with navigational params when they are set', () => {
    const url = formatOverviewPageUrl({search: '?q=css'});
    assert.equal(url, '/?q=css');
  });

  it('returns a URL with navigational params when wpt_metric_view param is set', () => {
    const url = formatOverviewPageUrl({
      search: '?wpt_metric_view=subtest_counts',
    });
    assert.equal(url, '/?wpt_metric_view=subtest_counts');
  });

  it('returns a URL with overrideparameters set', () => {
    const url = formatOverviewPageUrl({search: ''}, {q: 'memory'});
    assert.equal(url, '/?q=memory');
  });

  it('returns a URL with existing parameters overridden', () => {
    const url = formatOverviewPageUrl({search: '?q=css'}, {q: 'memory'});
    assert.equal(url, '/?q=memory');
  });

  it('can add the column spec parameter', () => {
    const url = formatOverviewPageUrl(
      {search: '?q=css'},
      {columns: ['name', 'baseline_stats']},
    );
    assert.equal(url, '/?q=css&columns=name%2Cbaseline_stats');
  });

  it('can override and existing column spec parameter', () => {
    const url = formatOverviewPageUrl(
      {search: '?q=css&columns=name'},
      {columns: ['name', 'baseline_stats']},
    );
    assert.equal(url, '/?q=css&columns=name%2Cbaseline_stats');
  });

  it('can clear the column spec parameter', () => {
    const url = formatOverviewPageUrl(
      {search: '?q=css&columns=name'},
      {columns: []},
    );
    assert.equal(url, '/?q=css');
  });

  it('can override the column_options parameter', () => {
    const url = formatOverviewPageUrl(
      {search: '?q=css&column_options=baseline_stats_high_date'},
      {column_options: ['baseline_stats_high_date']},
    );
    assert.equal(url, '/?q=css&column_options=baseline_stats_high_date');
  });

  it('can clear the column_options parameter', () => {
    const url = formatOverviewPageUrl(
      {search: '?q=css&column_options=baseline_stats_high_date'},
      {column_options: []},
    );
    assert.equal(url, '/?q=css');
  });

  it('returns a URL with navigational params when search_id param is set', () => {
    const url = formatOverviewPageUrl({search: ''}, {search_id: 'fake_uuid'});
    assert.equal(url, '/?search_id=fake_uuid');
  });

  it('returns a URL with navigational params when edit_saved_search param is set', () => {
    const url = formatOverviewPageUrl({search: ''}, {edit_saved_search: true});
    assert.equal(url, '/?edit_saved_search=true');
  });
});

describe('formatFeaturePageUrl', () => {
  const feature: components['schemas']['Feature'] = {
    feature_id: 'grid',
    name: 'test feature',
    baseline: {
      status: 'limited',
    },
  };
  it('returns a plain URL when no location is passed', () => {
    const url = formatFeaturePageUrl(feature);
    assert.equal(url, '/features/grid');
  });

  it('returns a plain URL when no navigational params were set', () => {
    const url = formatFeaturePageUrl(feature, {search: ''});
    assert.equal(url, '/features/grid');
  });

  it('returns a URL with navigational params when they are set', () => {
    const url = formatFeaturePageUrl(feature, {search: '?q=css'});
    assert.equal(url, '/features/grid?q=css');
  });

  it('returns a URL with navigational params when wpt_metric_view param is set', () => {
    const url = formatFeaturePageUrl(feature, {
      search: '?wpt_metric_view=subtest_counts',
    });
    assert.equal(url, '/features/grid?wpt_metric_view=subtest_counts');
  });
});
