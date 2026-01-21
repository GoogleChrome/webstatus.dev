/**
 * Copyright 2026 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law of a an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import {consume} from '@lit/context';
import {LitElement, css, html} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {Task} from '@lit/task';

import {
  UserContext,
  firebaseUserContext,
} from '../contexts/firebase-user-context.js';
import {apiClientContext} from '../contexts/api-client-context.js';
import {APIClient} from '../api/client.js';
import {components} from 'webstatus.dev-backend';
import {toast} from '../utils/toast.js';
import {navigateToUrl} from '../utils/app-router.js';

import './webstatus-notification-email-channels.js';
import './webstatus-notification-rss-channels.js';
import './webstatus-notification-webhook-channels.js';

type NotificationChannelResponse =
  components['schemas']['NotificationChannelResponse'];

@customElement('webstatus-notification-channels-page')
export class WebstatusNotificationChannelsPage extends LitElement {
  static styles = css`
    .container {
      display: flex;
      flex-direction: column;
      gap: 16px;
    }
  `;

  @consume({context: firebaseUserContext, subscribe: true})
  @state()
  userContext: UserContext | null | undefined;

  @consume({context: apiClientContext})
  @state()
  apiClient!: APIClient;

  @state()
  private emailChannels: NotificationChannelResponse[] = [];

  private _channelsTask = new Task(this, {
    task: async () => {
      if (this.userContext === null) {
        navigateToUrl('/');
        void toast('You must be logged in to view this page.', 'danger');
        return;
      }
      if (this.userContext === undefined) {
        return;
      }

      const token = await this.userContext.user.getIdToken();
      const channels = await this.apiClient
        .listNotificationChannels(token)
        .catch(e => {
          const errorMessage = e instanceof Error ? e.message : 'unknown error';
          void toast(
            `Failed to load notification channels: ${errorMessage}`,
            'danger',
          );
          return [];
        });
      this.emailChannels = channels.filter(c => c.type === 'email');
    },
    args: () => [this.userContext],
  });

  render() {
    return html`
      <div class="container">
        ${this._channelsTask.render({
          pending: () => html`
            <webstatus-notification-email-channels .loading=${true}>
            </webstatus-notification-email-channels>

            <webstatus-notification-rss-channels .loading=${true}>
            </webstatus-notification-rss-channels>

            <webstatus-notification-webhook-channels .loading=${true}>
            </webstatus-notification-webhook-channels>
          `,

          complete: () => html`
            <webstatus-notification-email-channels
              .channels=${this.emailChannels}
            >
            </webstatus-notification-email-channels>
            <webstatus-notification-rss-channels>
            </webstatus-notification-rss-channels>
            <webstatus-notification-webhook-channels>
            </webstatus-notification-webhook-channels>
          `,
          error: e => {
            const errorMessage =
              e instanceof Error ? e.message : 'unknown error';
            return html`<p>Error: ${errorMessage}</p>`;
          },
        })}
      </div>
    `;
  }
}
