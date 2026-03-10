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

import {AppLocation} from './router-types.js';
export {AppLocation, Route} from './router-types.js';

export type WebstatusNavigateEvent = CustomEvent<{url: string}>;
export type NavigationChangedEvent = CustomEvent<AppLocation>;

let polyfillPromise: Promise<void> | null = null;

/** Ensures the URLPattern polyfill is loaded if needed. */
export async function ensurePolyfill() {
  if (!(globalThis && 'URLPattern' in globalThis)) {
    if (!polyfillPromise) {
      polyfillPromise = import('urlpattern-polyfill').then(() => {});
    }
    await polyfillPromise;
  }
}

export const navigateToUrl = (url: string, event?: MouseEvent) => {
  if (event) {
    if (event.ctrlKey || event.metaKey) {
      window.open(url, '_blank');
      return;
    }
    event.preventDefault();
  }

  window.dispatchEvent(
    new CustomEvent('webstatus-navigate', {
      detail: {url},
      bubbles: true,
      composed: true,
    }),
  );
};

export const getCurrentLocation = (): AppLocation => {
  return {
    search: window.location.search,
    href: window.location.href,
    pathname: window.location.pathname,
    params: {},
  };
};
