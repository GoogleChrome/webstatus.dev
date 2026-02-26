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

import {consume} from '@lit/context';
import {LitElement, css, html} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {themeContext, Theme} from '../contexts/theme-context.js';

@customElement('webstatus-theme-button')
export class WebstatusThemeButton extends LitElement {
  static styles = css`
    .theme-toggle {
      margin-right: var(--content-padding);
      font-size: 1.5rem;
    }
  `;

  @consume({context: themeContext, subscribe: true})
  @state()
  theme: Theme | undefined = 'light';

  _fireEvent(eventName: string, detail: CustomEventInit | undefined): void {
    const event = new CustomEvent(eventName, {
      bubbles: true,
      composed: true,
      detail,
    });
    this.dispatchEvent(event);
  }

  handleThemeToggle(): void {
    this._fireEvent('theme-toggle', {});
  }

  render() {
    const isDark = this.theme === 'dark';
    const detectingTheme = this.theme === undefined;
    let themeBtnLabel = '';
    if (detectingTheme) {
      themeBtnLabel = 'Detecting theme...';
    } else {
      themeBtnLabel = isDark ? 'Switch to light theme' : 'Switch to dark theme';
    }
    return html`
      <sl-tooltip content="${themeBtnLabel}" placement="bottom">
        <sl-icon-button
          class="theme-toggle"
          name="${isDark ? 'sun' : 'moon'}"
          @click="${this.handleThemeToggle}"
          label="${themeBtnLabel}"
        ></sl-icon-button>
      </sl-tooltip>
    `;
  }
}
