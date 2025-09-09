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

/**
 * Formats a number into a compact, human-readable string.
 * For example, 1234 becomes "1.2K".
 * @param input The number to format.
 * @returns A compact string representation of the number.
 */
export function formatNumberShorthand(input: number): string {
  return new Intl.NumberFormat(undefined, {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(input);
}

/**
 * Formats a number with thousand separators.
 * For example, 1234567 becomes "1,234,567".
 * @param input The number to format.
 * @returns A string representation of the number with thousand separators.
 */
export function formatRawNumber(input: number): string {
  return new Intl.NumberFormat(undefined).format(input);
}

interface NumberMessages {
  shorthandNumber: string;
  rawNumber: string;
  message: string;
  shortMessage: string;
}

/**
 * Generates formatted numbers and messages related to developer upvotes.
 * @param upvotes The number of developer upvotes.
 * @returns An object containing formatted numbers and messages.
 */
export function formatDeveloperUpvotesMessages(
  upvotes: number,
): NumberMessages {
  const shorthandNumber = formatNumberShorthand(upvotes);
  const rawNumber = formatRawNumber(upvotes);

  return {
    shorthandNumber,
    rawNumber,
    message: `${rawNumber} developer upvotes. Need this feature across browsers? Click this and upvote it on GitHub.`,
    shortMessage: `${rawNumber} developer upvotes`,
  };
}
