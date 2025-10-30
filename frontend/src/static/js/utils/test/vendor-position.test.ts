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

import {expect} from '@open-wc/testing';
import {VendorPosition, VendorPositions} from '../vendor-position.js';

describe('VendorPosition', () => {
  describe('create', () => {
    it.skip('should create a VendorPosition instance from valid data', () => {
      const data = {
        vendor: 'mozilla',
        position: 'positive',
        url: 'https://example.com',
      };
      const vendorPosition = VendorPosition.create(data);
      expect(vendorPosition).to.be.an.instanceOf(VendorPosition);
      expect(vendorPosition!.vendor).to.equal('Mozilla');
      expect(vendorPosition!.position).to.equal('Positive');
      expect(vendorPosition!.url).to.equal('https://example.com');
    });

    it('should return null if data is not an object', () => {
      const data = 'not an object';
      const vendorPosition = VendorPosition.create(data);
      expect(vendorPosition).to.be.null;
    });

    it('should return null if data is null', () => {
      const data = null;
      const vendorPosition = VendorPosition.create(data);
      expect(vendorPosition).to.be.null;
    });

    it('should return null if position is missing', () => {
      const data = {
        vendor: 'mozilla',
        url: 'https://example.com',
      };
      const vendorPosition = VendorPosition.create(data);
      expect(vendorPosition).to.be.null;
    });

    it('should return null if url is missing', () => {
      const data = {
        vendor: 'mozilla',
        position: 'positive',
      };
      const vendorPosition = VendorPosition.create(data);
      expect(vendorPosition).to.be.null;
    });

    it('should return null if position is not a string', () => {
      const data = {
        vendor: 'mozilla',
        position: 123,
        url: 'https://example.com',
      };
      const vendorPosition = VendorPosition.create(data);
      expect(vendorPosition).to.be.null;
    });

    it('should return null if url is not a string', () => {
      const data = {
        vendor: 'mozilla',
        position: 'positive',
        url: 123,
      };
      const vendorPosition = VendorPosition.create(data);
      expect(vendorPosition).to.be.null;
    });

    it('should return null if position is an empty string', () => {
      const data = {
        vendor: 'mozilla',
        position: '',
        url: 'https://example.com',
      };
      const vendorPosition = VendorPosition.create(data);
      expect(vendorPosition).to.be.null;
    });

    it('should return null if url is an empty string', () => {
      const data = {
        vendor: 'mozilla',
        position: 'positive',
        url: '',
      };
      const vendorPosition = VendorPosition.create(data);
      expect(vendorPosition).to.be.null;
    });

    it('should return null if vendor is missing', () => {
      const data = {
        position: 'positive',
        url: 'https://example.com',
      };
      const vendorPosition = VendorPosition.create(data);
      expect(vendorPosition).to.be.null;
    });

    it('should capitalize vendor and position names', () => {
      const data = {
        vendor: 'mozilla',
        position: 'positive',
        url: 'https://example.com',
      };
      const vendorPosition = VendorPosition.create(data);
      expect(vendorPosition).to.be.an.instanceOf(VendorPosition);
      expect(vendorPosition!.vendor).to.equal('Mozilla');
      expect(vendorPosition!.position).to.equal('Positive');
    });

    it('should handle unknown vendor and position names', () => {
      const data = {
        vendor: 'unknown_vendor',
        position: 'unknown_position',
        url: 'https://example.com',
      };
      const vendorPosition = VendorPosition.create(data);
      expect(vendorPosition).to.be.an.instanceOf(VendorPosition);
      expect(vendorPosition!.vendor).to.equal('unknown_vendor');
      expect(vendorPosition!.position).to.equal('unknown_position');
    });
  });

  describe('VendorPositions', () => {
    it('should create an array of VendorPosition instances from valid data', () => {
      const data = [
        {
          vendor: 'mozilla',
          position: 'positive',
          url: 'https://example.com/mozilla',
        },
        {
          vendor: 'apple',
          position: 'negative',
          url: 'https://example.com/webkit',
        },
      ];
      const vendorPositions = VendorPositions.create(data);
      expect(vendorPositions).to.be.an('array');
      expect(vendorPositions!.length).to.equal(2);
      expect(vendorPositions![0]).to.be.an.instanceOf(VendorPosition);
      expect(vendorPositions![0].vendor).to.equal('Mozilla');
      expect(vendorPositions![1].vendor).to.equal('Apple');
    });

    it('should filter out invalid data', () => {
      const data = [
        {
          vendor: 'mozilla',
          position: 'positive',
          url: 'https://example.com/mozilla',
        },
        {
          position: 'negative',
          url: 'https://example.com/webkit',
        },
      ];
      const vendorPositions = VendorPositions.create(data);
      expect(vendorPositions).to.be.an('array');
      expect(vendorPositions!.length).to.equal(1);
      expect(vendorPositions![0].vendor).to.equal('Mozilla');
    });

    it('should return an empty array if all data is invalid', () => {
      const data = [
        {
          position: 'positive',
          url: 'https://example.com/mozilla',
        },
        {
          position: 'negative',
          url: 'https://example.com/webkit',
        },
      ];
      const vendorPositions = VendorPositions.create(data);
      expect(vendorPositions).to.be.an('array');
      expect(vendorPositions!.length).to.equal(0);
    });

    it('should return null if data is not an array', () => {
      const data = 'not an array';
      const vendorPositions = VendorPositions.create(data);
      expect(vendorPositions).to.be.null;
    });
  });
});
