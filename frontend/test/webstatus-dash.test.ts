import { html } from 'lit';
import { fixture, expect } from '@open-wc/testing';

import type { WebstatusDash } from '../src/webstatus-dash.js';
import '../src/webstatus-dash.js';

describe('WebstatusDash', () => {
  let element: WebstatusDash;
  beforeEach(async () => {
    element = await fixture(html`<webstatus-dash></webstatus-dash>`);
  });

  it('renders a h1', () => {
    const h1 = element.shadowRoot!.querySelector('h1')!;
    expect(h1).to.exist;
    expect(h1.textContent).to.equal('My app');
  });

  it('passes the a11y audit', async () => {
    await expect(element).shadowDom.to.be.accessible();
  });
});
