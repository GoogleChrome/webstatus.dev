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

import {CSSResultGroup, LitElement, css, html} from 'lit';
import {getDateRange, updatePageUrl} from '../utils/urls.js';
import {DateChangeEvent} from './webstatus-form-date-range-picker.js';
import './webstatus-form-date-range-picker.js';
import {IndexedParams} from '@vaadin/router';
import {SHARED_STYLES} from '../css/shared-css.js';
import {state} from 'lit/decorators.js';

// Date.now()
const DEFAULT_END_DATE = new Date(Date.now());
// Date.now() - 1 year.
const DEFAULT_START_DATE = new Date(Date.now() - 365 * 24 * 60 * 60 * 1000);

const DEFAULT_MINIMUM_DATE = new Date(2000, 0, 1);

// 1 day after DEFAULT_END_DATE
const DEFAULT_MAXIMUM_DATE = new Date(
  DEFAULT_END_DATE.getTime() + 24 * 60 * 60 * 1000,
);

export class BaseChartsPage extends LitElement {
  minDate: Date = DEFAULT_MINIMUM_DATE;
  maxDate: Date = DEFAULT_MAXIMUM_DATE;
  // Make startDate and endDate reactive so that @lit/task can detect the changes.
  // TODO: Remove the state decorators this when we move the loading task into a non-page component.
  @state()
  startDate: Date = DEFAULT_START_DATE;
  @state()
  endDate: Date = DEFAULT_END_DATE;

  location!: {params: IndexedParams; search: string; pathname: string}; // Set by router.

  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        .hbox,
        .vbox {
          gap: var(--content-padding-large);
        }
      `,
    ];
  }

  async firstUpdated(): Promise<void> {
    // Get date range from query parameters.
    const dateRange = getDateRange({search: location.search});
    if (dateRange) {
      // Use default values if the URL dates are invalid
      this.startDate =
        dateRange.start &&
        dateRange.start >= this.minDate &&
        dateRange.start <= this.maxDate
          ? dateRange.start
          : DEFAULT_START_DATE;

      this.endDate =
        dateRange.end &&
        dateRange.end >= this.minDate &&
        dateRange.end <= this.maxDate &&
        dateRange.end >= this.startDate
          ? dateRange.end
          : DEFAULT_END_DATE;

      // Update the URL with the potentially reset dates
      // TODO. We should display a message that we reset the values.
      updatePageUrl(this.location.pathname, this.location, {
        dateRange: {start: this.startDate, end: this.endDate},
      });
    }
  }

  async handleStartDateChange(event: CustomEvent<DateChangeEvent>) {
    this.startDate = event.detail.date;
    updatePageUrl(this.location.pathname, this.location, {
      dateRange: {start: this.startDate, end: this.endDate},
    });
  }

  async handleEndDateChange(event: CustomEvent<DateChangeEvent>) {
    this.endDate = event.detail.date;
    updatePageUrl(this.location.pathname, this.location, {
      dateRange: {start: this.startDate, end: this.endDate},
    });
  }
  renderDateRangePicker() {
    return html`
      <webstatus-form-date-range-picker
        .startDate=${this.startDate}
        .endDate=${this.endDate}
        .minimumDate=${this.minDate}
        .maximumDate=${this.maxDate}
        @webstatus-start-date-change=${this.handleStartDateChange}
        @webstatus-end-date-change=${this.handleEndDateChange}
      ></webstatus-form-date-range-picker>
    `;
  }
}
