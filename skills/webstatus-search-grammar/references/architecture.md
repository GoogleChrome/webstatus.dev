# Search Grammar Architecture & Implementation

This document provides a technical guide for the ANTLR4-based search infrastructure used across the webstatus.dev platform.

## 1. Grammar Definition

The core search language is defined using ANTLR4, allowing for complex, nested queries (e.g., `baseline:widely (wpt:chrome[>0.9] OR bcd:firefox[supported])`).

- **Grammar File**: [`antlr/FeatureSearch.g4`](../../../antlr/FeatureSearch.g4)
- **Generated Code**: The grammar is compiled into Go code located in [`lib/gen/featuresearch/`](../../../lib/gen/featuresearch/).

## 2. Technical Pipeline

The transition from a raw search string to a Spanner SQL query follows a precise transformation path.

1.  **Lexer/Parser**: Generated ANTLR4 code tokenizes the input string.
2.  **SearchNode Tree**: The parsed output is converted into a `SearchNode` tree structure in Go, which provides a type-safe representation of the query's boolean logic.
3.  **FeaturesSearchVisitor**: The [FeaturesSearchVisitor](../../../lib/gcpspanner/searchtypes/features_search_visitor.go) traverses the `SearchNode` tree.
4.  **SQL Generation**: The visitor translates each node into equivalent Spanner SQL fragments using `UNNEST`, `JOIN`, and parameterization to protect against SQL injection.

## 3. Extending the Search Grammar

To add a new search term (e.g., `availability:public`):

1.  Update the **G4 grammar** in [`antlr/FeatureSearch.g4`](../../../antlr/FeatureSearch.g4).
2.  Regenerate the parser using `make gen -B`.
3.  Update the **Visitor** in [`lib/gcpspanner/searchtypes/features_search_visitor.go`](../../../lib/gcpspanner/searchtypes/features_search_visitor.go) to handle the new token and generate the appropriate SQL.
