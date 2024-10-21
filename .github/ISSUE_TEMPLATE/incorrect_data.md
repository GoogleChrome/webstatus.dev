---
name: Incorrect data
about: Report incorrect data being shown on the dashboard
title: '[Incorrect Data] Title'
labels: bug
assignees: ''
---

## Is the data shown different from the original source?

Before submitting a bug report, please verify that the data displayed on
webstatus.dev is different from the data in the original source. We ingest most
data from other sources and display it; we don't modify it.

**Here's how to check:**

1. **Identify the source of the data:**

   - [ ] **Baseline Status, Browser Availability, Feature Groups, Feature Snapshots:**
         Check the [Web DX Features repository](https://github.com/web-platform-dx/web-features)
   - [ ] **Web Platform Test scores:** Check the
         [Web Platform Tests repository](https://github.com/web-platform-tests/wpt)
   - [ ] **Browser release dates:** Check
         [Browser Compat Data](https://github.com/mdn/browser-compat-data)
         (Please provide a specific link to the relevant section within Browser
         Compat Data if possible)

2. **Compare the data:** Carefully compare the data displayed on webstatus.dev
   with the data in the source you identified. Note any discrepancies.

**If the data is the same in both places:**

- [ ] **Do not submit an issue here.** The issue lies with the original data
      source. Please submit an issue in the relevant repository:

  - [Web DX Features repository](https://github.com/web-platform-dx/web-features)
  - [Web Platform Tests repository](https://github.com/web-platform-tests/wpt)
  - [Browser Compat Data](https://github.com/mdn/browser-compat-data)

**If the data is different:**

- [ ] Please provide the following information in your bug report:

  - **Screenshot:** A screenshot clearly showing the discrepancy between
    webstatus.dev and the original data source.
  - **URL:** The URL of the page on webstatus.dev where the incorrect data is
    displayed.
  - **Source URL:** The URL of the original data source.
  - **Specific details:** A clear description of the difference between the two
    data sources.

This will help us to quickly identify and fix the issue. Thank you!
