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

import {
  SlButton,
  SlChangeEvent,
  SlInput,
  SlInputEvent,
} from '@shoelace-style/shoelace';
import {CSSResultGroup, LitElement, PropertyValues, css, html} from 'lit';
import {customElement, property, query, state} from 'lit/decorators.js';
import {SHARED_STYLES} from '../css/shared-css.js';

export interface DateRangeChangeEvent {
  startDate: Date;
  endDate: Date;
}

/**
 * @summary Date range picker
 * @event CustomEvent<DateRangeChangeEvent> webstatus-date-range-change - Emitted when the the date range is changed.
 * @property {Date} minimumDate - The minimum selectable date. **Required.**
 * @property {Date} maximumDate - The maximum selectable date. **Required.**
 * @property {Date} startDate - The initial start date for the range. **Required.**
 * @property {Date} endDate - The initial end date for the range. **Required.**
 */
@customElement('webstatus-form-date-range-picker')
export class WebstatusFormDateRangePicker extends LitElement {
  @property({type: Object})
  minimumDate?: Date;

  @property({type: Object})
  maximumDate?: Date;

  @property({type: Object})
  startDate?: Date;

  @property({type: Object})
  endDate?: Date;

  @query('#start-date')
  readonly startDateEl?: SlInput;

  @query('#end-date')
  readonly endDateEl?: SlInput;

  @query('#date-range-picker-btn')
  readonly submitBtn?: SlButton;

  @state()
  private _startHasChanged = false;

  @state()
  private _endHasChanged = false;

  updated(changedProperties: PropertyValues<this>) {
    if (
      (changedProperties.has('minimumDate') ||
        changedProperties.has('maximumDate') ||
        changedProperties.has('startDate') ||
        changedProperties.has('endDate')) &&
      (!this.minimumDate ||
        !this.maximumDate ||
        !this.startDate ||
        !this.endDate)
    ) {
      const errorMessage =
        'WebstatusFormDateRangePicker: minimumDate, maximumDate, startDate, and endDate are required properties.';
      // Print the error and throw an error.
      console.error(errorMessage);
      throw new Error(errorMessage);
    }
  }

  isValidDate(d: Date): boolean {
    return !isNaN(d.getTime());
  }

  toIsoDate(date?: Date): string {
    return date?.toISOString().slice(0, 10) ?? '';
  }

  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        .hbox,
        .vbox {
          gap: var(--content-padding-large);
        }
        #date-range-picker-btn {
          justify-content: center;
          margin-top: var(--sl-input-label-font-size-medium);
        }
      `,
    ];
  }

  showPicker(input?: SlInput) {
    input?.showPicker();
  }

  async handleStartDateChange(_: SlChangeEvent) {
    const newStartDate = new Date(this.startDateEl?.valueAsDate || '');
    if (
      !this.isValidDate(newStartDate) ||
      (this.minimumDate && this.minimumDate > newStartDate) ||
      (this.endDate && this.endDate < newStartDate)
    ) {
      this.startDateEl?.setCustomValidity(
        `Date range should be ${this.toIsoDate(this.minimumDate)} to ${this.toIsoDate(this.endDate)} inclusive`,
      );
      this.startDateEl?.reportValidity();
      this._startHasChanged = false;
      return;
    }

    const currentStartDate = this.startDate;
    if (newStartDate.getTime() !== currentStartDate?.getTime()) {
      this.startDateEl?.setCustomValidity('');
      this.startDateEl?.reportValidity();
      this.startDate = newStartDate;
      this._startHasChanged = true;
    }
  }

  async handleEndDateChange(_: SlInputEvent) {
    const newEndDate = new Date(this.endDateEl?.valueAsDate || '');
    if (
      !this.isValidDate(newEndDate) ||
      (this.startDate && this.startDate > newEndDate) ||
      (this.maximumDate && this.maximumDate < newEndDate)
    ) {
      this.endDateEl?.setCustomValidity(
        `Date range should be ${this.toIsoDate(this.startDate)} to ${this.toIsoDate(this.maximumDate)} inclusive`,
      );
      this.endDateEl?.reportValidity();
      this._endHasChanged = false;
      return;
    }

    const currentEndDate = this.endDate;
    if (newEndDate.getTime() !== currentEndDate?.getTime()) {
      this.endDateEl?.setCustomValidity('');
      this.endDateEl?.reportValidity();
      this.endDate = newEndDate;
      this._endHasChanged = true;
    }
  }

  handleSubmit() {
    // Reset pending flags
    this._startHasChanged = false;
    this._endHasChanged = false;

    if (!this.startDate || !this.endDate) return;

    // Dispatch a single event with both dates
    const event = new CustomEvent<DateRangeChangeEvent>(
      'webstatus-date-range-change',
      {
        detail: {
          startDate: this.startDate,
          endDate: this.endDate,
        },
      },
    );
    this.dispatchEvent(event);
  }

  isSubmitButtonEnabled() {
    // Only enable the button if there component has validated new date(s) that
    // are ready to be emitted.
    return this._startHasChanged || this._endHasChanged;
  }
  render() {
    return html`
      <div class="hbox wrap">
        <label>
          Start date
          <sl-input
            id="start-date"
            @sl-change=${this.handleStartDateChange}
            @click=${() => {
              this.showPicker(this.startDateEl);
            }}
            type="date"
            .min=${this.toIsoDate(this.minimumDate)}
            .max=${this.toIsoDate(this.endDate)}
            .valueAsDate="${this.startDate ?? null}"
          ></sl-input>
        </label>
        <label>
          End date
          <sl-input
            id="end-date"
            @sl-change=${this.handleEndDateChange}
            @click=${() => {
              this.showPicker(this.endDateEl);
            }}
            type="date"
            .min=${this.toIsoDate(this.startDate)}
            .max=${this.toIsoDate(this.maximumDate)}
            .valueAsDate="${this.endDate ?? null}"
          ></sl-input>
        </label>
        <sl-button
          id="date-range-picker-btn"
          class="vbox"
          variant="primary"
          size="medium"
          ?disabled=${!this.isSubmitButtonEnabled()}
          @click=${this.handleSubmit}
        >
          <sl-icon name="search" slot="prefix"></sl-icon>
        </sl-button>
      </div>
    `;
  }
}
