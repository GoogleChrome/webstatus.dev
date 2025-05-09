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

import {assert, expect, fixture} from '@open-wc/testing';

import {
  ColumnKey,
  parseColumnsSpec,
  DEFAULT_COLUMNS,
  isJavaScriptFeature,
  didFeatureCrash,
  parseColumnOptions,
  DEFAULT_COLUMN_OPTIONS,
  ColumnOptionKey,
  renderBaselineStatus,
  renderAvailablity,
  renderChromeUsage,
  renderHeaderCell,
  renderUnsortableHeaderCell,
  CELL_DEFS,
  calcColGroupSpans,
  renderColgroups,
  renderGroupsRow,
} from '../webstatus-overview-cells.js';
import {components} from 'webstatus.dev-backend';
import {render} from 'lit';

describe('renderChromeUsage', () => {
  let container: HTMLElement;
  let feature: components['schemas']['Feature'];
  beforeEach(() => {
    container = document.createElement('div');

    feature = {
      feature_id: 'id',
      name: 'name',
      baseline: {
        status: 'widely',
        low_date: '2015-07-29',
        high_date: '2018-01-29',
      },
      usage: {},
    };
  });
  it('does render usage as a percentage', async () => {
    // 10.3% Chrome usage.
    feature.usage = {chrome: {daily: 0.1034}};
    const result = renderChromeUsage(feature, {search: ''}, {});
    render(result, container);
    const el = await fixture(container);
    const usageEl = el.querySelector('#chrome-usage');
    expect(usageEl).to.exist;
    expect(usageEl!.textContent!.trim()).to.equal('10.3%');
  });
  it('does render usage as a percentage, and rounds correctly', async () => {
    // Should round to 10.4% Chrome usage.
    feature.usage = {chrome: {daily: 0.1036}};
    const result = renderChromeUsage(feature, {search: ''}, {});
    render(result, container);
    const el = await fixture(container);
    const usageEl = el.querySelector('#chrome-usage');
    expect(usageEl).to.exist;
    expect(usageEl!.textContent!.trim()).to.equal('10.4%');
  });
  it('does render 0 usage amounts as "0.0%"', async () => {
    // Explicitly 0 Chrome usage.
    feature.usage = {chrome: {daily: 0.0}};
    const result = renderChromeUsage(feature, {search: ''}, {});
    render(result, container);
    const el = await fixture(container);
    const usageEl = el.querySelector('#chrome-usage');
    expect(usageEl).to.exist;
    expect(usageEl!.textContent!.trim()).to.equal('0.0%');
  });
  it('does render 100% usage amounts as "100%"', async () => {
    // Explicitly 100% Chrome usage.
    feature.usage = {chrome: {daily: 1.0}};
    const result = renderChromeUsage(feature, {search: ''}, {});
    render(result, container);
    const el = await fixture(container);
    const usageEl = el.querySelector('#chrome-usage');
    expect(usageEl).to.exist;
    expect(usageEl!.textContent!.trim()).to.equal('100.0%');
  });
  it('does render very small usage amounts as "<0.1%"', async () => {
    // 0.0003%.
    feature.usage = {chrome: {daily: 0.000003}};
    const result = renderChromeUsage(feature, {search: ''}, {});
    render(result, container);
    const el = await fixture(container);
    const usageEl = el.querySelector('#chrome-usage');
    expect(usageEl).to.exist;
    expect(usageEl!.textContent!.trim()).to.equal('<0.1%');
  });
  it('does render null usage amounts as "N/A"', async () => {
    // No usage found.
    feature.usage = undefined;
    const result = renderChromeUsage(feature, {search: ''}, {});
    render(result, container);
    const el = await fixture(container);
    const usageEl = el.querySelector('#chrome-usage');
    expect(usageEl).to.exist;
    expect(usageEl!.textContent!.trim()).to.equal('N/A');
  });
});

