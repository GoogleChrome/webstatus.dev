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

import { ContextProvider } from '@lit/context'
import { LitElement, type TemplateResult, html } from 'lit'
import { customElement, property } from 'lit/decorators.js'

import { type AppSettings } from '../../../common/app-settings.js'
import { APIClient } from '../api/client.js'
import { apiClientContext } from '../contexts/api-client-context.js'
import { appSettingsContext } from '../contexts/settings-context.js'

@customElement('webstatus-app-settings')
export class WebstatusAppSettings extends LitElement {
  apiClientProvider?: ContextProvider<typeof apiClientContext>

  @property({ type: Object })
    appSettings!: AppSettings

  appSettingsProvider?: ContextProvider<typeof appSettingsContext>

  connectedCallback (): void {
    super.connectedCallback()
    this.apiClientProvider = new ContextProvider(this, { context: apiClientContext })
    this.apiClientProvider.setValue(new APIClient(this.appSettings.apiUrl))

    this.appSettingsProvider = new ContextProvider(this, { context: appSettingsContext })
    this.appSettingsProvider.setValue(this.appSettings)
  }

  protected render (): TemplateResult {
    return html`<slot></slot>`
  }
}
