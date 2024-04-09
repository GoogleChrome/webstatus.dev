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

import {consume} from '@lit/context';
import {assert, fixture, html} from '@open-wc/testing';
import {LitElement, render} from 'lit';
import {customElement, property} from 'lit/decorators.js';
import {WebstatusGChartsLoader} from '../webstatus-gcharts-loader.js';
import {gchartsContext} from '../../contexts/gcharts-context.js';

describe('webstatus-gcharts-loader', () => {
  it('can be added to the page', async () => {
    const component = await fixture<WebstatusGChartsLoader>(
      html`<webstatus-gcharts-loader></webstatus-gcharts-loader>`
    );
    assert.exists(component);
    assert.equal(component.gchartsLibraryLoaded, true);
  });

  it('can have child components which are provided the load state via context', async () => {
    @customElement('fake-child-element')
    class FakeChildElement extends LitElement {
      @consume({context: gchartsContext, subscribe: true})
      @property({attribute: false})
      gchartsLibraryLoaded!: boolean;
    }

    const root = document.createElement('div');
    document.body.appendChild(root);
    render(
      html` <webstatus-gcharts-loader>
        <fake-child-element></fake-child-element>
      </webstatus-gcharts-loader>`,
      root
    );

    const component = root.querySelector(
      'webstatus-gcharts-loader'
    ) as WebstatusGChartsLoader;
    const childComponent = root.querySelector(
      'fake-child-element'
    ) as FakeChildElement;

    await component.updateComplete;
    await childComponent.updateComplete;

    assert.exists(component);
    assert.equal(component.gchartsLibraryLoaded, true);

    assert.exists(childComponent);
    assert.equal(childComponent.gchartsLibraryLoaded, true);
  });
});
