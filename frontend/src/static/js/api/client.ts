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
import {createAPIError} from './errors.js';

// import type {PathsWithMethod} from 'openapi-typescript-helpers';

// Local copy of PathsWithMethod from openapi-typescript-helpers,
// since eslint complains that the type is not required, but it is.
type HttpMethod =
  | 'get'
  | 'put'
  | 'post'
  | 'delete'
  | 'options'
  | 'head'
  | 'patch'
  | 'trace';

type PathsWithMethod<Paths extends {}, PathnameMethod extends HttpMethod> = {
  [Pathname in keyof Paths]: Paths[Pathname] extends {
    [K in PathnameMethod]: unknown;
  }
    ? Pathname
    : never;
}[keyof Paths];

export type FeatureSortOrderType = NonNullable<
  paths['/v1/features']['get']['parameters']['query']
>['sort'];

export type FeatureSearchType = NonNullable<
  paths['/v1/features']['get']['parameters']['query']
>['q'];

export type FeatureWPTMetricViewType = NonNullable<
  paths['/v1/features']['get']['parameters']['query']
>['wpt_metric_view'];

export type BrowsersParameter = components['parameters']['browserPathParam'];

/**
 * Iterable list of browsers we have data for.
 * This is the same as the items in the BrowsersParameter enum,
 * but there is no way to get the values from the parameter types,
 * so we have to redundantly specify them here.
 */
export const ALL_BROWSERS: BrowsersParameter[] = [
  'chrome',
  'firefox',
  'safari',
  'edge',
];

/** Map from browser id to label */
export const BROWSER_ID_TO_LABEL: Record<BrowsersParameter, string> = {
  chrome: 'Chrome',
  firefox: 'Firefox',
  safari: 'Safari',
  edge: 'Edge',
};

export const BROWSER_ID_TO_COLOR: Record<BrowsersParameter | 'total', string> =
  {
    chrome: '#FF0000',
    firefox: '#F48400',
    safari: '#4285F4',
    edge: '#0F9D58',
    total: '#888888',
  };

export type ChannelsParameter = components['parameters']['channelPathParam'];

export const STABLE_CHANNEL: ChannelsParameter = 'stable';
export const EXPERIMENTAL_CHANNEL: ChannelsParameter = 'experimental';

/**
 * Iterable list of all channels.
 */
export const ALL_CHANNELS: ChannelsParameter[] = [
  STABLE_CHANNEL,
  EXPERIMENTAL_CHANNEL,
];

const DEFAULT_TEST_VIEW: components['schemas']['WPTMetricView'] =
  'subtest_counts';

export type WPTRunMetric = components['schemas']['WPTRunMetric'];
export type WPTRunMetricsPage = components['schemas']['WPTRunMetricsPage'];
export type BrowserReleaseFeatureMetric =
  components['schemas']['BrowserReleaseFeatureMetric'];
export type BrowserReleaseFeatureMetricsPage =
  components['schemas']['BrowserReleaseFeatureMetricsPage'];

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

interface PageParams extends Record<string, unknown> {
  page_token?: components['parameters']['paginationTokenParam'];
  page_size?: components['parameters']['paginationSizeParam'];
}

export class APIClient {
  private readonly client: ReturnType<typeof createClient<paths>>;
  constructor(baseUrl: string) {
    this.client = createClient<paths>({
      baseUrl,
      headers: temporaryHeaders,
    });
  }

  // Internal client detail for constructing a FeatureResultOffsetCursor pagination token.
  // Typically, users of the /v1/features endpoint should use the provided pagination token.
  // However, this token can be used to facilitate a UI with where we have selectable page numbers.
  // Disclaimer: External users should be aware that the format of this token is subject to change and should not be
  // treated as a stable interface. Instead, external users should rely on the returned pagination token long term.
  private createOffsetPaginationTokenForGetFeatures(offset: number): string {
    return base64urlEncode(JSON.stringify({offset: offset}));
  }

  /**
   * Returns one page of data.
   * TODO: needs to be generalized further to
   *   handle more variations of queries. */
  public async getPageOfData<
    ResponseType extends {
      data: unknown[];
      metadata?: {next_page_token?: string};
    },
    ParamsType extends Record<string, unknown>,
  >(
    path: PathsWithMethod<paths, 'get'>,
    params: ParamsType = {} as ParamsType,
    offset?: number,
    pageSize?: number
  ): Promise<ResponseType> {
    const qsParams: ParamsType & PageParams = {
      ...params,
      page_token: offset
        ? this.createOffsetPaginationTokenForGetFeatures(offset)
        : undefined,
      page_size: pageSize,
    };

    const {data, error} = await this.client.GET(path, {
      ...temporaryFetchOptions,
      params: {
        query: qsParams,
      },
    });

    if (error !== undefined) {
      throw createAPIError(error);
    }

    return data as ResponseType;
  }

