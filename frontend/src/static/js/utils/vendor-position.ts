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

// The VendorPosition class provides a type-safe way to handle vendor position data from the API.
// The `vendor_positions` field in the OpenAPI spec is intentionally loosely defined to accommodate
// the evolving schema of the web-features-mappings data. This class ensures that the frontend
// can safely parse and handle this data, even if the structure changes in the future.
// See @jsonschema/web-platform-dx_web-features-mappings/combined-schema.gen.json for the full schema.
// This follows the StandardsPosition definition in lib/webfeaturesmappingtypes/types.go.
export class VendorPosition {
  // The vendor name.
  vendor: string;
  // The vendor's position on the feature.
  position: string;
  // A URL to the official position statement.
  url: string;

  private constructor(vendor: string, position: string, url: string) {
    this.vendor = vendor;
    this.position = position;
    this.url = url;
  }

  static create(data: unknown): VendorPosition | null {
    if (
      typeof data === 'object' &&
      data !== null &&
      'vendor' in data &&
      typeof (data as {vendor: unknown}).vendor === 'string' &&
      (data as {vendor: string}).vendor !== '' &&
      'position' in data &&
      typeof (data as {position: unknown}).position === 'string' &&
      (data as {position: string}).position !== '' &&
      'url' in data &&
      typeof (data as {url: unknown}).url === 'string' &&
      (data as {url: string}).url !== ''
    ) {
      const vendor = (data as {vendor: string}).vendor;
      const position = (data as {position: string}).position;
      const url = (data as {url: string}).url;
      return new VendorPosition(vendor, position, url);
    }
    return null;
  }
}

export class VendorPositions {
  static create(data: unknown): VendorPosition[] | null {
    if (Array.isArray(data)) {
      return data
        .map(item => VendorPosition.create(item))
        .filter((item): item is VendorPosition => item !== null);
    }
    return null;
  }
}
