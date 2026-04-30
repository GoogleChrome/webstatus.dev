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

import {fixture, html, expect, oneEvent} from '@open-wc/testing';
import sinon from 'sinon';
import {WebstatusLoginPromptDialog} from '../webstatus-login-prompt-dialog.js';
import {
  AuthConfig,
  firebaseAuthContext,
} from '../../contexts/firebase-auth-context.js';
import {provide} from '@lit/context';
import {LitElement} from 'lit';
import {customElement, property} from 'lit/decorators.js';

import '../webstatus-login-prompt-dialog.js';

@customElement('test-auth-provider')
class TestAuthProvider extends LitElement {
  @property({type: Object})
  @provide({context: firebaseAuthContext})
  authConfig!: AuthConfig;

  render() {
    return html`<slot></slot>`;
  }
}

describe('webstatus-login-prompt-dialog', () => {
  let mockAuthConfig: AuthConfig;
  let signInStub: sinon.SinonStub;

  beforeEach(() => {
    signInStub = sinon.stub().resolves();
    mockAuthConfig = {
      auth: {} as any,
      provider: {} as any,
      icon: 'github',
      signIn: signInStub,
    };
  });

  it('renders nothing when closed', async () => {
    const el = await fixture<WebstatusLoginPromptDialog>(html`
      <webstatus-login-prompt-dialog
        .open=${false}
      ></webstatus-login-prompt-dialog>
    `);
    const dialog = el.shadowRoot?.querySelector('sl-dialog');
    expect(dialog?.open).to.be.false;
  });

  it('renders dialog when open', async () => {
    const el = await fixture<WebstatusLoginPromptDialog>(html`
      <webstatus-login-prompt-dialog
        .open=${true}
        .savedSearchName=${'My Search'}
      ></webstatus-login-prompt-dialog>
    `);
    const dialog = el.shadowRoot?.querySelector('sl-dialog');
    expect(dialog?.open).to.be.true;
    expect(el.shadowRoot?.textContent).to.contain('My Search');
  });

  it('calls signIn on button click and dispatches login-success', async () => {
    const wrapper = await fixture<TestAuthProvider>(html`
      <test-auth-provider .authConfig=${mockAuthConfig}>
        <webstatus-login-prompt-dialog
          .open=${true}
        ></webstatus-login-prompt-dialog>
      </test-auth-provider>
    `);
    const el = wrapper.querySelector<WebstatusLoginPromptDialog>(
      'webstatus-login-prompt-dialog',
    )!;
    const button = el.shadowRoot?.querySelector('sl-button[variant="primary"]');
    expect(button).to.exist;

    const eventPromise = oneEvent(el, 'login-success');
    (button as HTMLElement).click();

    await eventPromise;
    expect(signInStub.calledOnce).to.be.true;
    expect(el.open).to.be.false;
  });
});
