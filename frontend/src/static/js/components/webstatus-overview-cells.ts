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
import {type TemplateResult, html, nothing} from 'lit';
import {type components} from 'webstatus.dev-backend';
import {
  formatFeaturePageUrl,
  formatOverviewPageUrl,
  getColumnOptions,
} from '../utils/urls.js';
import {BROWSER_ID_TO_ICON_NAME, FeatureSortOrderType} from '../api/client.js';
import {ifDefined} from 'lit/directives/if-defined.js';
import {
  TOP_CSS_INTEROP_ISSUES,
  TOP_HTML_INTEROP_ISSUES,
} from '../utils/constants.js';
import './webstatus-feature-badge.js';

const MISSING_VALUE = html``;

type CellRenderer = {
  (
    feature: components['schemas']['Feature'],
    routerLocation: {search: string},
    options: {
      browser?: components['parameters']['browserPathParam'];
      channel?: components['parameters']['channelPathParam'];
      platform?: string;
    },
  ): TemplateResult | typeof nothing;
};

type ColumnDefinition = {
  nameInDialog: string;
  group?: string;
  headerHtml?: TemplateResult;
  iconName?: string;
  cellRenderer: CellRenderer;
  cellClass?: string;
  unsortable?: boolean;
  options: {
    browser?: components['parameters']['browserPathParam'];
    channel?: components['parameters']['channelPathParam'];
    columnOptions?: Array<ColumnOptionDefinition>;
    platform?: string;
  };
};

// Currently, the widely available date is defined as 30 months after the newly available date.
// https://github.com/web-platform-dx/web-features/blob/6ac2ef2325d26b0c430c6dd08665d2361fa4653d/docs/baseline.md?plain=1#L152
// In the event a newly available feature is not widely available yet, we use this constant to estimate widely available date.
const NEWLY_TO_WIDELY_MONTH_OFFSET = 30;

export enum ColumnKey {
  Name = 'name',
  BaselineStatus = 'baseline_status',
  AvailabilityChrome = 'availability_chrome',
  AvailabilityEdge = 'availability_edge',
  AvailabilityFirefox = 'availability_firefox',
  AvailabilitySafari = 'availability_safari',
  AvailabilityChromeAndroid = 'availability_chrome_android',
  AvailabilityFirefoxAndroid = 'availability_firefox_android',
  AvailabilitySafariIos = 'availability_safari_ios',
  StableChrome = 'stable_chrome',
  StableEdge = 'stable_edge',
  StableFirefox = 'stable_firefox',
  StableSafari = 'stable_safari',
  StableChromeAndroid = 'stable_chrome_android',
  StableFirefoxAndroid = 'stable_firefox_android',
  StableSafariIos = 'stable_safari_ios',
  ExpChrome = 'experimental_chrome',
  ExpEdge = 'experimental_edge',
  ExpFirefox = 'experimental_firefox',
  ExpSafari = 'experimental_safari',
  ExpChromeAndroid = 'experimental_chrome_android',
  ExpFirefoxAndroid = 'experimental_firefox_android',
  ExpSafariIos = 'experimental_safari_ios',
  ChromeUsage = 'chrome_usage',
}

const columnKeyMapping = Object.entries(ColumnKey).reduce(
  (mapping, [enumKey, enumValue]) => {
    mapping[enumValue] = ColumnKey[enumKey as keyof typeof ColumnKey];
    return mapping;
  },
  {} as Record<string, ColumnKey>,
);

type ColumnOptionDefinition = {
  nameInDialog: string;
  columnOptionKey: ColumnOptionKey;
};

export enum ColumnOptionKey {
  BaselineStatusHighDate = 'baseline_status_high_date',
  BaselineStatusLowDate = 'baseline_status_low_date',
}

const columnOptionKeyMapping = Object.entries(ColumnOptionKey).reduce(
  (mapping, [enumKey, enumValue]) => {
    mapping[enumValue] =
      ColumnOptionKey[enumKey as keyof typeof ColumnOptionKey];
    return mapping;
  },
  {} as Record<string, ColumnOptionKey>,
);

