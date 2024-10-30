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
export class CSVUtils {
  /**
   * Returns a string in CSV format given header strings and rows.
   */
  static convertToCSV(header: string[], rows: string[][]): string {
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

  static downloadCSV(
    columns: string[],
    rows: string[][],
    filename: string,
  ): Promise<void> {
    const csv = this.convertToCSV(columns, rows);

    // Create blob which will be saved via a link.
    const blob = new Blob([csv], {type: 'text/csv'});
    const url = window.URL.createObjectURL(blob);

    const saveRequest = (path: string, filename?: string) =>
      // Use fetch to download the csv.
      fetch(path)
        .then(response => response.blob())
        .then(blob => {
          if (!filename) {
            const blobType = blob.type.split('/').pop();
            const type = blobType === 'plain' ? 'txt' : blobType;
            filename = 'file-' + new Date().getTime() + '.' + type;
          }

          const link = document.createElement('a');
          link.className = 'download';
          link.download = filename;
          const href = URL.createObjectURL(blob);
          console.info('href', href);
          link.href = href;
          document.body.appendChild(link);
          link.click();
          link.parentElement!.removeChild(link);
        });

    return saveRequest(url, filename);
  }
}
