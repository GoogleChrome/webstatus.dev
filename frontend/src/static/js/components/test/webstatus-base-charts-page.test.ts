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
  BaseChartsPage,
  DEFAULT_END_DATE,
  DEFAULT_START_DATE,
} from '../webstatus-base-charts-page.js';
import {DateRangeChangeEvent} from '../webstatus-form-date-range-picker.js';
import {customElement} from 'lit/decorators.js';
import sinon from 'sinon';
import '../webstatus-base-charts-page.js';

// Create a subclass for testing purposes
@customElement('test-base-charts-page')
class TestBaseChartsPage extends BaseChartsPage {
  render() {
    return html`${this.renderDateRangePicker()}`;
  }
}

describe('BaseChartsPage', () => {
  const mockNow = new Date(2024, 5, 1).getTime(); // June 1, 2024
  const mockDefaultEndDate = new Date(mockNow);
  const mockDefaultStartDate = new Date(2023, 5, 2);
  // let el: TestBaseChartsPage;
  const location = {
    params: {},
    search: '',
    pathname: '/some-path',
  };

  let getDateRangeStub: sinon.SinonStub;
  let updatePageUrlStub: sinon.SinonStub;

  beforeEach(async () => {
    sinon.useFakeTimers({now: mockNow});
    // Change the values of the constants before creating the fixture
    DEFAULT_END_DATE.setTime(mockDefaultEndDate.getTime());
    DEFAULT_START_DATE.setTime(mockDefaultStartDate.getTime());
    getDateRangeStub = sinon.stub();
    updatePageUrlStub = sinon.stub();
  });

  afterEach(() => {
    sinon.restore();
  });

  it('should render the date range picker', async () => {
    const el = await fixture<TestBaseChartsPage>(
      html`<test-base-charts-page
        .location=${location}
      ></test-base-charts-page>`,
    );
    await el.updateComplete;
    const picker = el.shadowRoot!.querySelector(
      'webstatus-form-date-range-picker',
    );
    expect(picker).to.exist;
  });

  it('should initialize with default dates if no query params', async () => {
    const el = await fixture<TestBaseChartsPage>(
      html`<test-base-charts-page
        .location=${location}
        ._getDateRange=${getDateRangeStub}
        ._updatePageUrl=${updatePageUrlStub}
      ></test-base-charts-page>`,
    );
    getDateRangeStub.returns({});
    await el.updateComplete;
    expect(el.startDate).to.deep.equal(mockDefaultStartDate);
    expect(el.endDate).to.deep.equal(mockDefaultEndDate);
    expect(updatePageUrlStub).to.not.be.called;
  });

  it('should use default start date if start date param is invalid', async () => {
    location.search = '?startDate=invalid-date';

    getDateRangeStub.returns({start: new Date('invalid-date'), end: undefined});
    const el = await fixture<TestBaseChartsPage>(
      html`<test-base-charts-page
        .location=${location}
        ._getDateRange=${getDateRangeStub}
        ._updatePageUrl=${updatePageUrlStub}
      ></test-base-charts-page>`,
    );

    await el.updateComplete;

    expect(el.startDate).to.deep.equal(mockDefaultStartDate);
    expect(el.endDate).to.deep.equal(mockDefaultEndDate);
    expect(updatePageUrlStub).to.have.been.calledWith(
      location.pathname,
      location,
      {
        dateRange: {start: mockDefaultStartDate, end: mockDefaultEndDate},
      },
    );
  });

  it('should use default end date if start date param is invalid', async () => {
    location.search = '?endDate=invalid-date';

    getDateRangeStub.returns({start: undefined, end: new Date('invalid-date')});
    const el = await fixture<TestBaseChartsPage>(
      html`<test-base-charts-page
        .location=${location}
        ._getDateRange=${getDateRangeStub}
        ._updatePageUrl=${updatePageUrlStub}
      ></test-base-charts-page>`,
    );

    await el.updateComplete;

    expect(el.startDate).to.deep.equal(mockDefaultStartDate);
    expect(el.endDate).to.deep.equal(mockDefaultEndDate);
    expect(updatePageUrlStub).to.have.been.calledWith(
      location.pathname,
      location,
      {
        dateRange: {start: mockDefaultStartDate, end: mockDefaultEndDate},
      },
    );
  });

  it('should use default dates if both start and end date params are invalid', async () => {
    location.search = '?startDate=invalid-date&endDate=invalid-date';

    getDateRangeStub.returns({
      start: new Date('invalid-date'),
      end: new Date('invalid-date'),
    });
    const el = await fixture<TestBaseChartsPage>(
      html`<test-base-charts-page
        .location=${location}
        ._getDateRange=${getDateRangeStub}
        ._updatePageUrl=${updatePageUrlStub}
      ></test-base-charts-page>`,
    );

    await el.updateComplete;

    expect(el.startDate).to.deep.equal(mockDefaultStartDate);
    expect(el.endDate).to.deep.equal(mockDefaultEndDate);
    expect(updatePageUrlStub).to.have.been.calledWith(
      location.pathname,
      location,
      {
        dateRange: {start: mockDefaultStartDate, end: mockDefaultEndDate},
      },
    );
  });

  it('should use updateUrl if valid start date is provided', async () => {
    location.search = '?startDate=2023-06-15';
    const newStartDate = new Date(2023, 5, 15);

    getDateRangeStub.returns({start: newStartDate, end: undefined});
    const el = await fixture<TestBaseChartsPage>(
      html`<test-base-charts-page
        .location=${location}
        ._getDateRange=${getDateRangeStub}
        ._updatePageUrl=${updatePageUrlStub}
      ></test-base-charts-page>`,
    );

    await el.updateComplete;

    expect(el.startDate).to.deep.equal(newStartDate);
    expect(el.endDate).to.deep.equal(mockDefaultEndDate);
    expect(updatePageUrlStub).to.have.been.calledWith(
      location.pathname,
      location,
      {
        dateRange: {start: newStartDate, end: mockDefaultEndDate},
      },
    );
  });

  it('should use updateUrl if valid end date is provided', async () => {
    location.search = '?endDate=2023-06-15';
    const newEndDate = new Date(2023, 5, 15);

    getDateRangeStub.returns({start: undefined, end: newEndDate});
    const el = await fixture<TestBaseChartsPage>(
      html`<test-base-charts-page
        .location=${location}
        ._getDateRange=${getDateRangeStub}
        ._updatePageUrl=${updatePageUrlStub}
      ></test-base-charts-page>`,
    );

    await el.updateComplete;

    expect(el.startDate).to.deep.equal(mockDefaultStartDate);
    expect(el.endDate).to.deep.equal(newEndDate);
    expect(updatePageUrlStub).to.have.been.calledWith(
      location.pathname,
      location,
      {
        dateRange: {start: mockDefaultStartDate, end: newEndDate},
      },
    );
  });

  it('should use updateUrl if valid start and end dates are provided', async () => {
    location.search = '?startDate=2022-06-15&endDate=2023-06-15';
    const newStartDate = new Date(2022, 5, 15);
    const newEndDate = new Date(2023, 5, 15);

    getDateRangeStub.returns({start: newStartDate, end: newEndDate});
    const el = await fixture<TestBaseChartsPage>(
      html`<test-base-charts-page
        .location=${location}
        ._getDateRange=${getDateRangeStub}
        ._updatePageUrl=${updatePageUrlStub}
      ></test-base-charts-page>`,
    );

    await el.updateComplete;

    expect(el.startDate).to.deep.equal(newStartDate);
    expect(el.endDate).to.deep.equal(newEndDate);
    expect(updatePageUrlStub).to.have.been.calledWith(
      location.pathname,
      location,
      {
        dateRange: {start: newStartDate, end: newEndDate},
      },
    );
  });

  it('should handle date range change event and update URL', async () => {
    const newStartDate = new Date(2023, 5, 15);
    const newEndDate = new Date(2024, 10, 20);

    getDateRangeStub.returns({});
    const el = await fixture<TestBaseChartsPage>(
      html`<test-base-charts-page
        .location=${location}
        ._getDateRange=${getDateRangeStub}
        ._updatePageUrl=${updatePageUrlStub}
      ></test-base-charts-page>`,
    );

    // Simulate date range change
    const dateRangeEvent = new CustomEvent<DateRangeChangeEvent>(
      'webstatus-date-range-change',
      {detail: {startDate: newStartDate, endDate: newEndDate}},
    );
    el.shadowRoot
      ?.querySelector('webstatus-form-date-range-picker')
      ?.dispatchEvent(dateRangeEvent);

    // Assert that _updatePageUrl was called with the new start date and the current end date
    expect(updatePageUrlStub).to.have.been.calledWith(
      location.pathname,
      location,
      {
        dateRange: {start: newStartDate, end: newEndDate},
      },
    );

    expect(el.startDate).to.deep.equal(newStartDate);
    expect(el.endDate).to.deep.equal(newEndDate);
  });
});
