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

import {TaskStatus} from '@lit/task';
import {LitElement, html, css, nothing} from 'lit';
import {customElement, state} from 'lit/decorators.js';

@customElement('webstatus-loading-overlay')
export class WebstatusLoadingOverlay extends LitElement {
  @state()
  status?: TaskStatus;

  static styles = css`
    .spinner-container {
      display: flex; /* Use flexbox for centering */
      align-items: center;
      justify-content: center;
      position: fixed; /* Position relative to viewport, not just the parent component */
      top: 0;
      left: 0;
      width: 100vw;
      height: 100vh;
      background-color: rgba(255, 255, 255, 0.7);
      z-index: 10;
      pointer-events: none;
    }
  `;

  render() {
    if (
      this.status === TaskStatus.COMPLETE ||
      this.status === TaskStatus.ERROR
    ) {
      return nothing; // Return nothing when not active
    }

    return html`
      <div class="spinner-container">
        <sl-spinner></sl-spinner>
      </div>
    `;
  }
}
