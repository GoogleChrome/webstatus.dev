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

import { consume } from '@lit/context'
import { LitElement, type TemplateResult, html } from 'lit'
import { customElement, state } from 'lit/decorators.js'
import { type components } from 'webstatus.dev-backend'

import { type APIClient } from '../api/client.js'
import { apiClientContext } from '../contexts/api-client-context.js'

@customElement('webstatus-feature-page')
export class FeaturePage extends LitElement {
  @consume({ context: apiClientContext })
  apiClient!: APIClient

  @state()
  feature?: components['schemas']['Feature'] | undefined

  id!: string

  @state()
  loading: boolean = true

  async firstUpdated(): Promise<void> {
    this.feature = await this.apiClient.getFeature(this.id)
    this.loading = false
  }

  render(): TemplateResult {
    if (this.loading) {
      return html`Loading`
    } else {
      return html`
        <h1>Feature Page</h1>
        spec size: ${this.feature?.spec != null ? this.feature.spec.length : 0}
        <br />
        Specs:
      `
    }
  }
}
