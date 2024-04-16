/**
 * Copyright 2023 Google LLC
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

import createClient, {HeadersOptions, type FetchOptions} from 'openapi-fetch';
import {type components, type paths} from 'webstatus.dev-backend';

export type FeatureSortOrderType = NonNullable<
  paths['/v1/features']['get']['parameters']['query']
>['sort'];

export type FeatureSearchType = NonNullable<
  paths['/v1/features']['get']['parameters']['query']
>['q'];

export type BrowsersParameter = components['parameters']['browserPathParam'];
export type ChannelsParameter = components['parameters']['channelPathParam'];
export type WPTRunMetric = components['schemas']['WPTRunMetric'];
export type WPTRunMetricsPage = components['schemas']['WPTRunMetricsPage'];

// TODO. Remove once not behind UbP
const temporaryFetchOptions: FetchOptions<unknown> = {
  credentials: 'include',
};

// TODO. Remove once not behind UbP
// https://github.com/drwpow/openapi-typescript/issues/1431
const temporaryHeaders: HeadersOptions = {
  'Content-Type': null,
};

// Create a base64 string that is URL safe.
function base64urlEncode(str: string): string {
  return btoa(str)
    .replace(/\+/g, '-') // Replace '+' with '-'
    .replace(/\//g, '_') // Replace '/' with '_'
    .replace(/=+$/, ''); // Remove trailing '='
}

export class APIClient {
  private readonly client: ReturnType<typeof createClient<paths>>;
  constructor(baseUrl: string) {
    this.client = createClient<paths>({
      baseUrl,
      headers: temporaryHeaders,
    });
  }

  public async getFeature(
    featureId: string
  ): Promise<components['schemas']['Feature']> {
    const {data, error} = await this.client.GET('/v1/features/{feature_id}', {
      ...temporaryFetchOptions,
      params: {path: {feature_id: featureId}},
    });
    if (error !== undefined) {
      throw new Error(error?.message);
    }
    return data;
  }

  // Internal client detail for constructing a FeatureResultOffsetCursor pagination token.
  // Typically, users of the /v1/features endpoint should use the provided pagination token.
  // However, this token can be used to facilitate a UI with where we have selectable page numbers.
  // Disclaimer: External users should be aware that the format of this token is subject to change and should not be
  // treated as a stable interface. Instead, external users should rely on the returned pagination token long term.
  private createOffsetPaginationTokenForGetFeatures(offset: number): string {
    return base64urlEncode(JSON.stringify({offset: offset}));
  }

  public async getFeatures(
    q: FeatureSearchType,
    sort: FeatureSortOrderType,
    offset?: number
  ): Promise<components['schemas']['FeaturePage']> {
    const qsParams: paths['/v1/features']['get']['parameters']['query'] = {};
    if (q) qsParams.q = q;
    if (sort) qsParams.sort = sort;
    if (offset)
      qsParams.page_token =
        this.createOffsetPaginationTokenForGetFeatures(offset);
    const {data, error} = await this.client.GET('/v1/features', {
      ...temporaryFetchOptions,
      params: {
        query: qsParams,
      },
    });
    if (error !== undefined) {
      throw new Error(error?.message);
    }
    return data;
  }

  public async getStatsByBrowserAndChannel(
    browser: BrowsersParameter,
    channel: ChannelsParameter,
    startAtDate: Date,
    endAtDate: Date
  ): Promise<WPTRunMetric[]> {
    const startAt: string = startAtDate.toISOString().substring(0, 10);
    const endAt: string = endAtDate.toISOString().substring(0, 10);

    let nextPageToken;
    const allData: WPTRunMetric[] = [];
    do {
      const response = await this.client.GET(
        '/v1/stats/wpt/browsers/{browser}/channels/{channel}/test_counts',
        {
          ...temporaryFetchOptions,
          params: {
            query: {startAt, endAt, page_token: nextPageToken},
            path: {browser, channel},
          },
        }
      );
      const error = response.error;
      if (error !== undefined) {
        throw new Error(error?.message);
      }
      const page: WPTRunMetricsPage = response.data as WPTRunMetricsPage;
      nextPageToken = page?.metadata?.next_page_token;
      if (page != null) {
        allData.push(...page.data);
      }
    } while (nextPageToken !== undefined);

    return allData;
  }
}
