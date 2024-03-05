grammar FeatureSearch;

// Keywords
AND: 'AND';
OR: 'OR';
SORT_BY: 'SORT_BY:';
ASC: 'ASC';
DESC: 'DESC';

// Identifiers
BROWSER_NAME: [a-z]+;
FEATURE_NAME: [a-zA-Z][a-zA-Z0-9_]*;

// Search criteria (flexible)
search_criteria:
	availability_information
	| negation_of_availability
	| specific_feature_name
	| missing_in_one_of;

// Availability information
availability_information: 'available_on:' BROWSER_NAME;

// Negation of availability
negation_of_availability: 'NOT available_on:' BROWSER_NAME;

// Specific feature name
specific_feature_name: 'name:' FEATURE_NAME;

// Missing in one of
missing_in_one_of: 'missing_in_one_of' '(' browser_list ')';

// Browser list
browser_list:
	BROWSER_NAME (',' BROWSER_NAME) {0,3}; // 1-4 Browsers

// Feature expression
feature_expr: search_criteria;

// Combined search criteria
combined_search_criteria:
	feature_expr (AND | OR) feature_expr
	| feature_expr; // Allow single expression

// Sorting specification
sorting_spec: SORT_BY ':' FEATURE_NAME (ASC | DESC)?;

// Search query
query: combined_search_criteria (sorting_spec)? EOF;