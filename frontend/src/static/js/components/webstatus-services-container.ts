/**
 * Copyright 2024 Google LLC
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

import {LitElement, TemplateResult, html} from 'lit';
import {customElement, property} from 'lit/decorators.js';
import {AppSettings} from '../../../common/app-settings.js';
import '../services/webstatus-app-settings-service.js';
import '../services/webstatus-firebase-app-service.js';
import '../services/webstatus-firebase-auth-service.js';
import '../services/webstatus-gcharts-loader-service.js';
import '../services/webstatus-api-client-service.js';
import '../services/webstatus-bookmarks-service.js';
import '../services/webstatus-theme-service.js';

/**
 * WebstatusServiceContainer: Centralized container for web status services.
 *
 * This component manages the initialization and nesting of essential services
 * required for the webstatus application. It provides a structured environment
 * where services can share context and data, ensuring proper communication and
 * dependency management.
 *
 * The container acts as a parent for service components, allowing them to
 * access shared configurations and resources. It also provides a slot for
 * injecting UI components that depend on these services.
 *
 * Key Responsibilities:
 * - Initializes and nests core service components.
 * - Manages the propagation of configuration data to services.
 * - Provides a slot for UI components to access service context.
 */
@customElement('webstatus-services-container')
export class WebstatusServicesContainer extends LitElement {
  @property({type: Object})
  settings!: AppSettings;
  protected render(): TemplateResult {
    return html`
      <webstatus-gcharts-loader-service>
        <webstatus-app-settings-service .appSettings="${this.settings}">
          <webstatus-api-client-service .url="${this.settings.apiUrl}">
            <webstatus-firebase-app-service
              .settings="${this.settings.firebase.app}"
            >
              <webstatus-firebase-auth-service
                .settings="${this.settings.firebase.auth}"
              >
                <webstatus-bookmarks-service>
                  <webstatus-theme-service>
                    <slot></slot>
                  </webstatus-theme-service>
                </webstatus-bookmarks-service>
              </webstatus-firebase-auth-service>
            </webstatus-firebase-app-service>
          </webstatus-api-client-service>
        </webstatus-app-settings-service>
      </webstatus-gcharts-loader-service>
    `;
  }
}
