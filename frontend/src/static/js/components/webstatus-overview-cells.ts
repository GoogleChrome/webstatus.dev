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
import {formatFeaturePageUrl, formatOverviewPageUrl} from '../utils/urls.js';
import {FeatureSortOrderType} from '../api/client.js';

const MISSING_VALUE = html`---`;

type CellRenderer = {
  (
    feature: components['schemas']['Feature'],
    routerLocation: {search: string},
    options: {
      browser?: components['parameters']['browserPathParam'];
      channel?: components['parameters']['channelPathParam'];
    }
  ): TemplateResult | typeof nothing;
};

type ColumnDefinition = {
  nameInDialog: string;
  headerHtml: TemplateResult;
  cellRenderer: CellRenderer;
  options: {
    browser?: components['parameters']['browserPathParam'];
    channel?: components['parameters']['channelPathParam'];
  };
};

export enum ColumnKey {
  Name = 'name',
  BaselineStatus = 'baseline_status',
  StableChrome = 'stable_chrome',
  StableEdge = 'stable_edge',
  StableFirefox = 'stable_firefox',
  StableSafari = 'stable_safari',
  ExpChrome = 'experimental_chrome',
  ExpEdge = 'experimental_edge',
  ExpFirefox = 'experimental_firefox',
  ExpSafari = 'experimental_safari',
}

const columnKeyMapping = Object.entries(ColumnKey).reduce(
  (mapping, [enumKey, enumValue]) => {
    mapping[enumValue] = ColumnKey[enumKey as keyof typeof ColumnKey];
    return mapping;
  },
  {} as Record<string, ColumnKey>
);

export const DEFAULT_COLUMNS = [
  ColumnKey.Name,
  ColumnKey.BaselineStatus,
  ColumnKey.StableChrome,
  ColumnKey.StableEdge,
  ColumnKey.StableFirefox,
  ColumnKey.StableSafari,
];

export type BrowserChannelColumnKeys =
  | ColumnKey.StableChrome
  | ColumnKey.StableEdge
  | ColumnKey.StableFirefox
  | ColumnKey.StableSafari
  | ColumnKey.ExpChrome
  | ColumnKey.ExpEdge
  | ColumnKey.ExpFirefox
  | ColumnKey.ExpSafari;

export const DEFAULT_SORT_SPEC: FeatureSortOrderType = 'baseline_status_desc';

interface BaselineChipConfig {
  cssClass: string;
  icon: string;
  word: string;
}

export const BASELINE_CHIP_CONFIGS: Record<
  NonNullable<components['schemas']['BaselineInfo']['status']>,
  BaselineChipConfig
> = {
  limited: {
    cssClass: 'limited',
    icon: 'cross.svg',
    word: 'Limited availability',
  },
  newly: {
    cssClass: 'newly',
    icon: 'newly.svg',
    word: 'Newly available',
  },
  widely: {
    cssClass: 'widely',
    icon: 'check.svg',
    word: 'Widely available',
  },
};

const renderFeatureName: CellRenderer = (feature, routerLocation, _options) => {
  const featureUrl = formatFeaturePageUrl(feature, routerLocation);
  return html` <a href=${featureUrl}>${feature.name}</a> `;
};

const renderBaselineStatus: CellRenderer = (
  feature,
  _routerLocation,
  _options
) => {
  const baselineStatus = feature.baseline?.status;
  if (baselineStatus === undefined) return html``;
  const chipConfig = BASELINE_CHIP_CONFIGS[baselineStatus];
  const lowDate = feature.baseline?.low_date;
  const baselineSince = lowDate
    ? `Baseline since ${lowDate}`
    : 'Not yet available';

  return html`
    <sl-tooltip content="${baselineSince}" placement="right-start">
      <span class="chip ${chipConfig.cssClass}">
        <img height="16" src="/public/img/${chipConfig.icon}" />
        ${chipConfig.word}
      </span>
    </sl-tooltip>
  `;
};

const BROWSER_IMPL_ICONS: Record<
  NonNullable<components['schemas']['BrowserImplementation']['status']>,
  string
