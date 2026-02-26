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

import {provide} from '@lit/context';
import {customElement, state} from 'lit/decorators.js';
import {themeContext, type Theme} from '../contexts/theme-context.js';
import {ServiceElement} from './service-element.js';

const THEME_STORAGE_KEY = 'webstatus-theme';

@customElement('webstatus-theme-service')
export class WebstatusThemeService extends ServiceElement {
  @provide({context: themeContext})
  @state()
  theme: Theme = 'light';

  willUpdate() {
    this.applyTheme(this.getInitialTheme());
  }

  connectedCallback() {
    super.connectedCallback();
    this.addEventListener('theme-toggle', this.handleThemeToggle);

    // Listen for system theme changes
    window
      .matchMedia('(prefers-color-scheme: dark)')
      .addEventListener('change', this.handleSystemThemeChange);
  }

  disconnectedCallback() {
    this.removeEventListener('theme-toggle', this.handleThemeToggle);
    window
      .matchMedia('(prefers-color-scheme: dark)')
      .removeEventListener('change', this.handleSystemThemeChange);
    super.disconnectedCallback();
  }

  private getInitialTheme(): Theme {
    const savedTheme = localStorage.getItem(THEME_STORAGE_KEY);
    if (savedTheme === 'light' || savedTheme === 'dark') {
      return savedTheme;
    }
    return window.matchMedia('(prefers-color-scheme: dark)').matches
      ? 'dark'
      : 'light';
  }

  private handleThemeToggle = () => {
    const newTheme = this.theme === 'light' ? 'dark' : 'light';
    this.applyTheme(newTheme);
    localStorage.setItem(THEME_STORAGE_KEY, newTheme);
    this.requestUpdate();
  };

  private handleSystemThemeChange = (e: MediaQueryListEvent) => {
    // Only auto-switch if user hasn't set a preference?
    // For now, let's just stick to the current theme unless toggled.
    // Or if we want to follow system by default if no storage exists.
    if (!localStorage.getItem(THEME_STORAGE_KEY)) {
      const newTheme = e.matches ? 'dark' : 'light';
      this.applyTheme(newTheme);
    }
  };

  private applyTheme(theme: Theme) {
    if (theme === 'dark') {
      document.documentElement.classList.add('sl-theme-dark');
      document.documentElement.classList.remove('sl-theme-light');
    } else {
      document.documentElement.classList.remove('sl-theme-dark');
      document.documentElement.classList.add('sl-theme-light');
    }
    this.theme = theme;
  }
}
