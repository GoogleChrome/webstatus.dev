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
import { customElement } from 'lit/decorators.js'

@customElement('webstatus-overview-sidebar-menu')
export class WebstatusOverviewSidebarMenu extends LitElement {
  static get styles(): CSSResultGroup {
    return [
      css`
        .material-icons {
          font-family: 'Material Icons';
          font-size: 24px;
          vertical-align: middle;
        }
      `
    ]
  }

  render(): TemplateResult {
    return html`
      <sl-tree>
        <sl-icon name="caret-right-fill" slot="expand-icon"></sl-icon>
        <sl-icon name="caret-right-fill" slot="collapse-icon"></sl-icon>

        <sl-tree-item>
          <sl-icon name="menu-button"></sl-icon> Features
          <sl-tree-item>
            <sl-icon name="bookmark"></sl-icon> Baseline 2023
          </sl-tree-item>
          <sl-tree-item>
            <sl-icon name="bookmark"></sl-icon> Top Developer Needs
          </sl-tree-item>
          <sl-tree-item> <sl-icon name="star"></sl-icon> Starred </sl-tree-item>
        </sl-tree-item>
        <sl-tree-item>
          <sl-icon name="heart-pulse"></sl-icon> Statistics
        </sl-tree-item>
        <sl-tree-item> <sl-icon name="bell"></sl-icon> Updates </sl-tree-item>
        <sl-tree-item>
          <sl-icon name="info-circle"></sl-icon> About
        </sl-tree-item>
      </sl-tree>
    `
  }
}
