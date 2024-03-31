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
import {
  type components,
  type paths,
  type operations,
} from 'webstatus.dev-backend';

export type FeatureSortOrderType = NonNullable<
  paths['/v1/features']['get']['parameters']['query']
>['sort'];

export type FeatureSearchType = NonNullable<
  paths['/v1/features']['get']['parameters']['query']
>['q'];

export type BrowsersParameter = components['parameters']['browserPathParam'];
export type ChannelsParameter = components['parameters']['channelPathParam'];
export type WPTRunMetric = components['schemas']['WPTRunMetric'];

// TODO. Remove once not behind UbP
const temporaryFetchOptions: FetchOptions<unknown> = {
  credentials: 'include',
};

// TODO. Remove once not behind UbP
// https://github.com/drwpow/openapi-typescript/issues/1431
const temporaryHeaders: HeadersOptions = {
  'Content-Type': null,
};

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

  public async getFeatures(
    q: FeatureSearchType,
    sort: FeatureSortOrderType
  ): Promise<components['schemas']['FeaturePage']['data']> {
    const qsParams: {q?: FeatureSearchType; sort?: FeatureSortOrderType} = {};
    if (q) qsParams.q = q;
    if (sort) qsParams.sort = sort;
    const {data, error} = await this.client.GET('/v1/features', {
      params: {
        query: qsParams,
      },
      ...temporaryFetchOptions,
    });
    if (error !== undefined) {
      throw new Error(error?.message);
    }
    return data?.data;
  }

  public async getStatsByBrowserAndChannel(
    browser: BrowsersParameter,
    channel: ChannelsParameter,
    startAtDate: Date,
    endAtDate: Date
  ): Promise<WPTRunMetric[]> {
    const startAt: string = startAtDate.toISOString().substring(0, 10);
    const endAt: string = endAtDate.toISOString().substring(0, 10);
    const parameters: operations['listAggregatedWPTMetrics']['parameters'] = {
      query: {startAt, endAt},
      path: {browser, channel},
    };
    // Mystery: This seems to hide temporaryFetchOptions inside the params.
    // But isn't it needed at the top level of the dict passed to GET?
    // Why the type error when passed in tope level dict?
    const options = Object.assign(parameters, temporaryFetchOptions);
    const {data, error} = await this.client.GET(
      '/v1/stats/wpt/browsers/{browser}/channels/{channel}/test_counts',
      {params: options}
    );
    if (error !== undefined) {
      throw new Error(error?.message);
    }
    return data?.data;
  }
}
