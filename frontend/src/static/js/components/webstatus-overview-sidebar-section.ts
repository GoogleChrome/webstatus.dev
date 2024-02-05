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

import {
  type CSSResultGroup,
  LitElement,
  type TemplateResult,
  css,
  html
} from 'lit'
import { customElement, property } from 'lit/decorators.js'

@customElement('webstatus-overview-sidebar-section')
export class WebstatusOverviewSidebarSection extends LitElement {
  @property()
  header!: string

  @property({ attribute: 'open', type: Boolean })
  open!: boolean

  constructor() {
    super()
    this.open = false
  }

  static get styles(): CSSResultGroup {
    return [
      css`
        .material-icons {
          font-family: 'Material Icons';
          font-size: 24px;
          vertical-align: middle;
        }

        .preset {
          border-radius: 25px;
          border: 1px solid #c4c7c5;
          padding: 5px;
          max-width: max-content;
        }
      `
    ]
  }

  render(): TemplateResult {
    return html`
      <sl-tree selection="leaf">
        <sl-icon name="caret-right-fill" slot="expand-icon"></sl-icon>
        <sl-icon name="caret-right-fill" slot="collapse-icon"></sl-icon>

        <sl-tree-item class="header">
          <span class="material-icons md-24">folder</span> Features
          <sl-tree-item class="preset">
            <sl-icon name="bookmark-star"></sl-icon> Baseline 2023
          </sl-tree-item>
          <sl-tree-item class="preset">
            <span class="material-icons md-24">workspace_premium</span> Top
            Developer Needs
          </sl-tree-item>
          <sl-tree-item class="preset">
            <span class="material-icons md-24">label</span> WPT score &lt; 99%
          </sl-tree-item>
          <sl-tree-item class="preset">
            <span class="material-icons md-24">bookmark</span> Subscribed by me
          </sl-tree-item>
        </sl-tree-item>
        <sl-tree-item class="header">
          <span class="material-icons md-24">query_stats</span> Statistics
        </sl-tree-item>
        <sl-tree-item class="header">
          <span class="material-icons md-24">info</span> About
        </sl-tree-item>
      </sl-tree>
    `
  }
}
