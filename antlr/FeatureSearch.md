# Feature Search Query Language

This query language enables you to construct flexible searches to find features based on various criteria.

## Key Concepts

- Terms: The basic building blocks of a search query. Each term has an identifier (available_on, baseline_status, name) followed by a colon (:) and its corresponding value without any spaces.
- **Value Types:**
  - browsers (`BROWSER_NAME`)
    - Accepted Values: `[a-z]+`
    - Example:
      - chrome
  - features (`FEATURE_NAME`)
    - Accepted Values: `'"' [a-zA-Z][a-zA-Z0-9_-]* [ ]* '"' | [a-zA-Z][a-zA-Z0-9_-]*`
    - Examples:
      - Grid
      - "CSS Grid"
  - baseline statuses (`BASELINE_STATUS`)
    - Accepted Values: 'none' | 'low' | 'high'
    - Examples:
      - none
  - browser list (`BROWSER_LIST`)
    - Accepted Values: `BROWSER_NAME (',' BROWSER_NAME)*`
    - Examples:
      - chrome
      - chrome,safari
- ## **Terms:**
  - `available_on`: Indicates whether a feature is available on a specific browser. Expects a browser name (BROWSER_NAME) as its value.
    - Example: `available_on:chrome`
  - `baseline_status`: Represents a feature's baseline status. Expects an enum value (BASELINE_STATUS) as its value.
    - Example: `baseline_status:low`
  - `name`: Searches for features by their name. Expects a feature name (FEATURE_NAME) as its value.
    - Examples:
      - `name:grid`
      - `name:"CSS Grid"`
  - `missing_in_one_of`: Searches for features that are almost universally supported, meaning they are available on all
    browsers **except one**. Expects a browser list (BROWSER_LIST) as its value.
    - Example:
      - missing_in_one_of(chrome, edge, firefox)
        - Explanation: Look at all the features supported among the 4 specified browsers. Find the features that are
          supported in N-1 browsers.
- **Negation:** Prepend a term with a minus sign (-) to indicate negation (search for features not matching that criterion).
- **Keywords:** These are reserved words used in the grammar, such as `AND`, `OR`
  - `AND`: Combine terms with the AND keyword for explicit logical AND, or use a space between terms for implied AND.
  - `OR`: Combine terms with OR for logical OR operations.
- **Standalone Feature Names:** Search by feature name without a 'name:' prefix.

## Example Queries

### Simple Term Examples

- `available_on:chrome` - Find features available on Chrome.
- `-available_on:firefox` - Find features not available on Firefox.
- `baseline_status:high` - Find features with a high baseline status.
- `name:"Dark Mode"` - Find features named "Dark Mode" (including spaces).

### Complex Queries

- `available_on:chrome AND baseline_status:low` - Find features available on Chrome and having a low baseline status.
- `-available_on:firefox OR name:"CSS Grid"` - Find features either not available on Firefox or named "CSS Grid".
- `missing_in_one_of(chrome,firefox,safari)` - Find features missing from at least one of the listed browsers.
- `"CSS Grid" baseline_status:none` - Find features named "CSS Grid" with a baseline status of none (implied AND).
