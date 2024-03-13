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
import {ifDefined} from 'lit/directives/if-defined.js';
import {range} from 'lit/directives/range.js';
import {map} from 'lit/directives/map.js';
import {formatOverviewPageUrl, getPaginationStart} from '../utils/urls.js';
import {type components} from 'webstatus.dev-backend';

import {SHARED_STYLES} from '../css/shared-css.js';

const ITEMS_PER_PAGE = 25;

@customElement('webstatus-pagination')
export class WebstatusPagination extends LitElement {
  @state()
  features: Array<components['schemas']['Feature']> = [];

  @state()
  start = 0; // Index of first result among total results.

  @state()
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

        sl-button::part(base):hover {
          background: var(--pagination-hover-background);
        }
      `,
    ];
  }

  getTotalCount() {
    // TODO(jrobbins): Get from JSON response field when available.
    return this.features?.length || 0;
  }

  formatUrlForOffset(offset: number): string {
    return formatOverviewPageUrl(this.location, {start: offset});
  }

  formatUrlForRelativeOffset(delta: number): string | undefined {
    const offset = this.start + delta;
    if (offset <= -ITEMS_PER_PAGE || offset > this.getTotalCount()) {
      return undefined;
    }
    return this.formatUrlForOffset(Math.max(0, offset));
  }

  renderPageButtons(): TemplateResult {
    const currentPage = Math.floor(this.start / ITEMS_PER_PAGE);
    const numPages = Math.floor(this.getTotalCount() / ITEMS_PER_PAGE) + 1;

    return html`
      ${map(
        range(numPages),
        i => html`
          <sl-button
            variant="text"
            class="page-button ${i === currentPage ? 'active' : ''}"
            href=${this.formatUrlForOffset(i * ITEMS_PER_PAGE)}
          >
            ${i + 1}
          </sl-button>
        `
      )}
    `;
  }

  render(): TemplateResult {
    if (this.features.length === 0) {
      return html``;
    }

    this.start = getPaginationStart(this.location);
    const prevUrl = this.formatUrlForRelativeOffset(-ITEMS_PER_PAGE);
    const nextUrl = this.formatUrlForRelativeOffset(ITEMS_PER_PAGE);

    return html`
      <div id="main" class="hbox halign-items-space-between">
        <div class="spacer"></div>
        <sl-button
          variant="text"
          class="stepper"
          href=${ifDefined(prevUrl)}
          ?disabled=${prevUrl === undefined}
          >Previous</sl-button
        >

        ${this.renderPageButtons()}

        <sl-button
          variant="text"
          class="stepper"
          href=${ifDefined(nextUrl)}
          ?disabled=${nextUrl === undefined}
          >Next</sl-button
        >
        <div class="spacer"></div>
      </div>
    `;
  }
}