export const DEFAULT_COLUMNS = [
  ColumnKey.Name,
  ColumnKey.BaselineStatus,
  ColumnKey.AvailabilityChrome,
  ColumnKey.AvailabilityEdge,
  ColumnKey.AvailabilityFirefox,
  ColumnKey.AvailabilitySafari,
  ColumnKey.StableChrome,
  ColumnKey.StableEdge,
  ColumnKey.StableFirefox,
  ColumnKey.StableSafari,
  ColumnKey.ChromeUsage,
];

export const DEFAULT_COLUMN_OPTIONS: ColumnOptionKey[] = [
  // None, but here is an example of what could be added:
  // ColumnOptionKey.BaselineStatusHighDate,
  // ColumnOptionKey.BaselineStatusLowDate,
];

export type BrowserChannelColumnKeys =
  | ColumnKey.StableChrome
  | ColumnKey.StableEdge
  | ColumnKey.StableFirefox
  | ColumnKey.StableSafari
  | ColumnKey.StableChromeAndroid
  | ColumnKey.StableFirefoxAndroid
  | ColumnKey.StableSafariIos
  | ColumnKey.ExpChrome
  | ColumnKey.ExpEdge
  | ColumnKey.ExpFirefox
  | ColumnKey.ExpSafari
  | ColumnKey.ExpChromeAndroid
  | ColumnKey.ExpFirefoxAndroid
  | ColumnKey.ExpSafariIos;

export const DEFAULT_SORT_SPEC: FeatureSortOrderType = 'baseline_status_desc';

const DEFAULT_SORTABLE_HEADER = html`<sl-icon
    class="sortable-icon"
    name="arrow-up"
  ></sl-icon
  ><sl-icon class="sortable-icon" name="arrow-down"></sl-icon>`;

interface BaselineChipConfig {
  icon: string;
  word: string;
}

export const BASELINE_CHIP_CONFIGS: Record<
  NonNullable<components['schemas']['BaselineInfo']['status']>,
  BaselineChipConfig
> = {
  limited: {
    icon: 'cross.svg',
    word: 'Limited availability',
  },
  newly: {
    icon: 'newly.svg',
    word: 'Newly available',
  },
  widely: {
    icon: 'check.svg',
    word: 'Widely available',
  },
};

function getFeatureBadges(featureId: string): TemplateResult[] {
  const extraIdentifiers: TemplateResult[] = [];
  if (TOP_CSS_INTEROP_ISSUES.includes(featureId)) {
    extraIdentifiers.push(
      html`<webstatus-feature-badge
        .badgeType=${'css'}
      ></webstatus-feature-badge>`,
    );
  }
  if (TOP_HTML_INTEROP_ISSUES.includes(featureId)) {
    extraIdentifiers.push(
      html`<webstatus-feature-badge
        .badgeType=${'html'}
      ></webstatus-feature-badge>`,
    );
  }
  return extraIdentifiers;
}

const renderFeatureName: CellRenderer = (feature, routerLocation, _options) => {
  const featureUrl = formatFeaturePageUrl(feature, routerLocation);
  return html`<div class="feature-name-cell">
    <a class="feature-page-link" href=${featureUrl}>${feature.name}</a
    >${getFeatureBadges(feature.feature_id)}
  </div>`;
};

export const renderChromeUsage: CellRenderer = (
  feature,
  _routerLocation,
  _options,
) => {
  let usage = 'N/A';
  if (feature.usage?.chrome?.daily && feature.usage.chrome.daily > 0) {
    // If the feature has some usage, but the usage is less than 0.1%,
    // display it as "<0.1%".
    if (feature.usage.chrome.daily < 0.001) {
      usage = '<0.1%';
    } else {
      // Format to display percentage with single decimal e.g. 0.8371 -> 83.7%.
      usage = `${(feature.usage.chrome.daily * 100).toFixed(1)}%`;
    }
  } else if (feature.usage?.chrome?.daily === 0) {
    usage = '0.0%';
  } else if (feature.usage?.chrome?.daily && feature.usage.chrome.daily >= 1) {
    usage = '100%';
  }
  return html`<span id="chrome-usage">${usage}</span>`;
};

