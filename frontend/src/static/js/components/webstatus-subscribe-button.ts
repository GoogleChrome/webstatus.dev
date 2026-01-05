/**
 * Copyright 2025 Google LLC
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

import {LitElement, html, css, TemplateResult} from 'lit';
import {customElement, property} from 'lit/decorators.js';

export class SubscribeEvent extends CustomEvent<{savedSearchId: string}> {
  constructor(savedSearchId: string) {
    super('subscribe', {
      bubbles: true,
      composed: true,
      detail: {savedSearchId},
    });
  }
}

@customElement('webstatus-subscribe-button')
export class SubscribeButton extends LitElement {
  @property({type: String, attribute: 'saved-search-id'})
  savedSearchId = '';

  static styles = css`
    sl-button::part(base) {
      font-size: var(--sl-button-font-size-medium);
    }
  `;

  private _handleClick() {
    this.dispatchEvent(new SubscribeEvent(this.savedSearchId));
  }

  render(): TemplateResult {
    return html`
      <sl-button variant="primary" @click=${this._handleClick}>
        <sl-icon slot="prefix" name="bell"></sl-icon>
        Subscribe to updates
      </sl-button>
    `;
  }
}
