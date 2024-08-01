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

/**
 * Base class for service components in LitElement.
 *
 * Service components typically don't render UI elements directly. Instead, they:
 *
 * 1. Encapsulate application logic, data fetching, or other non-visual functionality.
 * 2. Provide context to descendant components using Lit's `@lit/context` mechanism.
 *
 * Extending this class establishes a standardized structure for service component creation,
 * ensuring consistent integration within the LitElement application architecture.
 */
export class ServiceElement extends LitElement {
  protected render(): TemplateResult {
    return html`<slot></slot>`;
  }
}
