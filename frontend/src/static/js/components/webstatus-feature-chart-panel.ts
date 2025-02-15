/**
 * Copyright 2025 Google LLC
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

// import {
//   LineChartMetricData,
//   WebstatusLineChartPanel,
// } from './webstatus-line-chart-panel.js';

// // Type for the data fetched event (using type alias)
// type DataFetchedEvent<T> = CustomEvent<{label: string; data: T[]}>;
// // Generic interface for any additional metric data
// interface AdditionalMetricData<T> {
//   [key: string]: T[];
// }

// // Type for series calculator functions
// type SeriesCalculator<T> = (
//   dataPoint: T,
//   additionalMetricData: AdditionalMetricData<T>,
// ) => AdditionalMetricData<T>;

// // Type for extracting timestamp from a data point
// type TimestampExtractor<T> = (dataPoint: T) => Date;

// // Type for extracting value from a data point
// type ValueExtractor<T> = (dataPoint: T) => number;

// // Interface for additional series configuration
// interface AdditionalSeriesConfig<T> {
//   label: string;
//   calculator: SeriesCalculator<T>;
//   timestampExtractor: TimestampExtractor<T>;
//   valueExtractor: ValueExtractor<T>;
// }

// // Interface for fetch function configuration
// interface FetchFunctionConfig<T> {
//   label: string;
//   fetchFunction: () => AsyncIterable<T[]>;
//   timestampExtractor: TimestampExtractor<T>;
//   valueExtractor: ValueExtractor<T>;
// }

// export abstract class WebstatusFeatureChartPanel extends WebstatusLineChartPanel {
//   totalDetails: {label: string} | undefined;

//   async _fetchAndAggregateData<T>(
//     fetchFunctionConfigs: FetchFunctionConfig<T>[],
//     additionalSeriesConfigs?: AdditionalSeriesConfig<T>[],
//   ) {
//     // Create an array of metric data objects for each fetch function
//     const metricDataArray: Array<LineChartMetricData<T>> =
//       fetchFunctionConfigs.map(
//         ({label, timestampExtractor, valueExtractor}) => ({
//           label,
//           data: [],
//           getTimestamp: timestampExtractor,
//           getValue: valueExtractor,
//         }),
//       );

//     // Dispatch an event to signal the start of data fetching
//     const event = new CustomEvent('data-fetch-starting');
//     this.dispatchEvent(event);

//     // Fetch data for each configuration concurrently
//     const promises = fetchFunctionConfigs.map(
//       async ({fetchFunction, label}) => {
//         for await (const page of fetchFunction()) {
//           // Find the corresponding metric data object
//           const metricData = metricDataArray.find(data => data.label === label);
//           if (metricData) {
//             metricData.data.push(...page);
//           }
//         }
//       },
//     );

//     await Promise.all(promises);

//     // Apply additionalSeriesConfigs if provided
//     let additionalMetricData: AdditionalMetricData<T> = {};
//     if (additionalSeriesConfigs) {
//       fetchFunctionConfigs.forEach(({label}) => {
//         const metricData = metricDataArray.find(data => data.label === label);
//         if (metricData) {
//           metricData.data.forEach((dataPoint: T) => {
//             additionalSeriesConfigs.forEach(({calculator}) => {
//               additionalMetricData = calculator(
//                 dataPoint,
//                 additionalMetricData,
//               );
//             });
//           });
//         }
//       });

//       // Convert additionalMetricData to LineChartMetricData
//       const additionalMetricDataArray: Array<LineChartMetricData<T>> =
//         Object.entries(additionalMetricData).reduce(
//           (acc: Array<LineChartMetricData<T>>, [label, data]) => {
//             const config = additionalSeriesConfigs.find(
//               config => config.label === label,
//             );
//             if (config) {
//               acc.push({
//                 label,
//                 data: data,
//                 getTimestamp: config.timestampExtractor,
//                 getValue: config.valueExtractor,
//               });
//             }
//             return acc;
//           },
//           [],
//         );

//       if (additionalMetricDataArray !== undefined) {
//         metricDataArray.push(...additionalMetricDataArray);
//       }
//     }

//     this.setDisplayDataFromMap(metricDataArray);
//   }

//   // async _fetchAndAggregateBrowserData(
//   //   apiClient: APIClient,
//   //   fetchFunction: (browser: BrowsersParameter) => AsyncIterable<T[]>,
//   //   browserDataReference: (browser: BrowsersParameter) => string,
//   //   browsers: BrowsersParameter[],
//   // ) {
//   //   if (typeof apiClient !== 'object') return;

//   //   const browserMetricData: Array<
//   //     LineChartMetricData<T> & {
//   //       browser: BrowsersParameter;
//   //     }
//   //   > = browsers.map(browser => ({
//   //     label: BROWSER_ID_TO_LABEL[browser],
//   //     browser: browser,
//   //     data: [],
//   //     getTimestamp: (dataPoint: T) => new Date(dataPoint.timestamp),
//   //     getValue: (dataPoint: T) => dataPoint.count,
//   //   }));
//   //   const event = new CustomEvent('browser-feature-data-fetch-starting');
//   //   this.dispatchEvent(event);

//   //   const promises = browsers.map(async browser => {
//   //     for await (const page of fetchFunction(browser)) {
//   //       // Append the new data to existing data
//   //       const existingData =
//   //         this.fetchData.get(browserDataReference(browser)) || [];
//   //       this.fetchData.set(browserDataReference(browser), [
//   //         ...existingData,
//   //         ...page,
//   //       ]);
//   //     }
//   //   });

//   //   await Promise.all(promises); // Wait for all browsers to finish

//   //   // TODO. If T has a total data point along with the regular data point, create a new
//   // }

//   override setDisplayDataFromMap<D>(
//     metricDataArray: Array<LineChartMetricData<D>>,
//   ) {
//     if (this.totalDetails) {
//     }
//     super.setDisplayDataFromMap(metricDataArray);
//   }
// }
