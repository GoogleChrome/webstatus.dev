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

import {html, LitElement} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {BADGE_PARAMS_BY_TYPE} from '../utils/constants.js';

export type BadgeType = 'css' | 'html' | 'interop';

@customElement('webstatus-feature-badge')
export class WebstatusFeatureBadge extends LitElement {
  @state()
  badgeType?: BadgeType;

  render() {
    if (!this.badgeType) {
      return html``;
    }
    const badgeInfo = BADGE_PARAMS_BY_TYPE[this.badgeType];
    return html`<sl-tag
      size="small"
      variant="${badgeInfo.name === 'CSS' ? 'success' : 'primary'}"
      pill
    >
      <div class="survey-result" title="${badgeInfo.description}">
        <span class="survey-result-span">
          <a href="${badgeInfo.url}" target="_blank">${badgeInfo.name}</a>
        </span>
      </div>
    </sl-tag>`;
  }
}
