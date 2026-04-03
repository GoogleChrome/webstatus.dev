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
import {fixture, html, assert} from '@open-wc/testing';
import {WebstatusOverviewDataLoader} from '../webstatus-overview-data-loader.js';

describe('webstatus-overview-data-loader', () => {
  it('instantiates correctly', async () => {
    const component: WebstatusOverviewDataLoader =
      await fixture<WebstatusOverviewDataLoader>(
        html`<webstatus-overview-data-loader
          .location=${{search: ''}}
        ></webstatus-overview-data-loader>`,
      );
    assert.instanceOf(component, WebstatusOverviewDataLoader);
    assert.exists(component);
  });
});
