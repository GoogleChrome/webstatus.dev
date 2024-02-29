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
import {formatFeaturePageUrl} from '../utils/urls.js';

type CellRenderer = {
  (
    feature: components['schemas']['Feature'],
    routerLocation: {search: string}
  ): TemplateResult | typeof nothing;
};

type ColumnDefinition = {
  nameInDialog: string;
  headerHtml: TemplateResult;
  cellRenderer: CellRenderer;
};

export enum ColumnKey {
  Name = 'name',
  BaselineStatus = 'baseline_status',
  WptChrome = 'wpt_chrome',
  WptEdge = 'wpt_edge',
  WptFirefox = 'wpt_firefox',
  WptSafari = 'wpt_safari',
}

const columnKeyMapping = Object.entries(ColumnKey).reduce(
  (mapping, [enumKey, enumValue]) => {
    mapping[enumValue] = ColumnKey[enumKey as keyof typeof ColumnKey];
    return mapping;
  },
  {} as Record<string, ColumnKey>
);

interface BaselineChipConfig {
  cssClass: string;
  icon: string;
  word: string;
}

const BASELINE_CHIP_CONFIGS: Record<
  components['schemas']['Feature']['baseline_status'],
  BaselineChipConfig
> = {
  none: {
    cssClass: 'limited',
    icon: 'cross.svg',
    word: 'Limited',
  },
  low: {
    cssClass: 'newly',
    icon: 'cross.svg', // TODO(jrobbins): need dotted check
    word: 'New',
  },
  high: {
    cssClass: 'widely',
    icon: 'check.svg',
    word: 'Widely available',
  },
};

const renderFeatureName: CellRenderer = (feature, routerLocation) => {
  const featureUrl = formatFeaturePageUrl(feature, routerLocation);
  return html` <a href=${featureUrl}>${feature.name}</a> `;
};

const renderBaselineStatus: CellRenderer = (feature, _routerLocation) => {
  const baselineStatus = feature.baseline_status;
  const chipConfig = BASELINE_CHIP_CONFIGS[baselineStatus];
  return html`
    <span class="chip ${chipConfig.cssClass}">
      <img height="16" src="/public/img/${chipConfig.icon}" />
      ${chipConfig.word}
    </span>
  `;
};

const renderWPTChrome: CellRenderer = (_feature, _routerLocation) => {
  return html` 100% `;
};

const renderWPTEdge: CellRenderer = (_feature, _routerLocation) => {
  return html` 100% `;
};

const renderWPTFirefox: CellRenderer = (_feature, _routerLocation) => {
  return html` 100% `;
};

const renderWPTSafari: CellRenderer = (_feature, _routerLocation) => {
  return html` 100% `;
};

export const CELL_DEFS: Record<ColumnKey, ColumnDefinition> = {
  [ColumnKey.Name]: {
    nameInDialog: 'Feature name',
    headerHtml: html`Feature`,
    cellRenderer: renderFeatureName,
  },
  [ColumnKey.BaselineStatus]: {
    nameInDialog: 'Baseline status',
    headerHtml: html`Baseline`,
    cellRenderer: renderBaselineStatus,
  },
  [ColumnKey.WptChrome]: {
    nameInDialog: 'WPT score in Chrome',
    headerHtml: html`<img src="/public/img/chrome-dev_24x24.png" />`,
    cellRenderer: renderWPTChrome,
  },
  [ColumnKey.WptEdge]: {
    nameInDialog: 'WPT score in Edge',
    headerHtml: html`<img src="/public/img/edge-dev_24x24.png" />`,
    cellRenderer: renderWPTEdge,
  },
  [ColumnKey.WptFirefox]: {
    nameInDialog: 'WPT score in Firefox',
    headerHtml: html`<img src="/public/img/firefox-nightly_24x24.png" />`,
    cellRenderer: renderWPTFirefox,
  },
  [ColumnKey.WptSafari]: {
    nameInDialog: 'WPT score in Safari',
    headerHtml: html`<img src="/public/img/safari-preview_24x24.png" />`,
    cellRenderer: renderWPTSafari,
  },
};

export function renderHeaderCell(column: ColumnKey): TemplateResult {
  const colDef = CELL_DEFS[column];
  return colDef?.headerHtml || nothing;
}

export function renderFeatureCell(
  feature: components['schemas']['Feature'],
  routerLocation: {search: string},
  column: ColumnKey
): TemplateResult | typeof nothing {
  const colDef = CELL_DEFS[column];
  if (colDef?.cellRenderer) {
    return colDef.cellRenderer(feature, routerLocation);
  } else {
    return nothing;
  }
}

export function parseColumnsSpec(
  colSpec: string,
  defaults: ColumnKey[]
): ColumnKey[] {
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
    return defaults;
  }
}
