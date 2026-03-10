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

import {fixture, html, expect, oneEvent} from '@open-wc/testing';
import {
  WebstatusRouterService,
  routerServiceHelpers,
} from '../webstatus-router-service.js';
import {Route, AppLocation, navigateToUrl} from '../../utils/router-utils.js';
import {customElement, state} from 'lit/decorators.js';
import {LitElement} from 'lit';
import {consume} from '@lit/context';
import {locationContext} from '../../contexts/location-context.js';
import {WebstatusBasePage} from '../../components/webstatus-base-page.js';
import sinon from 'sinon';

@customElement('test-page-a')
class TestPageA extends WebstatusBasePage {
  render() {
    return html`<div id="page-a">Page A</div>`;
  }
}

@customElement('test-page-b')
class TestPageB extends WebstatusBasePage {
  render() {
    return html`<div id="page-b">Page B</div>`;
  }
}

@customElement('test-location-consumer')
class TestLocationConsumer extends LitElement {
  @consume({context: locationContext, subscribe: true})
  @state()
  location?: AppLocation;

  render() {
    return html`<div id="location">${this.location?.pathname}</div>`;
  }
}

describe('WebstatusRouterService', () => {
  let el: WebstatusRouterService;
  let host: HTMLElement | undefined;
  let pushStateStub: sinon.SinonStub;
  let safetyListener: (e: Event) => void;

  const routes: Route[] = [
    {path: '/a', component: 'test-page-a'},
    {path: '/b/:id', component: 'test-page-b'},
  ];

  beforeEach(async () => {
    // Stub history.pushState to prevent real navigation in tests
    pushStateStub = sinon.stub(window.history, 'pushState');

    // Global safety listener to prevent any actual navigation from interrupting the test runner.
    // We use CAPTURE phase and register it FIRST to ensure it catches the event before any
    // other listeners. This is our primary defense against Firefox site redirections.
    safetyListener = (e: Event) => {
      if (e instanceof MouseEvent && e.type === 'click') {
        const path = e.composedPath();
        const target = path[0];
        if (target instanceof Element && target.closest('a')) {
          e.preventDefault();
        }
      }
    };
    window.addEventListener('click', safetyListener, true);

    // Prevent page unloads (which the test runner perceives as interruptions)
    window.onbeforeunload = () => {
      return 'Test attempted to navigate. This was blocked.';
    };

    // Initialize the fixture.
    el = await fixture(html`
      <webstatus-router-service .routes=${routes}>
        <test-location-consumer></test-location-consumer>
      </webstatus-router-service>
    `);

    // Ensure it's correctly upgraded.
    expect(el).to.be.instanceOf(WebstatusRouterService);

    host = document.createElement('div');
    document.body.appendChild(host);
    el.host = host;
    await el.updateComplete;
  });

  afterEach(() => {
    sinon.restore();
    window.removeEventListener('click', safetyListener, true);
    window.onbeforeunload = null;
    if (host && host.parentElement) {
      document.body.removeChild(host);
    }
    host = undefined;
  });

  it('renders the correct component for a route', async () => {
    sinon.stub(routerServiceHelpers, 'getHref').returns('http://localhost/a');
    sinon.stub(routerServiceHelpers, 'getPathname').returns('/a');

    window.dispatchEvent(new PopStateEvent('popstate'));
    await el.updateComplete;

    const pageA = host?.querySelector<TestPageA>('test-page-a');
    expect(pageA).to.exist;
    expect(pageA?.tagName.toLowerCase()).to.equal('test-page-a');
  });

  it('handles parameters in routes', async () => {
    sinon
      .stub(routerServiceHelpers, 'getHref')
      .returns('http://localhost/b/123');
    sinon.stub(routerServiceHelpers, 'getPathname').returns('/b/123');

    window.dispatchEvent(new PopStateEvent('popstate'));
    await el.updateComplete;

    const pageB = host?.querySelector<TestPageB>('test-page-b');
    expect(pageB).to.exist;
    expect(pageB?.location.params?.id).to.equal('123');
  });

  it('updates location context', async () => {
    sinon.stub(routerServiceHelpers, 'getHref').returns('http://localhost/a');
    sinon.stub(routerServiceHelpers, 'getPathname').returns('/a');

    window.dispatchEvent(new PopStateEvent('popstate'));
    await el.updateComplete;

    const consumer = el.querySelector<TestLocationConsumer>(
      'test-location-consumer',
    );
    expect(consumer).to.exist;
    await consumer?.updateComplete;
    expect(
      consumer?.shadowRoot?.querySelector('#location')?.textContent,
    ).to.equal('/a');
  });

  it('handles webstatus-navigate event', async () => {
    sinon.stub(routerServiceHelpers, 'getHref').returns('http://localhost/old');
    sinon.stub(routerServiceHelpers, 'getOrigin').returns('http://localhost');

    navigateToUrl('/b/456');
    await el.updateComplete;

    expect(pushStateStub.calledOnce).to.be.true;
    expect(pushStateStub.firstCall.args[2]).to.equal('/b/456');
  });

  it('logic: _shouldInterceptClick correctly identifies internal links', () => {
    const anchor = document.createElement('a');
    sinon.stub(anchor, 'href').value('http://localhost/a');
    sinon.stub(anchor, 'origin').value('http://localhost');
    sinon.stub(routerServiceHelpers, 'getOrigin').returns('http://localhost');

    expect(el._shouldInterceptClick(anchor)).to.be.true;
  });

  it('logic: _shouldInterceptClick correctly identifies external links', () => {
    const anchor = document.createElement('a');
    sinon.stub(anchor, 'href').value('https://www.google.com/');
    sinon.stub(anchor, 'origin').value('https://www.google.com');
    sinon.stub(routerServiceHelpers, 'getOrigin').returns('http://localhost');

    expect(el._shouldInterceptClick(anchor)).to.be.false;
  });

  it('intercepts internal anchor clicks', async () => {
    const anchor = document.createElement('a');
    anchor.href = '/a';
    el.appendChild(anchor);

    const clickEvent = new MouseEvent('click', {
      bubbles: true,
      cancelable: true,
      composed: true,
    });

    sinon.stub(anchor, 'origin').value(window.location.origin);
    sinon
      .stub(routerServiceHelpers, 'getOrigin')
      .returns(window.location.origin);

    const preventSpy = sinon.spy(clickEvent, 'preventDefault');

    anchor.dispatchEvent(clickEvent);

    // Call count breakdown:
    // 1. safetyListener (capture phase) calls preventDefault
    // 2. handleGlobalClick (capture phase) calls preventDefault
    // 3. navigateToUrl (inside handleGlobalClick) calls preventDefault
    expect(preventSpy.calledThrice).to.be.true;
    expect(pushStateStub.called).to.be.true;
  });

  it('does not intercept external anchor clicks', () => {
    const anchor = document.createElement('a');
    anchor.href = 'https://www.google.com';
    el.appendChild(anchor);

    const clickEvent = new MouseEvent('click', {
      bubbles: true,
      cancelable: true,
      composed: true,
    });

    sinon.stub(anchor, 'origin').value('https://www.google.com');
    sinon.stub(routerServiceHelpers, 'getOrigin').returns('http://localhost');

    const preventSpy = sinon.spy(clickEvent, 'preventDefault');

    anchor.dispatchEvent(clickEvent);

    // Call count breakdown:
    // 1. safetyListener (capture phase) calls preventDefault (ensuring no real navigation)
    // 2. handleGlobalClick (capture phase) sees external link, does NOT call preventDefault
    expect(preventSpy.calledOnce).to.be.true;
    expect(pushStateStub.called).to.be.false;
  });

  it('dispatches navigation-changed event', async () => {
    // getHref is called twice: once in handleWebstatusNavigate and once in handleRouting.
    sinon
      .stub(routerServiceHelpers, 'getHref')
      .onFirstCall()
      .returns('http://localhost/old')
      .returns('http://localhost/a');
    // getPathname is only called once inside handleRouting (within appLocation object).
    sinon.stub(routerServiceHelpers, 'getPathname').returns('/a');
    sinon.stub(routerServiceHelpers, 'getOrigin').returns('http://localhost');

    const navPromise = oneEvent(host!, 'navigation-changed');

    const navEvent = new CustomEvent('webstatus-navigate', {
      detail: {url: '/a'},
      bubbles: true,
      composed: true,
    });
    window.dispatchEvent(navEvent);

    const event = await navPromise;
    expect(event.detail.pathname).to.equal('/a');
  });
});
