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
});

describe('formatFeaturePageUrl', () => {
  const feature: components['schemas']['Feature'] = {
    feature_id: 'grid',
    name: 'test feature',
    baseline_status: 'none',
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
});