function formatDateString(dateString: string): string {
  return formatDate(new Date(dateString));
}

function formatDate(date: Date): string {
  const year = date.getFullYear();
  const month = (date.getMonth() + 1).toString().padStart(2, '0'); // Months are 0-indexed
  const day = date.getDate().toString().padStart(2, '0'); // Days are 1-indexed

  // Assuming the original format was YYYY-MM-DD
  return `${year}-${month}-${day}`;
}

export const renderBaselineStatus: CellRenderer = (
  feature,
  routerLocation,
  _options,
) => {
  const baselineStatus = feature.baseline?.status;
  if (baselineStatus === undefined) return html``;
  const chipConfig = BASELINE_CHIP_CONFIGS[baselineStatus];
  const columnOptions: ColumnOptionKey[] = parseColumnOptions(
    getColumnOptions(routerLocation),
  );
  const columnHighDateOption = columnOptions.includes(
    ColumnOptionKey.BaselineStatusHighDate,
  );
  const columnLowDateOption = columnOptions.includes(
    ColumnOptionKey.BaselineStatusLowDate,
  );

  function generateDateHtml(
    header: string,
    date: string | number,
    blockType: 'widely' | 'newly',
  ) {
    return html`<div
      class="baseline-date-block baseline-date-block-${blockType}"
    >
      <span class="baseline-date-header">${header}:</span>
      <span class="baseline-date">${date}</span>
    </div>`;
  }

  let baselineStatusLowDateHtml = html``;
  const baselineStatusLowDate = feature.baseline?.low_date;
  if (baselineStatusLowDate && columnLowDateOption) {
    baselineStatusLowDateHtml = generateDateHtml(
      'Newly available',
      formatDateString(baselineStatusLowDate),
      'newly',
    );
  }

  let baselineStatusHighDateHtml = html``;
  const baselineStatusHighDate = feature.baseline?.high_date;
  if (baselineStatusHighDate && columnHighDateOption) {
    baselineStatusHighDateHtml = generateDateHtml(
      'Widely available',
      formatDateString(baselineStatusHighDate),
      'widely',
    );
  } else if (baselineStatusLowDate && columnHighDateOption) {
    // Add the month offset to the low date to get the projected high date.
    const projectedHighDate = new Date(baselineStatusLowDate);
    projectedHighDate.setMonth(
      projectedHighDate.getMonth() + NEWLY_TO_WIDELY_MONTH_OFFSET,
    );
    baselineStatusHighDateHtml = generateDateHtml(
      'Projected widely available',
      formatDate(projectedHighDate),
      'widely',
    );
  }

  return html`
    <img
      height="16"
      src="/public/img/${chipConfig.icon}"
      title=${chipConfig.word}
    />
    ${baselineStatusLowDateHtml} ${baselineStatusHighDateHtml}
  `;
};

function renderPlatformIcon(platform?: string): TemplateResult {
  if (platform) {
    return html` /
      <img
        class="platform"
        alt="${platform} logo"
        src="/public/img/${platform}.svg"
      />`;
  }
  return html``;
}

export const renderAvailablity: CellRenderer = (
  feature,
  _routerLocation,
  {browser, platform},
) => {
  const browserImpl = feature.browser_implementations?.[browser!];
  const browserImplStatus = browserImpl?.status || 'unavailable';
  const browserImplVersion = browserImpl?.version;
  return html`
    <div class="browser-impl-${browserImplStatus} browser-cell">
      <sl-tooltip
        content=${browserImplVersion
          ? `Since version ${browserImplVersion}`
          : 'Not available'}
      >
        <img
          class="availability-icon"
          src="/public/img/${BROWSER_ID_TO_ICON_NAME[browser!]}_24x24.png"
        />
      </sl-tooltip>
      ${renderPlatformIcon(platform)}
    </div>
  `;
};

