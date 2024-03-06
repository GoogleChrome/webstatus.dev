grammar FeatureSearch;

// Keywords
AND: 'AND';
OR: 'OR';
NOT: '-';

// Identifiers
BROWSER_NAME: [a-z]+;
FEATURE_NAME:
	'"' [a-zA-Z][a-zA-Z0-9_-]* [ ]* '"'
	| [a-zA-Z][a-zA-Z0-9_-]*;
BASELINE_STATUS: 'none' | 'low' | 'high';
BROWSER_LIST: BROWSER_NAME (',' BROWSER_NAME)*;

// Value Types
value_type: BROWSER_NAME | BASELINE_STATUS | FEATURE_NAME;

term:
	'available_on' // Value Type = BROWSER_NAME
	| 'baseline_status' // Value Type = BASELINE_STATUS
	| 'name'; // Value Type = FEATURE_NAME

generic_search_term: (NOT)? term ':' value_type;

// Search criteria
search_criteria:
	generic_search_term
	| missing_in_one_of
	| FEATURE_NAME; // Default to FEATURE_NAME search without "name:" prefix.

// Missing in one of
missing_in_one_of: 'missing_in_one_of' '(' BROWSER_LIST ')';

// Combined search criteria
combined_search_criteria:
	search_criteria (AND search_criteria)* // Explicit AND
	| search_criteria (' ' search_criteria)* // Implied AND
	| search_criteria (OR search_criteria)* // Explicit OR
	| search_criteria; // Allow single expression

// Search query
query: combined_search_criteria EOF;