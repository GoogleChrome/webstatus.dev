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

import {parseColumnsSpec} from '../webstatus-overview-cells.js';

describe('parseColumnsSpec', () => {
  it('returns empty array when there was no column spec', () => {
    const cols = parseColumnsSpec('');
    assert.deepEqual(cols, []);
  });

  it('returns an array when given a column spec', () => {
    const cols = parseColumnsSpec('name, baseline_status ');
    assert.deepEqual(cols, ['name', 'baseline_status']);
  });
});