function renderMissingPercentage(): TemplateResult {
  return html`<span class="missing percent">${MISSING_VALUE}</span>`;
}

function renderPercentage(score?: number): TemplateResult {
  if (score === undefined) {
    return renderMissingPercentage();
  }
  let percent = Number(score * 100).toFixed(1);
  if (percent === '100.0') {
    percent = '100';
  }
  return html`<span class="percent">${percent}%</span>`;
}

export const renderBrowserQuality: CellRenderer = (
  feature,
  _routerLocation,
  {browser},
) => {
  const score: number | undefined = feature.wpt?.stable?.[browser!]?.score;
  let percentage = renderPercentage(score);
  const browserImpl = feature.browser_implementations?.[browser!];
  const browserImplStatus = browserImpl?.status || 'unavailable';
  if (browserImplStatus === 'unavailable') {
    percentage = renderMissingPercentage();
  }
  if (feature.spec && isJavaScriptFeature(feature.spec)) {
    percentage = renderJavaScriptFeatureValue();
  }
  if (hasInsufficientTestCoverage(feature.feature_id)) {
    percentage = renderInsufficentTestCoverage();
  }
  if (didFeatureCrash(feature.wpt?.stable?.[browser!]?.metadata)) {
    percentage = renderFeatureCrash();
  }
  return html`
      ${percentage}
    </div>
  `;
};

export const renderBrowserQualityExp: CellRenderer = (
  feature,
  _routerLocation,
  {browser},
) => {
  const score: number | undefined =
    feature.wpt?.experimental?.[browser!]?.score;
  return renderPercentage(score);
};

export const getBrowserAndChannel = (
  browserColumnKey: BrowserChannelColumnKeys,
): {
  browser: components['parameters']['browserPathParam'];
  channel: components['parameters']['channelPathParam'];
} => {
  const browser = CELL_DEFS[browserColumnKey].options.browser;
  if (!browser) {
    throw new Error('browser is undefined');
  }
  const channel = CELL_DEFS[browserColumnKey].options.channel;
  if (!channel) {
    throw new Error('channel is undefined');
  }
  return {browser, channel};
};

