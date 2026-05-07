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

import {fixture, html, expect, waitUntil} from '@open-wc/testing';
import sinon from 'sinon';
import {WebstatusSavedSearchShareDialog} from '../webstatus-saved-search-share-dialog.js';
import {UserSavedSearch} from '../../utils/constants.js';
import {Toast} from '../../utils/toast.js';

import '../webstatus-saved-search-share-dialog.js';

describe('webstatus-saved-search-share-dialog', () => {
  let el: WebstatusSavedSearchShareDialog;
  let toastStub: sinon.SinonStub;
  let clipboardStub: sinon.SinonStub;

  const mockSavedSearch: UserSavedSearch = {
    id: 'test-id',
    name: 'Test Search',
    query: 'feature:css',
    description: 'A test search',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  };

  beforeEach(async () => {
    toastStub = sinon.stub(Toast.prototype, 'toast').resolves();

    if (!navigator.clipboard) {
      Object.defineProperty(navigator, 'clipboard', {
        value: {
          writeText: async () => {},
        },
        writable: true,
      });
    }
    clipboardStub = sinon.stub(navigator.clipboard, 'writeText').resolves();

    el = await fixture<WebstatusSavedSearchShareDialog>(html`
      <webstatus-saved-search-share-dialog></webstatus-saved-search-share-dialog>
    `);
  });

  afterEach(() => {
    sinon.restore();
  });

  it('computes effectiveUrl correctly', async () => {
    el.shareableUrl = 'http://localhost:8080/features?q=saved:test-id';
    await el.updateComplete;
    expect(el.effectiveUrl).to.equal(
      'http://localhost:8080/features?q=saved%3Atest-id&subscribe=true',
    );
  });

  it('computes effectiveUrl correctly when fallback is needed', async () => {
    el.shareableUrl = 'http://localhost:8080/features';
    await el.updateComplete;
    expect(el.effectiveUrl).to.equal(
      'http://localhost:8080/features?subscribe=true',
    );
  });

  it('copies effectiveUrl to clipboard on button click', async () => {
    el.shareableUrl = 'http://localhost:8080/features?q=saved:test-id';
    await el.updateComplete;

    await el.openWithContext(mockSavedSearch, el.shareableUrl);
    await el.updateComplete;

    const dialog = el.shadowRoot?.querySelector('sl-dialog');
    expect(dialog).to.exist;

    const copyButton = dialog?.querySelector<HTMLElement>(
      'sl-button[variant="primary"]',
    );
    expect(copyButton).to.exist;

    copyButton!.click();

    await waitUntil(() => toastStub.calledOnce);

    expect(
      clipboardStub.calledOnceWith(
        'http://localhost:8080/features?q=saved%3Atest-id&subscribe=true',
      ),
    ).to.be.true;
    expect(toastStub.calledOnce).to.be.true;
  });

  it('renders QR code when open', async () => {
    el.shareableUrl = 'http://localhost:8080/features?q=saved:test-id';
    await el.updateComplete;

    await el.openWithContext(mockSavedSearch, el.shareableUrl);
    await el.updateComplete;

    const qrCode = el.shadowRoot?.querySelector('sl-qr-code');
    expect(qrCode).to.exist;
    expect(qrCode?.getAttribute('value')).to.equal(
      'http://localhost:8080/features?q=saved%3Atest-id&subscribe=true',
    );
  });
});
