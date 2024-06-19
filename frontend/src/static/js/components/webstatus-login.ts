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

/// <reference types="@types/google.accounts" />

import {consume} from '@lit/context';
import {LitElement, type TemplateResult, html} from 'lit';
import {customElement, property, query, state} from 'lit/decorators.js';

import {LoadingState} from '../../../common/loading-state.js';
import {
  type AppSettings,
  appSettingsContext,
} from '../contexts/settings-context.js';

@customElement('webstatus-login')
export class WebstatusLogin extends LitElement {
  @consume({context: appSettingsContext})
  appSettings?: AppSettings;

  @query('#login-container')
  @state()
  protected container?: HTMLElement;

  protected libraryLoaded: LoadingState = LoadingState.NOT_STARTED;

  @property()
  public declare redirectTo: null | string;

  protected scriptInserted: boolean = false;

  constructor() {
    super();
    this.redirectTo = '';
  }

  async _signin(_token: string): Promise<void> {
    // TODO: Handle the token
  }

  firstUpdated(): void {
    this.loadScript().then(
      // TODO. Success case
      () => {},
      // TODO. Failure case
      () => {}
    );
  }

  initializeLibrary(): void {
    if (
      this.libraryLoaded === LoadingState.COMPLETE ||
      this.libraryLoaded === LoadingState.COMPLETE_WITH_ERRORS ||
      this.appSettings === undefined ||
      this.container === undefined ||
      this.appSettings?.gsiClientId === undefined
    ) {
      return;
    }

    google.accounts.id.initialize({
      callback: (response: google.accounts.id.CredentialResponse) => {
        this._signin(response.credential).then(
          () => {
            // TODO. Do successful redirect
          },
          () => {
            // TODO. Handle the error case
          }
        );
      },
      client_id: this.appSettings?.gsiClientId,
    });

    google.accounts.id.renderButton(this.container, {type: 'standard'});
    // TODO: Revisit this for playwright tests.
    // google.accounts.id.prompt();

    this.libraryLoaded = LoadingState.COMPLETE;
  }

  async loadScript(): Promise<void> {
    if (this.scriptInserted) {
      return;
    }

    // Load the script.
    const script = document.createElement('script');
    script.src = 'https://accounts.google.com/gsi/client';
    document.head.appendChild(script);

    this.scriptInserted = true;

    const promise = new Promise<void>(resolve => {
      script.addEventListener('load', () => {
        resolve();
      });
    });

    // When the script is loaded, request an update.
    await promise.then(() => {
      this.scriptLoaded();
      this.requestUpdate();
    });
  }

  render(): TemplateResult {
    return html` <div id="login-container"></div> `;
  }

  // TODO: remove eslint exemption when token handling is complete.
  scriptLoaded(): void {
    this.initializeLibrary();
  }
}