export const CELL_DEFS: Record<ColumnKey, ColumnDefinition> = {
  [ColumnKey.Name]: {
    nameInDialog: 'Feature name',
    headerHtml: html`Feature`,
    cellRenderer: renderFeatureName,
    options: {},
  },
  [ColumnKey.BaselineStatus]: {
    nameInDialog: 'Baseline status',
    headerHtml: html`Baseline`,
    cellRenderer: renderBaselineStatus,
    options: {
      columnOptions: [
        {
          nameInDialog: 'Show Baseline status low date',
          columnOptionKey: ColumnOptionKey.BaselineStatusLowDate,
        },
        {
          nameInDialog: 'Show Baseline status high date',
          columnOptionKey: ColumnOptionKey.BaselineStatusHighDate,
        },
      ],
    },
  },
  [ColumnKey.AvailabilityChrome]: {
    nameInDialog: 'Availibility in desktop Chrome',
    group: 'Availability',
    iconName: 'chrome',
    cellClass: 'centered',
    cellRenderer: renderAvailablity,
    options: {browser: 'chrome'},
  },
  [ColumnKey.AvailabilityEdge]: {
    nameInDialog: 'Availibility in desktop Edge',
    group: 'Availability',
    iconName: 'edge',
    cellClass: 'centered',
    cellRenderer: renderAvailablity,
    options: {browser: 'edge'},
  },
  [ColumnKey.AvailabilityFirefox]: {
    nameInDialog: 'Availibility in desktop Firefox',
    group: 'Availability',
    iconName: 'firefox',
    cellClass: 'centered',
    cellRenderer: renderAvailablity,
    options: {browser: 'firefox'},
  },
  [ColumnKey.AvailabilitySafari]: {
    nameInDialog: 'Availibility in desktop Safari',
    group: 'Availability',
    iconName: 'safari',
    cellClass: 'centered',
    cellRenderer: renderAvailablity,
    options: {browser: 'safari'},
  },
  [ColumnKey.AvailabilityChromeAndroid]: {
    nameInDialog: 'Availibility in mobile Chrome (Android)',
    group: 'Availability',
    iconName: 'chrome',
    cellClass: 'centered',
    cellRenderer: renderAvailablity,
    options: {browser: 'chrome_android', platform: 'android'},
  },
  [ColumnKey.AvailabilityFirefoxAndroid]: {
    nameInDialog: 'Availibility in mobile Firefox (Android)',
    group: 'Availability',
    iconName: 'firefox',
    cellClass: 'centered',
    cellRenderer: renderAvailablity,
    options: {browser: 'firefox_android', platform: 'android'},
  },
  [ColumnKey.AvailabilitySafariIos]: {
    nameInDialog: 'Availibility in mobile Safari (iOS)',
    group: 'Availability',
    iconName: 'safari',
    cellClass: 'centered',
    cellRenderer: renderAvailablity,
    options: {browser: 'safari_ios', platform: 'ios'},
  },
  [ColumnKey.StableChrome]: {
    nameInDialog: 'Browser Implementation in Chrome',
    group: 'WPT',
    headerHtml: html`<img src="/public/img/chrome_24x24.png" />`,
    cellClass: 'centered',
    cellRenderer: renderBrowserQuality,
    options: {browser: 'chrome', channel: 'stable'},
  },
  [ColumnKey.StableEdge]: {
    nameInDialog: 'Browser Implementation in Edge',
    group: 'WPT',
    headerHtml: html`<img src="/public/img/edge_24x24.png" />`,
    cellClass: 'centered',
    cellRenderer: renderBrowserQuality,
    options: {browser: 'edge', channel: 'stable'},
  },
  [ColumnKey.StableFirefox]: {
    nameInDialog: 'Browser Implementation in Firefox',
    group: 'WPT',
    headerHtml: html`<img src="/public/img/firefox_24x24.png" />`,
    cellClass: 'centered',
    cellRenderer: renderBrowserQuality,
    options: {browser: 'firefox', channel: 'stable'},
  },
  [ColumnKey.StableSafari]: {
    nameInDialog: 'Browser Implementation in Safari',
    group: 'WPT',
    headerHtml: html`<img src="/public/img/safari_24x24.png" />`,
    cellClass: 'centered',
    cellRenderer: renderBrowserQuality,
    options: {browser: 'safari', channel: 'stable'},
  },
  [ColumnKey.StableChromeAndroid]: {
    nameInDialog: 'Browser Implementation in Chrome (Android)',
    group: 'WPT',
    headerHtml: html`<img src="/public/img/chrome_24x24.png" />
      <img
        class="platform"
        alt="android logo"
        src="/public/img/android.svg"
      />`,
    cellClass: 'centered',
    cellRenderer: renderBrowserQuality,
    options: {
      browser: 'chrome_android',
      channel: 'stable',
      platform: 'android',
    },
  },
  [ColumnKey.StableFirefoxAndroid]: {
    nameInDialog: 'Browser Implementation in Firefox (Android)',
    group: 'WPT',
    headerHtml: html`<img src="/public/img/firefox_24x24.png" />
      <img
        class="platform"
        alt="android logo"
        src="/public/img/android.svg"
      />`,
    cellClass: 'centered',
    cellRenderer: renderBrowserQuality,
    options: {
      browser: 'firefox_android',
      channel: 'stable',
      platform: 'android',
    },
  },
  [ColumnKey.StableSafariIos]: {
    nameInDialog: 'Browser Implementation in Safari (iOS)',
    group: 'WPT',
    headerHtml: html`<img src="/public/img/safari_24x24.png" />
      <img class="platform" alt="ios logo" src="/public/img/ios.svg" />`,
    cellClass: 'centered',
    cellRenderer: renderBrowserQuality,
    options: {browser: 'safari_ios', channel: 'stable', platform: 'ios'},
  },
  [ColumnKey.ExpChrome]: {
    nameInDialog: 'Browser Implementation in Chrome Experimental',
    group: 'WPT Experimental',
    headerHtml: html`<img src="/public/img/chrome-canary_24x24.png" />`,
    cellClass: 'centered',
    cellRenderer: renderBrowserQualityExp,
    options: {browser: 'chrome', channel: 'experimental'},
  },
  [ColumnKey.ExpEdge]: {
    nameInDialog: 'Browser Implementation in Edge Experimental',
    group: 'WPT Experimental',
    headerHtml: html`<img src="/public/img/edge-dev_24x24.png" />`,
    cellClass: 'centered',
    cellRenderer: renderBrowserQualityExp,
    options: {browser: 'edge', channel: 'experimental'},
  },
  [ColumnKey.ExpFirefox]: {
    nameInDialog: 'Browser Implementation in Firefox Experimental',
    group: 'WPT Experimental',
    headerHtml: html`<img src="/public/img/firefox-nightly_24x24.png" />`,
    cellClass: 'centered',
    cellRenderer: renderBrowserQualityExp,
    options: {browser: 'firefox', channel: 'experimental'},
  },
  [ColumnKey.ExpSafari]: {
    nameInDialog: 'Browser Implementation in Safari Experimental',
    group: 'WPT Experimental',
    headerHtml: html`<img src="/public/img/safari-preview_24x24.png" />`,
    cellClass: 'centered',
    cellRenderer: renderBrowserQualityExp,
    options: {browser: 'safari', channel: 'experimental'},
  },
  [ColumnKey.ExpChromeAndroid]: {
    nameInDialog: 'Browser Implementation in Chrome Experimental (Android)',
    group: 'WPT Experimental',
    headerHtml: html`<img src="/public/img/chrome-canary_24x24.png" />
      <img
        class="platform"
        alt="android logo"
        src="/public/img/android.svg"
      />`,
    cellClass: 'centered',
    cellRenderer: renderBrowserQualityExp,
    options: {
      browser: 'chrome_android',
      channel: 'experimental',
      platform: 'android',
    },
  },
  [ColumnKey.ExpFirefoxAndroid]: {
    nameInDialog: 'Browser Implementation in Firefox Experimental (Android)',
    group: 'WPT Experimental',
    headerHtml: html`<img src="/public/img/firefox-nightly_24x24.png" />
      <img
        class="platform"
        alt="android logo"
        src="/public/img/android.svg"
      />`,
    cellClass: 'centered',
    cellRenderer: renderBrowserQualityExp,
    options: {
      browser: 'firefox_android',
      channel: 'experimental',
      platform: 'android',
    },
  },
  [ColumnKey.ExpSafariIos]: {
    nameInDialog: 'Browser Implementation in Safari Experimental (iOS)',
    group: 'WPT Experimental',
    headerHtml: html`<img src="/public/img/safari-preview_24x24.png" />
      <img class="platform" alt="ios logo" src="/public/img/ios.svg" />`,
    cellClass: 'centered',
    cellRenderer: renderBrowserQualityExp,
    options: {browser: 'safari_ios', channel: 'experimental', platform: 'ios'},
  },
  [ColumnKey.ChromeUsage]: {
    nameInDialog: 'Chrome Usage',
    headerHtml: html`Usage`,
    cellRenderer: renderChromeUsage,
    options: {},
  },
};