> = {
  unavailable: 'minus-circle',
  available: 'check-circle',
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
  {browser}
) => {
  const score: number | undefined = feature.wpt?.stable?.[browser!]?.score;
  let percentage = renderPercentage(score);
  const browserImpl =
    feature.browser_implementations?.[browser!]?.status || 'unavailable';
  if (browserImpl === 'unavailable') {
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
  const iconName = BROWSER_IMPL_ICONS[browserImpl];
  return html`
    <div class="browser-impl-${browserImpl}">
      <sl-icon name="${iconName}" library="custom"></sl-icon>
      ${percentage}
    </div>
  `;
};

export const renderBrowserQualityExp: CellRenderer = (
  feature,
  _routerLocation,
  {browser}
) => {
  const score: number | undefined =
    feature.wpt?.experimental?.[browser!]?.score;
  return renderPercentage(score);
};

export const getBrowserAndChannel = (
  browserColumnKey: BrowserChannelColumnKeys
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
    options: {},
  },
  [ColumnKey.StableChrome]: {
    nameInDialog: 'Browser Implementation in Chrome',
    headerHtml: html`<img src="/public/img/chrome_24x24.png" />`,
    cellRenderer: renderBrowserQuality,
    options: {browser: 'chrome', channel: 'stable'},
  },
  [ColumnKey.StableEdge]: {
    nameInDialog: 'Browser Implementation in Edge',
    headerHtml: html`<img src="/public/img/edge_24x24.png" />`,
    cellRenderer: renderBrowserQuality,
    options: {browser: 'edge', channel: 'stable'},
  },
  [ColumnKey.StableFirefox]: {
    nameInDialog: 'Browser Implementation in Firefox',
    headerHtml: html`<img src="/public/img/firefox_24x24.png" />`,
    cellRenderer: renderBrowserQuality,
    options: {browser: 'firefox', channel: 'stable'},
  },
  [ColumnKey.StableSafari]: {
    nameInDialog: 'Browser Implementation in Safari',
    headerHtml: html`<img src="/public/img/safari_24x24.png" />`,
    cellRenderer: renderBrowserQuality,
    options: {browser: 'safari', channel: 'stable'},
  },
  [ColumnKey.ExpChrome]: {
    nameInDialog: 'Browser Implementation in Chrome Experimental',
    headerHtml: html`<img src="/public/img/chrome-canary_24x24.png" />
      Experimental`,
    cellRenderer: renderBrowserQualityExp,
    options: {browser: 'chrome', channel: 'experimental'},
  },
  [ColumnKey.ExpEdge]: {
    nameInDialog: 'Browser Implementation in Edge Experimental',
    headerHtml: html`<img src="/public/img/edge-dev_24x24.png" /> Experimental`,
    cellRenderer: renderBrowserQualityExp,
    options: {browser: 'edge', channel: 'experimental'},
  },
  [ColumnKey.ExpFirefox]: {
    nameInDialog: 'Browser Implementation in Firefox Experimental',
    headerHtml: html`<img src="/public/img/firefox-nightly_24x24.png" />
      Experimental`,
    cellRenderer: renderBrowserQualityExp,
    options: {browser: 'firefox', channel: 'experimental'},
  },
  [ColumnKey.ExpSafari]: {
    nameInDialog: 'Browser Implementation in Safari Experimental',
    headerHtml: html`<img src="/public/img/safari-preview_24x24.png" />
      Experimental`,
    cellRenderer: renderBrowserQualityExp,
    options: {browser: 'safari', channel: 'experimental'},
  },
};

export function renderHeaderCell(
  routerLocation: {search: string},
  column: ColumnKey,
  sortSpec: string
): TemplateResult {
  let sortIndicator = html``;
  let urlWithSort = formatOverviewPageUrl(routerLocation, {
    sort: column + '_asc',
    start: 0,
  });
  if (sortSpec === column + '_asc') {
    sortIndicator = html` <sl-icon name="arrow-up"></sl-icon> `;
    urlWithSort = formatOverviewPageUrl(routerLocation, {
      sort: column + '_desc',
      start: 0,
    });
  } else if (sortSpec === column + '_desc') {
    sortIndicator = html` <sl-icon name="arrow-down"></sl-icon> `;
  }

  const colDef = CELL_DEFS[column];
  return html`
    <th title="Click to sort">
      <a href=${urlWithSort}> ${sortIndicator} ${colDef?.headerHtml} </a>
    </th>
  `;
}

export function renderFeatureCell(
  feature: components['schemas']['Feature'],
  routerLocation: {search: string},
  column: ColumnKey
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
      linkObj.link?.startsWith(JS_FEATURE_LINK_PREFIX)
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