describe('renderBaselineStatus', () => {
  let container: HTMLElement;
  beforeEach(() => {
    container = document.createElement('div');
  });
  describe('widely available feature', () => {
    const feature: components['schemas']['Feature'] = {
      feature_id: 'id',
      name: 'name',
      baseline: {
        status: 'widely',
        low_date: '2015-07-29',
        high_date: '2018-01-29',
      },
    };
    it('renders only the word and icon by default', async () => {
      const result = renderBaselineStatus(feature, {search: ''}, {});
      render(result, container);
      const el = await fixture(container);
      const icon = el.querySelector('img');
      expect(icon).to.exist;
      expect(icon!.getAttribute('title')).to.equal('Widely available');

      // Assert the absence of the low date block and the high date blocks.
      const lowDateBlock = el.querySelector('.baseline-date-block-newly');
      expect(lowDateBlock).to.not.exist;
      const highDateBlock = el.querySelector('.baseline-date-block-widely');
      expect(highDateBlock).to.not.exist;
    });
    it('additionally renders the low date when selected', async () => {
      const result = renderBaselineStatus(
        feature,
        {search: 'column_options=baseline_status_low_date'},
        {},
      );
      render(result, container);
      const el = await fixture(container);
      const icon = el.querySelector('img');
      expect(icon).to.exist;
      expect(icon!.getAttribute('title')).to.equal('Widely available');

      // Assert the presence of the low date block and absence of the high date block.
      const lowDateBlock = el.querySelector('.baseline-date-block-newly');
      expect(lowDateBlock).to.exist;
      expect(
        lowDateBlock
          ?.querySelector('.baseline-date-header')
          ?.textContent?.trim(),
      ).to.equal('Newly available:');
      expect(
        lowDateBlock?.querySelector('.baseline-date')?.textContent?.trim(),
      ).to.equal('2015-07-29');
      const highDateBlock = el.querySelector('.baseline-date-block-widely');
      expect(highDateBlock).to.not.exist;
    });
    it('additionally renders the high date when selected', async () => {
      const result = renderBaselineStatus(
        feature,
        {search: 'column_options=baseline_status_high_date'},
        {},
      );
      render(result, container);
      const el = await fixture(container);
      const icon = el.querySelector('img');
      expect(icon).to.exist;
      expect(icon!.getAttribute('title')).to.equal('Widely available');

      // Assert the presence of the high date block and absence of the low date block.
      const lowDateBlock = el.querySelector('.baseline-date-block-newly');
      expect(lowDateBlock).to.not.exist;
      const highDateBlock = el.querySelector('.baseline-date-block-widely');
      expect(highDateBlock).to.exist;
      expect(
        highDateBlock
          ?.querySelector('.baseline-date-header')
          ?.textContent?.trim(),
      ).to.equal('Widely available:');
      expect(
        highDateBlock?.querySelector('.baseline-date')?.textContent?.trim(),
      ).to.equal('2018-01-29');
    });
    it('additionally renders the low date and high date when both are selected', async () => {
      const result = renderBaselineStatus(
        feature,
        {
          search:
            'column_options=baseline_status_low_date%2Cbaseline_status_high_date',
        },
        {},
      );
      render(result, container);
      const el = await fixture(container);
      const icon = el.querySelector('img');
      expect(icon).to.exist;
      expect(icon!.getAttribute('title')).to.equal('Widely available');

      // Assert the presence of the low date block and the high date blocks.
      const lowDateBlock = el.querySelector('.baseline-date-block-newly');
      expect(lowDateBlock).to.exist;
      expect(
        lowDateBlock
          ?.querySelector('.baseline-date-header')
          ?.textContent?.trim(),
      ).to.equal('Newly available:');
      expect(
        lowDateBlock?.querySelector('.baseline-date')?.textContent?.trim(),
      ).to.equal('2015-07-29');
      const highDateBlock = el.querySelector('.baseline-date-block-widely');
      expect(highDateBlock).to.exist;
      expect(
        highDateBlock
          ?.querySelector('.baseline-date-header')
          ?.textContent?.trim(),
      ).to.equal('Widely available:');
      expect(
        highDateBlock?.querySelector('.baseline-date')?.textContent?.trim(),
      ).to.equal('2018-01-29');
    });
  });
  describe('newly available feature', () => {
    const feature: components['schemas']['Feature'] = {
      feature_id: 'id',
      name: 'name',
      baseline: {
        status: 'newly',
        low_date: '2015-07-29',
      },
    };
    it('renders only the word and icon by default', async () => {
      const result = renderBaselineStatus(feature, {search: ''}, {});
      render(result, container);
      const el = await fixture(container);
      const icon = el.querySelector('img');
      expect(icon).to.exist;
      expect(icon!.getAttribute('title')).to.equal('Newly available');

      // Assert the absence of the low date block and the high date blocks.
      const lowDateBlock = el.querySelector('.baseline-date-block-newly');
      expect(lowDateBlock).to.not.exist;
      const highDateBlock = el.querySelector('.baseline-date-block-widely');
      expect(highDateBlock).to.not.exist;
    });
    it('additionally renders the low date when selected', async () => {
      const result = renderBaselineStatus(
        feature,
        {search: 'column_options=baseline_status_low_date'},
        {},
      );
      render(result, container);
      const el = await fixture(container);
      const icon = el.querySelector('img');
      expect(icon).to.exist;
      expect(icon!.getAttribute('title')).to.equal('Newly available');

      // Assert the presence of the low date block and absence of the high date block.
      const lowDateBlock = el.querySelector('.baseline-date-block-newly');
      expect(lowDateBlock).to.exist;
      expect(
        lowDateBlock
          ?.querySelector('.baseline-date-header')
          ?.textContent?.trim(),
      ).to.equal('Newly available:');
      expect(
        lowDateBlock?.querySelector('.baseline-date')?.textContent?.trim(),
      ).to.equal('2015-07-29');
      const highDateBlock = el.querySelector('.baseline-date-block-widely');
      expect(highDateBlock).to.not.exist;
    });
    it('additionally renders the projected high date when selected', async () => {
      const result = renderBaselineStatus(
        feature,
        {search: 'column_options=baseline_status_high_date'},
        {},
      );
      render(result, container);
      const el = await fixture(container);
      const icon = el.querySelector('img');
      expect(icon).to.exist;
      expect(icon!.getAttribute('title')).to.equal('Newly available');

      // Assert the presence of the high date block and absence of the low date block.
      const lowDateBlock = el.querySelector('.baseline-date-block-newly');
      expect(lowDateBlock).to.not.exist;
      const highDateBlock = el.querySelector('.baseline-date-block-widely');
      expect(highDateBlock).to.exist;
      expect(
        highDateBlock
          ?.querySelector('.baseline-date-header')
          ?.textContent?.trim(),
      ).to.equal('Projected widely available:');
      expect(
        highDateBlock?.querySelector('.baseline-date')?.textContent?.trim(),
      ).to.equal('2018-01-29');
    });
    it('additionally renders the low date and projected high date when both are selected', async () => {
      const result = renderBaselineStatus(
        feature,
        {
          search:
            'column_options=baseline_status_low_date%2Cbaseline_status_high_date',
        },
        {},
      );
      render(result, container);
      const el = await fixture(container);
      const icon = el.querySelector('img');
      expect(icon).to.exist;
      expect(icon!.getAttribute('title')).to.equal('Newly available');

      // Assert the presence of the low date block and the high date blocks.
      const lowDateBlock = el.querySelector('.baseline-date-block-newly');
      expect(lowDateBlock).to.exist;
      expect(
        lowDateBlock
          ?.querySelector('.baseline-date-header')
          ?.textContent?.trim(),
      ).to.equal('Newly available:');
      expect(
        lowDateBlock?.querySelector('.baseline-date')?.textContent?.trim(),
      ).to.equal('2015-07-29');
      const highDateBlock = el.querySelector('.baseline-date-block-widely');
      expect(highDateBlock).to.exist;
      expect(
        highDateBlock
          ?.querySelector('.baseline-date-header')
          ?.textContent?.trim(),
      ).to.equal('Projected widely available:');
      expect(
        highDateBlock?.querySelector('.baseline-date')?.textContent?.trim(),
      ).to.equal('2018-01-29');
    });
  });
  describe('limited feature', () => {
    const feature: components['schemas']['Feature'] = {
      feature_id: 'id',
      name: 'name',
      baseline: {
        status: 'limited',
      },
    };
    it('renders only the word and icon by default', async () => {
      const result = renderBaselineStatus(feature, {search: ''}, {});
      render(result, container);
      const el = await fixture(container);
      const icon = el.querySelector('img');
      expect(icon).to.exist;
      expect(icon!.getAttribute('title')).to.equal('Limited availability');

      // Assert the absence of the low date block and the high date blocks.
      const lowDateBlock = el.querySelector('.baseline-date-block-newly');
      expect(lowDateBlock).to.not.exist;
      const highDateBlock = el.querySelector('.baseline-date-block-widely');
      expect(highDateBlock).to.not.exist;
    });
    it('does not render the low date even when selected', async () => {
      const result = renderBaselineStatus(
        feature,
        {search: 'column_options=baseline_status_low_date'},
        {},
      );
      render(result, container);
      const el = await fixture(container);
      const icon = el.querySelector('img');
      expect(icon).to.exist;
      expect(icon!.getAttribute('title')).to.equal('Limited availability');

      // Assert the absence of the low date block and the high date blocks.
      const lowDateBlock = el.querySelector('.baseline-date-block-newly');
      expect(lowDateBlock).to.not.exist;
      const highDateBlock = el.querySelector('.baseline-date-block-widely');
      expect(highDateBlock).to.not.exist;
    });
    it('does not render the projected high date even when selected', async () => {
      const result = renderBaselineStatus(
        feature,
        {search: 'column_options=baseline_status_high_date'},
        {},
      );
      render(result, container);
      const el = await fixture(container);
      const icon = el.querySelector('img');
      expect(icon).to.exist;
      expect(icon!.getAttribute('title')).to.equal('Limited availability');

      // Assert the absence of the low date block and the high date blocks.
      const lowDateBlock = el.querySelector('.baseline-date-block-newly');
      expect(lowDateBlock).to.not.exist;
      const highDateBlock = el.querySelector('.baseline-date-block-widely');
      expect(highDateBlock).to.not.exist;
    });
    it('does render the low date and projected high date even when both are selected', async () => {
      const result = renderBaselineStatus(
        feature,
        {
          search:
            'column_options=baseline_status_low_date%2Cbaseline_status_high_date',
        },
        {},
      );
      render(result, container);
      const el = await fixture(container);
      const icon = el.querySelector('img');
      expect(icon).to.exist;
      expect(icon!.getAttribute('title')).to.equal('Limited availability');

      // Assert the absence of the low date block and the high date blocks.
      const lowDateBlock = el.querySelector('.baseline-date-block-newly');
      expect(lowDateBlock).to.not.exist;
      const highDateBlock = el.querySelector('.baseline-date-block-widely');
      expect(highDateBlock).to.not.exist;
    });
  });
});