export function calcColGroupSpans(
  columns: ColumnKey[],
): {group?: string; count: number}[] {
  const result: {group?: string; count: number}[] = [];
  for (let i = 0; i < columns.length; i++) {
    const colDef = CELL_DEFS[columns[i]];
    if (colDef.group === undefined) {
      result.push({count: 1});
    } else {
      let colspan = 1;
      while (
        i + colspan < columns.length &&
        colDef.group === CELL_DEFS[columns[i + colspan]].group
      ) {
        colspan++;
      }
      result.push({group: colDef.group, count: colspan});
      i += colspan - 1;
    }
  }
  return result;
}

export function renderColgroups(columns: ColumnKey[]): TemplateResult {
  const colGroupSpans = calcColGroupSpans(columns);
  return html`
    ${colGroupSpans.map(({count}) => html`<colgroup span=${count}></colgroup>`)}
  `;
}

export function renderGroupCells(
  routerLocation: {search: string},
  columns: ColumnKey[],
  sortSpec: string,
): TemplateResult[] {
  const colGroupSpans = calcColGroupSpans(columns);
  let spanTotal = 0;
  return colGroupSpans.map(({group, count}) => {
    let template;
    if (count === 1) {
      template = renderHeaderCell(routerLocation, columns[spanTotal], sortSpec);
    } else {
      template = html`<th colspan=${count}>${group}</th>`;
    }
    spanTotal += count;
    return template;
  });
}

