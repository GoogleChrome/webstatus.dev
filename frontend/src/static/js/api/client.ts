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

import createClient, {type FetchOptions} from 'openapi-fetch';

import {type components, type paths} from 'webstatus.dev-backend';
import {
  createAPIError,
  FeatureGoneSplitError,
  FeatureMovedError,
} from './errors.js';

import {FilterKeys} from 'openapi-typescript-helpers';

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

/**
 * Union of API paths that support pagination.
 * We explicitly list these to enable path-aware type inference for paginated data.
 */
type PageablePath =
  | '/v1/features'
  | '/v1/features/{feature_id}/stats/wpt/browsers/{browser}/channels/{channel}/{metric_view}'
  | '/v1/features/{feature_id}/stats/usage/chrome/daily_stats'
  | '/v1/users/me/notification-channels'
  | '/v1/stats/features/browsers/{browser}/feature_counts'
  | '/v1/users/me/saved-searches'
  | '/v1/global-saved-searches'
  | '/v1/users/me/subscriptions'
  | '/v1/stats/baseline_status/low_date_feature_counts'
  | '/v1/stats/features/browsers/{browser}/missing_one_implementation_counts'
  | '/v1/stats/features/browsers/{browser}/missing_one_implementation_counts/{date}/features';

/**
 * Utility to extract the item type from a paginated API response.
 *
 * Uses 'infer' to automatically discover the array element type from the OpenAPI schema.
 * If the path is not a valid paginated path, it resolves to 'never' to prevent unsafe usage.
 */
type PageItems<Path extends PageablePath> = paths[Path]['get'] extends {
  responses: {
    200: {
      content: {
        'application/json': {
          data: (infer U)[];
        };
      };
    };
  };
}
  ? U
  : never;

type PageMetadata = components['schemas']['PageMetadata'];
type PageMetadataWithTotal = components['schemas']['PageMetadataWithTotal'];

export type AnyPageMetadata = PageMetadata & Partial<PageMetadataWithTotal>;

export type SuccessResponsePageableData<Path extends PageablePath> = {
  metadata: AnyPageMetadata;
  data: PageItems<Path>[];
};

type ResponsesObject<
  Path extends keyof paths,
  Method extends keyof paths[Path],
> = paths[Path][Method] extends {responses: infer R} ? R : never;

/**
 * Extracts the payload type for a specific status code of an API endpoint.
 * This ensures that status code handlers receive the correctly typed data
 * from the OpenAPI schema.
 */
type ResponsePayload<
  Path extends keyof paths,
  Method extends keyof paths[Path],
  Status extends keyof ResponsesObject<Path, Method>,
> = ResponsesObject<Path, Method>[Status] extends {
  content: {'application/json': infer T};
}
  ? T
  : undefined;

type ManualOffsetPagination = (offset: number) => string;

export type UpdateSavedSearchInput = {
  id: string;
  name?: string;
  description?: string | null;
  query?: string;
};

