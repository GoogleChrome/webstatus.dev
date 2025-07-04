# Feature Search Query Language

This query language enables you to construct flexible searches to find features based on various criteria.

## Key Concepts

- Terms: The basic building blocks of a search query. Each term has an identifier (available_on, baseline_status, name) followed by a colon (:) and its corresponding value without any spaces.
- **Value Types:**
  - browsers (`BROWSER_NAME`)
    - Accepted Values: `chrome`, `chrome_android`, `firefox`, `firefox_android`, `edge`, `safari`, `safari_ios`
    - Example:
      - chrome
  - features (`FEATURE_NAME`)
    - Accepted Values: `'"' [a-zA-Z][a-zA-Z0-9_-]* [ ]* '"' | [a-zA-Z][a-zA-Z0-9_-]*`
    - Examples:
      - Grid
      - "CSS Grid"
  - baseline statuses (`BASELINE_STATUS`)
    - Accepted Values: 'limited' | 'newly' | 'widely'
    - Examples:
      - none
  - browser list (`BROWSER_LIST`)
    - Accepted Values: `BROWSER_NAME (',' BROWSER_NAME)*`
    - Examples:
      - chrome
      - chrome,safari
- **Terms:**
  - `available_date`: Represents the date a feature became available
    - Option 1: Specifies an inclusive date range (DATE1..DATE2) during which features became available.
  - `available_on`: Indicates whether a feature is available on a specific browser. Expects a browser name (BROWSER_NAME) as its value.
    - Example: `available_on:chrome`
  - `baseline_status`: Represents a feature's baseline status. Expects an enum value (BASELINE_STATUS) as its value.
    - Example: `baseline_status:low`
  - `name`: Searches for features by their name. Expects a feature name (FEATURE_NAME) as its value.
    - Examples:
      - `name:grid`
      - `name:"CSS Grid"`
  - `desc`: Searches for features by their description. Expects a string as its value.
  - Examples:
    - `desc:@container`
    - `desc:"CSS Property"`
  - `baseline_date`: Represents the date a feature reached baseline.
    - Option 1: Searches for an inclusive date range (DATE..DATE) where features reached baseline.
  - `group`: Searches for features that belong to a group defined in the web-features [repository](https://github.com/web-platform-dx/web-features/tree/main/groups).
  - `snapshot`: Searches for features that belong to a snapshot defined in the web-features [repository](https://github.com/web-platform-dx/web-features/tree/main/snapshots).
  - `id`: Searches for features whose feature identifiers (FeatureKey) match the search value exactly.
    - Examples:
    - `id:grid`
    - `id:"CSS Grid"`
- **Negation:** Prepend a term with a minus sign (-) to indicate negation (search for features not matching that criterion).
- **Keywords:** These are reserved words used in the grammar, such as `AND`, `OR`
  - `AND`: Combine terms with the AND keyword for explicit logical AND, or use a space between terms for implied AND.
  - `OR`: Combine terms with OR for logical OR operations.
- **Standalone Feature Names:** Search by feature description and name without a `name:` or `desc:` prefix.

## Example Queries

### Simple Term Examples

- `available_date:chrome:2023-01-01..2023-12-31` - Searches for all features that became available on Chrome in 2023.
- `available_on:chrome` - Find features available on Chrome.
- `-available_on:firefox` - Find features not available on Firefox.
- `baseline_status:high` - Find features with a high baseline status.
- `name:"Dark Mode"` - Find features named "Dark Mode" (including spaces).
- `baseline_date:2023-01-01..2023-12-31` - Searches for all features that reached baseline in 2023.
- `group:css` - Searches for features that belong to the `css` group and any groups that are descendants of that group.
- `snapshot:ecmascript-5` - Searches for features that belong to the `ecmascript-5` snapshot.
- `id:css` - Searches for a feature whose feature identifier (featurekey) is `css`.

### Complex Queries

- `available_on:chrome AND baseline_status:newly` - Find features available on Chrome and having a newly baseline status.
- `-available_on:firefox OR name:"CSS Grid"` - Find features either not available on Firefox or named "CSS Grid".
- `"CSS Grid" baseline_status:limited` - Find features named "CSS Grid" with a baseline status of none (implied AND).
