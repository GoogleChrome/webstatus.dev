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
import {LitElement, type TemplateResult, html} from 'lit';
import {customElement} from 'lit/decorators.js';

import {gchartsContext} from '../contexts/gcharts-context.js';

@customElement('webstatus-gcharts-loader')
export class WebstatusGChartsLoader extends LitElement {
  @provide({context: gchartsContext})
  gchartsLibraryLoaded = false;

  scriptInserted: boolean = false;

  constructor() {
    super();
    console.log('WebstatusGChartsLoader constructor');
  }

  firstUpdated(): void {
    console.log('WebstatusGChartsLoader firstUpdated');
    this.loadGoogleChartsLoaderAndPackages().then(
      // TODO. Success case
      () => {},
      // TODO. Failure case.  We could progagate an event or signal
      // that will render a useful message to the user to reload the page.
      () => {}
    );
  }

  async loadGoogleChartsLoaderAndPackages(): Promise<void> {
    console.log('WebstatusGChartsLoader loadGoogleChartsLoaderAndPackages');
    if (this.scriptInserted) {
      return;
    }

    // Insert script to load the loader.
    const script = document.createElement('script');
    script.src = 'https://www.gstatic.com/charts/loader.js';
    document.head.appendChild(script);

    this.scriptInserted = true;

    const loaderPromise = new Promise<void>(resolve => {
      script.addEventListener('load', () => {
        // resolve();
        google.charts
        .load('current', {
          packages: ['corechart'],
        })
        .then(() => {
          this.gchartsLibraryLoaded = true;
          console.log('WebstatusGChartsLoader loadGoogleChartsLoaderAndPackages resolved');
          resolve();
        });
      });
    });

    return loaderPromise;

    // // After the loader script is loaded, we can load packages.
    // await loaderPromise.then(async () => {
    //   // this.loadGoogleChartsPackages();
    // });
  }

  // async loadGoogleChartsPackages(): Promise<void> {
  //  return google.charts
  //     .load('current', {
  //       packages: ['corechart'],
  //     })
  //     .then(() => {
  //       this.gchartsLibraryLoaded = true;
  //     });
  // }

  render(): TemplateResult {
    console.log('WebstatusGChartsLoader render');
    return html` <slot></slot> `;
  }
}
