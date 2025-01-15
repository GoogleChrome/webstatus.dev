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
import {CSSResultGroup, LitElement, css, html} from 'lit';
import {customElement, property, query, state} from 'lit/decorators.js';
import {SHARED_STYLES} from '../css/shared-css.js';

export interface DateRangeChangeEvent {
  startDate: Date;
  endDate: Date;
}

/**
 * @summary Date range picker
 * @event CustomEvent<DateRangeChangeEvent> webstatus-date-range-change - Emitted when the the date range is changed.
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

  @query('#date-range-picker-btn')
  submitBtn?: SlButton;

  @state()
  private _pendingStartDate = false;

  @state()
  private _pendingEndDate = false;

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
    const currentStartDate = this.startDate;
    const newStartDate = new Date(this.startDateEl?.valueAsDate || '');
    if (
      !this.isValidDate(newStartDate) ||
      this.minimumDate > newStartDate ||
      this.endDate < newStartDate
    ) {
      this.startDateEl?.setCustomValidity(
        `Date range should be ${this.toIsoDate(this.minimumDate)} to ${this.toIsoDate(this.endDate)} inclusive`,
      );
      this.startDateEl?.reportValidity();
      this._pendingStartDate = false;
      return;
    }
    if (newStartDate.getTime() !== currentStartDate.getTime()) {
      this.startDateEl?.setCustomValidity('');
      this.startDateEl?.reportValidity();
      this.startDate = newStartDate;
      this._pendingStartDate = true;
    }
  }

  async handleEndDateChange(_: SlInputEvent) {
    const currentEndDate = this.endDate;
    const newEndDate = new Date(this.endDateEl?.valueAsDate || '');
    if (
      !this.isValidDate(newEndDate) ||
      this.startDate > newEndDate ||
      this.maximumDate < newEndDate
    ) {
      this.endDateEl?.setCustomValidity(
        `Date range should be ${this.toIsoDate(this.startDate)} to ${this.toIsoDate(this.maximumDate)} inclusive`,
      );
      this.endDateEl?.reportValidity();
      this._pendingEndDate = false;
      return;
    }
    if (newEndDate.getTime() !== currentEndDate.getTime()) {
      this.endDateEl?.setCustomValidity('');
      this.endDateEl?.reportValidity();
      this.endDate = newEndDate;
      this._pendingEndDate = true;
    }
  }

  handleSubmit() {
    // Reset pending flags
    this._pendingStartDate = false;
    this._pendingEndDate = false;

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
    return this._pendingStartDate || this._pendingEndDate;
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
            .valueAsDate="${this.startDate}"
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
            .valueAsDate="${this.endDate}"
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
