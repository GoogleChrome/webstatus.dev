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
  DateRangeChangeEvent,
} from '../webstatus-form-date-range-picker.js';
import {customElement, property} from 'lit/decorators.js';
import {LitElement} from 'lit';
import '../webstatus-form-date-range-picker.js';
import '@shoelace-style/shoelace/dist/components/input/input.js';
import '@shoelace-style/shoelace/dist/components/button/button.js';
import sinon from 'sinon';

// TestComponent to listen for events from WebstatusFormDateRangePicker
@customElement('test-component')
class TestComponent extends LitElement {
  @property({type: Object}) startDate: Date | undefined;
  @property({type: Object}) endDate: Date | undefined;

  handleDateRangeChange(event: CustomEvent<DateRangeChangeEvent>) {
    this.startDate = event.detail.startDate;
    this.endDate = event.detail.endDate;
  }

  render() {
    return html`
      <webstatus-form-date-range-picker
        .minimumDate=${new Date(2023, 0, 1)}
        .maximumDate=${new Date(2024, 11, 31)}
        .startDate=${new Date(2023, 5, 1)}
        .endDate=${new Date(2023, 10, 31)}
        @webstatus-date-range-change=${this.handleDateRangeChange}
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
    expect(el.submitBtn).to.exist;
    expect(startDateInput.valueAsDate).to.deep.equal(el.startDate);
    expect(endDateInput.valueAsDate).to.deep.equal(el.endDate);
    expect(startDateInput.min).to.equal(el.toIsoDate(el.minimumDate));
    expect(startDateInput.max).to.equal(el.toIsoDate(el.endDate));
    expect(endDateInput.min).to.equal(el.toIsoDate(el.startDate));
    expect(endDateInput.max).to.equal(el.toIsoDate(el.maximumDate));
  });

  describe('Initialization Validation', () => {
    it('should throw an error if minimumDate is not provided', async () => {
      try {
        await fixture(
          html`<webstatus-form-date-range-picker
            .maximumDate="${new Date('2024-01-01')}"
            .startDate="${new Date('2023-01-01')}"
            .endDate="${new Date('2023-12-31')}"
          ></webstatus-form-date-range-picker>`,
        );
        throw new Error('Expected an error to be thrown');
      } catch (error) {
        expect((error as Error).message).to.eq(
          'WebstatusFormDateRangePicker: minimumDate, maximumDate, startDate, and endDate are required properties.',
        );
      }
    });
    it('should throw an error if maximumDate is not provided', async () => {
      try {
        await fixture(
          html`<webstatus-form-date-range-picker
            .minimumDate="${new Date('2023-01-01')}"
            .startDate="${new Date('2023-01-01')}"
            .endDate="${new Date('2023-12-31')}"
          ></webstatus-form-date-range-picker>`,
        );
        throw new Error('Expected an error to be thrown');
      } catch (error: unknown) {
        expect((error as Error).message).to.eq(
          'WebstatusFormDateRangePicker: minimumDate, maximumDate, startDate, and endDate are required properties.',
        );
      }
    });

    it('should throw an error if startDate is not provided', async () => {
      try {
        await fixture(
          html`<webstatus-form-date-range-picker
            .minimumDate="${new Date('2023-01-01')}"
            .maximumDate="${new Date('2024-01-01')}"
            .endDate="${new Date('2023-12-31')}"
          ></webstatus-form-date-range-picker>`,
        );
        throw new Error('Expected an error to be thrown');
      } catch (error: unknown) {
        expect((error as Error).message).to.eq(
          'WebstatusFormDateRangePicker: minimumDate, maximumDate, startDate, and endDate are required properties.',
        );
      }
    });

    it('should throw an error if endDate is not provided', async () => {
      try {
        await fixture(
          html`<webstatus-form-date-range-picker
            .minimumDate="${new Date('2023-01-01')}"
            .maximumDate="${new Date('2024-01-01')}"
            .startDate="${new Date('2023-01-01')}"
          ></webstatus-form-date-range-picker>`,
        );
        throw new Error('Expected an error to be thrown');
      } catch (error: unknown) {
        expect((error as Error).message).to.eq(
          'WebstatusFormDateRangePicker: minimumDate, maximumDate, startDate, and endDate are required properties.',
        );
      }
    });
  });

  describe('showPicker', () => {
    it('should call showPicker on the startDateEl when clicked', async () => {
      // Stub showPicker to avoid the "NotAllowedError" in the unit test
      // since showPicker requires a user gesture.
      const showPickerStub = sinon.stub(el.startDateEl!, 'showPicker'); // Stub showPicker on startDateEl
      el.startDateEl?.click();
      expect(showPickerStub.calledOnce).to.be.true;
    });

    it('should call showPicker on the endDateEl when clicked', async () => {
      // Stub showPicker to avoid the "NotAllowedError" in the unit test
      // since showPicker requires a user gesture.
      const showPickerStub = sinon.stub(el.endDateEl!, 'showPicker'); // Stub showPicker on endDateEl
      el.endDateEl?.click();
      expect(showPickerStub.calledOnce).to.be.true;
    });
  });

  describe('Date Range Validation and Events', () => {
    it('should update both dates and emit a single event when valid dates are entered', async () => {
      expect(el.submitBtn?.disabled).to.be.true;
      const newStartDate = new Date(2023, 5, 15);
      const newEndDate = new Date(2023, 10, 16);

      el.startDateEl!.valueAsDate = newStartDate;
      el.endDateEl!.valueAsDate = newEndDate;
      await el.updateComplete;
      await parent.updateComplete;
      el.startDateEl!.dispatchEvent(new Event('sl-change'));
      el.endDateEl!.dispatchEvent(new Event('sl-change'));
      await el.updateComplete;
      await parent.updateComplete;

      // Simulate button click to submit
      expect(el.submitBtn?.disabled).to.be.false;
      el.submitBtn?.click();
      await el.updateComplete;
      await parent.updateComplete;
      expect(el.submitBtn?.disabled).to.be.true;

      expect(parent.startDate).to.deep.equal(newStartDate);
      expect(parent.endDate).to.deep.equal(newEndDate);
    });

    it('should not emit an event if no changes were made', async () => {
      // Button should be disabled.
      expect(el.submitBtn?.disabled).to.be.true;
      el.submitBtn?.click();
      await el.updateComplete;
      await parent.updateComplete;

      // Parent's start and end dates should be undefined
      expect(parent.startDate).to.be.undefined;
      expect(parent.endDate).to.be.undefined;
    });

    it('should not update if the start date is invalid', async () => {
      expect(el.submitBtn?.disabled).to.be.true;
      const newStartDate = new Date('invalid');

      el.startDateEl!.valueAsDate = newStartDate;
      await el.updateComplete;
      await parent.updateComplete;
      el.startDateEl!.dispatchEvent(new Event('sl-change'));
      el.endDateEl!.dispatchEvent(new Event('sl-change'));
      await el.updateComplete;
      await parent.updateComplete;

      // Button should still be disabled
      expect(el.submitBtn?.disabled).to.be.true;

      // Parent's start and end dates should be undefined
      expect(parent.startDate).to.be.undefined;
      expect(parent.endDate).to.be.undefined;
    });

    it('should not update if the end date is invalid', async () => {
      expect(el.submitBtn?.disabled).to.be.true;
      const newEndDate = new Date('invalid');

      el.endDateEl!.valueAsDate = newEndDate;
      await el.updateComplete;
      await parent.updateComplete;
      el.startDateEl!.dispatchEvent(new Event('sl-change'));
      el.endDateEl!.dispatchEvent(new Event('sl-change'));
      await el.updateComplete;
      await parent.updateComplete;

      // Button should still be disabled
      expect(el.submitBtn?.disabled).to.be.true;

      // Parent's start and end dates should be undefined
      expect(parent.startDate).to.be.undefined;
      expect(parent.endDate).to.be.undefined;
    });

    it('should not update if the start date is after the end date', async () => {
      expect(el.submitBtn?.disabled).to.be.true;
      const newStartDate = new Date(2024, 10, 15);
      el.startDateEl!.valueAsDate = newStartDate;
      await el.updateComplete;
      await parent.updateComplete;
      el.startDateEl!.dispatchEvent(new Event('sl-change'));
      el.endDateEl!.dispatchEvent(new Event('sl-change'));
      await el.updateComplete;
      await parent.updateComplete;

      expect(el.submitBtn?.disabled).to.be.true;

      // Parent's start and end dates should be undefined
      expect(parent.startDate).to.be.undefined;
      expect(parent.endDate).to.be.undefined;
    });
  });
});
