/**
 * Copyright 2026 Google LLC
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

import {LitElement} from 'lit';
import {property} from 'lit/decorators.js';
import {AppLocation} from '../utils/router-types.js';

/**
 * Base class for all page components in webstatus.dev.
 *
 * This provides a standard way to receive routing information and a centralized
 * location for future page-level instrumentation (e.g., analytics).
 */
export abstract class WebstatusBasePage extends LitElement {
  @property({type: Object})
  location!: AppLocation;
}