describe('renderAvailablity', () => {
  let container: HTMLElement;
  beforeEach(() => {
    container = document.createElement('td');
  });
  it('renders a full-color icon for an available feature', async () => {
    const feature: components['schemas']['Feature'] = {
      feature_id: 'id',
      name: 'name',
      browser_implementations: {
        chrome: {status: 'available', version: '123'},
      },
    };
    const result = renderAvailablity(
      feature,
      {search: ''},
      {browser: 'chrome'},
    );
    render(result, container);
    const el = await fixture(container);
    const div = el.querySelector('div');
    expect(div).to.exist;
    expect(div!.getAttribute('class')).to.equal('browser-impl-available');
    expect(div!.innerHTML).to.include('Since ');
    expect(div!.innerHTML).to.include('chrome_24x24.png');
    // The mobile platform icon should not be rendered got desktop browsers.
    expect(div!.innerHTML).to.not.include('android.svg');
  });
  it('renders a grayscale icon for an available feature', async () => {
    const feature: components['schemas']['Feature'] = {
      feature_id: 'id',
      name: 'name',
      browser_implementations: {
        chrome: {status: 'unavailable'},
      },
    };
    const result = renderAvailablity(
      feature,
      {search: ''},
      {browser: 'chrome'},
    );
    render(result, container);
    const el = await fixture(container);
    const div = el.querySelector('div');
    expect(div).to.exist;
    expect(div!.getAttribute('class')).to.equal('browser-impl-unavailable');
    expect(div!.innerHTML).to.include('Not available');
    expect(div!.innerHTML).to.include('chrome_24x24.png');
  });
  it('renders a mobile  browser icon with platform logo', async () => {
    const feature: components['schemas']['Feature'] = {
      feature_id: 'id',
      name: 'name',
      browser_implementations: {
        chrome_android: {status: 'available', version: '123'},
      },
    };
    const result = renderAvailablity(
      feature,
      {search: ''},
      {browser: 'chrome_android', platform: 'android'},
    );
    render(result, container);
    const el = await fixture(container);
    const div = el.querySelector('div');
    expect(div).to.exist;
    expect(div!.getAttribute('class')).to.equal('browser-impl-available');
    expect(div!.innerHTML).to.include('Since ');
    expect(div!.innerHTML).to.include('chrome_24x24.png');
    expect(div!.innerHTML).to.include('android.svg');
  });
});

