---
name: webstatus-search-grammar
description: Use when modifying the ANTLR search grammar, adding new search terms, or working with the query parser and builder.
---

# webstatus-search-grammar

This skill provides instructions for modifying the feature search syntax in `webstatus.dev`, which is built on ANTLR v4.

## Source of Truth

- The canonical source of truth for the search syntax is `antlr/FeatureSearch.g4`.
- **DON'T** edit the generated parser files in `lib/gen/featuresearch/parser/` directly.

## How to Add a New Search Term (e.g., `is:discouraged`)

1. **Update Grammar (`antlr/FeatureSearch.g4`)**:
   - Add the new term to the `search_criteria` rule in the grammar file (e.g., add `| discouraged_term`).
   - Define the new rule: `discouraged_term: 'is' ':' 'discouraged';`.
2. **Regenerate Parser**:
   - Run `make antlr-gen`. This will update the files in `lib/gen/featuresearch/parser/`.
3. **Update Visitor (`lib/gcpspanner/searchtypes/`)**:
   - Add a new `SearchIdentifier` for your term in `searchtypes.go` (e.g., `IdentifierIsDiscouraged`).
   - In `features_search_visitor.go`, implement the `VisitDiscouraged_termContext` method to create and return a `SearchNode` with the new identifier.
4. **Update Query Builder (`lib/gcpspanner/feature_search_query.go`)**:
   - In `FeatureSearchFilterBuilder.traverseAndGenerateFilters`, add a `case` for your new `SearchIdentifier`.
   - This case should generate the appropriate Spanner SQL `WHERE` clause for the filter.
5. **Add Tests**:
   - Add a parsing test in `lib/gcpspanner/searchtypes/features_search_parse_test.go`.
   - Add a SQL generation test in `lib/gcpspanner/feature_search_query_test.go`.
   - Add an integration test in `lib/gcpspanner/feature_search_test.go`.
6. **Update Frontend UI**:
   - Add the new search term to the search builder UI vocabulary in `frontend/src/static/js/utils/constants.ts` to make it discoverable to users.

## Documentation Updates

When you add a new search grammar term or modify parsing:

- Trigger the "Updating the Knowledge Base" prompt in `GEMINI.md` to ensure I am aware of the changes.
- Ensure that `docs/ARCHITECTURE.md` is updated if there are broader system impacts.
