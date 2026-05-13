/**
 * Copyright 2026 Google LLC
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

import {LitElement, TemplateResult, css, html} from 'lit';
import {customElement, property} from 'lit/decorators.js';
import {consume} from '@lit/context';
import {UserSavedSearch} from '../utils/constants.js';
import {themeContext, type Theme} from '../contexts/theme-context.js';
import {Toast} from '../utils/toast.js';
import {SHARED_STYLES} from '../css/shared-css.js';

@customElement('webstatus-saved-search-share-dialog')
export class WebstatusSavedSearchShareDialog extends LitElement {
  @property({type: Object})
  savedSearch?: UserSavedSearch;

  @consume({context: themeContext, subscribe: true})
  @property({attribute: false})
  theme?: Theme;

  @property({type: String})
  shareableUrl: string = '';

  get effectiveUrl(): string {
    try {
      const url = new URL(this.shareableUrl);
      url.searchParams.set('subscribe', 'true');
      return url.toString();
    } catch {
      // Fallback if shareableUrl is relative or invalid.
      const separator = this.shareableUrl.includes('?') ? '&' : '?';
      return `${this.shareableUrl}${separator}subscribe=true`;
    }
  }

  static styles = [
    SHARED_STYLES,
    css`
      sl-dialog::part(body) {
        padding-top: 0;
      }

      .qr-code-box {
        border: 1px solid var(--border-color, #ccc);
        border-radius: var(--border-radius, 4px);
        padding: var(--content-padding-half) var(--content-padding)
          var(--content-padding-quarter) var(--content-padding);
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: var(--content-padding);
        margin: 0 auto;
        margin-bottom: var(--content-padding-quarter);
        width: 25em;
      }

      .qr-code-box h3 {
        margin: 0.5em;
        font-weight: bold;
        font-size: 1rem;
      }

      .qr-code-box sl-button::part(base) {
        padding-bottom: var(--content-padding-quarter);
        font-weight: normal;
      }

      .link-share-box {
        display: grid;
        grid-template-columns: 1fr auto;
        grid-template-rows: auto auto;
        gap: 0;
        width: 100%;
        border-radius: var(--border-radius);
        align-items: stretch;
        margin-top: 1em;
      }

      .link-share-box h3 {
        grid-column: 1;
        grid-row: 1;
        margin: 0;
        font-size: 1rem;
        font-weight: bold;
      }

      .link-share-box sl-input {
        grid-column: 1;
        grid-row: 2;
      }

      .link-share-box sl-input::part(base) {
        border: none;
        background-color: transparent;
        box-shadow: none;
      }

      .link-share-box sl-input::part(input) {
        padding-left: 0;
        text-overflow: ellipsis;
        white-space: nowrap;
        overflow: hidden;
        font-size: 0.875rem;
      }

      .link-share-box sl-button {
        grid-column: 2;
        grid-row: 1 / span 2;
        align-self: stretch;
      }
    `,
  ];

  async openWithContext(savedSearch: UserSavedSearch, shareableUrl: string) {
    this.savedSearch = savedSearch;
    this.shareableUrl = shareableUrl;
    const dialog = this.shadowRoot?.querySelector('sl-dialog');
    if (dialog?.show) await dialog.show();
  }

  async hide() {
    const dialog = this.shadowRoot?.querySelector('sl-dialog');
    if (dialog?.hide) await dialog.hide();
  }

  async drawLogoOnCanvas(qrCodeElement: Element) {
    if (!qrCodeElement.shadowRoot) return;
    const canvas = qrCodeElement.shadowRoot.querySelector('canvas');
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const logo = new Image();
    logo.src = '/public/img/cross.svg';
    await logo.decode();

    const logoSize = canvas.width * 0.2;
    const x = (canvas.width - logoSize) / 2;
    const y = (canvas.height - logoSize) / 2;

    ctx.drawImage(logo, x, y, logoSize, logoSize);
  }

  saveQRCode() {
    const canvas = this.shadowRoot
      ?.querySelector('sl-qr-code')
      ?.shadowRoot?.querySelector('canvas');
    if (!canvas) return;
    const dataURL = canvas.toDataURL('image/png');
    const a = document.createElement('a');
    a.href = dataURL;
    a.download = `qr-code-${this.savedSearch?.id}.png`;
    a.click();
  }

  async copyToClipboard() {
    try {
      await navigator.clipboard.writeText(this.effectiveUrl);
      await new Toast().toast(
        'Link copied to clipboard',
        'success',
        'info-circle',
      );
    } catch (err) {
      console.error('Failed to copy: ', err);
      await new Toast().toast(
        'Failed to copy link',
        'danger',
        'exclamation-triangle',
      );
    }
  }

  render(): TemplateResult {
    const isDark = this.theme === 'dark';
    const fill = isDark ? 'white' : 'black';
    const background = isDark ? 'black' : 'white';

    return html`
      <sl-dialog
        label="Share bookmark"
        aria-label="Share bookmark"
        style="--width:fit-content"
      >
        <div class="vbox gap-large">
          <div class="qr-code-box">
            <h3>Share via QR code</h3>
            <sl-qr-code
              value="${this.effectiveUrl}"
              size="180"
              fill="${fill}"
              background="${background}"
              radius="0"
              error-correction="H"
              @sl-after-render=${(e: CustomEvent) => {
                if (e.target instanceof Element) {
                  void this.drawLogoOnCanvas(e.target);
                }
              }}
            ></sl-qr-code>
            <sl-button variant="text" @click=${this.saveQRCode}>
              <sl-icon name="download"></sl-icon> Save QR code
            </sl-button>
          </div>

          <div class="link-share-box">
            <h3>Share via link</h3>
            <sl-input value="${this.effectiveUrl}" readonly></sl-input>
            <sl-button
              data-testid="copy-link-button"
              variant="primary"
              @click=${this.copyToClipboard}
            >
              <sl-icon name="link"></sl-icon> Copy link
            </sl-button>
          </div>
        </div>
      </sl-dialog>
    `;
  }
}

let shareDialogEl: WebstatusSavedSearchShareDialog | null = null;

export async function openShareDialog(
  savedSearch: UserSavedSearch,
  shareableUrl: string,
): Promise<WebstatusSavedSearchShareDialog> {
  if (!shareDialogEl) {
    shareDialogEl = new WebstatusSavedSearchShareDialog();
    const app = document.querySelector('webstatus-app');
    const servicesContainer = app?.shadowRoot?.querySelector(
      'webstatus-services-container',
    );
    if (servicesContainer) {
      servicesContainer.appendChild(shareDialogEl);
    } else {
      document.body.appendChild(shareDialogEl);
    }
    await shareDialogEl.updateComplete;
  }
  await shareDialogEl.openWithContext(savedSearch, shareableUrl);
  return shareDialogEl;
}
