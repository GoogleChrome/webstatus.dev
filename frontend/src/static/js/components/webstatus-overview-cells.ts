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

export const CELL_DEFS: Record<string, ColumnDefinition> = {
  name: {
    nameInDialog: 'Feature name',
    headerHtml: html`Feature`,
    cellRenderer: renderFeatureName,
  },
  baseline_status: {
    nameInDialog: 'Feature name',
    headerHtml: html`Baseline`,
    cellRenderer: renderBaselineStatus,
  },
  wpt_chrome: {
    nameInDialog: 'Feature name',
    headerHtml: html`<img src="/public/img/chrome-dev_24x24.png" />`,
    cellRenderer: renderWPTChrome,
  },
  wpt_edge: {
    nameInDialog: 'Feature name',
    headerHtml: html`<img src="/public/img/edge-dev_24x24.png" />`,
    cellRenderer: renderWPTEdge,
  },
  wpt_firefox: {
    nameInDialog: 'Feature name',
    headerHtml: html`<img src="/public/img/firefox-nightly_24x24.png" />`,
    cellRenderer: renderWPTFirefox,
  },
  wpt_safari: {
    nameInDialog: 'Feature name',
    headerHtml: html`<img src="/public/img/safari-preview_24x24.png" />`,
    cellRenderer: renderWPTSafari,
  },
};

export function renderHeaderCell(column: string): TemplateResult {
  const colDef = CELL_DEFS[column];
  return colDef?.headerHtml || nothing;
}

export function renderFeatureCell(
  feature: components['schemas']['Feature'],
  routerLocation: {search: string},
  column: string
): TemplateResult | typeof nothing {
  const colDef = CELL_DEFS[column];
  if (colDef?.cellRenderer) {
    return colDef.cellRenderer(feature, routerLocation);
  } else {
    return nothing;
  }
}

export function parseColumnsSpec(colSpec: string) {
  let cols = colSpec.toLowerCase().split(',');
  cols = cols.map(s => s.trim()).filter(c => c);
  return cols;
}