describe('calcColGroupSpans', () => {
  const TEST_COLS = [
    ColumnKey.Name,
    ColumnKey.BaselineStatus,
    ColumnKey.StableChrome,
    ColumnKey.StableEdge,
    ColumnKey.ExpChrome,
    ColumnKey.ExpEdge,
    ColumnKey.ExpFirefox,
  ];
  it('returns empty list when there was no column spec', () => {
    const colSpans = calcColGroupSpans([]);
    assert.deepEqual(colSpans, []);
  });

  it('keeps undefined groups separate and spans matching groups', () => {
    const colSpans = calcColGroupSpans(TEST_COLS);
    assert.deepEqual(colSpans, [
      {count: 1},
      {count: 1},
      {group: 'WPT', count: 2},
      {group: 'WPT Experimental', count: 3},
    ]);
  });
});

describe('renderColgroups', () => {
  const TEST_COLS = [
    ColumnKey.Name,
    ColumnKey.BaselineStatus,
    ColumnKey.StableChrome,
    ColumnKey.StableEdge,
  ];
  let container: HTMLElement;
  beforeEach(() => {
    container = document.createElement('table');
  });
  it('Renders <colgroup>s for column groups', async () => {
    const result = renderColgroups(TEST_COLS);
    render(result, container);
    const el = await fixture(container);
    const colGroups = el.querySelectorAll('colgroup');
    expect(colGroups![0]).to.exist;
    expect(colGroups![0].getAttribute('span')).to.equal('1');
    expect(colGroups![1]).to.exist;
    expect(colGroups![1].getAttribute('span')).to.equal('1');
    expect(colGroups![2]).to.exist;
    expect(colGroups![2].getAttribute('span')).to.equal('2');
  });
});

