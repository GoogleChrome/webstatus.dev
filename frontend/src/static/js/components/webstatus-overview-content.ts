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

import {LitElement, type TemplateResult, CSSResultGroup, css, html} from 'lit';
import {Task, TaskStatus} from '@lit/task';
import {customElement, state} from 'lit/decorators.js';
import {type components} from 'webstatus.dev-backend';

import './webstatus-overview-filters.js';
import './webstatus-overview-table.js';
import {SHARED_STYLES} from '../css/shared-css.js';

@customElement('webstatus-overview-content')
export class WebstatusOverviewContent extends LitElement {
  @state()
  features: Array<components['schemas']['Feature']> = [];

  loadingTask!: Task; // Set by parent.

  location!: {search: string}; // Set by parent.

  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        .header-line {
          gap: 1em;
        }
        .stats-summary {
          color: #6c7381;
        }
      `,
    ];
  }

  renderCount(): TemplateResult {
    const numFeatures = this.features.length;
    const date = new Date().toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });

    return html`
      <span class="stats-summary">
        <sl-icon library="phosphor" name="clock-clockwise"></sl-icon>
        ${numFeatures} features
      </span>
      <span class="stats-summary">
        <sl-icon library="phosphor" name="clock-clockwise"></sl-icon>
        Updated ${date}
      </span>
    `;
  }

  render(): TemplateResult {
    return html`
      <div class="main">
        <div class="hbox halign-items-space-between header-line">
          <h1 class="halign-stretch">Features overview</h1>
          <sl-button
            ><sl-icon
              slot="prefix"
              name="link-simple"
              library="phosphor"
            ></sl-icon
          ></sl-button>
          <sl-button
            ><sl-icon name="bookmark"></sl-icon> Save this view</sl-button
          >
        </div>
        <div class="hbox">
          ${this.loadingTask.status !== TaskStatus.COMPLETE
            ? html`Loading features...`
            : this.renderCount()}
        </div>
        <br />
        <webstatus-overview-filters
          .location=${this.location}
        ></webstatus-overview-filters>
        <br />

        <webstatus-overview-table
          .location=${this.location}
          .features=${this.features}
          .loadingTask=${this.loadingTask}
        >
        </webstatus-overview-table>
        <button>Modify Columns</button>
      </div>
    `;
  }
}
