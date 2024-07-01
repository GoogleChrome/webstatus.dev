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

import {html} from 'lit';
import {assert, fixture} from '@open-wc/testing';
import {
  WebstatusTypeahead,
  WebstatusTypeaheadDropdown,
  WebstatusTypeaheadItem,
} from '../webstatus-typeahead.js';
import {SlInput, SlMenuItem} from '@shoelace-style/shoelace';

describe('webstatus-typeahead', () => {
  it('reflects the value of the sl-input as its own value', async () => {
    const component = await fixture<WebstatusTypeahead>(html`
      <webstatus-typeahead> </webstatus-typeahead>
    `);
    assert.exists(component);
    assert.instanceOf(component, WebstatusTypeahead);
    const slInput = component.shadowRoot!.querySelector('sl-input') as SlInput;

    slInput.value = 'test value';
    component.reflectValue();
    assert.equal('test value', component.value);
  });

  it('has a menu that can show and hide', async () => {
    const component = await fixture<WebstatusTypeahead>(html`
      <webstatus-typeahead>
        <webstatus-typeahead-item
          id="item0"
          value="aaa"
          doc="Docs about aaa"
        ></webstatus-typeahead-item>
      </webstatus-typeahead>
    `);
    assert.exists(component);
    assert.instanceOf(component, WebstatusTypeahead);
    const dropdown = component.shadowRoot!.querySelector(
      'webstatus-typeahead-dropdown'
    ) as WebstatusTypeaheadDropdown;

    assert.equal(false, dropdown.open);

    component.show();
    assert.equal(true, dropdown.open);

    component.hide();
    assert.equal(false, dropdown.open);
  });

  it('can determine the prefix of what the user typed', async () => {
    const component = await fixture<WebstatusTypeahead>(html`
      <webstatus-typeahead>
        <webstatus-typeahead-item
          id="item0"
          value="aaa"
          doc="Docs about aaa"
        ></webstatus-typeahead-item>
      </webstatus-typeahead>
    `);
    assert.exists(component);
    assert.instanceOf(component, WebstatusTypeahead);
    const slInput = component.shadowRoot!.querySelector('sl-input') as SlInput;

    slInput.value = '';
    await slInput.updateComplete;
    component.findPrefix();
    assert.equal(0, component.chunkStart);
    assert.equal(0, component.chunkEnd);
    assert.equal('', component.prefix);

    slInput.value = 'term1 term2 -term3=2024-06-28..2025-02-03';
    await slInput.updateComplete;

    // Caret is at the start of the input field.
    slInput.input.selectionStart = 0;
    slInput.input.selectionEnd = 0;
    component.findPrefix();
    assert.equal(0, component.chunkStart);
    assert.equal(5, component.chunkEnd);
    assert.equal('', component.prefix);

    // User has selected a range.
    slInput.input.selectionStart = 0;
    slInput.input.selectionEnd = 3; // A range
    component.findPrefix();
    assert.equal(null, component.prefix);

    // Caret is in middle of term2.
    slInput.input.selectionStart = 8;
    slInput.input.selectionEnd = 8;
    component.findPrefix();
    assert.equal(6, component.chunkStart);
    assert.equal(11, component.chunkEnd);
    assert.equal('te', component.prefix);

    // Caret is after the negation operator, at start of term3.
    slInput.input.selectionStart = 13;
    slInput.input.selectionEnd = 13;
    component.findPrefix();
    assert.equal(13, component.chunkStart);
    assert.equal(41, component.chunkEnd);
    assert.equal('', component.prefix);

    // Caret is near the end of -term3=2024-06-28..2025-02-03.
    slInput.input.selectionStart = 40;
    slInput.input.selectionEnd = 40;
    component.findPrefix();
    assert.equal(13, component.chunkStart);
    assert.equal(41, component.chunkEnd);
    assert.equal('term3=2024-06-28..2025-02-0', component.prefix);
  });

  it('determines whether a candidate should be shown', async () => {
    const component = new WebstatusTypeahead();

    const candidate1 = {name: 'term=', doc: 'Some term'};
    assert.isFalse(component.shouldShowCandidate(candidate1, null));
    assert.isTrue(component.shouldShowCandidate(candidate1, ''));
    assert.isTrue(component.shouldShowCandidate(candidate1, 'te'));
    assert.isTrue(component.shouldShowCandidate(candidate1, 'term='));
    assert.isTrue(component.shouldShowCandidate(candidate1, 'Som'));
    assert.isFalse(component.shouldShowCandidate(candidate1, 'other'));
    assert.isFalse(component.shouldShowCandidate(candidate1, 'erm'));
    assert.isFalse(component.shouldShowCandidate(candidate1, 'erm'));

    assert.isTrue(component.shouldShowCandidate(candidate1, 'TERM='));
    assert.isTrue(component.shouldShowCandidate(candidate1, 'SOM'));

    const candidate2 = {name: 'path.dot.term=', doc: 'Some term'};
    assert.isTrue(component.shouldShowCandidate(candidate2, ''));
    assert.isTrue(component.shouldShowCandidate(candidate2, 'do'));
    assert.isTrue(component.shouldShowCandidate(candidate2, 'Ter'));
    assert.isTrue(component.shouldShowCandidate(candidate2, 'path.dot.'));
    assert.isFalse(component.shouldShowCandidate(candidate2, 'th'));
    assert.isFalse(component.shouldShowCandidate(candidate2, 'th.dot'));
  });
});

