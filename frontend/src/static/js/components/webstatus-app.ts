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

import { LitElement, html, type TemplateResult, type CSSResultGroup, isServer } from 'lit'
import { SHARED_STYLES } from '../css/shared-css.js'
import { customElement } from 'lit/decorators.js'
import './webstatus-header.js'
import './webstatus-page.js'
import './webstatus-app-settings.js'
import { ContextProvider } from '@lit/context'
import { apiClientContext } from '../contexts/api-client-context.js'
import { APIClient } from '../api/client.js'
import { SettingsMixin } from '../mixins/settings-mixin.js'

@customElement('webstatus-app')
export class WebstatusApp extends SettingsMixin(LitElement) {
  static get styles (): CSSResultGroup {
    return [
      SHARED_STYLES
    ]
  }

  connectedCallback(): void {
    super.connectedCallback()
    if(!isServer){
      console.log("adding provs")
      this.apiClientProvider = new ContextProvider(this, { context: apiClientContext });
      this.apiClientProvider.setValue(new APIClient('http://localhost:8080'))
    } else {
      console.log("not addings")
    }
  }

  apiClientProvider:any;
  // @provide({ context: apiClientContext })
  //   apiClient = new APIClient('http://localhost:8080')

  protected render (): TemplateResult {
    return html`
      <webstatus-app-settings apiURL=${this.apiURL}>
        <webstatus-header></webstatus-header>
        <webstatus-page>
          <slot></slot>
        </webstatus-page>
      </webstatus-app-settings>
    `
  }
}
