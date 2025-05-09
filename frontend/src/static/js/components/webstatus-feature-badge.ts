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

import {css, CSSResultGroup, html, LitElement} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {BADGE_PARAMS_BY_TYPE, BadgeType} from '../utils/constants.js';
import {SHARED_STYLES} from '../css/shared-css.js';

@customElement('webstatus-feature-badge')
export class WebstatusFeatureBadge extends LitElement {
  @state()
  badgeType?: BadgeType;

  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        .badge,
        .badge:hover,
        .badge a {
          font-size: 10px;
          text-decoration: none;
          cursor: help;
        }
      `,
    ];
  }

  render() {
    console.log(this.badgeType);
    if (!this.badgeType) {
      return html``;
    }
    const badgeInfo = BADGE_PARAMS_BY_TYPE[this.badgeType];
    return html`<sl-tag size="small" variant="${badgeInfo.variant}" pill>
      <div class="badge" title="${badgeInfo.description}">
        <span>
          <a href="${badgeInfo.url}" target="_blank">${badgeInfo.name}</a>
        </span>
      </div>
    </sl-tag>`;
  }
}
