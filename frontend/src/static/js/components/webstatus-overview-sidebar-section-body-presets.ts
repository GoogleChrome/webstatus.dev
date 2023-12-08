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

import { LitElement, type TemplateResult, css, html } from 'lit'
import { customElement } from 'lit/decorators.js'

@customElement('webstatus-overview-sidebar-section-body-presets')
export class WebstatusOverviewSidebarSectionBodyPresets extends LitElement {
  // TODO: move to base template
  static styles = css`
    .material-icons {
      font-family: 'Material Icons';
      font-size: 24px;
      vertical-align: middle;
    }

    .preset {
      border-radius: 25px;
      border: 1px solid #C4C7C5;
      padding: 5px;
      max-width: max-content;
    }
  `
  render (): TemplateResult {
    return html`
      <div class="preset"><span class="material-icons md-24">label</span> Baseline 2023</div><br />
      <div class="preset"><span class="material-icons md-24">label</span> Top Developer Needs</div><br />
      <div class="preset"><span class="material-icons md-24">label</span> WPT score &lt; 99%</div><br />
      <div class="preset"><span class="material-icons md-24">bookmark</span> Subscribed by me</div><br />
    `
  }
}
