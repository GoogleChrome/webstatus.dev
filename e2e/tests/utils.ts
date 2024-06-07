import { Page } from '@playwright/test';

const DEFAULT_FAKE_NOW = 'Dec 1 2020 12:34:56';

export async function setupFakeNow(page: Page,
  fakeNowDateString = DEFAULT_FAKE_NOW): Promise<void> {
  // Get fakeNow from UTC to extract the timeZone offset used in the test
  const fakeNowFromUTC = new Date(fakeNowDateString);
  const offset = fakeNowFromUTC.getTimezoneOffset();
  const offsetSign = offset < 0 ? '-' : '+';
  const offsetHours = `${Math.abs(Math.floor(offset / 60))}`.padStart(2, '0');
  const offsetMinutes = `${Math.abs(offset % 60)}`.padStart(2, '0');
  const offsetText = `${offsetSign}${offsetHours}:${offsetMinutes}`;

  // Get fakeNow from the test timeZone
  const fakeNow = new Date(`${fakeNowDateString}Z${offsetText}`).valueOf();

  // Update the Date accordingly in your test pages
  await page.addInitScript(`{
    // Extend Date constructor to default to fakeNow
    Date = class extends Date {
      constructor(...args) {
        if (args.length === 0) {
          super(${fakeNow});
        } else {
          super(...args);
        }
      }
    }
    // Override Date.now() to start from fakeNow
    const __DateNowOffset = ${fakeNow} - Date.now();
    const __DateNow = Date.now;
    Date.now = () => __DateNow() + __DateNowOffset;
  }`);
}
