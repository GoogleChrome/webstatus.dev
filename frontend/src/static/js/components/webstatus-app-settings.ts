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

import { LitElement, TemplateResult, html } from "lit";
import { customElement, property } from "lit/decorators.js";
import { apiClientContext } from '../contexts/api-client-context.js'
import { APIClient } from '../api/client.js'
import { ContextProvider } from "@lit/context";
import { AppSettings } from "../../../common/app-settings.js";
import { appSettingsContext } from "../contexts/app-settings-context.js";


@customElement('webstatus-app-settings')
export class WebstatusAppSettings extends LitElement {
  @property({type: Object})
  appSettings!: AppSettings

  // Providers
  // We must create the context provider manually in connectedCallback because
  // currently SSR calls the constructor server side. And there's an issue
  // where context is not supported by SSR [1]. Once it is resolved, we can do
  // things like:
  //    @provide()
  //    @property()
  //    varName: type
  // [1] https://github.com/lit/lit/issues/3301
  apiClientProvider?:ContextProvider<typeof apiClientContext>;
  appSettingsProvider?: ContextProvider<typeof appSettingsContext>

  connectedCallback(): void {
    super.connectedCallback()
    console.log("found these settings")
    console.log(this.appSettings)
    this.apiClientProvider = new ContextProvider(this, { context: apiClientContext });
    this.apiClientProvider.setValue(new APIClient(this.appSettings.apiUrl))

    this.appSettingsProvider = new ContextProvider(this, {context: appSettingsContext})
    this.appSettingsProvider.setValue(this.appSettings)
  }

  protected render(): TemplateResult {
    return html`<slot></slot>`
  }
}