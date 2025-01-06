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

import {SlChangeEvent, SlInput, SlInputEvent} from '@shoelace-style/shoelace';
import {CSSResultGroup, LitElement, css, html} from 'lit';
import {customElement, property, query} from 'lit/decorators.js';
import {SHARED_STYLES} from '../css/shared-css.js';

export interface DateChangeEvent {
  date: Date;
}

/**
 * @summary Date range picker
 * @event CustomEvent<DateChangeEvent> webstatus-start-date-change - Emitted when the start date is changed.
 * @event CustomEvent<DateChangeEvent> webstatus-end-date-change - Emitted when the end date is changed.
 */
@customElement('webstatus-form-date-range-picker')
export class WebstatusFormDateRangePicker extends LitElement {
  @property({type: Object})
  minimumDate: Date = new Date();

  @property({type: Object})
  maximumDate: Date = new Date();

  @property({type: Object})
  startDate: Date = new Date();

  @property({type: Object})
  endDate: Date = new Date();

  @query('#start-date')
  startDateEl?: SlInput;

  @query('#end-date')
  endDateEl?: SlInput;

  isValidDate(d: Date): boolean {
    return !isNaN(d.getTime());
  }

  toIsoDate(date: Date): string {
    return date.toISOString().slice(0, 10);
  }

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

  async handleStartDateChange(_: SlChangeEvent) {
    const currentStartDate = this.startDate;
    const newStartDate = new Date(this.startDateEl?.value || '');
    if (
      !this.isValidDate(newStartDate) ||
      this.minimumDate > newStartDate ||
      this.endDate < newStartDate
    ) {
      this.startDateEl?.setCustomValidity(
        `Date range should be ${this.toIsoDate(this.minimumDate)} to ${this.toIsoDate(this.endDate)} inclusive`,
      );
      this.startDateEl?.reportValidity();
      return;
    }
    if (
      this.isValidDate(newStartDate) &&
      newStartDate.getTime() !== currentStartDate.getTime()
    ) {
      this.startDateEl?.setCustomValidity('');
      this.startDateEl?.reportValidity();
      this.startDate = newStartDate;
      const event = new CustomEvent<DateChangeEvent>(
        'webstatus-start-date-change',
        {
          detail: {
            date: this.startDate,
          },
        },
      );
      this.dispatchEvent(event);
    }
  }

  async handleEndDateChange(_: SlInputEvent) {
    const currentEndDate = this.endDate;
    const newEndDate = new Date(this.endDateEl?.value || '');
    if (
      !this.isValidDate(newEndDate) ||
      this.startDate > newEndDate ||
      this.maximumDate < newEndDate
    ) {
      this.endDateEl?.setCustomValidity(
        `Date range should be ${this.toIsoDate(this.startDate)} to ${this.toIsoDate(this.maximumDate)} inclusive`,
      );
      this.endDateEl?.reportValidity();
      return;
    }
    if (newEndDate.getTime() !== currentEndDate.getTime()) {
      // Clear the message.
      this.endDateEl?.setCustomValidity('');
      this.endDateEl?.reportValidity();
      this.endDate = newEndDate;
      const event = new CustomEvent<DateChangeEvent>(
        'webstatus-end-date-change',
        {
          detail: {
            date: this.endDate,
          },
        },
      );
      this.dispatchEvent(event);
    }
  }
  render() {
    return html`
      <div class="hbox wrap">
        <label>
          Start date
          <sl-input
            id="start-date"
            @sl-blur=${this.handleStartDateChange}
            type="date"
            .min=${this.toIsoDate(this.minimumDate)}
            .max=${this.toIsoDate(this.endDate)}
            .valueAsDate="${this.startDate}"
          ></sl-input>
        </label>
        <label>
          End date
          <sl-input
            id="end-date"
            @sl-blur=${this.handleEndDateChange}
            type="date"
            .min=${this.toIsoDate(this.startDate)}
            .max=${this.toIsoDate(this.maximumDate)}
            .valueAsDate="${this.endDate}"
          ></sl-input>
        </label>
      </div>
    `;
  }
}
