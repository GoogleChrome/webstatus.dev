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

import {
  formatDeveloperUpvotesMessages,
  formatNumberShorthand,
  formatRawNumber,
} from '../format.js';
import {expect} from '@open-wc/testing';

describe('format', () => {
  it('formatNumberShorthand', () => {
    expect(formatNumberShorthand(123)).to.equal('123');
    expect(formatNumberShorthand(1234)).to.equal('1.2K');
    expect(formatNumberShorthand(12345)).to.equal('12.3K');
    expect(formatNumberShorthand(123456)).to.equal('123.5K');
    expect(formatNumberShorthand(1234567)).to.equal('1.2M');
  });

  it('formatRawNumber', () => {
    expect(formatRawNumber(123)).to.equal('123');
    expect(formatRawNumber(1234)).to.equal('1,234');
    expect(formatRawNumber(1234567)).to.equal('1,234,567');
  });

  it('formatDeveloperUpvotesMessages', () => {
    const result = formatDeveloperUpvotesMessages(12345);
    expect(result.shorthandNumber).to.equal('12.3K');
    expect(result.rawNumber).to.equal('12,345');
    expect(result.message).to.equal(
      '12,345 developer upvotes. Need this feature across browsers? Upvote it on GitHub.',
    );
    expect(result.shortMessage).to.equal('12,345 developer upvotes');
  });
});
