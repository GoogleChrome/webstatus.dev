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

import createClient, {
  HeadersOptions,
  type FetchOptions,
  ParamsOption,
  ParseAsResponse,
} from 'openapi-fetch';
import {type components, type paths} from 'webstatus.dev-backend';
import {
  createAPIError,
  FeatureGoneSplitError,
  FeatureMovedError,
} from './errors.js';

import {
  MediaType,
  SuccessResponse,
  ResponseObjectMap,
  FilterKeys,
} from 'openapi-typescript-helpers';

export type FeatureSortOrderType = NonNullable<
  paths['/v1/features']['get']['parameters']['query']
>['sort'];

export type FeatureSearchType = NonNullable<
  paths['/v1/features']['get']['parameters']['query']
>['q'];

export type FeatureWPTMetricViewType = Exclude<
  NonNullable<
    paths['/v1/features']['get']['parameters']['query']
  >['wpt_metric_view'],
  undefined
>;

export type BrowsersParameter = components['parameters']['browserPathParam'];

type PageablePath =
  | '/v1/features'
  | '/v1/features/{feature_id}/stats/wpt/browsers/{browser}/channels/{channel}/{metric_view}'
  | '/v1/users/me/notification-channels'
  | '/v1/stats/features/browsers/{browser}/feature_counts'
  | '/v1/users/me/saved-searches'
  | '/v1/users/me/subscriptions'
  | '/v1/stats/baseline_status/low_date_feature_counts';

type SuccessResponsePageableData<
  T,
  Options,
  Media extends MediaType,
  Path extends PageablePath,
> = ParseAsResponse<
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  SuccessResponse<ResponseObjectMap<T> & Record<string | number, any>, Media>,
  Options
> & {
  metadata: Path extends '/v1/features' ? PageMetadataWithTotal : PageMetadata;
};

type PageMetadata = components['schemas']['PageMetadata'];
type PageMetadataWithTotal = components['schemas']['PageMetadataWithTotal'];

type ManualOffsetPagination = (offset: number) => string;

export type UpdateSavedSearchInput = {
  id: string;
  name?: string;
  description?: string | null;
  query?: string;
};

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
  'chrome_android',
  'firefox_android',
  'safari_ios',
];

/** Map from browser id to label */
export const BROWSER_ID_TO_LABEL: Record<BrowsersParameter, string> = {
  chrome: 'Chrome',
  firefox: 'Firefox',
  safari: 'Safari',
  edge: 'Edge',
  chrome_android: 'Chrome Android',
  firefox_android: 'Firefox Android',
  safari_ios: 'Safari iOS',
};

/** Map from label to browser id */
export const BROWSER_LABEL_TO_ID: Record<string, BrowsersParameter> =
  Object.fromEntries(
    Object.entries(BROWSER_ID_TO_LABEL).map(([key, value]) => [
      value,
      key as BrowsersParameter,
    ]),
  );

export const BROWSER_ID_TO_COLOR: Record<BrowsersParameter | 'total', string> =
  {
    chrome: '#34A853',
    chrome_android: '#34A853',
    firefox: '#F48400',
    firefox_android: '#F48400',
    safari: '#4285F4',
    safari_ios: '#4285F4',
    edge: '#7851A9',
    total: '#888888',
  };

export const BROWSER_ID_TO_ICON_NAME: Record<BrowsersParameter, string> = {
  chrome: 'chrome',
  chrome_android: 'chrome',
  firefox: 'firefox',
  firefox_android: 'firefox',
  safari: 'safari',
  safari_ios: 'safari',
  edge: 'edge',
};

