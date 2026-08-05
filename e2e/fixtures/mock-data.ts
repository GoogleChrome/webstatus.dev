/**
 * Copyright 2026 Google LLC
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

export const globalSavedSearchesFixture = {
  metadata: {},
  data: [
    {
      id: 'a09386fe-65f1-4640-b28d-3cf2f2de69c9',
      name: 'I like queries',
      query: 'baseline_status:limited OR available_on:chrome',
      description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit.',
      display_order: 1,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
    {
      id: 'b19386fe-65f1-4640-b28d-3cf2f2de69c0',
      name: 'Baseline Newly Available',
      query: 'baseline_status:newly',
      description: 'Newly available web features across browsers',
      display_order: 2,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
    {
      id: 'c29386fe-65f1-4640-b28d-3cf2f2de69c1',
      name: 'Baseline Widely Available',
      query: 'baseline_status:widely',
      description: 'Widely available web features',
      display_order: 3,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
  ],
};

export const userSavedSearchesFixture = {
  metadata: {},
  data: [
    {
      id: 'u09386fe-65f1-4640-b28d-3cf2f2de69c9',
      name: 'my first project query',
      query: 'available_on:chrome AND available_on:firefox',
      description: 'My custom user query for project monitoring',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
      permissions: {
        role: 'saved_search_owner' as const,
      },
      bookmark_status: {
        status: 'bookmark_active' as const,
      },
    },
  ],
};

export const userNotificationChannelsFixture = {
  metadata: {},
  data: [
    {
      id: 'nc-email-001',
      name: 'Primary Email',
      type: 'email' as const,
      status: 'enabled' as const,
      config: {
        type: 'email' as const,
        address: 'test.user.1@example.com',
      },
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
    {
      id: 'nc-webhook-001',
      name: 'Team Slack Alerts',
      type: 'webhook' as const,
      status: 'enabled' as const,
      config: {
        type: 'webhook' as const,
        url: 'https://hooks.slack.com/services/T000/B000/XXXX',
      },
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
  ],
};

export const featuresPage1Fixture = {
  metadata: {
    next_page_token: 'page_2_token',
    total: 50,
  },
  data: [
    {
      feature_id: 'accent-color',
      name: 'accent-color',
      baseline: {
        status: 'widely' as const,
        low_date: '2022-03-14',
        high_date: '2024-09-14',
      },
      browser_implementations: {
        chrome: {
          status: 'available' as const,
          date: '2021-08-31',
          version: '93',
        },
        firefox: {
          status: 'available' as const,
          date: '2021-09-07',
          version: '92',
        },
        safari: {
          status: 'available' as const,
          date: '2022-03-14',
          version: '15.4',
        },
        edge: {status: 'available' as const, date: '2021-08-31', version: '93'},
        chrome_android: {
          status: 'available' as const,
          date: '2021-08-31',
          version: '93',
        },
        firefox_android: {
          status: 'available' as const,
          date: '2021-09-07',
          version: '92',
        },
        safari_ios: {
          status: 'available' as const,
          date: '2022-03-14',
          version: '15.4',
        },
      },
      wpt: {
        stable: {
          chrome: {score: 1.0},
          firefox: {score: 1.0},
          safari: {score: 1.0},
        },
      },
      developer_signals: {
        upvotes: 42,
      },
    },
    {
      feature_id: 'anchor-positioning',
      name: 'Anchor Positioning',
      baseline: {
        status: 'limited' as const,
        low_date: undefined,
        high_date: undefined,
      },
      browser_implementations: {
        chrome: {
          status: 'available' as const,
          date: '2024-05-21',
          version: '125',
        },
        firefox: {status: 'unavailable' as const},
        safari: {status: 'unavailable' as const},
        edge: {
          status: 'available' as const,
          date: '2024-05-21',
          version: '125',
        },
        chrome_android: {
          status: 'available' as const,
          date: '2024-05-21',
          version: '125',
        },
        firefox_android: {status: 'unavailable' as const},
        safari_ios: {status: 'unavailable' as const},
      },
      wpt: {
        stable: {
          chrome: {score: 0.98},
          firefox: {score: 0.12},
          safari: {score: 0.05},
        },
      },
      developer_signals: {
        upvotes: 128,
      },
    },
    {
      feature_id: 'aspect-ratio',
      name: 'aspect-ratio',
      baseline: {
        status: 'widely' as const,
        low_date: '2021-09-20',
        high_date: '2024-03-20',
      },
      browser_implementations: {
        chrome: {
          status: 'available' as const,
          date: '2021-01-19',
          version: '88',
        },
        firefox: {
          status: 'available' as const,
          date: '2021-07-13',
          version: '89',
        },
        safari: {
          status: 'available' as const,
          date: '2021-09-20',
          version: '15',
        },
        edge: {status: 'available' as const, date: '2021-01-19', version: '88'},
      },
      developer_signals: {
        upvotes: 75,
      },
    },
    {
      feature_id: 'backdrop-filter',
      name: 'backdrop-filter',
      baseline: {
        status: 'newly' as const,
        low_date: '2023-10-24',
        high_date: undefined,
      },
      browser_implementations: {
        chrome: {
          status: 'available' as const,
          date: '2019-09-10',
          version: '76',
        },
        firefox: {
          status: 'available' as const,
          date: '2022-07-26',
          version: '103',
        },
        safari: {
          status: 'available' as const,
          date: '2015-09-30',
          version: '9',
        },
        edge: {status: 'available' as const, date: '2020-01-15', version: '79'},
      },
      developer_signals: {
        upvotes: 55,
      },
    },
    {
      feature_id: 'flexbox',
      name: 'CSS Flexible Box Layout',
      baseline: {
        status: 'widely' as const,
        low_date: '2015-09-30',
        high_date: '2018-03-30',
      },
      browser_implementations: {
        chrome: {
          status: 'available' as const,
          date: '2013-08-20',
          version: '29',
        },
        firefox: {
          status: 'available' as const,
          date: '2014-04-29',
          version: '28',
        },
        safari: {
          status: 'available' as const,
          date: '2015-09-30',
          version: '9',
        },
        edge: {status: 'available' as const, date: '2015-07-29', version: '12'},
      },
      developer_signals: {
        upvotes: 350,
      },
    },
  ],
};

export const featuresPage2Fixture = {
  metadata: {
    next_page_token: undefined,
    total: 50,
  },
  data: [
    {
      feature_id: 'grid',
      name: 'CSS Grid Layout',
      baseline: {
        status: 'widely' as const,
        low_date: '2017-10-17',
        high_date: '2020-04-17',
      },
      browser_implementations: {
        chrome: {
          status: 'available' as const,
          date: '2017-03-09',
          version: '57',
        },
        firefox: {
          status: 'available' as const,
          date: '2017-03-07',
          version: '52',
        },
        safari: {
          status: 'available' as const,
          date: '2017-03-27',
          version: '10.1',
        },
        edge: {status: 'available' as const, date: '2017-10-17', version: '16'},
      },
      developer_signals: {
        upvotes: 280,
      },
    },
    {
      feature_id: 'webgpu',
      name: 'WebGPU API',
      baseline: {
        status: 'limited' as const,
        low_date: undefined,
        high_date: undefined,
      },
      browser_implementations: {
        chrome: {
          status: 'available' as const,
          date: '2023-05-02',
          version: '113',
        },
        firefox: {status: 'unavailable' as const},
        safari: {status: 'unavailable' as const},
        edge: {
          status: 'available' as const,
          date: '2023-05-02',
          version: '113',
        },
      },
      developer_signals: {
        upvotes: 310,
      },
    },
  ],
};

export const featuresSortFixture = {
  metadata: {
    next_page_token: undefined,
    total: 5,
  },
  data: featuresPage1Fixture.data,
};

export const featuresUpvotesFixture = {
  metadata: {
    next_page_token: undefined,
    total: 5,
  },
  data: featuresPage1Fixture.data,
};

export const featureDetailAnchorPositioningFixture = {
  feature_id: 'anchor-positioning',
  name: 'Anchor Positioning',
  baseline: {
    status: 'limited' as const,
    low_date: undefined,
    high_date: undefined,
  },
  browser_implementations: {
    chrome: {status: 'available' as const, date: '2024-05-21', version: '125'},
    firefox: {status: 'unavailable' as const},
    safari: {status: 'unavailable' as const},
    edge: {status: 'available' as const, date: '2024-05-21', version: '125'},
    chrome_android: {
      status: 'available' as const,
      date: '2024-05-21',
      version: '125',
    },
    firefox_android: {status: 'unavailable' as const},
    safari_ios: {status: 'unavailable' as const},
  },
  spec: {
    links: [{link: 'https://drafts.csswg.org/css-anchor-position-1/'}],
  },
  wpt: {
    stable: {
      chrome: {score: 0.98},
      firefox: {score: 0.12},
      safari: {score: 0.05},
    },
  },
  developer_signals: {
    upvotes: 128,
  },
};

export const featureDetailDiscouragedFixture = {
  feature_id: 'discouraged',
  name: 'Discouraged Feature Example',
  baseline: {
    status: 'limited' as const,
    low_date: undefined,
    high_date: undefined,
  },
  discouraged: {
    according_to: [
      {link: 'https://html.spec.whatwg.org/multipage/obsolete.html'},
    ],
    alternatives: [{id: 'dialog'}],
  },
  browser_implementations: {
    chrome: {status: 'available' as const, date: '2010-01-01', version: '4'},
    firefox: {status: 'available' as const, date: '2010-01-01', version: '3.6'},
    safari: {status: 'available' as const, date: '2010-01-01', version: '4'},
  },
  wpt: {
    stable: {
      chrome: {score: 0.5},
      firefox: {score: 0.5},
      safari: {score: 0.5},
    },
  },
};

// =============================================================================
// Feature WPT Metrics (Per-Browser & Paginated)
// =============================================================================

export const featureWptMetricsChromePage1 = {
  metadata: {next_page_token: 'page_2', total: 12},
  data: [
    {
      run_timestamp: '2023-01-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 82,
    },
    {
      run_timestamp: '2023-02-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 84,
    },
    {
      run_timestamp: '2023-03-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 86,
    },
    {
      run_timestamp: '2023-04-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 88,
    },
    {
      run_timestamp: '2023-05-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 89,
    },
    {
      run_timestamp: '2023-06-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 91,
    },
  ],
};

export const featureWptMetricsChromePage2 = {
  metadata: {total: 12},
  data: [
    {
      run_timestamp: '2023-07-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 93,
    },
    {
      run_timestamp: '2023-08-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 95,
    },
    {
      run_timestamp: '2023-09-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 96,
    },
    {
      run_timestamp: '2023-10-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 97,
    },
    {
      run_timestamp: '2023-11-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 98,
    },
    {
      run_timestamp: '2023-12-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 99,
    },
  ],
};

export const featureWptMetricsFirefoxPage1 = {
  metadata: {next_page_token: 'page_2', total: 12},
  data: [
    {
      run_timestamp: '2023-01-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 70,
    },
    {
      run_timestamp: '2023-02-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 72,
    },
    {
      run_timestamp: '2023-03-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 74,
    },
    {
      run_timestamp: '2023-04-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 76,
    },
    {
      run_timestamp: '2023-05-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 78,
    },
    {
      run_timestamp: '2023-06-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 80,
    },
  ],
};

export const featureWptMetricsFirefoxPage2 = {
  metadata: {total: 12},
  data: [
    {
      run_timestamp: '2023-07-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 82,
    },
    {
      run_timestamp: '2023-08-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 84,
    },
    {
      run_timestamp: '2023-09-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 86,
    },
    {
      run_timestamp: '2023-10-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 88,
    },
    {
      run_timestamp: '2023-11-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 90,
    },
    {
      run_timestamp: '2023-12-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 92,
    },
  ],
};

export const featureWptMetricsSafariPage1 = {
  metadata: {next_page_token: 'page_2', total: 12},
  data: [
    {
      run_timestamp: '2023-01-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 55,
    },
    {
      run_timestamp: '2023-02-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 58,
    },
    {
      run_timestamp: '2023-03-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 60,
    },
    {
      run_timestamp: '2023-04-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 63,
    },
    {
      run_timestamp: '2023-05-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 65,
    },
    {
      run_timestamp: '2023-06-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 68,
    },
  ],
};

export const featureWptMetricsSafariPage2 = {
  metadata: {total: 12},
  data: [
    {
      run_timestamp: '2023-07-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 70,
    },
    {
      run_timestamp: '2023-08-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 73,
    },
    {
      run_timestamp: '2023-09-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 75,
    },
    {
      run_timestamp: '2023-10-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 78,
    },
    {
      run_timestamp: '2023-11-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 80,
    },
    {
      run_timestamp: '2023-12-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 83,
    },
  ],
};

export const featureWptMetricsEdgePage1 = {
  metadata: {next_page_token: 'page_2', total: 12},
  data: [
    {
      run_timestamp: '2023-01-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 80,
    },
    {
      run_timestamp: '2023-02-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 82,
    },
    {
      run_timestamp: '2023-03-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 84,
    },
    {
      run_timestamp: '2023-04-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 86,
    },
    {
      run_timestamp: '2023-05-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 87,
    },
    {
      run_timestamp: '2023-06-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 89,
    },
  ],
};

export const featureWptMetricsEdgePage2 = {
  metadata: {total: 12},
  data: [
    {
      run_timestamp: '2023-07-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 91,
    },
    {
      run_timestamp: '2023-08-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 93,
    },
    {
      run_timestamp: '2023-09-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 94,
    },
    {
      run_timestamp: '2023-10-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 95,
    },
    {
      run_timestamp: '2023-11-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 96,
    },
    {
      run_timestamp: '2023-12-01T00:00:00Z',
      total_tests_count: 100,
      test_pass_count: 97,
    },
  ],
};

// Fallback/combined export
export const featureWptMetricsFixture = featureWptMetricsChromePage1;

// =============================================================================
// Feature UMA Daily Usage (Paginated)
// =============================================================================

export const featureUmaMetricsPage1 = {
  metadata: {next_page_token: 'page_2', total: 12},
  data: [
    {timestamp: '2023-01-01T00:00:00Z', usage: 0.12},
    {timestamp: '2023-02-01T00:00:00Z', usage: 0.18},
    {timestamp: '2023-03-01T00:00:00Z', usage: 0.25},
    {timestamp: '2023-04-01T00:00:00Z', usage: 0.32},
    {timestamp: '2023-05-01T00:00:00Z', usage: 0.4},
    {timestamp: '2023-06-01T00:00:00Z', usage: 0.48},
  ],
};

export const featureUmaMetricsPage2 = {
  metadata: {total: 12},
  data: [
    {timestamp: '2023-07-01T00:00:00Z', usage: 0.55},
    {timestamp: '2023-08-01T00:00:00Z', usage: 0.62},
    {timestamp: '2023-09-01T00:00:00Z', usage: 0.7},
    {timestamp: '2023-10-01T00:00:00Z', usage: 0.78},
    {timestamp: '2023-11-01T00:00:00Z', usage: 0.83},
    {timestamp: '2023-12-01T00:00:00Z', usage: 0.89},
  ],
};

export const featureUmaMetricsFixture = featureUmaMetricsPage1;

// =============================================================================
// Stats Page: Global Feature Support (Per-Browser & Paginated)
// =============================================================================

export const statsFeatureCountsChromePage1 = {
  metadata: {next_page_token: 'page_2', total: 10},
  data: [
    {timestamp: '2019-01-01T00:00:00Z', count: 320},
    {timestamp: '2020-01-01T00:00:00Z', count: 450},
    {timestamp: '2021-01-01T00:00:00Z', count: 590},
    {timestamp: '2022-01-01T00:00:00Z', count: 720},
    {timestamp: '2023-01-01T00:00:00Z', count: 850},
  ],
};

export const statsFeatureCountsChromePage2 = {
  metadata: {total: 10},
  data: [
    {timestamp: '2023-07-01T00:00:00Z', count: 910},
    {timestamp: '2024-01-01T00:00:00Z', count: 980},
    {timestamp: '2024-07-01T00:00:00Z', count: 1040},
    {timestamp: '2025-01-01T00:00:00Z', count: 1110},
    {timestamp: '2025-07-01T00:00:00Z', count: 1170},
  ],
};

export const statsFeatureCountsFirefoxPage1 = {
  metadata: {next_page_token: 'page_2', total: 10},
  data: [
    {timestamp: '2019-01-01T00:00:00Z', count: 280},
    {timestamp: '2020-01-01T00:00:00Z', count: 390},
    {timestamp: '2021-01-01T00:00:00Z', count: 510},
    {timestamp: '2022-01-01T00:00:00Z', count: 640},
    {timestamp: '2023-01-01T00:00:00Z', count: 760},
  ],
};

export const statsFeatureCountsFirefoxPage2 = {
  metadata: {total: 10},
  data: [
    {timestamp: '2023-07-01T00:00:00Z', count: 820},
    {timestamp: '2024-01-01T00:00:00Z', count: 890},
    {timestamp: '2024-07-01T00:00:00Z', count: 950},
    {timestamp: '2025-01-01T00:00:00Z', count: 1020},
    {timestamp: '2025-07-01T00:00:00Z', count: 1080},
  ],
};

export const statsFeatureCountsSafariPage1 = {
  metadata: {next_page_token: 'page_2', total: 10},
  data: [
    {timestamp: '2019-01-01T00:00:00Z', count: 220},
    {timestamp: '2020-01-01T00:00:00Z', count: 310},
    {timestamp: '2021-01-01T00:00:00Z', count: 430},
    {timestamp: '2022-01-01T00:00:00Z', count: 560},
    {timestamp: '2023-01-01T00:00:00Z', count: 680},
  ],
};

export const statsFeatureCountsSafariPage2 = {
  metadata: {total: 10},
  data: [
    {timestamp: '2023-07-01T00:00:00Z', count: 740},
    {timestamp: '2024-01-01T00:00:00Z', count: 810},
    {timestamp: '2024-07-01T00:00:00Z', count: 870},
    {timestamp: '2025-01-01T00:00:00Z', count: 940},
    {timestamp: '2025-07-01T00:00:00Z', count: 1000},
  ],
};

// =============================================================================
// Stats Page: Total Baseline Feature Counts (Paginated)
// =============================================================================

export const statsLowDateCountsPage1 = {
  metadata: {next_page_token: 'page_2', total: 10},
  data: [
    {timestamp: '2019-01-01T00:00:00Z', count: 120},
    {timestamp: '2020-01-01T00:00:00Z', count: 210},
    {timestamp: '2021-01-01T00:00:00Z', count: 350},
    {timestamp: '2022-01-01T00:00:00Z', count: 520},
    {timestamp: '2023-01-01T00:00:00Z', count: 710},
  ],
};

export const statsLowDateCountsPage2 = {
  metadata: {total: 10},
  data: [
    {timestamp: '2023-07-01T00:00:00Z', count: 820},
    {timestamp: '2024-01-01T00:00:00Z', count: 940},
    {timestamp: '2024-07-01T00:00:00Z', count: 1050},
    {timestamp: '2025-01-01T00:00:00Z', count: 1180},
    {timestamp: '2025-07-01T00:00:00Z', count: 1290},
  ],
};

export const statsLowDateCountsFixture = statsLowDateCountsPage1;

// =============================================================================
// Stats Page: Features Missing in Only One Browser (Per-Browser & Paginated)
// =============================================================================

export const statsMissingOneChromePage1 = {
  metadata: {next_page_token: 'page_2', total: 10},
  data: [
    {timestamp: '2019-01-01T00:00:00Z', count: 45},
    {timestamp: '2020-01-01T00:00:00Z', count: 38},
    {timestamp: '2021-01-01T00:00:00Z', count: 28},
    {timestamp: '2022-01-01T00:00:00Z', count: 20},
    {timestamp: '2023-01-01T00:00:00Z', count: 15},
  ],
};

export const statsMissingOneChromePage2 = {
  metadata: {total: 10},
  data: [
    {timestamp: '2023-07-01T00:00:00Z', count: 12},
    {timestamp: '2024-01-01T00:00:00Z', count: 9},
    {timestamp: '2024-07-01T00:00:00Z', count: 7},
    {timestamp: '2025-01-01T00:00:00Z', count: 5},
    {timestamp: '2025-07-01T00:00:00Z', count: 3},
  ],
};

export const statsMissingOneFirefoxPage1 = {
  metadata: {next_page_token: 'page_2', total: 10},
  data: [
    {timestamp: '2019-01-01T00:00:00Z', count: 65},
    {timestamp: '2020-01-01T00:00:00Z', count: 55},
    {timestamp: '2021-01-01T00:00:00Z', count: 42},
    {timestamp: '2022-01-01T00:00:00Z', count: 32},
    {timestamp: '2023-01-01T00:00:00Z', count: 24},
  ],
};

export const statsMissingOneFirefoxPage2 = {
  metadata: {total: 10},
  data: [
    {timestamp: '2023-07-01T00:00:00Z', count: 19},
    {timestamp: '2024-01-01T00:00:00Z', count: 15},
    {timestamp: '2024-07-01T00:00:00Z', count: 11},
    {timestamp: '2025-01-01T00:00:00Z', count: 8},
    {timestamp: '2025-07-01T00:00:00Z', count: 5},
  ],
};

export const statsMissingOneSafariPage1 = {
  metadata: {next_page_token: 'page_2', total: 10},
  data: [
    {timestamp: '2019-01-01T00:00:00Z', count: 85},
    {timestamp: '2020-01-01T00:00:00Z', count: 72},
    {timestamp: '2021-01-01T00:00:00Z', count: 58},
    {timestamp: '2022-01-01T00:00:00Z', count: 45},
    {timestamp: '2023-01-01T00:00:00Z', count: 35},
  ],
};

export const statsMissingOneSafariPage2 = {
  metadata: {total: 10},
  data: [
    {timestamp: '2023-07-01T00:00:00Z', count: 28},
    {timestamp: '2024-01-01T00:00:00Z', count: 22},
    {timestamp: '2024-07-01T00:00:00Z', count: 16},
    {timestamp: '2025-01-01T00:00:00Z', count: 12},
    {timestamp: '2025-07-01T00:00:00Z', count: 8},
  ],
};

export const statsMissingOneImplPage1 = statsMissingOneChromePage1;
export const statsMissingOneImplPage2 = statsMissingOneChromePage2;
export const statsMissingOneImplFixture = statsMissingOneChromePage1;

// =============================================================================
// Stats Page: Interactive Point-Selected Missing Features List (Paginated)
// =============================================================================

export const statsMissingFeaturesListPage1 = {
  metadata: {next_page_token: 'page_2', total: 6},
  data: [
    {feature_id: 'anchor-positioning'},
    {feature_id: 'subgrid'},
    {feature_id: 'view-transitions'},
  ],
};

export const statsMissingFeaturesListPage2 = {
  metadata: {total: 6},
  data: [
    {feature_id: 'popover'},
    {feature_id: 'has'},
    {feature_id: 'container-queries'},
  ],
};

export const statsMissingFeaturesListFixture = statsMissingFeaturesListPage1;
