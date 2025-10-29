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

import {type Vendor, type Position} from 'web-features-mappings';

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

  // Base record so that TypeScript will be exhaustive when checking vendors.
  private static readonly VENDOR_DISPLAY_MAP: Record<Vendor, string> = {
    webkit: 'WebKit',
    mozilla: 'Mozilla',
  };

  // Actual map used for safe string lookup for vendors.
  private static readonly VENDOR_STRING_MAP: Record<string, string> =
    VendorPosition.VENDOR_DISPLAY_MAP;

  // Base record so that TypeScript will be exhaustive when checking positions.
  private static readonly POSITION_DISPLAY_MAP: Record<Position, string> = {
    blocked: 'Blocked',
    defer: 'Defer',
    '': '',
    negative: 'Negative',
    neutral: 'Neutral',
    oppose: 'Oppose',
    positive: 'Positive',
    support: 'Support',
  };

  // Actual map used for safe string lookup for positions.
  private static readonly POSITION_STRING_MAP: Record<string, string> =
    VendorPosition.POSITION_DISPLAY_MAP;

  private constructor(vendor: string, position: string, url: string) {
    const displayVendorValue = VendorPosition.VENDOR_STRING_MAP[vendor];
    if (displayVendorValue !== undefined) {
      this.vendor = displayVendorValue;
    } else {
      // Leave as-is so that we can determine unknown positions later.
      this.vendor = vendor;
    }

    const displayPositionValue = VendorPosition.POSITION_STRING_MAP[position];
    if (displayPositionValue !== undefined) {
      this.position = displayPositionValue;
    } else {
      // Leave as-is so that we can determine unknown positions later.
      this.position = position;
    }

    this.url = url;
  }

  static create(data: unknown): VendorPosition | null {
    if (typeof data !== 'object' || data === null) {
      return null;
    }

    const record = data as Record<string, unknown>;

    const {vendor, position, url} = record;

    if (typeof vendor !== 'string' || vendor === '') {
      return null;
    }

    if (typeof position !== 'string' || position === '') {
      return null;
    }

    if (typeof url !== 'string' || url === '') {
      return null;
    }

    return new VendorPosition(vendor, position, url);
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
