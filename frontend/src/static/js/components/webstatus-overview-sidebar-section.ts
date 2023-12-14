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
        .collapsible-section .header {
          display: flex;
          align-items: center;
          cursor: pointer;
        }

        .collapsible-section .arrow {
          margin-left: auto;
        }

        .collapsible-section .content {
          max-height: 0;
          overflow: hidden;
          transition: max-height 0.5s ease;
        }

        .collapsible-section .content.open {
          display: block;
          max-height: 1000px;
        }
      `
    ]
  }

  render(): TemplateResult {
    const arrow = this.open ? '▲' : '▼'
    const contentClass = this.open ? 'content open' : 'content'
    return html`
      <div class="collapsible-section">
        <div class="header" @click=${this.toggleOpen}>
          ${this.header}
          <span class="arrow">${arrow}</span>
        </div>
        <div class="${contentClass}" ?hidden=${!this.open}>
          <slot></slot>
        </div>
      </div>
    `
  }

  toggleOpen(): void {
    this.open = !this.open
  }
}
