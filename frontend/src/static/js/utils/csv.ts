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

/**
 * Returns a string in CSV format given header strings and rows.
 */
export function convertToCSV(header: string[], rows: string[][]): string {
  const encodeCell = (cell: string) => {
    // Prepend all double quotes with another double quote, RFC 4810 Section 2.7
    let escaped = cell.replace(/"/g, '""');

    // Prevent CSV injection: owasp.org/index.php/CSV_Injection
    if (
      cell[0] === '=' ||
      cell[0] === '+' ||
      cell[0] === '-' ||
      cell[0] === '@'
    ) {
      escaped = `'${escaped}`;
    }
    // Wrap cell with double quotes, RFC 4810 Section 2.7
    return `"${escaped}"`;
  };
  const csvRows = rows.map((row: string[]) => {
    row = row.map(encodeCell);
    return row.join(',');
  });

  let csv = header.map(encodeCell).join(',');

  if (csvRows.length > 0) {
    csv += '\n' + csvRows.join('\n');
  }
  return csv;
}
