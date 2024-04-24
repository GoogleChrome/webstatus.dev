/**
 * Copyright 2023 Google LLC
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

import {assert, fixture, html} from '@open-wc/testing';
import '../webstatus-gcharts-loader.js';
import '../webstatus-gchart.js';
import {type WebstatusGChartsLoader} from '../webstatus-gcharts-loader.js';
import {type WebstatusGChart} from '../webstatus-gchart.js';
import {render} from 'lit';

describe('webstatus-gchart', () => {
  it('can receive loaded state via loader context', async () => {
    const component = await fixture<WebstatusGChartsLoader>(
      html`<webstatus-gcharts-loader>
        <webstatus-gchart></webstatus-gchart>
      </webstatus-gcharts-loader>`
    );

    assert.exists(component);
    await component.updateComplete;
    await component.waitForGChartsLibraryLoaded();

    const childComponent = component.querySelector(
      'webstatus-gchart'
    ) as WebstatusGChart;
    assert.exists(childComponent);
    await childComponent.updateComplete;

    assert.equal(component.gchartsLibraryLoaded, true);
    assert.equal(childComponent.gchartsLibraryLoaded, true);
  });

  it('can subscribe to the parent gchart loader', async () => {
    // This also tests adding components via lit render.
    const root = document.createElement('div');
    document.body.appendChild(root);
    render(
      html`
        <webstatus-gcharts-loader>
          <webstatus-gchart></webstatus-gchart>
        </webstatus-gcharts-loader>
      `,
      root
    );
    const loader = root.querySelector(
      'webstatus-gcharts-loader'
    ) as WebstatusGChartsLoader;
    assert.exists(loader);
    await loader.updateComplete;
    await loader.waitForGChartsLibraryLoaded();

    const gchart = root.querySelector('webstatus-gchart') as WebstatusGChart;
    await gchart.updateComplete;

    // Wait for both loader and gchart to have the library loaded
    await new Promise<void>(resolve => {
      const intervalId = setInterval(() => {
        if (loader.gchartsLibraryLoaded && gchart.gchartsLibraryLoaded) {
          clearInterval(intervalId);
          resolve();
        }
      }, 10);
    });

    assert.equal(loader.gchartsLibraryLoaded, true);
    assert.equal(gchart.gchartsLibraryLoaded, true);
  });
});
