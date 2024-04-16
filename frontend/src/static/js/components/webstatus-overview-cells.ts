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
  WptChrome = 'wpt_chrome',
  WptEdge = 'wpt_edge',
  WptFirefox = 'wpt_firefox',
  WptSafari = 'wpt_safari',
  WptChromeExp = 'wpt_chrome_exp',
  WptEdgeExp = 'wpt_edge_exp',
  WptFirefoxExp = 'wpt_firefox_exp',
  WptSafariExp = 'wpt_safari_exp',
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
  ColumnKey.WptChrome,
  ColumnKey.WptEdge,
  ColumnKey.WptFirefox,
  ColumnKey.WptSafari,
];

export const DEFAULT_SORT_SPEC = 'name_asc';

interface BaselineChipConfig {
  cssClass: string;
  icon: string;
  word: string;
}

export const BASELINE_CHIP_CONFIGS: Record<
  components['schemas']['Feature']['baseline_status'],
  BaselineChipConfig
> = {
  undefined: {
    cssClass: 'limited',
    icon: 'cross.svg',
    word: 'Limited availability',
  },
  limited: {
    cssClass: 'limited',
    icon: 'cross.svg',
    word: 'Limited availability',
  },
  newly: {
    cssClass: 'newly',
    icon: 'cross.svg', // TODO(jrobbins): need dotted check
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
  const baselineStatus = feature.baseline_status;
  const chipConfig = BASELINE_CHIP_CONFIGS[baselineStatus];
  return html`
    <span class="chip ${chipConfig.cssClass}">
      <img height="16" src="/public/img/${chipConfig.icon}" />
      ${chipConfig.word}
    </span>
  `;
};

const BROWSER_IMPL_ICONS = {
  unknown: 'check-partial-circle',
  not: 'minus-circle',
  fully: 'check-circle',
};

export const renderWPTScore: CellRenderer = (
  feature,
  _routerLocation,
  {browser, channel}
) => {
  const score: number | undefined =
    channel === 'experimental'
      ? feature.wpt?.experimental?.[browser!]?.score
      : feature.wpt?.stable?.[browser!]?.score;
  let percentage = MISSING_VALUE;
  const browserImpl =
    feature.browser_implementations?.status.status || 'unknown';
  if (score !== undefined && browserImpl !== 'not') {
    percentage = html`${Number(score * 100).toFixed(1)}%`;
  }
  const iconName = BROWSER_IMPL_ICONS[browserImpl];
  return html`
    <div class="browser-impl-${browserImpl}">
      <sl-icon name="${iconName}" library="custom"></sl-icon>
      <span class="percent">${percentage}</span>
    </div>
  `;
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
  [ColumnKey.WptChrome]: {
    nameInDialog: 'WPT score in Chrome',
    headerHtml: html`<img src="/public/img/chrome_24x24.png" />`,
    cellRenderer: renderWPTScore,
    options: {browser: 'chrome', channel: 'stable'},
  },
  [ColumnKey.WptEdge]: {
    nameInDialog: 'WPT score in Edge',
    headerHtml: html`<img src="/public/img/edge_24x24.png" />`,
    cellRenderer: renderWPTScore,
    options: {browser: 'edge', channel: 'stable'},
  },
  [ColumnKey.WptFirefox]: {
    nameInDialog: 'WPT score in Firefox',
    headerHtml: html`<img src="/public/img/firefox_24x24.png" />`,
    cellRenderer: renderWPTScore,
    options: {browser: 'firefox', channel: 'stable'},
  },
  [ColumnKey.WptSafari]: {
    nameInDialog: 'WPT score in Safari',
    headerHtml: html`<img src="/public/img/safari_24x24.png" />`,
    cellRenderer: renderWPTScore,
    options: {browser: 'safari', channel: 'stable'},
  },
  [ColumnKey.WptChromeExp]: {
    nameInDialog: 'WPT score in Chrome Experimental',
    headerHtml: html`<img src="/public/img/chrome-canary_24x24.png" />
      Experimental`,
    cellRenderer: renderWPTScore,
    options: {browser: 'chrome', channel: 'experimental'},
  },
  [ColumnKey.WptEdgeExp]: {
    nameInDialog: 'WPT score in Edge Experimental',
    headerHtml: html`<img src="/public/img/edge-dev_24x24.png" /> Experimental`,
    cellRenderer: renderWPTScore,
    options: {browser: 'edge', channel: 'experimental'},
  },
  [ColumnKey.WptFirefoxExp]: {
    nameInDialog: 'WPT score in Firefox Experimental',
    headerHtml: html`<img src="/public/img/firefox-nightly_24x24.png" />
      Experimental`,
    cellRenderer: renderWPTScore,
    options: {browser: 'firefox', channel: 'experimental'},
  },
  [ColumnKey.WptSafariExp]: {
    nameInDialog: 'WPT score in Safari Experimental',
    headerHtml: html`<img src="/public/img/safari-preview_24x24.png" />
      Experimental`,
    cellRenderer: renderWPTScore,
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
  });
  if (sortSpec === column + '_asc') {
    sortIndicator = html` <sl-icon name="arrow-up"></sl-icon> `;
    urlWithSort = formatOverviewPageUrl(routerLocation, {
      sort: column + '_desc',
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
