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

import {assert, fixture} from '@open-wc/testing';

import {WebstatusOverviewPagination} from '../webstatus-overview-pagination.js';

describe('webstatus-overview-pagination', () => {
  let el: WebstatusOverviewPagination;

  beforeEach(async () => {
    el = await fixture<WebstatusOverviewPagination>(
      '<webstatus-overview-pagination></webstatus-overview-pagination>',
    );
    el.totalCount = 123;
    el.location = {search: ''};
    await el.updateComplete;
  });

  it('can be added to the page', async () => {
    assert.exists(el);
    assert.instanceOf(el, WebstatusOverviewPagination);
  });

  it('renders navigation buttons', async () => {
    const previous = el.shadowRoot!.querySelector('#previous');
    assert.exists(previous);

    const jump_1 = el.shadowRoot!.querySelector('#jump_1');
    assert.exists(jump_1);
    const jump_2 = el.shadowRoot!.querySelector('#jump_2');
    assert.exists(jump_2);
    const jump_3 = el.shadowRoot!.querySelector('#jump_3');
    assert.exists(jump_3);
    const jump_4 = el.shadowRoot!.querySelector('#jump_4');
    assert.exists(jump_4);
    const jump_5 = el.shadowRoot!.querySelector('#jump_5');
    assert.exists(jump_5);
    const jump_6 = el.shadowRoot!.querySelector('#jump_6');
    assert.isNull(jump_6);

    const next = el.shadowRoot!.querySelector('#next');
    assert.exists(next);
  });

  it('renders skips distant navigation buttons', async () => {
    el.location = {search: 'start=20&num=2'};
    el.totalCount = 40;
    await el.updateComplete;

    const previous = el.shadowRoot!.querySelector('#previous');
    assert.exists(previous);

    const jump_1 = el.shadowRoot!.querySelector('#jump_1');
    assert.exists(jump_1);
    const jump_2 = el.shadowRoot!.querySelector('#jump_2');
    assert.notExists(jump_2);
    const jump_3 = el.shadowRoot!.querySelector('#jump_3');
    assert.notExists(jump_3);
    const jump_7 = el.shadowRoot!.querySelector('#jump_7');
    assert.exists(jump_7);
    const jump_11 = el.shadowRoot!.querySelector('#jump_11');
    assert.exists(jump_11);
    const jump_15 = el.shadowRoot!.querySelector('#jump_15');
    assert.exists(jump_15);
    const jump_19 = el.shadowRoot!.querySelector('#jump_19');
    assert.notExists(jump_19);
    const jump_20 = el.shadowRoot!.querySelector('#jump_20');
    assert.exists(jump_20);

    const next = el.shadowRoot!.querySelector('#next');
    assert.exists(next);
  });

  it('renders standard options for items-per-page', async () => {
    const opt_25 = el.shadowRoot!.querySelector('#opt_25');
    assert.exists(opt_25);
    const opt_50 = el.shadowRoot!.querySelector('#opt_50');
    assert.exists(opt_50);
    const opt_77 = el.shadowRoot!.querySelector('#opt_77');
    assert.notExists(opt_77);
    const opt_100 = el.shadowRoot!.querySelector('#opt_100');
    assert.exists(opt_100);
  });

  it('renders custom options for items-per-page', async () => {
    el.location = {search: 'num=77'};
    await el.updateComplete;
    const opt_25 = el.shadowRoot!.querySelector('#opt_25');
    assert.exists(opt_25);
    const opt_50 = el.shadowRoot!.querySelector('#opt_50');
    assert.exists(opt_50);
    const opt_77 = el.shadowRoot!.querySelector('#opt_77');
    assert.exists(opt_77);
    const opt_100 = el.shadowRoot!.querySelector('#opt_100');
    assert.exists(opt_100);
  });
});