describe('renderGroupsRow', () => {
  const TEST_COLS = [
    ColumnKey.Name,
    ColumnKey.BaselineStatus,
    ColumnKey.StableChrome,
    ColumnKey.StableEdge,
  ];
  let container: HTMLElement;
  beforeEach(() => {
    container = document.createElement('table');
  });
  it('Renders <th>s for column groups', async () => {
    const result = renderGroupsRow(TEST_COLS);
    render(result, container);
    const el = await fixture(container);
    const groupTHs = el.querySelectorAll('th');
    expect(groupTHs![0]).to.exist;
    expect(groupTHs![0].getAttribute('colspan')).to.equal('1');
    expect(groupTHs![0].innerText).to.equal('');
    expect(groupTHs![1]).to.exist;
    expect(groupTHs![1].getAttribute('colspan')).to.equal('1');
    expect(groupTHs![1].innerText).to.equal('');
    expect(groupTHs![2]).to.exist;
    expect(groupTHs![2].getAttribute('colspan')).to.equal('2');
    expect(groupTHs![2].innerText).to.equal('WPT');
  });
});

describe('parseColumnsSpec', () => {
  it('returns default columns when there was no column spec', () => {
    const cols = parseColumnsSpec('');
    assert.deepEqual(cols, DEFAULT_COLUMNS);
  });

  it('returns an array when given a column spec', () => {
    const cols = parseColumnsSpec('name, baseline_status ');
    assert.deepEqual(cols, [ColumnKey.Name, ColumnKey.BaselineStatus]);
  });
});