export const CHANNEL_ID_TO_LABEL: Record<ChannelsParameter, string> = {
  stable: 'Stable',
  experimental: 'Experimental',
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

export const TEST_COUNT_METRIC_VIEW: components['schemas']['WPTMetricView'] =
  'test_counts';
export const SUBTEST_COUNT_METRIC_VIEW: components['schemas']['WPTMetricView'] =
  'subtest_counts';
export const DEFAULT_TEST_VIEW: components['schemas']['WPTMetricView'] =
  TEST_COUNT_METRIC_VIEW;

export type WPTRunMetric = components['schemas']['WPTRunMetric'];
export type WPTRunMetricsPage = components['schemas']['WPTRunMetricsPage'];
export type ChromeUsageStat = components['schemas']['ChromeUsageStat'];
export type ChromeDailyUsageStatsPage =
  components['schemas']['ChromeDailyStatsPage'];
export type BrowserReleaseFeatureMetric =
  components['schemas']['BrowserReleaseFeatureMetric'];
export type BrowserReleaseFeatureMetricsPage =
  components['schemas']['BrowserReleaseFeatureMetricsPage'];
export type BaselineStatusMetricsPage =
  components['schemas']['BaselineStatusMetricsPage'];
export type BaselineStatusMetric =
  components['schemas']['BaselineStatusMetric'];
export type MissingOneImplFeaturesPage =
  components['schemas']['MissingOneImplFeaturesPage'];
export type MissingOneImplFeaturesList =
  components['schemas']['MissingOneImplFeature'][];
export type SavedSearchResponse = components['schemas']['SavedSearchResponse'];

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

  // Internal client detail for constructing a FeatureResultOffsetCursor pagination token.
  // Typically, users of the /v1/features endpoint should use the provided pagination token.
  // However, this token can be used to facilitate a UI where we have selectable page numbers.
  // Disclaimer: External users should be aware that the format of this token is subject to change and should not be
  // treated as a stable interface. Instead, external users should rely on the returned pagination token long term.
  private createOffsetPaginationTokenForGetFeatures(offset: number): string {
    return base64urlEncode(JSON.stringify({offset: offset}));
  }

  /**
   * Returns one page of data.
   */
  public async getPageOfData<
    Path extends PageablePath,
    ResponseData extends SuccessResponsePageableData<
      paths[PageablePath]['get'],
      ParamsOption<Path>,
      'application/json',
      Path
    >,
  >(
    path: Path,
    params: FetchOptions<FilterKeys<paths[Path], 'get'>>,
    pageToken?: string,
    pageSize?: number,
  ): Promise<ResponseData> {
    // Add the pagination parameters to the query
    if (params.params === undefined) params.params = {};
    if (params.params.query === undefined) params.params.query = {};

    params.params.query.page_token = pageToken;
    params.params.query.page_size = pageSize;

    const options = {
      ...temporaryFetchOptions,
      ...params,
    };
    const {data, error} = await this.client.GET(path, options);

    if (error !== undefined) {
      throw createAPIError(error);
    }

    if (data === undefined) {
      throw createAPIError();
    }

    return data as ResponseData;
  }

  /** Returns all pages of data.  */
  public async getAllPagesOfData<
    Path extends PageablePath,
    ResponseData extends SuccessResponsePageableData<
      paths[PageablePath]['get'],
      ParamsOption<Path>,
      'application/json',
      Path
    >,
  >(
    path: Path,
    params: FetchOptions<FilterKeys<paths[Path], 'get'>>,
    overridenOffsetPaginator?: ManualOffsetPagination,
  ): Promise<ResponseData['data'][number][]> {
    let offset = 0;
    let nextPageToken;
    const allData: ResponseData['data'][number][] = [];

    do {
      const page: ResponseData = await this.getPageOfData<Path, ResponseData>(
        path,
        params,
        overridenOffsetPaginator
          ? overridenOffsetPaginator(offset)
          : nextPageToken,
        100,
      );

      nextPageToken = page?.metadata?.next_page_token;
      allData.push(...(page.data ?? []));
      offset += (page.data || []).length;
    } while (nextPageToken !== undefined);

    return allData;
  }

  public async getFeature(
    featureId: string,
    wptMetricView: FeatureWPTMetricViewType,
  ): Promise<components['schemas']['Feature']> {
    const qsParams: paths['/v1/features/{feature_id}']['get']['parameters']['query'] =
      {};
    if (wptMetricView) qsParams.wpt_metric_view = wptMetricView;
    const resp = await this.client.GET('/v1/features/{feature_id}', {
      ...temporaryFetchOptions,
      params: {
        path: {feature_id: featureId},
        query: qsParams,
      },
    });
    if (resp.error !== undefined) {
      const data = resp.error;
      if (resp.response.status === 410 && 'new_features' in data) {
        // Type narrowing doesn't work.
        // https://github.com/openapi-ts/openapi-typescript/issues/1723
        // We have to force it.
        const featureGoneData =
          data as components['schemas']['FeatureGoneError'];
        throw new FeatureGoneSplitError(
          resp.error.message,
          featureGoneData.new_features.map(f => f.id),
        );
      }
      throw createAPIError(resp.error);
    }
    if (resp.response.redirected) {
      const featureId = resp.response.url.split('/').pop() || '';
      throw new FeatureMovedError(
        'redirected to feature',
        featureId,
        resp.data,
      );
    }
    return resp.data;
  }

  public async getFeatureMetadata(
    featureId: string,
  ): Promise<components['schemas']['FeatureMetadata']> {
    const {data, error} = await this.client.GET(
      '/v1/features/{feature_id}/feature-metadata',
      {
        ...temporaryFetchOptions,
        params: {
          path: {feature_id: featureId},
        },
      },
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
    pageSize?: number,
  ): Promise<components['schemas']['FeaturePage']> {
    const queryParams: paths['/v1/features']['get']['parameters']['query'] = {};
    if (q) queryParams.q = q;
    if (sort) queryParams.sort = sort;
    if (wptMetricView) queryParams.wpt_metric_view = wptMetricView;
    const pageToken = offset
      ? this.createOffsetPaginationTokenForGetFeatures(offset)
      : undefined;

    return this.getPageOfData(
      '/v1/features',
      {params: {query: queryParams}},
      pageToken,
      pageSize,
    );
  }

  // Get all features
  public async getAllFeatures(
    q: FeatureSearchType,
    sort: FeatureSortOrderType,
    wptMetricView?: FeatureWPTMetricViewType,
  ): Promise<components['schemas']['Feature'][]> {
    const queryParams: paths['/v1/features']['get']['parameters']['query'] = {};
    if (q) queryParams.q = q;
    if (sort) queryParams.sort = sort;
    if (wptMetricView) queryParams.wpt_metric_view = wptMetricView;
    return this.getAllPagesOfData<
      '/v1/features',
      components['schemas']['FeaturePage']
    >(
      '/v1/features',
      {params: {query: queryParams}},
      this.createOffsetPaginationTokenForGetFeatures,
    );
  }

  // Get all saved searches for a user
  public async getAllUserSavedSearches(
    token: string,
  ): Promise<SavedSearchResponse[]> {
    type SavedSearchResponsePage = SuccessResponsePageableData<
      components['schemas']['SavedSearchResponse'][],
      ParamsOption<'/v1/users/me/saved-searches'>,
      'application/json',
      '/v1/users/me/saved-searches'
    >;
    return this.getAllPagesOfData<
      '/v1/users/me/saved-searches',
      SavedSearchResponsePage
    >('/v1/users/me/saved-searches', {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
  }

  public async listNotificationChannels(
    token: string,
  ): Promise<components['schemas']['NotificationChannelResponse'][]> {
    type NotificationChannelPage = SuccessResponsePageableData<
      paths['/v1/users/me/notification-channels']['get'],
      FetchOptions<
        FilterKeys<paths['/v1/users/me/notification-channels'], 'get'>
      >,
      'application/json',
      '/v1/users/me/notification-channels'
    >;

    return this.getAllPagesOfData<
      '/v1/users/me/notification-channels',
      NotificationChannelPage
    >('/v1/users/me/notification-channels', {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
  }

  public async pingUser(
    token: string,
    pingOptions?: {githubToken?: string},
  ): Promise<void> {
    const options: FetchOptions<
      FilterKeys<paths['/v1/users/me/ping'], 'post'>
    > = {
      headers: {
        Authorization: `Bearer ${token}`,
      },
      credentials: temporaryFetchOptions.credentials,
      body: {
        github_token: pingOptions?.githubToken,
      },
    };
    const {error} = await this.client.POST('/v1/users/me/ping', options);
    if (error) {
      throw createAPIError(error);
    }
  }

  public async *getFeatureStatsByBrowserAndChannel(
    featureId: string,
    browser: BrowsersParameter,
    channel: ChannelsParameter,
    startAtDate: Date,
    endAtDate: Date,
    metricView: components['schemas']['WPTMetricView'],
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
              metric_view: metricView,
            },
          },
        },
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

  public async *getChromeDailyUsageStats(
    featureId: string,
    startAtDate: Date,
    endAtDate: Date,
  ): AsyncIterable<ChromeUsageStat[]> {
    const startAt: string = startAtDate.toISOString().substring(0, 10);
    const endAt: string = endAtDate.toISOString().substring(0, 10);
    let nextPageToken;
    do {
      const response = await this.client.GET(
        '/v1/features/{feature_id}/stats/usage/chrome/daily_stats',
        {
          ...temporaryFetchOptions,
          params: {
            query: {startAt, endAt, page_token: nextPageToken},
            path: {
              feature_id: featureId,
            },
          },
        },
      );
      const error = response.error;
      if (error !== undefined) {
        throw createAPIError(error);
      }
      const page: ChromeDailyUsageStatsPage =
        response.data as ChromeDailyUsageStatsPage;
      nextPageToken = page?.metadata?.next_page_token;
      yield page.data;
    } while (nextPageToken !== undefined);
  }

  // Fetches feature counts for a browser in a date range
  // via "/v1/stats/features/browsers/{browser}/feature_counts"
  public async *getFeatureCountsForBrowser(
    browser: BrowsersParameter,
    startAtDate: Date,
    endAtDate: Date,
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
            query: {
              startAt,
              endAt,
              page_token: nextPageToken,
              include_baseline_mobile_browsers: true,
            },
            path: {browser},
          },
        },
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

  // Returns the count of features supported that have reached baseline
  // via "/v1/stats/baseline_status/low_date_feature_counts"
  public async *listAggregatedBaselineStatusCounts(
    startAtDate: Date,
    endAtDate: Date,
  ): AsyncIterable<BaselineStatusMetric[]> {
    const startAt: string = startAtDate.toISOString().substring(0, 10);
    const endAt: string = endAtDate.toISOString().substring(0, 10);

    let nextPageToken;
    do {
      const response = await this.client.GET(
        '/v1/stats/baseline_status/low_date_feature_counts',
        {
          ...temporaryFetchOptions,
          params: {
            query: {startAt, endAt, page_token: nextPageToken},
          },
        },
      );
      const error = response.error;
      if (error !== undefined) {
        throw createAPIError(error);
      }
      const page: BaselineStatusMetricsPage =
        response.data as BaselineStatusMetricsPage;
      nextPageToken = page?.metadata?.next_page_token;
      yield page.data; // Yield the entire page
    } while (nextPageToken !== undefined);
  }

  // Fetches feature counts for a browser in a date range
  // via "/v1/stats/features/browsers/{browser}/feature_counts"
  public async *getMissingOneImplementationCountsForBrowser(
    browser: BrowsersParameter,
    otherBrowsers: BrowsersParameter[],
    startAtDate: Date,
    endAtDate: Date,
  ): AsyncIterable<BrowserReleaseFeatureMetric[]> {
    const startAt: string = startAtDate.toISOString().substring(0, 10);
    const endAt: string = endAtDate.toISOString().substring(0, 10);

    let nextPageToken;
    do {
      const response = await this.client.GET(
        '/v1/stats/features/browsers/{browser}/missing_one_implementation_counts',
        {
          ...temporaryFetchOptions,
          params: {
            query: {
              startAt,
              endAt,
              page_token: nextPageToken,
              browser: otherBrowsers,
              include_baseline_mobile_browsers: true,
            },
            path: {browser},
          },
        },
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

  // Fetches missing feature list for a browser for a give date
  // via "/v1/stats/features/browsers/{browser}/missing_one_implementation_counts/{date}/features"
  public async getMissingOneImplementationFeatures(
    targetBrowser: BrowsersParameter,
    otherBrowsers: BrowsersParameter[],
    date: Date,
  ): Promise<MissingOneImplFeaturesList> {
    const targetDate: string = date.toISOString().substring(0, 10);
    let nextPageToken: string | undefined;
    const allFeatures: MissingOneImplFeaturesList = [];

    do {
      const response = await this.client.GET(
        '/v1/stats/features/browsers/{browser}/missing_one_implementation_counts/{date}/features',
        {
          ...temporaryFetchOptions,
          params: {
            query: {
              page_token: nextPageToken,
              browser: otherBrowsers,
              include_baseline_mobile_browsers: true,
            },
            path: {browser: targetBrowser, date: targetDate},
          },
        },
      );
      const error = response.error;
      if (error !== undefined) {
        throw createAPIError(error);
      }
      const page: MissingOneImplFeaturesPage =
        response.data as MissingOneImplFeaturesPage;

      if (page?.data) {
        allFeatures.push(...page.data);
      }

      nextPageToken = page?.metadata?.next_page_token;
    } while (nextPageToken !== undefined);

    return allFeatures;
  }

  public async getSavedSearchByID(
    searchID: string,
    token?: string,
  ): Promise<SavedSearchResponse> {
    const options = {
      ...temporaryFetchOptions,
      params: {
        path: {
          search_id: searchID,
        },
      },
    };
    // If the token is there, add it to the options
    if (token) {
      options.headers = {
        Authorization: `Bearer ${token}`,
      };
    }
    const response = await this.client.GET(
      '/v1/saved-searches/{search_id}',
      options,
    );
    const error = response.error;
    if (error !== undefined) {
      throw createAPIError(error);
    }

    return response.data;
  }

  public async removeSavedSearchByID(searchID: string, token: string) {
    const options = {
      ...temporaryFetchOptions,
      params: {
        path: {
          search_id: searchID,
        },
      },
      headers: {
        Authorization: `Bearer ${token}`,
      },
    };
    const response = await this.client.DELETE(
      '/v1/saved-searches/{search_id}',
      options,
    );
    const error = response.error;
    if (error !== undefined) {
      throw createAPIError(error);
    }

    return response.data;
  }

  public async putUserSavedSearchBookmark(searchID: string, token: string) {
    const options = {
      ...temporaryFetchOptions,
      params: {
        path: {
          search_id: searchID,
        },
      },
      headers: {
        Authorization: `Bearer ${token}`,
      },
    };
    const response = await this.client.PUT(
      '/v1/users/me/saved-searches/{search_id}/bookmark_status',
      options,
    );
    const error = response.error;
    if (error !== undefined) {
      throw createAPIError(error);
    }

    return response.data;
  }

  public async removeUserSavedSearchBookmark(searchID: string, token: string) {
    const options = {
      ...temporaryFetchOptions,
      params: {
        path: {
          search_id: searchID,
        },
      },
      headers: {
        Authorization: `Bearer ${token}`,
      },
    };
    const response = await this.client.DELETE(
      '/v1/users/me/saved-searches/{search_id}/bookmark_status',
      options,
    );
    const error = response.error;
    if (error !== undefined) {
      throw createAPIError(error);
    }

    return response.data;
  }

  public async createSavedSearch(
    token: string,
    savedSearch: components['schemas']['SavedSearch'],
  ): Promise<components['schemas']['SavedSearchResponse']> {
    const options: FetchOptions<
      FilterKeys<paths['/v1/saved-searches'], 'post'>
    > = {
      headers: {
        Authorization: `Bearer ${token}`,
      },
      body: savedSearch,
      credentials: temporaryFetchOptions.credentials,
    };
    const response = await this.client.POST('/v1/saved-searches', options);
    const error = response.error;
    if (error !== undefined) {
      throw createAPIError(error);
    }
    return response.data;
  }

  public async updateSavedSearch(
    savedSearch: UpdateSavedSearchInput,
    token: string,
  ): Promise<components['schemas']['SavedSearchResponse']> {
    const req: components['schemas']['SavedSearchUpdateRequest'] = {
      update_mask: [],
    };
    if (savedSearch.name !== undefined) {
      req.update_mask.push('name');
      req.name = savedSearch.name;
    }
    if (savedSearch.description !== undefined) {
      req.update_mask.push('description');
      req.description = savedSearch.description;
    }
    if (savedSearch.query !== undefined) {
      req.update_mask.push('query');
      req.query = savedSearch.query;
    }
    const options: FetchOptions<
      FilterKeys<paths['/v1/saved-searches/{search_id}'], 'patch'>
    > = {
      headers: {
        Authorization: `Bearer ${token}`,
      },
      params: {
        path: {
          search_id: savedSearch.id,
        },
      },
      body: req,
      credentials: temporaryFetchOptions.credentials,
    };
    const response = await this.client.PATCH(
      '/v1/saved-searches/{search_id}',
      options,
    );
    const error = response.error;
    if (error !== undefined) {
      throw createAPIError(error);
    }
    return response.data;
  }

  public async getSubscription(
    subscriptionId: string,
    token: string,
  ): Promise<components['schemas']['SubscriptionResponse']> {
    const options = {
      ...temporaryFetchOptions,
      params: {
        path: {
          subscription_id: subscriptionId,
        },
      },
      headers: {
        Authorization: `Bearer ${token}`,
      },
    };
    const response = await this.client.GET(
      '/v1/users/me/subscriptions/{subscription_id}',
      options,
    );
    const error = response.error;
    if (error !== undefined) {
      throw createAPIError(error);
    }

    return response.data;
  }

  public async deleteSubscription(subscriptionId: string, token: string) {
    const options = {
      ...temporaryFetchOptions,
      params: {
        path: {
          subscription_id: subscriptionId,
        },
      },
      headers: {
        Authorization: `Bearer ${token}`,
      },
    };
    const response = await this.client.DELETE(
      '/v1/users/me/subscriptions/{subscription_id}',
      options,
    );
    const error = response.error;
    if (error !== undefined) {
      throw createAPIError(error);
    }

    return response.data;
  }

  public async listSubscriptions(
    token: string,
  ): Promise<components['schemas']['SubscriptionResponse'][]> {
    type SubscriptionPage = SuccessResponsePageableData<
      paths['/v1/users/me/subscriptions']['get'],
      ParamsOption<'/v1/users/me/subscriptions'>,
      'application/json',
      '/v1/users/me/subscriptions'
    >;

    return this.getAllPagesOfData<
      '/v1/users/me/subscriptions',
      SubscriptionPage
    >('/v1/users/me/subscriptions', {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
  }

  public async createSubscription(
    token: string,
    subscription: components['schemas']['Subscription'],
  ): Promise<components['schemas']['SubscriptionResponse']> {
    const options: FetchOptions<
      FilterKeys<paths['/v1/users/me/subscriptions'], 'post'>
    > = {
      headers: {
        Authorization: `Bearer ${token}`,
      },
      body: subscription,
      credentials: temporaryFetchOptions.credentials,
    };
    const response = await this.client.POST(
      '/v1/users/me/subscriptions',
      options,
    );
    const error = response.error;
    if (error !== undefined) {
      throw createAPIError(error);
    }
    return response.data;
  }

  public async updateSubscription(
    subscriptionId: string,
    token: string,
    updates: {
      triggers?: components['schemas']['SubscriptionTriggerWritable'][];
      frequency?: components['schemas']['SubscriptionFrequency'];
    },
  ): Promise<components['schemas']['SubscriptionResponse']> {
    const req: components['schemas']['UpdateSubscriptionRequest'] = {
      update_mask: [],
    };
    if (updates.triggers !== undefined) {
      req.update_mask.push('triggers');
      req.triggers = updates.triggers;
    }
    if (updates.frequency !== undefined) {
      req.update_mask.push('frequency');
      req.frequency = updates.frequency;
    }
    const options: FetchOptions<
      FilterKeys<paths['/v1/users/me/subscriptions/{subscription_id}'], 'patch'>
    > = {
      headers: {
        Authorization: `Bearer ${token}`,
      },
      params: {
        path: {
          subscription_id: subscriptionId,
        },
      },
      body: req,
      credentials: temporaryFetchOptions.credentials,
    };
    const response = await this.client.PATCH(
      '/v1/users/me/subscriptions/{subscription_id}',
      options,
    );
    const error = response.error;
    if (error !== undefined) {
      throw createAPIError(error);
    }
    return response.data;
  }
}
