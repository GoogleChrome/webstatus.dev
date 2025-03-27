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

import {
  assert,
  elementUpdated,
  expect,
  fixture,
  html,
  waitUntil,
} from '@open-wc/testing';
import '../webstatus-overview-filters.js';
import sinon from 'sinon';

import {CSVUtils} from '../../utils/csv.js';
import {Toast} from '../../utils/toast.js';
import {WebstatusOverviewFilters} from '../webstatus-overview-filters.js';
import {APIClient} from '../../api/client.js';

import {stub} from 'sinon'; // Make sure you have sinon installed
import {bookmarkHelpers} from '../../contexts/app-bookmark-info-context.js';

it('should correctly update _activeQuery based on getCurrentQuery return value', async () => {
  const apiClient = new APIClient('');
  const location = {search: ''};

  const getCurrentQueryStub = stub(bookmarkHelpers, 'getCurrentQuery');

  // Test case 1: Empty query
  getCurrentQueryStub.returns('');
  let filterComponent = await fixture<WebstatusOverviewFilters>(
    html`<webstatus-overview-filters
      .location=${location}
      .apiClient=${apiClient}
    ></webstatus-overview-filters>`,
  );
  await elementUpdated(filterComponent);
  expect(filterComponent._activeQuery).to.eq('');

  // Test case 2: A specific query
  getCurrentQueryStub.returns('my-test-query');
  filterComponent = await fixture<WebstatusOverviewFilters>(
    html`<webstatus-overview-filters
      .location=${location}
      .apiClient=${apiClient}
    ></webstatus-overview-filters>`,
  );
  await elementUpdated(filterComponent);
  expect(filterComponent._activeQuery).to.eq('my-test-query');

  // Test case 3: Another query
  getCurrentQueryStub.returns('another-test-query');
  filterComponent = await fixture<WebstatusOverviewFilters>(
    html`<webstatus-overview-filters
      .location=${location}
      .apiClient=${apiClient}
    ></webstatus-overview-filters>`,
  );
  await elementUpdated(filterComponent);
  expect(filterComponent._activeQuery).to.eq('another-test-query');

  getCurrentQueryStub.restore();
});

describe('downloadCSV', () => {
  it('should display an error toast when the CSVUtils.downloadCSV function throws an error', async () => {
    const apiClient = new APIClient(''); // TODO Can probably stub allFeaturesFetecher instead.
    const getAllFeaturesStub = sinon.stub(apiClient, 'getAllFeatures');
    getAllFeaturesStub.resolves([]);
    const location = {search: ''};

    const filterComponent = await fixture<WebstatusOverviewFilters>(
      html`<webstatus-overview-filters
        .location=${location}
        .apiClient=${apiClient}
      ></webstatus-overview-filters>`,
    );

    assert.exists(filterComponent);
    await filterComponent.updateComplete;

    const toastStub = sinon.stub(Toast.prototype, 'toast');
    const downloadCSVStub = sinon.stub(CSVUtils, 'downloadCSV');
    downloadCSVStub.rejects(new Error('Test error'));

    // Click the 'Export to CSV' button
    const downloadButton =
      filterComponent.shadowRoot?.querySelector<HTMLButtonElement>(
        '#export-to-csv-button',
      );
    assert.exists(downloadButton);
    downloadButton.click();

    await elementUpdated(filterComponent);
    await waitUntil(
      () => filterComponent.exportDataStatus,
      'Export data status failed to change',
    );

    expect(downloadCSVStub.calledOnce).to.be.true;
    expect(toastStub.calledOnce).to.be.true;
    expect(
      toastStub.calledWith(
        'Save file error: Test error',
        'danger',
        'exclamation-triangle',
      ),
    ).to.be.true;
    expect(getAllFeaturesStub.calledOnce).to.be.true;
    downloadCSVStub.restore();
    toastStub.restore();
  });

  it('should display an error toast when the allFeaturesFetcher promise rejects', async () => {
    const apiClient = new APIClient('');
    const getAllFeaturesStub = sinon.stub(apiClient, 'getAllFeatures');
    getAllFeaturesStub.rejects(new Error('Test error'));

    const location = {search: ''};
    const filterComponent = await fixture<WebstatusOverviewFilters>(
      html`<webstatus-overview-filters
        .location=${location}
        .apiClient=${apiClient}
      ></webstatus-overview-filters>`,
    );
    assert.exists(filterComponent);
    await filterComponent.updateComplete;

    // Click the 'Export to CSV' button
    const downloadButton =
      filterComponent.shadowRoot?.querySelector<HTMLButtonElement>(
        '#export-to-csv-button',
      );
    assert.exists(downloadButton);

    const toastStub = sinon.stub(Toast.prototype, 'toast');
    downloadButton.click();
    await elementUpdated(filterComponent);
    await waitUntil(
      () => filterComponent.exportDataStatus,
      'Export data status failed to change',
    );

    expect(toastStub.calledOnce).to.be.true;
    expect(
      toastStub.calledWith(
        'Download features error: Test error',
        'danger',
        'exclamation-triangle',
      ),
    ).to.be.true;
    expect(getAllFeaturesStub.calledOnce).to.be.true;
    toastStub.restore();
  });
});
