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

import {
  ColumnKey,
  parseColumnsSpec,
  DEFAULT_COLUMNS,
  isJavaScriptFeature,
  didFeatureCrash,
} from '../webstatus-overview-cells.js';

describe('parseColumnsSpec', () => {
  it('returns default columns when there was no column spec', () => {
    const cols = parseColumnsSpec('');
    assert.deepEqual(cols, DEFAULT_COLUMNS);
  });

  it('returns an array when given a column spec', () => {
    const cols = parseColumnsSpec('name, baseline_status ');
    assert.deepEqual(cols, [ColumnKey.Name, ColumnKey.BaselineStatus]);
  });
});

describe('isJavaScriptFeature', () => {
  it('returns true for a JavaScript feature (link prefix match)', () => {
    const featureSpecInfo = {
      links: [{link: 'https://tc39.es/proposal-temporal'}],
    };
    assert.isTrue(isJavaScriptFeature(featureSpecInfo));
  });

  it('returns false for a non-JavaScript feature (no link match)', () => {
    const featureSpecInfo = {
      links: [{link: 'https://www.w3.org/TR/webgpu/'}],
    };
    assert.isFalse(isJavaScriptFeature(featureSpecInfo));
  });

  it('returns false if links are missing', () => {
    const featureSpecInfo = {}; // No 'links' property
    assert.isFalse(isJavaScriptFeature(featureSpecInfo));
  });
});

describe('didFeatureCrash', () => {
  it('returns true if metadata contains a mapping of "status" to "C"', () => {
    const metadata = {
      status: 'C',
    };
    assert.isTrue(didFeatureCrash(metadata));
  });

  it('returns false for other status metadata', () => {
    const metadata = {
      status: 'hi',
    };
    assert.isFalse(didFeatureCrash(metadata));
  });

  it('returns false for no metadata', () => {
    const metadata = {};
    assert.isFalse(didFeatureCrash(metadata));
  });

  it('returns false for undefined metadata', () => {
    const metadata = undefined;
    assert.isFalse(didFeatureCrash(metadata));
  });
});
