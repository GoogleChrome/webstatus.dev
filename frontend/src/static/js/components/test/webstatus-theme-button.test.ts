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

import {expect, fixture, html} from '@open-wc/testing';
import {WebstatusThemeButton} from '../webstatus-theme-button.js';
import '../webstatus-theme-button.js';
import sinon from 'sinon';

describe('webstatus-theme-button', () => {
  it('renders correctly with light theme', async () => {
    const component = await fixture<WebstatusThemeButton>(
      html`<webstatus-theme-button .theme=${'light'}></webstatus-theme-button>`,
    );
    const iconButton = component.shadowRoot?.querySelector('sl-icon-button');
    expect(iconButton).to.exist;
    expect(iconButton?.getAttribute('name')).to.equal('moon');
    expect(iconButton?.getAttribute('label')).to.equal('Switch to dark theme');

    const tooltip = component.shadowRoot?.querySelector('sl-tooltip');
    expect(tooltip?.getAttribute('content')).to.equal('Switch to dark theme');
  });

  it('renders correctly with dark theme', async () => {
    const component = await fixture<WebstatusThemeButton>(
      html`<webstatus-theme-button .theme=${'dark'}></webstatus-theme-button>`,
    );
    const iconButton = component.shadowRoot?.querySelector('sl-icon-button');
    expect(iconButton).to.exist;
    expect(iconButton?.getAttribute('name')).to.equal('sun');
    expect(iconButton?.getAttribute('label')).to.equal('Switch to light theme');

    const tooltip = component.shadowRoot?.querySelector('sl-tooltip');
    expect(tooltip?.getAttribute('content')).to.equal('Switch to light theme');
  });

  it('renders correctly when theme is detecting (undefined)', async () => {
    const component = await fixture<WebstatusThemeButton>(
      html`<webstatus-theme-button
        .theme=${undefined}
      ></webstatus-theme-button>`,
    );
    const iconButton = component.shadowRoot?.querySelector('sl-icon-button');
    expect(iconButton).to.exist;
    expect(iconButton?.getAttribute('name')).to.equal('moon');
    expect(iconButton?.getAttribute('label')).to.equal('Detecting theme...');

    const tooltip = component.shadowRoot?.querySelector('sl-tooltip');
    expect(tooltip?.getAttribute('content')).to.equal('Detecting theme...');
  });

  it('dispatches theme-toggle event when clicked', async () => {
    const component = await fixture<WebstatusThemeButton>(
      html`<webstatus-theme-button .theme=${'light'}></webstatus-theme-button>`,
    );
    const toggleSpy = sinon.spy();
    component.addEventListener('theme-toggle', toggleSpy);

    const iconButton = component.shadowRoot?.querySelector('sl-icon-button');
    iconButton?.click();

    expect(toggleSpy).to.have.been.calledOnce;
  });
});
