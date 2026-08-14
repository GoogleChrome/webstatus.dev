// Code generated from antlr/FeatureSearch.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // FeatureSearch

import "github.com/antlr4-go/antlr/v4"

// A complete Visitor for a parse tree produced by FeatureSearchParser.
type FeatureSearchVisitor interface {
	antlr.ParseTreeVisitor

	// Visit a parse tree produced by FeatureSearchParser#available_on_term.
	VisitAvailable_on_term(ctx *Available_on_termContext) interface{}

	// Visit a parse tree produced by FeatureSearchParser#baseline_status_term.
	VisitBaseline_status_term(ctx *Baseline_status_termContext) interface{}

	// Visit a parse tree produced by FeatureSearchParser#available_date_term.
	VisitAvailable_date_term(ctx *Available_date_termContext) interface{}

	// Visit a parse tree produced by FeatureSearchParser#baseline_date_term.
	VisitBaseline_date_term(ctx *Baseline_date_termContext) interface{}

	// Visit a parse tree produced by FeatureSearchParser#name_term.
	VisitName_term(ctx *Name_termContext) interface{}

	// Visit a parse tree produced by FeatureSearchParser#description_term.
	VisitDescription_term(ctx *Description_termContext) interface{}

	// Visit a parse tree produced by FeatureSearchParser#group_term.
	VisitGroup_term(ctx *Group_termContext) interface{}

	// Visit a parse tree produced by FeatureSearchParser#snapshot_term.
	VisitSnapshot_term(ctx *Snapshot_termContext) interface{}

	// Visit a parse tree produced by FeatureSearchParser#id_term.
	VisitId_term(ctx *Id_termContext) interface{}

	// Visit a parse tree produced by FeatureSearchParser#saved_term.
	VisitSaved_term(ctx *Saved_termContext) interface{}

	// Visit a parse tree produced by FeatureSearchParser#hotlist_term.
	VisitHotlist_term(ctx *Hotlist_termContext) interface{}

	// Visit a parse tree produced by FeatureSearchParser#term.
	VisitTerm(ctx *TermContext) interface{}

	// Visit a parse tree produced by FeatureSearchParser#date_range_query.
	VisitDate_range_query(ctx *Date_range_queryContext) interface{}

	// Visit a parse tree produced by FeatureSearchParser#generic_search_term.
	VisitGeneric_search_term(ctx *Generic_search_termContext) interface{}

	// Visit a parse tree produced by FeatureSearchParser#search_criteria.
	VisitSearch_criteria(ctx *Search_criteriaContext) interface{}

	// Visit a parse tree produced by FeatureSearchParser#parenthesizedCriteria.
	VisitParenthesizedCriteria(ctx *ParenthesizedCriteriaContext) interface{}

	// Visit a parse tree produced by FeatureSearchParser#combined_search_criteria.
	VisitCombined_search_criteria(ctx *Combined_search_criteriaContext) interface{}

	// Visit a parse tree produced by FeatureSearchParser#operator.
	VisitOperator(ctx *OperatorContext) interface{}

	// Visit a parse tree produced by FeatureSearchParser#query.
	VisitQuery(ctx *QueryContext) interface{}
}
