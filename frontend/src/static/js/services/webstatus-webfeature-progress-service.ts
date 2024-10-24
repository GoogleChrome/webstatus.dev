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

import {customElement, property, state} from 'lit/decorators.js';
import {ServiceElement} from './service-element.js';
import {consume, provide} from '@lit/context';
import {AppSettings, appSettingsContext} from '../contexts/settings-context.js';
import {
  WebFeatureProgress,
  webFeatureProgressContext,
} from '../contexts/webfeature-progress-context.js';
import {PropertyValueMap} from 'lit';

@customElement('webstatus-webfeature-progress-service')
export class WebstatusWebFeatureProgressService extends ServiceElement {
  @consume({context: appSettingsContext, subscribe: true})
  @state()
  appSettings?: AppSettings;

  @provide({context: webFeatureProgressContext})
  @property({type: Object})
  progress?: WebFeatureProgress;

  async loadProgress(webFeaturesProgressUrl?: string) {
    // If we don't already have the url or we have already loaded it, return early.
    if (webFeaturesProgressUrl === undefined || this.progress !== undefined)
      return;

    try {
      const response = await fetch(webFeaturesProgressUrl);
      if (!response.ok) {
        this.progress = {
          error: `Received ${response.status} status trying to get web feature stats`,
        };
        return;
      }
      const data = await response.json();
      const progress: WebFeatureProgress = {};
      if (data.is_disabled) progress.isDisabled = data.is_disabled;
      if (data.bcd_map_progress)
        progress.bcdMapProgress = data.bcd_map_progress;
      this.progress = progress;
    } catch (e) {
      this.progress = {
        error: `Unexpected error ${e} trying to get web feature stats`,
      };
      return;
    }
  }

  protected async willUpdate(
    changedProperties: PropertyValueMap<this>
  ): Promise<void> {
    if (changedProperties.has('appSettings')) {
      await this.loadProgress(this.appSettings?.webFeaturesProgressUrl);
    }
  }
}
