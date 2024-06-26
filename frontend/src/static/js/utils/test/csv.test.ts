/**
 * Copyright 2024 Google LLC
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

import {assert} from '@open-wc/testing';
import {convertToCSV} from '../csv.js';

describe('convertToCSV', () => {
  it('should return an empty string when there are no columns', () => {
    const columns: string[] = [];
    const rows: string[][] = [];
    const csv = convertToCSV(columns, rows);
    assert.equal(csv, '');
  });

  it('should return just the column header when there is no rows to convert to CSV', () => {
    const columns = ['Column1', 'Column2'];
    const rows: string[][] = [];
    const csv = convertToCSV(columns, rows);
    assert.equal(csv, '"Column1","Column2"');
  });

  it('should return a CSV string with one row', () => {
    const columns = ['Column1', 'Column2'];
    const rows: string[][] = [['Value1', 'Value2']];
    const expectedCSV = '"Column1","Column2"\n"Value1","Value2"';
    const csv = convertToCSV(columns, rows);
    assert.equal(csv, expectedCSV);
  });

  it('should return a CSV string with multiple rows', () => {
    const columns = ['Column1', 'Column2', 'Column3'];
    const rows: string[][] = [
      ['Value1', 'Value2', 'Value3'],
      ['Value4', 'Value5', 'Value6'],
    ];
    const expectedCSV =
      '"Column1","Column2","Column3"\n"Value1","Value2","Value3"\n"Value4","Value5","Value6"';
    const csv = convertToCSV(columns, rows);
    assert.equal(csv, expectedCSV);
  });

  it('should handle missing values in rows', () => {
    const columns = ['Column1', 'Column2', 'Column3'];
    const rows: string[][] = [
      ['Value1', 'Value2', ''],
      ['Value4', '', 'Value6'],
    ];
    const expectedCSV =
      '"Column1","Column2","Column3"\n"Value1","Value2",""\n"Value4","","Value6"';
    const csv = convertToCSV(columns, rows);
    assert.equal(csv, expectedCSV);
  });

  // Test all cases where escaping is necessary.
  // Specifically, any cells or column header values with
  // double-quotes("), or starting with '=', '+', '-', or '@'.
  it('should escape all special characters', () => {
    const columns = ['Column1', '"Column2"'];
    const rows: string[][] = [
      ['"', '""'],
      ['Value"3"', '"Value4"'],
      ['=Value=5"', '+Value+6'],
      ['-Value-7', '@Value@8'],
    ];
    const expectedCSV = `"Column1","""Column2"""\n"""",""""""\n"Value""3""","""Value4"""\n"\'=Value=5""","\'+Value+6"\n"\'-Value-7","\'@Value@8"`;

    const csv = convertToCSV(columns, rows);
    assert.equal(csv, expectedCSV);
    if (csv !== expectedCSV) {
      console.error('Expected CSV:', expectedCSV);
      console.error('Actual CSV:', csv);
    }
    assert.equal(csv, expectedCSV, '"Expected CSV to be: " + expectedCSV');
  });
});