export function renderSavedSearchHeaderCells(
  name: string,
  columns: ColumnKey[],
): TemplateResult[] {
  const headerCells: TemplateResult[] = columns.map(col => {
    if (!CELL_DEFS[col].group) {
      return html`<th colspan="1"></th>`;
    }
    if (col === ColumnKey.Name) {
      const title = `Sorted by ${name} query order`;
      return html`${renderUnsortableHeaderCell(col, title)}`;
    } else {
      return html`${renderUnsortableHeaderCell(col)}`;
    }
  });
  return headerCells;
}

export function renderHeaderCells(
  routerLocation: {search: string},
  columns: ColumnKey[],
  sortSpec: string,
) {
  return columns.map(col => {
    if (!CELL_DEFS[col].group) {
      return html`<th colspan=${1}></th>`;
    }
    return html`${renderHeaderCell(routerLocation, col, sortSpec)}`;
  });
}

function renderHeaderCell(
  routerLocation: {search: string},
  column: ColumnKey,
  sortSpec: string,
): TemplateResult {
  if (CELL_DEFS[column].unsortable) {
    return renderUnsortableHeaderCell(column);
  }
  return renderSortableHeaderCell(routerLocation, column, sortSpec);
}

function renderSortableHeaderCell(
  routerLocation: {search: string},
  column: ColumnKey,
  sortSpec: string,
): TemplateResult {
  let sortIndicator = html``;
  let urlWithSort = formatOverviewPageUrl(routerLocation, {
    sort: column + '_asc',
    start: 0,
  });

  const colDef = CELL_DEFS[column];
  let headerHtml = colDef.headerHtml || DEFAULT_SORTABLE_HEADER;
  if (sortSpec === column + '_asc') {
    sortIndicator = html`<sl-icon name="arrow-up"></sl-icon>`;
    urlWithSort = formatOverviewPageUrl(routerLocation, {
      sort: column + '_desc',
      start: 0,
    });
    if (colDef.headerHtml === undefined) {
      headerHtml = html``;
    }
  } else if (sortSpec === column + '_desc') {
    sortIndicator = html`<sl-icon name="arrow-down"></sl-icon>`;
    console.log(colDef.headerHtml);
    if (colDef.headerHtml === undefined) {
      headerHtml = html``;
    }
  }

  return html`
    <th
      title="Click to sort"
      class="${colDef?.cellClass || ''} sortable"
      colspan="1"
    >
      <a href=${urlWithSort}>
        ${sortIndicator}
        <p>${headerHtml}</p></a
      >
    </th>
  `;
}

export function renderUnsortableHeaderCell(
  column: ColumnKey,
  customTitle?: string,
): TemplateResult {
  const colDef = CELL_DEFS[column];

  return html`
    <th
      title=${ifDefined(customTitle)}
      class="${colDef?.cellClass || ''} unsortable"
      colspan="1"
    >
      ${colDef?.headerHtml || nothing}
    </th>
  `;
}

export function renderFeatureCell(
  feature: components['schemas']['Feature'],
  routerLocation: {search: string},
  column: ColumnKey,
): TemplateResult | typeof nothing {
  const colDef = CELL_DEFS[column];
  if (colDef?.cellRenderer) {
    return colDef.cellRenderer(feature, routerLocation, colDef.options);
  } else {
    return nothing;
  }
}

