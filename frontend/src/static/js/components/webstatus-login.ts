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

import { LitElement, type TemplateResult, html } from 'lit'
import { customElement, property, query } from 'lit/decorators.js'

@customElement('webstatus-login')
export class WebstatusLogin extends LitElement {
  @property()
  declare public redirectTo: null | string

  @query('#login-container')
  protected container!: HTMLElement

  protected scriptInserted: boolean
  protected libraryLoaded: boolean

  constructor () {
    super()
    this.redirectTo = ''
    this.scriptInserted = false
    this.libraryLoaded = false
  }

  scriptLoaded (): void {
    this.initializeLibrary()
  }

  firstUpdated (): void {
    this.loadScript().then(
      // TODO. Success case
      () => {},
      // TODO. Failure case
      () => {}
    )
  }

  async loadScript (): Promise<void> {
    if (this.scriptInserted) {
      return
    }
    // Load the script.
    const script = document.createElement('script')
    script.src = 'https://accounts.google.com/gsi/client'
    document.head.appendChild(script)

    this.scriptInserted = true

    const promise = new Promise<void>((resolve) => {
      script.addEventListener('load', () => {
        resolve()
      })
    })

    // When the script is loaded, request an update.
    await promise.then(() => {
      this.scriptLoaded()
      this.requestUpdate()
    })
  }

  initializeLibrary (): void {
    if (this.libraryLoaded) {
      return
    }

    // @ts-expect-error TODO: figure out how to import nested namespace
    google.accounts.id.initialize({
      client_id: 'YOUR_GOOGLE_CLIENT_ID',
      // @ts-expect-error TODO: figure out how to import nested namespace
      callback: (response: google.accounts.id.CredentialResponse) => {
        this._signin(response.credential).then(() => {
          // window.location.href = this.redirectTo;
          console.log('hello')
        }, () => {
          console.log('something went wrong')
        })
      }
    })
    if (this.container !== null) {
      // @ts-expect-error TODO: figure out how to import nested namespace
      google.accounts.id.renderButton(
        this.container,
        { theme: 'outline', size: 'large', type: 'standard' } // customization attributes
      )
      // @ts-expect-error TODO: figure out how to import nested namespace
      google.accounts.id.prompt() // also display the One Tap dialog
    }

    this.libraryLoaded = true
  }

  render (): TemplateResult {
    return html`
      <script id="login-script" @load=${this.scriptLoaded} src="https://accounts.google.com/gsi/client"></script>
      <div id="login-container"></div>
    `
  }

  async _signin (token: string): Promise<void> {
    console.log(token)
  }
}
