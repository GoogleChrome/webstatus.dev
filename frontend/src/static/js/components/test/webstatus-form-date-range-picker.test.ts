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

import {expect, fixture, html} from '@open-wc/testing';
import {
  WebstatusFormDateRangePicker,
  DateChangeEvent,
} from '../webstatus-form-date-range-picker.js';
import {customElement, property} from 'lit/decorators.js';
import {LitElement} from 'lit';
import '../webstatus-form-date-range-picker.js';
import '@shoelace-style/shoelace/dist/components/input/input.js';

// TestComponent to listen for events from WebstatusFormDateRangePicker
@customElement('test-component')
class TestComponent extends LitElement {
  @property({type: Object}) startDate: Date | undefined;
  @property({type: Object}) endDate: Date | undefined;

  handleStartDateChange(event: CustomEvent<DateChangeEvent>) {
    this.startDate = event.detail.date;
  }

  handleEndDateChange(event: CustomEvent<DateChangeEvent>) {
    this.endDate = event.detail.date;
  }

  render() {
    return html`
      <webstatus-form-date-range-picker
        .minimumDate=${new Date(2023, 0, 1)}
        .maximumDate=${new Date(2024, 11, 31)}
        .startDate=${new Date(2023, 5, 1)}
        .endDate=${new Date(2023, 10, 31)}
        @webstatus-start-date-change=${this.handleStartDateChange}
        @webstatus-end-date-change=${this.handleEndDateChange}
      ></webstatus-form-date-range-picker>
    `;
  }
}

describe('WebstatusFormDateRangePicker', () => {
  let parent: TestComponent;
  let el: WebstatusFormDateRangePicker;

  beforeEach(async () => {
    // Create the parent component, which now renders the date picker
    parent = await fixture<TestComponent>(
      html`<test-component></test-component>`,
    );
    el = parent.shadowRoot!.querySelector<WebstatusFormDateRangePicker>(
      'webstatus-form-date-range-picker',
    )!;
  });

  it('should render the date range picker with default values', () => {
    const startDateInput = el.startDateEl!;
    const endDateInput = el.endDateEl!;

    expect(startDateInput).to.exist;
    expect(endDateInput).to.exist;
    expect(startDateInput.valueAsDate).to.deep.equal(el.startDate);
    expect(endDateInput.valueAsDate).to.deep.equal(el.endDate);
    expect(startDateInput.min).to.equal(el.toIsoDate(el.minimumDate));
    expect(startDateInput.max).to.equal(el.toIsoDate(el.endDate));
    expect(endDateInput.min).to.equal(el.toIsoDate(el.startDate));
    expect(endDateInput.max).to.equal(el.toIsoDate(el.maximumDate));
  });

  describe('Start Date Validation and Events', () => {
    it('should update start date and emit event when valid date is entered', async () => {
      const newStartDate = new Date(2023, 5, 15);

      el.startDateEl!.valueAsDate = newStartDate;
      await el.updateComplete;
      await parent.updateComplete;
      el.startDateEl!.dispatchEvent(new Event('sl-blur'));

      expect(el.startDate).to.deep.equal(newStartDate);
      expect(parent.startDate).to.deep.equal(newStartDate);
    });

    it('should not update start date if invalid date is entered', async () => {
      const newStartDate = new Date('invalid-date');

      el.startDateEl!.valueAsDate = newStartDate;
      await el.updateComplete;
      await parent.updateComplete;
      el.startDateEl!.dispatchEvent(new Event('sl-blur'));

      expect(el.startDate).to.deep.equal(el.startDate);
      expect(parent.startDate).to.be.undefined;
    });

    it('should not update start date if new date is before minimum date', async () => {
      const newStartDate = new Date(el.minimumDate.getFullYear() - 1, 0, 1);

      el.startDateEl!.valueAsDate = newStartDate;
      await el.updateComplete;
      await parent.updateComplete;
      el.startDateEl!.dispatchEvent(new Event('sl-blur'));

      expect(el.startDate).to.deep.equal(el.startDate);
      expect(parent.startDate).to.be.undefined;
    });

    it('should not update start date if new date is after end date', async () => {
      const newStartDate = new Date(el.endDate.getFullYear() + 1, 0, 1);

      el.startDateEl!.valueAsDate = newStartDate;
      await el.updateComplete;
      await parent.updateComplete;
      el.startDateEl!.dispatchEvent(new Event('sl-blur'));

      expect(el.startDate).to.deep.equal(el.startDate);
      expect(parent.startDate).to.be.undefined;
    });
  });

  describe('End Date Validation and Events', () => {
    it('should update end date and emit event when valid date is entered', async () => {
      const newEndDate = new Date(2023, 10, 15);

      el.endDateEl!.valueAsDate = newEndDate;
      await el.updateComplete;
      await parent.updateComplete;
      el.endDateEl!.dispatchEvent(new Event('sl-blur'));

      expect(el.endDate).to.deep.equal(newEndDate);
      expect(parent.endDate).to.deep.equal(newEndDate);
    });

    it('should not update end date if invalid date is entered', async () => {
      const newEndDate = new Date('invalid-date');

      el.endDateEl!.valueAsDate = newEndDate;
      await el.updateComplete;
      await parent.updateComplete;
      el.endDateEl!.dispatchEvent(new Event('sl-blur'));

      expect(el.endDate).to.deep.equal(el.endDate);
      expect(parent.endDate).to.be.undefined;
    });

    it('should not update end date if new date is before start date', async () => {
      const newEndDate = new Date(el.startDate.getFullYear() - 1, 0, 1);

      el.endDateEl!.valueAsDate = newEndDate;
      await el.updateComplete;
      await parent.updateComplete;
      el.endDateEl!.dispatchEvent(new Event('sl-blur'));

      expect(el.endDate).to.deep.equal(el.endDate);
      expect(parent.endDate).to.be.undefined;
    });

    it('should not update end date if new date is after maximum date', async () => {
      const newEndDate = new Date(el.maximumDate.getFullYear() + 1, 0, 1);

      el.endDateEl!.valueAsDate = newEndDate;
      await el.updateComplete;
      await parent.updateComplete;
      el.endDateEl!.dispatchEvent(new Event('sl-blur'));

      expect(el.endDate).to.deep.equal(el.endDate);
      expect(parent.endDate).to.be.undefined;
    });
  });
});