// Add test of parseColumnOptions here
describe('parseColumnOptions', () => {
  it('returns default column options when none are specified', () => {
    const options = parseColumnOptions('');
    assert.deepEqual(options, DEFAULT_COLUMN_OPTIONS);
  });

  it('returns an array when given a column options spec', () => {
    const options = parseColumnOptions('baseline_status_high_date');
    assert.deepEqual(options, [ColumnOptionKey.BaselineStatusHighDate]);
  });
});

describe('isJavaScriptFeature', () => {
  it('returns true for a JavaScript feature (link prefix match)', () => {
    const featureSpecInfo = {
      links: [{link: 'https://tc39.es/proposal-temporal'}],
    };
    assert.isTrue(isJavaScriptFeature(featureSpecInfo));
  });

  it('returns false for a non-JavaScript feature (no link match)', () => {
    const featureSpecInfo = {
      links: [{link: 'https://www.w3.org/TR/webgpu/'}],
    };
    assert.isFalse(isJavaScriptFeature(featureSpecInfo));
  });

  it('returns false if links are missing', () => {
    const featureSpecInfo = {}; // No 'links' property
    assert.isFalse(isJavaScriptFeature(featureSpecInfo));
  });
});

describe('didFeatureCrash', () => {
  it('returns true if metadata contains a mapping of "status" to "C"', () => {
    const metadata = {
      status: 'C',
    };
    assert.isTrue(didFeatureCrash(metadata));
  });

  it('returns false for other status metadata', () => {
    const metadata = {
      status: 'hi',
    };
    assert.isFalse(didFeatureCrash(metadata));
  });

  it('returns false for no metadata', () => {
    const metadata = {};
    assert.isFalse(didFeatureCrash(metadata));
  });

  it('returns false for undefined metadata', () => {
    const metadata = undefined;
    assert.isFalse(didFeatureCrash(metadata));
  });
});

describe('renderHeaderCell', () => {
  let container: HTMLElement;
  beforeEach(() => {
    container = document.createElement('tr');
  });
  it('renders a sortable header cell', async () => {
    const result = renderHeaderCell(
      {search: '/'},
      ColumnKey.BaselineStatus,
      '',
    );
    render(result, container);
    const el = await fixture(container);
    const th = el.querySelector('th');
    expect(th).to.exist;
    expect(th!.getAttribute('title')).to.equal('Click to sort');
    expect('' + th!.getAttribute('class')).to.include('sortable');
  });
  it('renders an unsortable header cell', async () => {
    CELL_DEFS[ColumnKey.BaselineStatus].unsortable = true;
    const result = renderHeaderCell(
      {search: '/'},
      ColumnKey.BaselineStatus,
      '',
    );
    render(result, container);
    const el = await fixture(container);
    const th = el.querySelector('th');
    expect(th).to.exist;
    expect(th!.getAttribute('title')).to.not.equal('Click to sort');
    expect(th!.getAttribute('class')).to.include('unsortable');
  });
  it('renders the name header cell for query order', async () => {
    const result = renderUnsortableHeaderCell(
      ColumnKey.Name,
      'bookmark1 query order',
    );
    render(result, container);
    const el = await fixture(container);
    const th = el.querySelector('th');
    expect(th).to.exist;
    expect(th!.getAttribute('title')).to.equal('bookmark1 query order');
    expect(th!.getAttribute('class')).to.include('unsortable');
  });
  it('renders a header cell with a cell class', async () => {
    CELL_DEFS[ColumnKey.BaselineStatus].cellClass = 'cell-class';
    const result = renderHeaderCell(
      {search: '/'},
      ColumnKey.BaselineStatus,
      '',
    );
    render(result, container);
    const el = await fixture(container);
    const th = el.querySelector('th');
    expect(th).to.exist;
    expect(th!.getAttribute('class')).to.include('cell-class');
  });
  it('renders a non-name header cell for query order', async () => {
    const result = renderUnsortableHeaderCell(ColumnKey.BaselineStatus);
    render(result, container);
    const el = await fixture(container);
    const th = el.querySelector('th');
    expect(th).to.exist;
    expect(th!.getAttribute('title')).to.not.exist;
    expect(th!.getAttribute('class')).to.include('unsortable');
  });
});
