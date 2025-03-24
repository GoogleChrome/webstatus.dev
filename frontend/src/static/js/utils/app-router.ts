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

import {Router} from '@vaadin/router';

import '../components/webstatus-overview-page.js';
import '../components/webstatus-feature-page.js';
import '../components/webstatus-stats-page.js';
import '../components/webstatus-notfound-error-page.js';

export const initRouter = async (element: HTMLElement): Promise<Router> => {
  const router = new Router(element);
  await router.setRoutes([
    {
      component: 'webstatus-overview-page',
      path: '/',
    },
    {
      component: 'webstatus-feature-page',
      path: '/features/:featureId',
    },
    {
      component: 'webstatus-stats-page',
      path: '/stats',
    },
    {
      path: '(.*)',
      component: 'webstatus-not-found-error-page',
    },
  ]);
  return router;
};

export interface AppLocation {
  search: string;
  href: string;
}

export const navigateToUrl = (url: string, event?: MouseEvent) => {
  if ((event !== undefined && event.ctrlKey) || event?.metaKey) {
    window.open(url, '_blank');
    return;
  }

  // Construct a full URL to handle relative paths correctly
  const fullNewUrl = new URL(url, window.location.origin).href; // TODO for future - handle try/catch case until we use AppLocation

  const currentUrl = window.location.href;
  if (currentUrl === fullNewUrl) {
    return;
  }

  // TODO. We should use the vaadin router and use the navigate method there.
  window.location.href = url;
};

export const getCurrentLocation = (): AppLocation => {
  // Return a copy of the location object to avoid modifying the original and
  // to have more control over when our copy is updated.
  return {...window.location};
};