describe('webstatus-typeahead-dropdown', () => {
  it('can get, set, and clear its current item', async () => {
    const component = await fixture<WebstatusTypeaheadDropdown>(html`
      <webstatus-typeahead-dropdown>
        <sl-input slot="trigger"> </sl-input>
        <sl-menu>
          <webstatus-typeahead-item
            id="item0"
            value="aaa"
            doc="Docs about aaa"
          ></webstatus-typeahead-item>
          <webstatus-typeahead-item
            id="item1"
            value="bbb"
            doc="Docs about bbb"
          ></webstatus-typeahead-item>
        </sl-menu>
      </webstatus-typeahead-dropdown>
    `);
    assert.exists(component);
    assert.instanceOf(component, WebstatusTypeaheadDropdown);
    const item0 = component.querySelector('#item0') as SlMenuItem;
    const item1 = component.querySelector('#item1') as SlMenuItem;

    assert.equal(item0, component.getCurrentItem());

    component.setCurrentItem(item1);
    assert.equal(item1, component.getCurrentItem());

    component.resetSelection();
    assert.equal(null, component.getCurrentItem());
  });
});

describe('webstatus-typeahead-item', () => {
  it('renders a value and doc string', async () => {
    const component = await fixture<WebstatusTypeaheadItem>(html`
      <webstatus-typeahead-item
        value="aValue"
        doc="Docs about it"
      ></webstatus-typeahead-item>
    `);
    assert.exists(component);
    assert.instanceOf(component, WebstatusTypeaheadItem);
    assert.equal(component.getAttribute('role'), 'menuitem');

    const div = component.shadowRoot!.querySelector('div') as Element;
    assert.exists(div);

    const divInnerHTML = div.innerHTML;
    assert.include(divInnerHTML, 'aValue');
    assert.include(divInnerHTML, 'Docs about it');
    assert.notInclude(divInnerHTML, 'active');
  });

  it('renders as active when sl-menu makes it current', async () => {
    const component = await fixture<WebstatusTypeaheadItem>(html`
      <webstatus-typeahead-item
        value="aValue"
        doc="Docs about it"
        tabindex="0"
      ></webstatus-typeahead-item>
    `);
    const div = component.shadowRoot!.querySelector('div') as Element;

    assert(div.classList.contains('active'));
  });

  it('renders with the prefix in bold', async () => {
    const component = await fixture<WebstatusTypeaheadItem>(html`
      <webstatus-typeahead-item
        value="aValue"
        doc="Docs about it"
        prefix="aVal"
      ></webstatus-typeahead-item>
    `);
    const valueEl = component.shadowRoot!.querySelector('#value') as Element;
    assert.include(valueEl.innerHTML, 'aVal');
    assert.include(valueEl.innerHTML, 'ue');
    const bold = component.shadowRoot!.querySelector('b') as Element;
    assert.include(bold.innerHTML, 'aVal');
    const docEl = component.shadowRoot!.querySelector('#doc') as Element;
    assert.include(docEl.innerHTML, 'Docs about it');
  });

  it('matches any word in value or doc', async () => {
    const component = await fixture<WebstatusTypeaheadItem>(html`
      <webstatus-typeahead-item
        value="a-search-keyword-search="
        doc="Search within keyword"
        prefix="sEar"
      ></webstatus-typeahead-item>
    `);
    const valueEl = component.shadowRoot!.querySelector('#value') as Element;
    assert.include(valueEl.innerHTML, 'a-');
    assert.include(valueEl.innerHTML, 'sear');
    assert.include(valueEl.innerHTML, 'ch-keyword-search');
    const vBold = valueEl.querySelector('b') as Element;
    assert.include(vBold.innerHTML, 'sear');
    const docEl = component.shadowRoot!.querySelector('#doc') as Element;
    assert.include(docEl.innerHTML, 'Sear');
    assert.include(docEl.innerHTML, 'ch within keyword');
    const dBold = docEl.querySelector('b') as Element;
    assert.include(dBold.innerHTML, 'Sear');
  });
});
