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
import {customElement, state} from 'lit/decorators.js';
import {range} from 'lit/directives/range.js';
import {map} from 'lit/directives/map.js';
import {type components} from 'webstatus.dev-backend';

import {SHARED_STYLES} from '../css/shared-css.js';

const ITEMS_PER_PAGE = 25;

@customElement('webstatus-pagination')
export class WebstatusPagination extends LitElement {
  @state()
  features: Array<components['schemas']['Feature']> = [];

  @state()
  currentPage = 0;

  location!: {search: string}; // Set by parent.

  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        .stepper {
          width: 7em;
        }

        .active {
          background: var(--pagination-active-background);
        }
      `,
    ];
  }

  getNumPages() {
    const totalCount = this.features?.length || 0;
    const numPages = Math.floor(totalCount / ITEMS_PER_PAGE) + 1;
    return numPages;
  }

  renderPageButtons(): TemplateResult {
    return html`
      ${map(
         range(this.getNumPages()),
         (i) => html`
          <sl-button
            variant="text"
            class="page-button ${i === this.currentPage ? 'active' : ''}"
            href="#TODO-${i*ITEMS_PER_PAGE}"
           >
             ${i + 1}
           </sl-button>
         `
      )}
    `;
  }

  render(): TemplateResult {
    const prevDisabled = this.currentPage === 0;
    const nextDisabled = this.currentPage === this.getNumPages();
    return html`
      <div id="main" class="hbox halign-items-space-between">
        <div class="spacer"></div>
        <sl-button
          variant="text" class="stepper"
          ?disabled=${prevDisabled}
        >Previous</sl-button>

        ${this.renderPageButtons()}

        <sl-button
          variant="text" class="stepper"
          ?disabled=${nextDisabled}
        >Next</sl-button>
        <div class="spacer"></div>
      </div>
    `;
  }
}
