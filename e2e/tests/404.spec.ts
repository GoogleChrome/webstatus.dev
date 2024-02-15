import {test, expect} from '@playwright/test';

const homePageUrl = 'http://localhost:5555/';

// TODO. Redirect to the 404 page once we have it.
test('Bad URL redirection to home page', async ({page}) => {
  const badUrls = [
    // Test for bad public asset
    'http://localhost:5555/public/junk',
    // TODO. Test for bad app urls (e.g. bad feature id)
  ];

  for (const badUrl of badUrls) {
    await test.step(`Testing redirection for: ${badUrl}`, async () => {
      await page.goto(badUrl);
      await expect(page).toHaveURL(homePageUrl);
    });
  }
});