export function parseColumnsSpec(colSpec: string): ColumnKey[] {
  let colStrs = colSpec.toLowerCase().split(',');
  colStrs = colStrs.map(s => s.trim()).filter(c => c);
  const colKeys: ColumnKey[] = [];
  for (const cs of colStrs) {
    if (columnKeyMapping[cs]) {
      colKeys.push(columnKeyMapping[cs]);
    }
  }
  if (colKeys.length > 0) {
    return colKeys;
  } else {
    return DEFAULT_COLUMNS;
  }
}

export function parseColumnOptions(columnOptions: string): ColumnOptionKey[] {
  let colStrs = columnOptions.toLowerCase().split(',');
  colStrs = colStrs.map(s => s.trim()).filter(c => c);
  const colKeys: ColumnOptionKey[] = [];
  for (const cs of colStrs) {
    if (columnOptionKeyMapping[cs]) {
      colKeys.push(columnOptionKeyMapping[cs]);
    }
  }
  if (colKeys.length > 0) {
    return colKeys;
  } else {
    return [];
  }
}

// JavaScript features will not have WPT scores for now. Instead of presenting MISSING_VALUE,
// these features can present an informative message describing the absence of the
// WPT score.
const JS_FEATURE_LINK_PREFIX = 'https://tc39.es/';

// FeatureSpecInfo represents the specification information for a feature,
// particularly the links that might indicate it's a JavaScript feature.
interface FeatureSpecInfo {
  links?: {
    link?: string;
  }[]; // Array of objects potentially containing a 'link' property
}

export function isJavaScriptFeature(featureSpecInfo: FeatureSpecInfo): boolean {
  return (
    featureSpecInfo?.links?.some(linkObj =>
      linkObj.link?.startsWith(JS_FEATURE_LINK_PREFIX),
    ) ?? false
  );
}

function renderJavaScriptFeatureValue(): TemplateResult {
  return html` <sl-tooltip
    class="missing percent"
    content="WPT metrics are not applicable to TC39 features."
  >
    <sl-icon-button name="info-circle" label="TC39 feature"></sl-icon-button>
  </sl-tooltip>`;
}

export function hasInsufficientTestCoverage(feature_id: string): boolean {
  return [
    'avif', // 1 test, for animated AVIF, and it fails in Edge+Firefox+Safari.
    'counter-set', // 2 tests, and counter-set-001.html failures need review. Probably valid.
    'declarative-shadow-dom', // Dominated by getHTML() tests which fail in Firefox+Safari. In other words, skewed coverage and not insufficient coverage.
    'device-orientation-events', // Failures are mostly because of permissions. Feature could be OK.
    'preserves-pitch', // Timeout in Firefox and Safari affect the scores a lot. Feature probably OK.
    'storage-access', // 2 tests. idlharness.js is shallow by design, and the other fails.
    'webtransport', // A big test suite, but harness errors could indicate a problem with the tests.
    'webvtt', // Widespread failures due to default styling, see https://github.com/web-platform-tests/wpt/issues/46453.
  ].includes(feature_id);
}

function renderInsufficentTestCoverage(): TemplateResult {
  return html` <sl-tooltip
    class="missing percent"
    content="Insufficient test coverage."
  >
    <sl-icon-button
      name="info-circle"
      label="insufficent-test-coverage"
    ></sl-icon-button>
  </sl-tooltip>`;
}

export function didFeatureCrash(metadata?: {[key: string]: unknown}): boolean {
  return !!metadata && 'status' in metadata && metadata['status'] === 'C';
}

function renderFeatureCrash(): TemplateResult {
  return html` <sl-tooltip
    class="missing percent"
    content="Feature's WPT run metrics are incomplete due to a crash. See wpt.fyi for more information."
  >
    <sl-icon-button
      name="exclamation-triangle"
      label="feature-crash-warning"
    ></sl-icon-button>
  </sl-tooltip>`;
}
