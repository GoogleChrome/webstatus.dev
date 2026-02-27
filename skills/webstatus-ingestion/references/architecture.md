# Ingestion Architecture & Implementation

This document provides a technical guide for the `workflows/` directory, detailing how external data is synchronized into the Spanner database.

## 1. Job Orchestration Pattern

All ingestion jobs (Cloud Run Jobs) follow a standardized three-stage pipeline to ensure consistency and testability.

1.  **Downloader**: Fetches raw data from external APIs (e.g., WPT.fyi) or repositories (e.g., MDN BCD).
2.  **Parser**: Translates the raw format (JSON, YAML, CSV) into internal Go structs.
3.  **Adapter/Mapper**: Uses the [Spanner Adapter Pattern](../../webstatus-backend/references/architecture.md#spanner-adapters) to upsert the parsed data into Spanner.

## 2. Supported Data Sources & Mappings

The system maintains distinct consumers for each data provider.

| Source             | Description                                                                            | Consumer Service                | Target Spanner Tables                                                                                                                                                                        |
| :----------------- | :------------------------------------------------------------------------------------- | :------------------------------ | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **MDN BCD**        | Browser Compatibility Data (releases, support status)                                  | `bcd_consumer`                  | [`BrowserReleases`](../../../docs/schema/BrowserReleases.md), [`BrowserFeatureAvailabilities`](../../../docs/schema/BrowserFeatureAvailabilities.md)                                         |
| **Web-Features**   | Baseline status, feature groupings, and snapshots                                      | `web_feature_consumer`          | [`WebFeatures`](../../../docs/schema/WebFeatures.md), [`FeatureGroupKeysLookup`](../../../docs/schema/FeatureGroupKeysLookup.md), [`WebDXSnapshots`](../../../docs/schema/WebDXSnapshots.md) |
| **WF Mappings**    | [web-features-mappings repo](https://github.com/web-platform-dx/web-features-mappings) | `web_features_mapping_consumer` | [`WebFeaturesMappingData`](../../../docs/schema/WebFeaturesMappingData.md)                                                                                                                   |
| **WPT.fyi**        | Test run results and pass/fail metrics                                                 | `wpt_consumer`                  | [`WPTRuns`](../../../docs/schema/WPTRuns.md), [`WPTRunFeatureMetrics`](../../../docs/schema/WPTRunFeatureMetrics.md)                                                                         |
| **Chromium UMA**   | Real-world usage metrics (Usage and Histograms)                                        | `uma_export`                    | [`DailyChromiumHistogramMetrics`](../../../docs/schema/DailyChromiumHistogramMetrics.md)                                                                                                     |
| **GitHub Signals** | Developer signals (upvotes, external interest)                                         | `developer_signals_consumer`    | [`LatestFeatureDeveloperSignals`](../../../docs/schema/LatestFeatureDeveloperSignals.md)                                                                                                     |
| **Chrome Enums**   | WebDX feature enum labels and bucket mappings                                          | `chromium_histogram_enums`      | [`ChromiumHistogramEnums`](../../../docs/schema/ChromiumHistogramEnums.md), [`ChromiumHistogramEnumValues`](../../../docs/schema/ChromiumHistogramEnumValues.md)                             |

## 3. Workflow Logic

- **Scheduling**: Triggers originate from Cloud Scheduler.
- **Delta Detection**: Some jobs (like the `event_producer`) use GCS snapshots to detect changes between runs.
- **Concurrency**: Large datasets are processed using the internal `lib/workerpool` to manage Spanner transaction limits and CPU usage.
