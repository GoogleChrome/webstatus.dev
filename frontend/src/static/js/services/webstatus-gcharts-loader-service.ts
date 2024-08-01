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

import {provide} from '@lit/context';
import {customElement} from 'lit/decorators.js';

import {gchartsContext} from '../contexts/gcharts-context.js';
import {ServiceElement} from './service-element.js';

@customElement('webstatus-gcharts-loader-service')
export class WebstatusGChartsLoaderService extends ServiceElement {
  @provide({context: gchartsContext})
  gchartsLibraryLoaded = false;

  scriptInserted: boolean = false;

  firstUpdated(): void {
    this.loadGoogleChartsLoaderAndPackages().then(
      // TODO. Success case
      () => {},
      // TODO. Failure case.  We could progagate an event or signal
      // that will render a useful message to the user to reload the page.
      () => {}
    );
  }

  async loadGoogleChartsLoaderAndPackages(): Promise<void> {
    if (this.scriptInserted) {
      return;
    }
    this.scriptInserted = true;

    // Insert script to load the loader.
    const script = document.createElement('script');
    script.src = 'https://www.gstatic.com/charts/loader.js';
    document.head.appendChild(script);

    const loaderPromise = new Promise<void>(resolve => {
      script.addEventListener('load', () => {
        google.charts
          .load('current', {
            packages: ['corechart'],
          })
          .then(() => {
            this.gchartsLibraryLoaded = true;
            resolve();
          });
      });
    });

    return loaderPromise;
  }

  // This is a test helper method and should not be used by live code.
  async waitForGChartsLibraryLoaded(timeoutMs = 5000): Promise<void> {
    const endTime = Date.now() + timeoutMs;
    let delay = 10;

    return new Promise((resolve, reject) => {
      const loaderLoop = () => {
        if (this.gchartsLibraryLoaded) {
          resolve();
        } else if (Date.now() < endTime) {
          delay *= 2;
          setTimeout(loaderLoop, delay);
        } else {
          reject('Timeout waiting for Google Charts to load');
        }
      };
      setTimeout(loaderLoop, delay);
    });
  }
}