export type UpdateNotificationChannelInput = {
  name?: string;
  config?: components['schemas']['WebhookConfig'];
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
  Object.fromEntries(ALL_BROWSERS.map(key => [BROWSER_ID_TO_LABEL[key], key]));

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

export const DEFAULT_SORT_ORDER: FeatureSortOrderType = 'name_asc';

export function isWPTMetricViewType(
  val: string | null | undefined,
): val is FeatureWPTMetricViewType {
  return val === TEST_COUNT_METRIC_VIEW || val === SUBTEST_COUNT_METRIC_VIEW;
}

export function isFeatureSortOrderType(
  val: string | null | undefined,
): val is FeatureSortOrderType {
  if (!val) return false;
  return val.endsWith('_asc') || val.endsWith('_desc');
}

/**
 * Type guard to verify if a response matches the expected paginated structure for a given path.
 *
 * This acts as a 'gatekeeper' between the raw API response and our typed internal logic,
 * replacing the need for unsafe 'as' assertions.
 */
function isPageAtPath<Path extends PageablePath>(
  val: {} | null | undefined,
  _path: Path,
): val is {metadata?: AnyPageMetadata; data?: PageItems<Path>[]} {
  return (
    val !== null &&
    typeof val === 'object' &&
    // Not all endpoints return a 'data' field, but if it does exist it must be an array.
    (('data' in val && Array.isArray(val.data)) || !Object.hasOwn(val, 'data'))
  );
}

function isFeatureGoneError(
  val: {} | null | undefined,
): val is components['schemas']['FeatureGoneError'] {
  return (
    val !== null &&
    typeof val === 'object' &&
    'new_features' in val &&
    Array.isArray(val.new_features)
  );
}

function isFeature(
  val: {} | null | undefined,
): val is components['schemas']['Feature'] {
  return (
    val !== null &&
    typeof val === 'object' &&
    'feature_id' in val &&
    'name' in val
  );
}

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

const fetchOptions = {
  // TODO. Remove once not behind UbP
  credentials: 'include' as const,
  // https://github.com/drwpow/openapi-typescript/issues/1431

  headers: {
    'Content-Type': null,
  },
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
      ...fetchOptions,
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
   * Retrieves a single page of data with path-aware type inference.
   *
   * This method ensures that the returned data is perfectly synchronized with the OpenAPI
   * schema for the specific path, eliminating the need for manual type casting.
   */
  public async getPageOfData<Path extends PageablePath>(
    path: Path,
    params: FetchOptions<FilterKeys<paths[Path], 'get'>>,
    pageToken?: string,
    pageSize?: number,
  ): Promise<SuccessResponsePageableData<Path>> {
    // Add the pagination parameters to the query
    if (params.params === undefined) params.params = {};
    if (params.params.query === undefined) params.params.query = {};

    params.params.query.page_token = pageToken;
    params.params.query.page_size = pageSize;

    const result = await this.handleResponse(
      this.client.GET(path, params),
      path,
      'get',
    );

    if (isPageAtPath(result, path)) {
      return {
        metadata: result.metadata ?? {},
        data: result.data ?? [],
      };
    }

    throw createAPIError(new Error('Response data missing data array'));
  }

  /** Returns all pages of data.  */
  public async getAllPagesOfData<Path extends PageablePath>(
    path: Path,
    params: FetchOptions<FilterKeys<paths[Path], 'get'>>,
    overridenOffsetPaginator?: ManualOffsetPagination,
  ): Promise<PageItems<Path>[]> {
    let offset = 0;
    let nextPageToken: string | undefined = undefined;
    const allData: PageItems<Path>[] = [];

    while (true) {
      const page: SuccessResponsePageableData<Path> = await this.getPageOfData(
        path,
        params,
        nextPageToken ||
          (overridenOffsetPaginator && overridenOffsetPaginator(offset)),
        100,
      );

      nextPageToken = page.metadata.next_page_token;
      allData.push(...page.data);
      offset += page.data.length;
      if (nextPageToken === undefined) {
        break;
      }
    }

    return allData;
  }

  /**
   * Returns an async iterable that yields pages of data.
   */
  public async *getAsyncIterableOfData<Path extends PageablePath>(
    path: Path,
    params: FetchOptions<FilterKeys<paths[Path], 'get'>>,
    pageSize?: number,
  ): AsyncIterable<PageItems<Path>[]> {
    let nextPageToken: string | undefined = undefined;
    while (true) {
      const page: SuccessResponsePageableData<Path> = await this.getPageOfData(
        path,
        params,
        nextPageToken,
        pageSize,
      );
      yield page.data;
      nextPageToken = page.metadata.next_page_token;
      if (nextPageToken === undefined) {
        break;
      }
    }
  }

  /**
   * Type-safe error handler for any API response.
   * Leverages openapi-fetch result types to ensure data extraction is safe and concise.
   */
  private async handleResponse<
    T,
    ErrorType,
    Path extends keyof paths,
    Method extends keyof paths[Path] & string,
  >(
    promise: Promise<{
      data?: T;
      error?: ErrorType;
      response: Response;
    }>,
    _path: Path,
    _method: Method,
    options?: {
      statusHandlers?: Partial<{
        [K in keyof ResponsesObject<Path, Method>]: (
          payload: ResponsePayload<Path, Method, K>,
          data: T | undefined,
          response: Response,
        ) => void;
      }>;
    },
  ): Promise<T> {
    const result = await promise;

    if (options?.statusHandlers) {
      // Use Object.entries to safely iterate without needing narrow type assertions.
      for (const [code, handler] of Object.entries(options.statusHandlers)) {
        if (
          Number(code) === result.response.status &&
          typeof handler === 'function'
        ) {
          const payload =
            result.response.status >= 200 && result.response.status < 300
              ? result.data
              : result.error;
          handler.call(
            options.statusHandlers,
            payload,
            result.data,
            result.response,
          );
        }
      }
    }

    if (result.error !== undefined) {
      throw createAPIError(result.error, result.response.status);
    }
    if (!result.response.ok) {
      throw createAPIError(undefined, result.response.status);
    }

    // Now we know it's a success response, and openapi-fetch guarantees data matches the success type.
    // In case of 204 No Content, result.data is undefined and T is void/undefined.
    // We use a non-null assertion here to bridge the gap between the complex union type and
    // the generic return type T, following the restriction on 'as' assertions.
    return result.data!;
  }

  public async getFeature(
    featureId: string,
    wptMetricView: FeatureWPTMetricViewType,
  ): Promise<components['schemas']['Feature']> {
    const qsParams: paths['/v1/features/{feature_id}']['get']['parameters']['query'] =
      {};
    if (wptMetricView) qsParams.wpt_metric_view = wptMetricView;
    return this.handleResponse(
      this.client.GET('/v1/features/{feature_id}', {
        params: {
          path: {feature_id: featureId},
          query: qsParams,
        },
      }),
      '/v1/features/{feature_id}',
      'get',
      {
        statusHandlers: {
          200: (data, _d, response) => {
            if (response.redirected && isFeature(data)) {
              const newId = response.url.split('/').pop() || '';
              throw new FeatureMovedError('Redirected', newId, data);
            }
          },
          410: error => {
            if (isFeatureGoneError(error)) {
              throw new FeatureGoneSplitError(
                error.message,
                error.new_features.map(f => f.id),
              );
            }
          },
        },
      },
    );
  }

  public getFeatureMetadata(
    featureId: string,
  ): Promise<components['schemas']['FeatureMetadata']> {
    return this.handleResponse(
      this.client.GET('/v1/features/{feature_id}/feature-metadata', {
        params: {
          path: {feature_id: featureId},
        },
      }),
      '/v1/features/{feature_id}/feature-metadata',
      'get',
    );
  }

  // Get one page of features
  public async getFeatures(
    q?: FeatureSearchType,
    sort?: FeatureSortOrderType,
    wptMetricView?: FeatureWPTMetricViewType,
    offset?: number,
    pageSize?: number,
  ): Promise<SuccessResponsePageableData<'/v1/features'>> {
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
    q?: FeatureSearchType,
    sort?: FeatureSortOrderType,
    wptMetricView?: FeatureWPTMetricViewType,
  ): Promise<components['schemas']['Feature'][]> {
    const queryParams: paths['/v1/features']['get']['parameters']['query'] = {};
    if (q) queryParams.q = q;
    if (sort) queryParams.sort = sort;
    if (wptMetricView) queryParams.wpt_metric_view = wptMetricView;
    return this.getAllPagesOfData(
      '/v1/features',
      {params: {query: queryParams}},
      this.createOffsetPaginationTokenForGetFeatures.bind(this),
    );
  }

  // Get all saved searches for a user
  public async getAllUserSavedSearches(
    token: string,
  ): Promise<SavedSearchResponse[]> {
    return this.getAllPagesOfData('/v1/users/me/saved-searches', {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
  }

  public async listNotificationChannels(
    token: string,
  ): Promise<components['schemas']['NotificationChannelResponse'][]> {
    return this.getAllPagesOfData('/v1/users/me/notification-channels', {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
  }

  public createNotificationChannel(
    token: string,
    channel: components['schemas']['CreateNotificationChannelRequest'],
  ): Promise<components['schemas']['NotificationChannelResponse']> {
    return this.handleResponse(
      this.client.POST('/v1/users/me/notification-channels', {
        headers: {
          Authorization: `Bearer ${token}`,
        },
        body: channel,
      }),
      '/v1/users/me/notification-channels',
      'post',
    );
  }

  public updateNotificationChannel(
    token: string,
    channelId: string,
    updates: UpdateNotificationChannelInput,
  ): Promise<components['schemas']['NotificationChannelResponse']> {
    const req: components['schemas']['UpdateNotificationChannelRequest'] = {
      update_mask: [],
    };
    if (updates.name !== undefined) {
      req.update_mask.push('name');
      req.name = updates.name;
    }
    if (updates.config !== undefined) {
      req.update_mask.push('config');
      req.config = updates.config;
    }
    return this.handleResponse(
      this.client.PATCH('/v1/users/me/notification-channels/{channel_id}', {
        headers: {
          Authorization: `Bearer ${token}`,
        },
        params: {
          path: {
            channel_id: channelId,
          },
        },
        body: req,
      }),
      '/v1/users/me/notification-channels/{channel_id}',
      'patch',
    );
  }

  public deleteNotificationChannel(
    token: string,
    channelId: string,
  ): Promise<void> {
    return this.handleResponse(
      this.client.DELETE('/v1/users/me/notification-channels/{channel_id}', {
        params: {
          path: {
            channel_id: channelId,
          },
        },
        headers: {
          Authorization: `Bearer ${token}`,
        },
      }),
      '/v1/users/me/notification-channels/{channel_id}',
      'delete',
    );
  }

  public async pingUser(
    token: string,
    pingOptions?: {githubToken?: string},
  ): Promise<void> {
    await this.handleResponse(
      this.client.POST('/v1/users/me/ping', {
        headers: {
          Authorization: `Bearer ${token}`,
        },
        body: {
          github_token: pingOptions?.githubToken,
        },
      }),
      '/v1/users/me/ping',
      'post',
    );
  }

  public getFeatureStatsByBrowserAndChannel(
    featureId: string,
    browser: BrowsersParameter,
    channel: ChannelsParameter,
    startAtDate: Date,
    endAtDate: Date,
    metricView: components['schemas']['WPTMetricView'],
  ): AsyncIterable<WPTRunMetric[]> {
    const startAt = startAtDate.toISOString().substring(0, 10);
    const endAt = endAtDate.toISOString().substring(0, 10);

    return this.getAsyncIterableOfData(
      '/v1/features/{feature_id}/stats/wpt/browsers/{browser}/channels/{channel}/{metric_view}',
      {
        params: {
          query: {startAt, endAt},
          path: {
            feature_id: featureId,
            browser,
            channel,
            metric_view: metricView,
          },
        },
      },
    );
  }

  public getChromeDailyUsageStats(
    featureId: string,
    startAtDate: Date,
    endAtDate: Date,
  ): AsyncIterable<ChromeUsageStat[]> {
    const startAt = startAtDate.toISOString().substring(0, 10);
    const endAt = endAtDate.toISOString().substring(0, 10);
    return this.getAsyncIterableOfData(
      '/v1/features/{feature_id}/stats/usage/chrome/daily_stats',
      {
        params: {
          query: {startAt, endAt},
          path: {
            feature_id: featureId,
          },
        },
      },
    );
  }

  // Fetches feature counts for a browser in a date range
  // via "/v1/stats/features/browsers/{browser}/feature_counts"
  public getFeatureCountsForBrowser(
    browser: BrowsersParameter,
    startAtDate: Date,
    endAtDate: Date,
  ): AsyncIterable<BrowserReleaseFeatureMetric[]> {
    const startAt = startAtDate.toISOString().substring(0, 10);
    const endAt = endAtDate.toISOString().substring(0, 10);

    return this.getAsyncIterableOfData(
      '/v1/stats/features/browsers/{browser}/feature_counts',
      {
        params: {
          query: {
            startAt,
            endAt,
            include_baseline_mobile_browsers: true,
          },
          path: {browser},
        },
      },
    );
  }

  // Returns the count of features supported that have reached baseline
  // via "/v1/stats/baseline_status/low_date_feature_counts"
  public listAggregatedBaselineStatusCounts(
    startAtDate: Date,
    endAtDate: Date,
  ): AsyncIterable<BaselineStatusMetric[]> {
    const startAt = startAtDate.toISOString().substring(0, 10);
    const endAt = endAtDate.toISOString().substring(0, 10);

    return this.getAsyncIterableOfData(
      '/v1/stats/baseline_status/low_date_feature_counts',
      {
        params: {
          query: {startAt, endAt},
        },
      },
    );
  }

  // Fetches feature counts for a browser in a date range
  // via "/v1/stats/features/browsers/{browser}/feature_counts"
  public getMissingOneImplementationCountsForBrowser(
    browser: BrowsersParameter,
    otherBrowsers: BrowsersParameter[],
    startAtDate: Date,
    endAtDate: Date,
  ): AsyncIterable<BrowserReleaseFeatureMetric[]> {
    const startAt = startAtDate.toISOString().substring(0, 10);
    const endAt = endAtDate.toISOString().substring(0, 10);

    return this.getAsyncIterableOfData(
      '/v1/stats/features/browsers/{browser}/missing_one_implementation_counts',
      {
        params: {
          query: {
            startAt,
            endAt,
            browser: otherBrowsers,
            include_baseline_mobile_browsers: true,
          },
          path: {browser},
        },
      },
    );
  }

  // Fetches missing feature list for a browser for a give date
  // via "/v1/stats/features/browsers/{browser}/missing_one_implementation_counts/{date}/features"
  public getMissingOneImplementationFeatures(
    targetBrowser: BrowsersParameter,
    otherBrowsers: BrowsersParameter[],
    date: Date,
  ): Promise<MissingOneImplFeaturesList> {
    const targetDate: string = date.toISOString().substring(0, 10);
    return this.getAllPagesOfData(
      '/v1/stats/features/browsers/{browser}/missing_one_implementation_counts/{date}/features',
      {
        params: {
          query: {
            browser: otherBrowsers,
            include_baseline_mobile_browsers: true,
          },
          path: {browser: targetBrowser, date: targetDate},
        },
      },
    );
  }

  public getSavedSearchByID(
    searchID: string,
    token?: string,
  ): Promise<SavedSearchResponse> {
    return this.handleResponse(
      this.client.GET('/v1/saved-searches/{search_id}', {
        params: {path: {search_id: searchID}},
        headers: token ? {Authorization: `Bearer ${token}`} : undefined,
      }),
      '/v1/saved-searches/{search_id}',
      'get',
    );
  }

  public removeSavedSearchByID(searchID: string, token: string): Promise<void> {
    return this.handleResponse(
      this.client.DELETE('/v1/saved-searches/{search_id}', {
        params: {path: {search_id: searchID}},
        headers: {Authorization: `Bearer ${token}`},
      }),
      '/v1/saved-searches/{search_id}',
      'delete',
    );
  }

  public async getGlobalSavedSearches(
    pageToken?: string,
    pageSize?: number,
  ): Promise<SuccessResponsePageableData<'/v1/global-saved-searches'>> {
    return this.getPageOfData(
      '/v1/global-saved-searches',
      {params: {query: {}}},
      pageToken,
      pageSize,
    );
  }

  public putUserSavedSearchBookmark(
    searchID: string,
    token: string,
  ): Promise<void> {
    return this.handleResponse(
      this.client.PUT(
        '/v1/users/me/saved-searches/{search_id}/bookmark_status',
        {
          params: {path: {search_id: searchID}},
          headers: {Authorization: `Bearer ${token}`},
        },
      ),
      '/v1/users/me/saved-searches/{search_id}/bookmark_status',
      'put',
    );
  }

  public removeUserSavedSearchBookmark(
    searchID: string,
    token: string,
  ): Promise<void> {
    return this.handleResponse(
      this.client.DELETE(
        '/v1/users/me/saved-searches/{search_id}/bookmark_status',
        {
          params: {path: {search_id: searchID}},
          headers: {Authorization: `Bearer ${token}`},
        },
      ),
      '/v1/users/me/saved-searches/{search_id}/bookmark_status',
      'delete',
    );
  }

  public createSavedSearch(
    token: string,
    savedSearch: components['schemas']['SavedSearch'],
  ): Promise<components['schemas']['SavedSearchResponse']> {
    return this.handleResponse(
      this.client.POST('/v1/saved-searches', {
        headers: {
          Authorization: `Bearer ${token}`,
        },
        body: savedSearch,
      }),
      '/v1/saved-searches',
      'post',
    );
  }

  public updateSavedSearch(
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
    return this.handleResponse(
      this.client.PATCH('/v1/saved-searches/{search_id}', {
        headers: {
          Authorization: `Bearer ${token}`,
        },
        params: {
          path: {
            search_id: savedSearch.id,
          },
        },
        body: req,
      }),
      '/v1/saved-searches/{search_id}',
      'patch',
    );
  }

  public getSubscription(
    subscriptionId: string,
    token: string,
  ): Promise<components['schemas']['SubscriptionResponse']> {
    return this.handleResponse(
      this.client.GET('/v1/users/me/subscriptions/{subscription_id}', {
        params: {path: {subscription_id: subscriptionId}},
        headers: {Authorization: `Bearer ${token}`},
      }),
      '/v1/users/me/subscriptions/{subscription_id}',
      'get',
    );
  }

  public deleteSubscription(
    subscriptionId: string,
    token: string,
  ): Promise<void> {
    return this.handleResponse(
      this.client.DELETE('/v1/users/me/subscriptions/{subscription_id}', {
        params: {path: {subscription_id: subscriptionId}},
        headers: {Authorization: `Bearer ${token}`},
      }),
      '/v1/users/me/subscriptions/{subscription_id}',
      'delete',
    );
  }

  public async listSubscriptions(
    token: string,
  ): Promise<components['schemas']['SubscriptionResponse'][]> {
    return this.getAllPagesOfData('/v1/users/me/subscriptions', {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
  }

  public createSubscription(
    token: string,
    subscription: components['schemas']['Subscription'],
  ): Promise<components['schemas']['SubscriptionResponse']> {
    return this.handleResponse(
      this.client.POST('/v1/users/me/subscriptions', {
        headers: {
          Authorization: `Bearer ${token}`,
        },
        body: subscription,
      }),
      '/v1/users/me/subscriptions',
      'post',
    );
  }

  public updateSubscription(
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
    return this.handleResponse(
      this.client.PATCH('/v1/users/me/subscriptions/{subscription_id}', {
        headers: {
          Authorization: `Bearer ${token}`,
        },
        params: {
          path: {
            subscription_id: subscriptionId,
          },
        },
        body: req,
      }),
      '/v1/users/me/subscriptions/{subscription_id}',
      'patch',
    );
  }
}
