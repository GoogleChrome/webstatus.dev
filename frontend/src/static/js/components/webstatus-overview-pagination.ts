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
import {
  DEFAULT_ITEMS_PER_PAGE,
  formatOverviewPageUrl,
  getPageSize,
  getPaginationStart,
} from '../utils/urls.js';
import {navigateToUrl} from '../utils/app-router.js';
import {SHARED_STYLES} from '../css/shared-css.js';

@customElement('webstatus-overview-pagination')
export class WebstatusOverviewPagination extends LitElement {
  @state()
  totalCount: number | undefined = undefined;

  @state()
  start = 0; // Index of first result among total results.

  @state()
  pageSize = DEFAULT_ITEMS_PER_PAGE; // Number of items to display per page

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

        #items-per-page {
          align-self: center;
          color: var(--unimportant-text-color);
          font-size: var(--sl-input-font-size-small);
        }

        sl-select {
          align-self: center;
          display: inline-block;
          margin: 0 var(--content-padding-quarter) 0 var(--content-padding);
          width: 5em;
        }
      `,
    ];
  }

  formatUrlForOffset(offset: number): string {
    return formatOverviewPageUrl(this.location, {start: offset});
  }

  formatUrlForRelativeOffset(delta: number): string | undefined {
    const offset = this.start + delta;
    if (
      this.totalCount === undefined ||
      offset <= -this.pageSize ||
      offset >= this.totalCount
    ) {
      return undefined;
    }
    return this.formatUrlForOffset(Math.max(0, offset));
  }

  renderPageButtons(): TemplateResult {
    if (this.totalCount === undefined || this.totalCount === 0) {
      return html``;
    }
    const currentPage = Math.floor(this.start / this.pageSize);
    const numPages = Math.ceil(this.totalCount / this.pageSize);

    return html`
      ${map(
        range(numPages),
        i => html`
          <sl-button
            variant="text"
            class="page-button ${i === currentPage ? 'active' : ''}"
            href=${this.formatUrlForOffset(i * this.pageSize)}
          >
            ${i + 1}
          </sl-button>
        `
      )}
    `;
  }

  setItemsPerPage(event: Event): void {
    const target = event.target as HTMLInputElement;
    const newSize = parseInt(target.value);
    const url = formatOverviewPageUrl(this.location, {num: newSize});
    navigateToUrl(url);
  }

  renderItemsPerPage(): TemplateResult {
    const options = [25, 50, 100];
    if (!options.includes(this.pageSize)) {
      options.push(this.pageSize);
      options.sort((a, b) => a - b);
    }
    return html`
      <sl-select
        value="${this.pageSize}"
        size="small"
        @sl-change=${this.setItemsPerPage}
      >
        ${options.map(
          opt => html` <sl-option value=${opt}>${opt}</sl-option> `
        )}
      </sl-select>
      <span id="items-per-page"> items per page </span>
    `;
  }

  render(): TemplateResult {
    if (this.totalCount === undefined || this.totalCount === 0) {
      return html``;
    }

    this.start = getPaginationStart(this.location);
    this.pageSize = getPageSize(this.location);
    const prevUrl = this.formatUrlForRelativeOffset(-this.pageSize);
    const nextUrl = this.formatUrlForRelativeOffset(this.pageSize);

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

        ${this.renderItemsPerPage()}

        <div class="spacer"></div>
      </div>
    `;
  }
}
