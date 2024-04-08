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

// import { consume } from '@lit/context';
import {Task} from '@lit/task';
import {
  LitElement,
  // type TemplateResult,
  html,
  CSSResultGroup,
  css,
} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {SHARED_STYLES} from '../css/shared-css.js';
// import { type components } from 'webstatus.dev-backend';

@customElement('webstatus-feature-page')
export class Chart extends LitElement {
  @state()
  loadingGCharts: Task;

  @state()
  gchartsPackagesLoaded: boolean;

  // @consume({ context: googleChartsContext })
  @state()
  chartWrapper!: google.visualization.ChartWrapper;

  @state()
  data?: google.visualization.DataTable;

  @state()
  specs?: google.visualization.ChartSpecs;

  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        :host {
          padding: 0;
          margin: 0;
          border: 0;
        }
      `,
    ];
  }

  loadGoogleCharts() {
    google.charts.load('current', {
      packages: ['corechart'],
    });
    google.charts.setOnLoadCallback(() => {
      this.gchartsPackagesLoaded = true;
    });
  }

  constructor() {
    super();
    this.gchartsPackagesLoaded = false;
    this.loadingGCharts = new Task(this, {
      args: () => [this.gchartsPackagesLoaded] as [boolean],
      task: async ([gcLoaded]: [boolean]) => {
        if (gcLoaded) {
          this.chartWrapper = new google.visualization.ChartWrapper(this.specs);
          this.chartWrapper.draw();
        }
      },
    });
  }

  render() {
    return html` <div class="container"></div> `;
  }
}
