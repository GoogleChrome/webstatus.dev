/**
 * Copyright 2025 Google LLC
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

import {fixture, html, assert} from '@open-wc/testing';
import sinon from 'sinon';
import {SubscribeButton} from '../webstatus-subscribe-button.js';
import {UserContext} from '../../contexts/firebase-user-context.js';
import {
  SubscriptionDeleteErrorEvent,
  SubscriptionSaveErrorEvent,
} from '../webstatus-manage-subscriptions-dialog.js';
import {SlDialog} from '@shoelace-style/shoelace';

describe('webstatus-subscribe-button', () => {
  const mockUserContext: UserContext = {
    user: {
      getIdToken: sinon.stub().resolves('test-token'),
    },
    loading: false,
  } as unknown as UserContext;

  it('renders button when user is logged in and savedSearchId is provided', async () => {
    const el = await fixture<SubscribeButton>(html`
      <webstatus-subscribe-button
        .userContext=${mockUserContext}
        .savedSearchId=${'test-id'}
      ></webstatus-subscribe-button>
    `);
    assert.isNotNull(el.shadowRoot?.querySelector('sl-button'));
  });

  it('does not render button when user is not logged in', async () => {
    const el = await fixture<SubscribeButton>(html`
      <webstatus-subscribe-button
        .userContext=${null}
        .savedSearchId=${'test-id'}
      ></webstatus-subscribe-button>
    `);
    assert.isNull(el.shadowRoot?.querySelector('sl-button'));
  });

  it('does not render button when savedSearchId is not provided', async () => {
    const el = await fixture<SubscribeButton>(html`
      <webstatus-subscribe-button
        .userContext=${mockUserContext}
        .savedSearchId=${''}
      ></webstatus-subscribe-button>
    `);
    assert.isNull(el.shadowRoot?.querySelector('sl-button'));
  });

  it('opens dialog on button click', async () => {
    const el = await fixture<SubscribeButton>(html`
      <webstatus-subscribe-button
        .userContext=${mockUserContext}
        .savedSearchId=${'test-id'}
      ></webstatus-subscribe-button>
    `);
    const button = el.shadowRoot?.querySelector('sl-button');
    const dialog = el.shadowRoot?.querySelector<SlDialog>(
      'webstatus-manage-subscriptions-dialog',
    );
    assert.isFalse(dialog?.open);
    button?.click();
    await el.updateComplete;
    assert.isTrue(dialog?.open);
  });

  it('calls toaster on successful save', async () => {
    const toasterSpy = sinon.spy();
    const el = await fixture<SubscribeButton>(html`
      <webstatus-subscribe-button
        .userContext=${mockUserContext}
        .savedSearchId=${'test-id'}
        .toaster=${toasterSpy}
      ></webstatus-subscribe-button>
    `);
    const dialog = el.shadowRoot?.querySelector(
      'webstatus-manage-subscriptions-dialog',
    );
    dialog?.dispatchEvent(new CustomEvent('subscription-save-success'));
    assert.isTrue(toasterSpy.calledWith('Subscription saved!', 'success'));
  });

  it('calls toaster on successful delete', async () => {
    const toasterSpy = sinon.spy();
    const el = await fixture<SubscribeButton>(html`
      <webstatus-subscribe-button
        .userContext=${mockUserContext}
        .savedSearchId=${'test-id'}
        .toaster=${toasterSpy}
      ></webstatus-subscribe-button>
    `);
    const dialog = el.shadowRoot?.querySelector(
      'webstatus-manage-subscriptions-dialog',
    );
    dialog?.dispatchEvent(new CustomEvent('subscription-delete-success'));
    assert.isTrue(toasterSpy.calledWith('Subscription deleted!', 'success'));
  });

  it('calls toaster on save error', async () => {
    const toasterSpy = sinon.spy();
    const el = await fixture<SubscribeButton>(html`
      <webstatus-subscribe-button
        .userContext=${mockUserContext}
        .savedSearchId=${'test-id'}
        .toaster=${toasterSpy}
      ></webstatus-subscribe-button>
    `);
    const dialog = el.shadowRoot?.querySelector(
      'webstatus-manage-subscriptions-dialog',
    );
    const error = new Error('Save failed');
    dialog?.dispatchEvent(new SubscriptionSaveErrorEvent(error));
    assert.isTrue(
      toasterSpy.calledWith('Error saving subscription: Save failed', 'danger'),
    );
  });

  it('calls toaster on delete error', async () => {
    const toasterSpy = sinon.spy();
    const el = await fixture<SubscribeButton>(html`
      <webstatus-subscribe-button
        .userContext=${mockUserContext}
        .savedSearchId=${'test-id'}
        .toaster=${toasterSpy}
      ></webstatus-subscribe-button>
    `);
    const dialog = el.shadowRoot?.querySelector(
      'webstatus-manage-subscriptions-dialog',
    );
    const error = new Error('Delete failed');
    dialog?.dispatchEvent(new SubscriptionDeleteErrorEvent(error));
    assert.isTrue(
      toasterSpy.calledWith(
        'Error deleting subscription: Delete failed',
        'danger',
      ),
    );
  });
});
