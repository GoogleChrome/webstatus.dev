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

// See https://github.com/DefinitelyTyped/DefinitelyTyped/blob/master/types/google.visualization/index.d.ts
/// <reference types="@types/google.visualization" />

import {consume} from '@lit/context';
import {assert, fixture, html} from '@open-wc/testing';
import {LitElement} from 'lit';
import {customElement, property} from 'lit/decorators.js';
import '../webstatus-gcharts-loader.js';
import {type WebstatusGChartsLoader} from '../webstatus-gcharts-loader.js';
import {gchartsContext} from '../../contexts/gcharts-context.js';

describe('webstatus-gcharts-loader', () => {

  it('can be added to the page via fixture', async () => {
    // console.log('1 does console.log work?');
    const component = await fixture<WebstatusGChartsLoader>(
      html`<webstatus-gcharts-loader> </webstatus-gcharts-loader>`
    );
    assert.exists(component);
    await component.updateComplete;

    await component.loadGoogleChartsLoaderAndPackages().then(() => {
      assert.exists(component.gchartsLibraryLoaded);
      assert.equal(component.gchartsLibraryLoaded, true);
    })
  });

  it('can provide child components the loaded state via context', async () => {
    @customElement('fake-child-element')
    class FakeChildElement extends LitElement {
      @consume({context: gchartsContext, subscribe: true})
      @property({attribute: false})
      gchartsLibraryLoaded!: boolean;
    }

    const component = await fixture<WebstatusGChartsLoader>(
      html`<webstatus-gcharts-loader>
        <fake-child-element></fake-child-element>
      </webstatus-gcharts-loader>`
    );

    assert.exists(component);
    await component.updateComplete;

    const childComponent = component.querySelector(
      'fake-child-element'
    ) as FakeChildElement;
    assert.exists(childComponent);
    await childComponent.updateComplete;

    assert.equal(component.gchartsLibraryLoaded, true);
    assert.equal(childComponent.gchartsLibraryLoaded, true);
  });
});