  /** Returns all pages of data.  */
  public async getAllPagesOfData<
    PageType extends WPTRunMetricsPage | BrowserReleaseFeatureMetricsPage,
    ResponseType extends {
      data: unknown[];
      metadata?: {next_page_token?: string};
    },
    ParamsType extends Record<string, unknown>,
  >(
    path: PathsWithMethod<paths, 'get'>,
    params: ParamsType = {} as ParamsType
  ): Promise<ResponseType[]> {
    let offset = 0;
    let nextPageToken;
    const allData: ResponseType[] = [];

    do {
      const page: PageType = (await this.getPageOfData<
        ResponseType,
        ParamsType
      >(path, params, offset, 100)) as unknown as PageType;
      nextPageToken = page?.metadata?.next_page_token;
      allData.push(...((page.data || []) as unknown as ResponseType[]));
      offset += (page.data || []).length;
    } while (nextPageToken !== undefined);

    return allData;
  }

  public async getFeature(
    featureId: string,
    wptMetricView: FeatureWPTMetricViewType
  ): Promise<components['schemas']['Feature']> {
    const qsParams: paths['/v1/features/{feature_id}']['get']['parameters']['query'] =
      {};
    if (wptMetricView) qsParams.wpt_metric_view = wptMetricView;
    const {data, error} = await this.client.GET('/v1/features/{feature_id}', {
      ...temporaryFetchOptions,
      params: {
        path: {feature_id: featureId},
        query: qsParams,
      },
    });
    if (error !== undefined) {
      throw createAPIError(error);
    }
    return data;
  }

  public async getFeatureMetadata(
    featureId: string
  ): Promise<components['schemas']['FeatureMetadata']> {
    const {data, error} = await this.client.GET(
      '/v1/features/{feature_id}/feature-metadata',
      {
        ...temporaryFetchOptions,
        params: {
          path: {feature_id: featureId},
        },
      }
    );
    if (error !== undefined) {
      throw createAPIError(error);
    }
    return data;
  }

  // Get one page of features
  public async getFeatures(
    q: FeatureSearchType,
    sort: FeatureSortOrderType,
    wptMetricView?: FeatureWPTMetricViewType,
    offset?: number,
    pageSize?: number
  ): Promise<components['schemas']['FeaturePage']> {
    const params: paths['/v1/features']['get']['parameters']['query'] = {};
    if (q) params.q = q;
    if (sort) params.sort = sort;
    if (wptMetricView) params.wpt_metric_view = wptMetricView;
    return this.getPageOfData('/v1/features', params, offset, pageSize);
  }

  // Get all features
  public async getAllFeatures(
    q: FeatureSearchType,
    sort: FeatureSortOrderType,
    wptMetricView?: FeatureWPTMetricViewType
  ): Promise<components['schemas']['Feature'][]> {
    const params: paths['/v1/features']['get']['parameters']['query'] = {};
    if (q) params.q = q;
    if (sort) params.sort = sort;
    if (wptMetricView) params.wpt_metric_view = wptMetricView;
    return this.getAllPagesOfData(
      '/v1/features',
      params
    ) as unknown as components['schemas']['Feature'][];
  }

  public async *getFeatureStatsByBrowserAndChannel(
    featureId: string,
    browser: BrowsersParameter,
    channel: ChannelsParameter,
    startAtDate: Date,
    endAtDate: Date
  ): AsyncIterable<WPTRunMetric[]> {
    const startAt: string = startAtDate.toISOString().substring(0, 10);
    const endAt: string = endAtDate.toISOString().substring(0, 10);

    let nextPageToken;
    do {
      const response = await this.client.GET(
        '/v1/features/{feature_id}/stats/wpt/browsers/{browser}/channels/{channel}/{metric_view}',
        {
          ...temporaryFetchOptions,
          params: {
            query: {startAt, endAt, page_token: nextPageToken},
            path: {
              feature_id: featureId,
              browser,
              channel,
              metric_view: DEFAULT_TEST_VIEW,
            },
          },
        }
      );
      const error = response.error;
      if (error !== undefined) {
        throw createAPIError(error);
      }
      const page: WPTRunMetricsPage = response.data as WPTRunMetricsPage;
      nextPageToken = page?.metadata?.next_page_token;

      yield page.data; // Yield the entire page
    } while (nextPageToken !== undefined);
  }

  // Fetches feature counts for a browser in a date range
  // via "/v1/stats/features/browsers/{browser}/feature_counts"
  public async *getFeatureCountsForBrowser(
    browser: BrowsersParameter,
    startAtDate: Date,
    endAtDate: Date
  ): AsyncIterable<BrowserReleaseFeatureMetric[]> {
    const startAt: string = startAtDate.toISOString().substring(0, 10);
    const endAt: string = endAtDate.toISOString().substring(0, 10);

    let nextPageToken;
    do {
      const response = await this.client.GET(
        '/v1/stats/features/browsers/{browser}/feature_counts',
        {
          ...temporaryFetchOptions,
          params: {
            query: {startAt, endAt, page_token: nextPageToken},
            path: {browser},
          },
        }
      );
      const error = response.error;
      if (error !== undefined) {
        throw createAPIError(error);
      }
      const page: BrowserReleaseFeatureMetricsPage =
        response.data as BrowserReleaseFeatureMetricsPage;
      nextPageToken = page?.metadata?.next_page_token;
      yield page.data; // Yield the entire page
    } while (nextPageToken !== undefined);
  }
}
