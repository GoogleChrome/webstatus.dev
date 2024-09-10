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

import {assert, expect, fixture, html} from '@open-wc/testing';
import '../webstatus-feature-page.js';
import sinon from 'sinon';

import {CSVUtils} from '../../utils/csv.js';
import {Toast} from '../../utils/toast.js';
import {WebstatusOverviewFilters} from '../webstatus-overview-filters.js';

describe('downloadCSV', () => {
  it.skip('should display an error toast when the CSVUtils.downloadCSV function throws an error', async () => {
    const downloadCSVStub = sinon.stub(CSVUtils, 'downloadCSV');
    downloadCSVStub.throws(new Error('Test error'));

    const toast = new Toast();
    const toastStub = sinon.stub(toast, 'toast');

    const filterComponent = await fixture<WebstatusOverviewFilters>(
      html`<webstatus-overview-filters></webstatus-overview-filters>`
    );
    assert.exists(filterComponent);
    await filterComponent.updateComplete;

    // Click the 'Export to CSV' button
    const downloadButton = filterComponent.shadowRoot!.querySelector(
      '#export-to-csv-button'
    )! as HTMLButtonElement;
    downloadButton.click();

    expect(toastStub.calledOnce).to.be.true;
    expect(toastStub.calledWith('Save file error: Test error', 'danger')).to.be
      .true;

    downloadCSVStub.restore();
    toastStub.restore();
  });
});
