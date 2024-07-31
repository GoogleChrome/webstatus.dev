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

import {provide} from '@lit/context';
import {customElement, property, state} from 'lit/decorators.js';

import {APIClient} from '../api/client.js';
import {apiClientContext} from '../contexts/api-client-context.js';
import {ServiceElement} from './service-element.js';
import {PropertyValueMap} from 'lit';

@customElement('webstatus-api-client-service')
export class WebstatusAPIClientService extends ServiceElement {
  @provide({context: apiClientContext})
  @state()
  apiClient!: APIClient;

  @property({type: String})
  url!: string;

  protected willUpdate(changedProperties: PropertyValueMap<this>): void {
    if (changedProperties.has('url')) {
      this.apiClient = new APIClient(this.url);
    }
  }
}
