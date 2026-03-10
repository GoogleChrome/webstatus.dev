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

import {provide} from '@lit/context';
import {customElement, property, state} from 'lit/decorators.js';
import {PropertyValues} from 'lit';
import {ServiceElement} from './service-element.js';
import {locationContext} from '../contexts/location-context.js';
import {
  getCurrentLocation,
  AppLocation,
  Route,
  navigateToUrl,
  ensurePolyfill,
  WebstatusNavigateEvent,
  NavigationChangedEvent,
} from '../utils/router-utils.js';
import {WebstatusBasePage} from '../components/webstatus-base-page.js';

/**
 * Helpers for accessing browser location properties.
 * Separated into an object to allow for easy stubbing in unit tests.
 */
export const routerServiceHelpers = {
  getHref: () => window.location.href,
  getOrigin: () => window.location.origin,
  getPathname: () => window.location.pathname,
  getSearch: () => window.location.search,
};

@customElement('webstatus-router-service')
export class WebstatusRouterService extends ServiceElement {
  @provide({context: locationContext})
  @state()
  location: AppLocation = getCurrentLocation();

  @property({attribute: false})
  routes: Route[] = [];

  @property({attribute: false})
  host?: HTMLElement;

  private routePatterns: {pattern: URLPattern; component: string}[] = [];

  private handlePopState = () => {
    void this.handleRouting();
  };

  private handleWebstatusNavigate = (event: Event) => {
    const {url} = (event as WebstatusNavigateEvent).detail;
    // Construct a full URL to handle relative paths correctly
    const fullNewUrl = new URL(url, routerServiceHelpers.getOrigin()).href;
    const currentUrl = routerServiceHelpers.getHref();

    if (currentUrl !== fullNewUrl) {
      window.history.pushState(null, '', url);
      void this.handleRouting();
    }
  };

  /**
   * Internal method to determine if a click on an anchor should be intercepted.
   * Extracted for easier unit testing.
   */
  _shouldInterceptClick(anchor: HTMLAnchorElement): boolean {
    return (
      !!anchor.href &&
      anchor.origin === routerServiceHelpers.getOrigin() &&
      !anchor.hasAttribute('download') &&
      anchor.target !== '_blank'
    );
  }

  private handleGlobalClick = (event: MouseEvent) => {
    const path = event.composedPath();
    const target = path[0];
    if (!(target instanceof Element)) return;

    const anchor = target.closest('a');

    if (
      anchor instanceof HTMLAnchorElement &&
      this._shouldInterceptClick(anchor)
    ) {
      event.preventDefault();
      navigateToUrl(anchor.pathname + anchor.search + anchor.hash, event);
    }
  };

  private async handleRouting() {
    if (!this.host || this.routePatterns.length === 0) return;

    await ensurePolyfill();
    const url = routerServiceHelpers.getHref();

    for (const route of this.routePatterns) {
      const match = route.pattern.exec(url);
      if (match) {
        const componentTag = route.component;
        let compElement = this.host.firstElementChild;

        if (
          !compElement ||
          compElement.tagName.toLowerCase() !== componentTag.toLowerCase()
        ) {
          // Clear old component
          while (this.host.firstChild) {
            this.host.removeChild(this.host.firstChild);
          }
          compElement = document.createElement(componentTag);
          this.host.appendChild(compElement);
        }

        // Pass location to component
        const appLocation: AppLocation = {
          search: routerServiceHelpers.getSearch(),
          href: routerServiceHelpers.getHref(),
          pathname: routerServiceHelpers.getPathname(),
          params: match.pathname.groups || {},
        };

        this.location = appLocation;

        if (compElement instanceof WebstatusBasePage) {
          compElement.location = appLocation;
        }

        // Dispatch event for centralized state management
        this.host.dispatchEvent(
          new CustomEvent<AppLocation>('navigation-changed', {
            detail: appLocation,
            bubbles: true,
            composed: true,
          }) as NavigationChangedEvent,
        );
        return;
      }
    }
  }

  async updated(changedProperties: PropertyValues<this>) {
    if (changedProperties.has('routes') && this.routes.length > 0) {
      await ensurePolyfill();
      this.routePatterns = this.routes.map(route => {
        const path = route.path.replace(/\(\.\*\)/g, '*');
        return {
          pattern: new URLPattern({pathname: path}),
          component: route.component,
        };
      });
      void this.handleRouting();
    } else if (changedProperties.has('host') && this.host) {
      void this.handleRouting();
    }
  }

  connectedCallback(): void {
    super.connectedCallback();
    window.addEventListener('click', this.handleGlobalClick, true);
    window.addEventListener('popstate', this.handlePopState);
    window.addEventListener('webstatus-navigate', this.handleWebstatusNavigate);
  }

  disconnectedCallback() {
    window.removeEventListener('click', this.handleGlobalClick, true);
    window.removeEventListener('popstate', this.handlePopState);
    window.removeEventListener(
      'webstatus-navigate',
      this.handleWebstatusNavigate,
    );
    super.disconnectedCallback();
  }
}
