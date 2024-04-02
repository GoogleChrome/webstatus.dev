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

import { consume } from '@lit/context';
import { Task } from '@lit/task';
import {
  LitElement,
  type TemplateResult,
  html,
  CSSResultGroup,
  css,
  nothing,
} from 'lit';
import { customElement, state } from 'lit/decorators.js';
import { SHARED_STYLES } from '../css/shared-css.js';
import { type components } from 'webstatus.dev-backend';


import { type APIClient } from '../api/client.js';
import { apiClientContext } from '../contexts/api-client-context.js';


@customElement('webstatus-feature-page')
export class Chart extends LitElement {
  _loadingTask: Task;

  @consume({ context: apiClientContext })
  @state()
  apiClient!: APIClient;

  // @consume({ context: googleChartsContext })
  @state()
  googviz!: google.visualization.ChartWrapper;

  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
      :host {
        padding: 0;
        margin: 0;
        border: 0;
      }
  `];
  }

  constructor() {
    super();
    this._loadingTask = new Task(this, {
      args: () => [this.apiClient, this.googviz],
      task: async ([apiClient, googviz]) => {
        if (typeof apiClient === 'object') {
          // this.feature = await apiClient.getFeature(featureId);
        }
        return; // this.feature;
      },
    });
  }

  render() {
    return html`
      <div class="container">
      </div>
      `;
  }

}
