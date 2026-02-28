/**
 * Copyright 2026 Google LLC
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

import {LitElement, html} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {consume} from '@lit/context';
import {themeContext, type Theme} from '../../contexts/theme-context.js';
import {WebstatusThemeService} from '../webstatus-theme-service.js';
import {fixture, expect} from '@open-wc/testing';
import '../webstatus-theme-service.js';
import sinon from 'sinon';

@customElement('test-theme-consumer')
class TestThemeConsumer extends LitElement {
  @consume({context: themeContext, subscribe: true})
  @state()
  theme?: Theme;

  render() {
    return html`<div id="theme-val">${this.theme}</div>`;
  }
}

describe('webstatus-theme-service', () => {
  let matchMediaStub: sinon.SinonStub;

  beforeEach(() => {
    localStorage.clear();
    matchMediaStub = sinon.stub(window, 'matchMedia');
    matchMediaStub.returns({
      matches: false,
      addEventListener: () => {},
      removeEventListener: () => {},
    });
    document.documentElement.classList.remove('sl-theme-dark');
  });

  afterEach(() => {
    matchMediaStub.restore();
  });

  it('can be added to the page and provides light theme by default', async () => {
    const el = await fixture<WebstatusThemeService>(html`
      <webstatus-theme-service>
        <test-theme-consumer></test-theme-consumer>
      </webstatus-theme-service>
    `);
    const consumer = el.querySelector<TestThemeConsumer>('test-theme-consumer');
    expect(el).to.exist;
    expect(consumer).to.exist;
    expect(consumer!.theme).to.equal('light');
    expect(document.documentElement.classList.contains('sl-theme-dark')).to.be
      .false;
    expect(document.documentElement.classList.contains('sl-theme-light')).to.be
      .true;
  });

  it('provides dark theme if system preference is dark', async () => {
    matchMediaStub.returns({
      matches: true,
      addEventListener: () => {},
      removeEventListener: () => {},
    });
    const el = await fixture<WebstatusThemeService>(html`
      <webstatus-theme-service>
        <test-theme-consumer></test-theme-consumer>
      </webstatus-theme-service>
    `);
    const consumer = el.querySelector<TestThemeConsumer>('test-theme-consumer');
    expect(consumer!.theme).to.equal('dark');
    expect(document.documentElement.classList.contains('sl-theme-dark')).to.be
      .true;
    expect(document.documentElement.classList.contains('sl-theme-light')).to.be
      .false;
  });

  it('prefers saved theme from localStorage', async () => {
    localStorage.setItem('webstatus-theme', 'dark');
    matchMediaStub.returns({
      matches: false, // system is light
      addEventListener: () => {},
      removeEventListener: () => {},
    });
    const el = await fixture<WebstatusThemeService>(html`
      <webstatus-theme-service>
        <test-theme-consumer></test-theme-consumer>
      </webstatus-theme-service>
    `);
    const consumer = el.querySelector<TestThemeConsumer>('test-theme-consumer');
    expect(consumer!.theme).to.equal('dark');
    expect(document.documentElement.classList.contains('sl-theme-dark')).to.be
      .true;
    expect(document.documentElement.classList.contains('sl-theme-light')).to.be
      .false;
  });

  it('toggles theme on toggle-theme event', async () => {
    const el = await fixture<WebstatusThemeService>(html`
      <webstatus-theme-service>
        <test-theme-consumer></test-theme-consumer>
      </webstatus-theme-service>
    `);
    const consumer = el.querySelector<TestThemeConsumer>('test-theme-consumer');

    expect(consumer!.theme).to.equal('light');

    el.dispatchEvent(
      new CustomEvent('theme-toggle', {bubbles: true, composed: true}),
    );

    expect(consumer!.theme).to.equal('dark');
    expect(document.documentElement.classList.contains('sl-theme-dark')).to.be
      .true;
    expect(document.documentElement.classList.contains('sl-theme-light')).to.be
      .false;
    expect(localStorage.getItem('webstatus-theme')).to.equal('dark');

    el.dispatchEvent(
      new CustomEvent('theme-toggle', {bubbles: true, composed: true}),
    );
    expect(consumer!.theme).to.equal('light');
    expect(document.documentElement.classList.contains('sl-theme-dark')).to.be
      .false;
    expect(document.documentElement.classList.contains('sl-theme-light')).to.be
      .true;
    expect(localStorage.getItem('webstatus-theme')).to.equal('light');
  });
});
