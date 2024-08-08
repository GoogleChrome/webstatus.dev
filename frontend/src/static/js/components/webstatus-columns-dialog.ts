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

import {formatOverviewPageUrl, getColumnsSpec} from '../utils/urls.js';
import {
  ColumnKey,
  parseColumnsSpec,
  CELL_DEFS,
} from './webstatus-overview-cells.js';

import {SHARED_STYLES} from '../css/shared-css.js';

@customElement('webstatus-columns-dialog')
export class WebstatusColumnsDialog extends LitElement {
  @state()
  location!: {search: string}; // Set by openWithContext().

  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        #checkboxes {
          gap: var(--content-padding);
        }
        #button-row {
          padding-top: var(--content-padding);
        }

        sl-dialog::part(body) {
          padding-top: 0;
        }
      `,
    ];
  }

  openWithContext(location: {search: string}) {
    this.location = location;
    const dialog = this.shadowRoot?.querySelector('sl-dialog');
    if (dialog?.show) dialog.show();
  }

  hide() {
    const dialog = this.shadowRoot?.querySelector('sl-dialog');
    if (dialog?.hide) dialog.hide();
  }

  handleSave() {
    const newColumns: string[] = [];
    this.shadowRoot!.querySelectorAll('sl-checkbox').forEach(cb => {
      if (cb.checked) {
        newColumns.push(cb.value);
      }
    });
    this.hide();
    const nextUrl = this.formatUrlWithColumns(newColumns);
    window.location.href = nextUrl;
  }

  formatUrlWithColumns(columns: string[]): string {
    return formatOverviewPageUrl(this.location, {columns});
  }

  renderDialogContent(): TemplateResult {
    if (!this.location) return html``;
    const columns: ColumnKey[] = parseColumnsSpec(
      getColumnsSpec(this.location)
    );
    const checkboxes: TemplateResult[] = [];
    for (const enumKeyStr of Object.keys(ColumnKey)) {
      const ck = enumKeyStr as keyof typeof ColumnKey;
      const columnId = ColumnKey[ck];
      const displayName = CELL_DEFS[columnId].nameInDialog;
      // For baseline status, include options to show the low and high dates.
      let baselineStatusOptions = html``;
      if (columnId === ColumnKey.BaselineStatus) {
        baselineStatusOptions = html`
          <sl-tree-item expanded>
            <sl-checkbox
              value="low_date"
              ?checked=${columns.includes('low_date' as ColumnKey)}
              >Show Baseline status low date</sl-checkbox
            >
          </sl-tree-item>
          <sl-tree-item expanded>
            <sl-checkbox
              value="high_date"
              ?checked=${columns.includes('high_date' as ColumnKey)}
              >Show Baseline status high date</sl-checkbox
            >
          </sl-tree-item>
        `;
      }

      checkboxes.push(html`
        <sl-tree-item expanded>
          <sl-checkbox
            value="${columnId}"
            ?checked=${columns.includes(ColumnKey[ck])}
          >
            ${displayName}
          </sl-checkbox>
          ${baselineStatusOptions}
        </sl-tree-item>
      `);
    }
    const tree = html`<sl-tree>${checkboxes}</sl-tree>`;
    return html`
      <div class="vbox" id="checkboxes">${tree}</div>
      <div id="button-row">
        <sl-button
          id="columns-save-button"
          size="small"
          variant="primary"
          @click=${this.handleSave}
          >Save
        </sl-button>
      </div>
    `;
  }

  render(): TemplateResult {
    return html`
      <sl-dialog label="Select table columns" style="--width:fit-content">
        ${this.renderDialogContent()}
      </sl-dialog>
    `;
  }
}

let columnsDialogEl: WebstatusColumnsDialog | null = null;

export async function openColumnsDialog(location: {
  search: string;
}): Promise<WebstatusColumnsDialog> {
  if (!columnsDialogEl) {
    columnsDialogEl = document.createElement(
      'webstatus-columns-dialog'
    ) as WebstatusColumnsDialog;
    document.body.appendChild(columnsDialogEl);
    await columnsDialogEl.updateComplete;
  }
  columnsDialogEl.openWithContext(location);
  return columnsDialogEl;
}
