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

import { css, type CSSResultGroup, html, LitElement, type TemplateResult } from 'lit'
import { customElement, property } from 'lit/decorators.js'
import { type APIClient } from '../api/client.js'
import { type components } from 'webstatus.dev-backend'
import './webstatus-overview-sidebar.js'
import './webstatus-overview-content.js'
import { SHARED_STYLES } from '../css/shared-css.js'
import { consume } from '@lit/context'
import { apiClientContext } from '../contexts/api-client-context.js'
import { ContextConsumer } from '@lit/context'

@customElement('webstatus-overview-page')
export class OverviewPage extends LitElement {
  static get styles (): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
          @media (max-width: 768px) {
            webstatus-overview-sidebar {
              display: none;
            }
          }
          .container {
            display: flex;
            flex-direction: row;
          }
  
          webstatus-overview-sidebar {
            flex: 1;
          }

          webstatus-overview-content {
            flex: 2;
            padding-left: 20px;
            padding-right: 20px;
            padding-top: 10px;
          }

          webstatus-overview-sidebar {
            min-width: 300px;
            padding-left: 20px;
            padding-right: 20px;
            padding-top: 10px;
          }
  
          @media (max-width: 768px) {
            .container {
              flex-direction: column;
            }
          }
        `
    ]
  }

  @property()
    items: Array<components['schemas']['Feature']> = []

  @property()
    loading: boolean = true

  @consume({ context: apiClientContext })
    apiClient?: APIClient
  
    _consumer: any
  updated(): void {
    if (this._consumer.value) {
      console.log("found the thing")
      this.apiClient = this._consumer.value
    }
    console.log("finish looking for the thing")
  }

  connectedCallback(): void {
    super.connectedCallback()
    console.log("starting the thing")
    this._consumer = new ContextConsumer(this, {context: apiClientContext})
  }

  async firstUpdated (): Promise<void> {
    if(this.apiClient) 
      this.items = await this.apiClient.getFeatures()
    this.loading = false
  }

  render (): TemplateResult {
    return html`
        <div class="container">
          <webstatus-overview-sidebar></webstatus-overview-sidebar>
          <webstatus-overview-content></webstatus-overview-content>
        </div>
      `
  }
}
