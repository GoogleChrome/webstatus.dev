grammar FeatureSearch;

// Keywords
AND: 'AND';
OR: 'OR';
NOT: '-';

COLON: ':';

// Capture Whitespace. This allows for flexibility in how users write their queries, tolerating
// spaces around operators and keywords.
WS: [ \t\r\n]+ -> skip;

// Identifiers
BASELINE_STATUS: 'limited' | 'newly' | 'widely';
BROWSER_NAME: 'chrome' | 'firefox' | 'edge' | 'safari';
BROWSER_LIST: BROWSER_NAME (',' BROWSER_NAME)*;
DATE: [2][0-9][0-9][0-9]'-'[01][0-9]'-'[0-3][0-9]; // YYYY-MM-DD (starting from 2000)
ANY_VALUE:
	'"' [a-zA-Z][a-zA-Z0-9_ -]* '"' // Words with spaces.
	| [a-zA-Z][a-zA-Z0-9_-]*; // Single words

// Terms
available_on_term: 'available_on' COLON BROWSER_NAME;
baseline_status_term: 'baseline_status' COLON BASELINE_STATUS;
// In the future support other operators by doing something like (date_operator_query | date_range_query)
baseline_date_term: 'baseline_date' COLON (date_range_query);
name_term: 'name' COLON ANY_VALUE;
term: available_on_term | baseline_status_term | baseline_date_term | name_term;

date_range_query: startDate=DATE '..' endDate=DATE;

generic_search_term: (NOT)? term;

// Search criteria
search_criteria:
	generic_search_term
	| missing_in_one_of
	| ANY_VALUE; // Default to ANY_VALUE search without "name:" prefix.

// Missing in one of
missing_in_one_of: 'missing_in_one_of' '(' BROWSER_LIST ')';

// Combined search criteria
combined_search_criteria:
	// Single term or grouped expression
	(search_criteria | '(' combined_search_criteria ')')
	// Optional chaining with implicit AND or explicit operators
	(
		(operator)? // Optional explicit operator
		(
			search_criteria
			| '(' combined_search_criteria ')'
		) // Next search term or group
	)*;

operator: AND | OR;

// Search query
query: combined_search_criteria EOF;