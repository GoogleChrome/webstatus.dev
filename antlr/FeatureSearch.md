## Feature Search Grammar Documentation

This document provides a comprehensive guide to the Feature Search grammar, explaining its functionalities, rules,
and usage with examples.

### Introduction

The Feature Search grammar enables the construction of queries to search for features based on various criteria,
including availability on specific browsers, specific feature names, and more. It offers flexibility for future
expansion with new search fields and functionalities.

### Grammar Rules

The grammar is composed of several key rules:

- **Keywords:** These are reserved words used in the grammar, such as `AND`, `OR`, `SORT_BY`, `ASC`, and `DESC`.
- **Identifiers:** These represent names of browsers (`BROWSER_NAME`) and features (`FEATURE_NAME`).
- **Search Criteria:** These are the fundamental building blocks of a search query, encompassing various conditions to
  filter results:
  - **Availability Information (`availability_information`):** Checks if a feature is available on a specific browser.
  - **Negation of Availability (`negation_of_availability`):** Checks if a feature is **not** available on a specific
    browser.
  - **Specific Feature Name (`specific_feature_name`):** Searches for features with a matching name.
- **Search Expression (`feature_expr`):** Represents a single search criterion.
- **Combined Search Criteria (`combined_search_criteria`):** Combines multiple search expressions using `AND` or `OR`
  operators.
- **Sorting Specification (`sorting_spec`):** (Optional) Specifies the criteria and direction for sorting the search
  results.
- **Search Query (`query`):** The top-level rule that defines a complete search query, consisting of combined search
  criteria (optionally with sorting).

### Search Criteria and Examples

Here are examples of basic search queries using different criteria:

**Example 1: Find features available on Chrome:**

```
available_on: chrome
```

**Example 2: Find features not available on Chrome:**

```
NOT available_on: chrome
```

**Example 3: Find features named "CSS Grid":**

```
name: CSS Grid
```

**Combining Criteria:**

You can combine multiple criteria using `AND` or `OR` operators within `combined_search_criteria`:

**Example 4: Find features available on Chrome or Edge and named "CSS Grid":**

```
(available_on: chrome OR available_on: edge) AND name: CSS Grid
```

**Example 5: Find features available on Chrome but not on Firefox:**

```
available_on: chrome AND NOT available_on: firefox
```

### Almost Universally Supported Features

The grammar allows searching for features that are almost universally supported, meaning they are available on all
browsers **except one**. This functionality is achieved using the `missing_in_one_of` condition:

**Example 6: Find features almost universally supported (missing in one browser)**

```
missing_in_one_of(chrome,firefox,edge,safari)
```

**Explanation:**

- `missing_in_one_of(chrome,firefox,edge,safari)`: Look at all the features supported among the 4 specified browsers.
  Find the features that are supported in N-1 browsers.

### Sorting

The grammar supports optional sorting of search results by specifying a feature name and direction (ascending or descending) using the `SORT_BY` keyword:

**Example 7: Find features available on Chrome, sorted by name in ascending order:**

```
available_on: Chrome
SORT_BY: name ASC
```
