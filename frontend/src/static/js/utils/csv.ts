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
    // Encode any double quotes in the cell as "", and then wrap the
    // cell in double quotes.
    return `"${cell.replace(/"/g, '""')}"`;
  };
  const csv = rows.map((row: string[]) => {
    row = row.map(encodeCell);
    return row.join(',');
  });

  if (csv.length > 0) {
    header = header.map(encodeCell);
    return header.join(',') + '\n' + csv.join('\n');
  } else {
    throw new Error('No rows to convert to CSV');
  }
}
