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

import {type SlAlert} from '@shoelace-style/shoelace';

// Escape HTML for text arguments
export function escapeHtml(html: string) {
  const div = document.createElement('div');
  div.textContent = html;
  return div.innerHTML;
}

// Custom function to emit toast notifications
export function toast(
  message: string,
  variant: SlAlert['variant'] = 'primary',
  icon: 'info-circle' | 'exclamation-triangle' = 'info-circle',
  duration = 3000
) {
  const alert = Object.assign(document.createElement('sl-alert'), {
    variant,
    closable: true,
    duration: duration,
    innerHTML: `
      <sl-icon name="${icon}" class="toast" slot="icon"></sl-icon>
      ${escapeHtml(message)}
    `,
  });

  document.body.append(alert);
  return alert.toast();
}
