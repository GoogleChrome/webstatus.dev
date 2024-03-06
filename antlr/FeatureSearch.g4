grammar FeatureSearch;

// Keywords
AND: 'AND';
OR: 'OR';

// Identifiers
BROWSER_NAME: [a-z]+;
FEATURE_NAME: [a-zA-Z][a-zA-Z0-9_-]*;
BASELINE_STATUS: 'none' | 'low' | 'high';

// Search criteria (flexible)
search_criteria:
	available_on_filter
	| negate_available_on_filter
	| specific_feature_name
	| missing_in_one_of
	| baseline_status_filter
	| negate_baseline_status_filter
	| specific_feature_name
	| FEATURE_NAME; // Default to FEATURE_NAME search without "name:" prefix.

// Availability information
available_on_filter: 'available_on:' BROWSER_NAME;
negate_available_on_filter: '-' available_on_filter;

// Baseline Status
baseline_status_filter: 'baseline_status:' BASELINE_STATUS;
negate_baseline_status_filter: '-' baseline_status_filter;

// Specific feature name
specific_feature_name: 'name:' FEATURE_NAME;

// Missing in one of
missing_in_one_of: 'missing_in_one_of' '(' browser_list ')';

// Browser list
browser_list:
	BROWSER_NAME (',' BROWSER_NAME) {0,3}; // 1-4 Browsers

// Combined search criteria
combined_search_criteria:
	search_criteria (AND search_criteria)* // Explicit AND
	| search_criteria (' ' search_criteria)* // Implied AND
	| search_criteria (OR search_criteria)* // Explicit OR
	| search_criteria; // Allow single expression

// Search query
query: combined_search_criteria EOF;