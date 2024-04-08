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

import { ContextProvider } from '@lit/context';
import { LitElement, type TemplateResult, html } from 'lit';
import { customElement, state } from 'lit/decorators.js';

import { gchartsContext } from '../contexts/gcharts-context.js';

@customElement('webstatus-gcharts-loader')
export class WebstatusAppSettings extends LitElement {

  @state()
  gchartsLibraryLoaded = false;

  @state()
  gchartsProvider = new ContextProvider(this, {
    context: gchartsContext,
  });

  constructor() {
    super();
  }

  loadGoogleCharts() {
    google.charts.load('current', {
      packages: ['corechart'],
    }).then(() => {
      this.gchartsLibraryLoaded = true;
      this.gchartsProvider.setValue(this.gchartsLibraryLoaded);
    });
  }

  // Render conditional on the loading state of the task.
  render(): TemplateResult {
    this.loadGoogleCharts();
    return html`<slot></slot>`;
  }
}
