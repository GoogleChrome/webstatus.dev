// Code generated from antlr/FeatureSearch.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // FeatureSearch

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/antlr4-go/antlr/v4"
)

// Suppress unused import errors
var _ = fmt.Printf
var _ = strconv.Itoa
var _ = sync.Once{}

type FeatureSearchParser struct {
	*antlr.BaseParser
}

var FeatureSearchParserStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	LiteralNames           []string
	SymbolicNames          []string
	RuleNames              []string
	PredictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func featuresearchParserInit() {
	staticData := &FeatureSearchParserStaticData
	staticData.LiteralNames = []string{
		"", "'available_on'", "'baseline_status'", "'available_date'", "'baseline_date'",
		"'name'", "'desc'", "'group'", "'snapshot'", "'id'", "'saved'", "'hotlist'",
		"'..'", "'('", "')'", "'AND'", "'OR'", "'-'", "':'",
	}
	staticData.SymbolicNames = []string{
		"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "AND", "OR",
		"NOT", "COLON", "WS", "BROWSER_NAME", "BASELINE_STATUS", "DATE", "ANY_VALUE",
	}
	staticData.RuleNames = []string{
		"available_on_term", "baseline_status_term", "available_date_term",
		"baseline_date_term", "name_term", "description_term", "group_term",
		"snapshot_term", "id_term", "saved_term", "hotlist_term", "term", "date_range_query",
		"generic_search_term", "search_criteria", "parenthesizedCriteria", "combined_search_criteria",
		"operator", "query",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 23, 136, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
		4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2, 10, 7,
		10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15, 7, 15,
		2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 1, 0, 1, 0, 1, 0, 1, 0, 1, 1,
		1, 1, 1, 1, 1, 1, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 3, 1, 3, 1, 3,
		1, 3, 1, 4, 1, 4, 1, 4, 1, 4, 1, 5, 1, 5, 1, 5, 1, 5, 1, 6, 1, 6, 1, 6,
		1, 6, 1, 7, 1, 7, 1, 7, 1, 7, 1, 8, 1, 8, 1, 8, 1, 8, 1, 9, 1, 9, 1, 9,
		1, 9, 1, 10, 1, 10, 1, 10, 1, 10, 1, 11, 1, 11, 1, 11, 1, 11, 1, 11, 1,
		11, 1, 11, 1, 11, 1, 11, 1, 11, 1, 11, 3, 11, 96, 8, 11, 1, 12, 1, 12,
		1, 12, 1, 12, 1, 13, 3, 13, 103, 8, 13, 1, 13, 1, 13, 1, 14, 1, 14, 3,
		14, 109, 8, 14, 1, 15, 1, 15, 1, 15, 1, 15, 1, 16, 1, 16, 3, 16, 117, 8,
		16, 1, 16, 3, 16, 120, 8, 16, 1, 16, 1, 16, 3, 16, 124, 8, 16, 5, 16, 126,
		8, 16, 10, 16, 12, 16, 129, 9, 16, 1, 17, 1, 17, 1, 18, 1, 18, 1, 18, 1,
		18, 0, 0, 19, 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30,
		32, 34, 36, 0, 1, 1, 0, 15, 16, 132, 0, 38, 1, 0, 0, 0, 2, 42, 1, 0, 0,
		0, 4, 46, 1, 0, 0, 0, 6, 52, 1, 0, 0, 0, 8, 56, 1, 0, 0, 0, 10, 60, 1,
		0, 0, 0, 12, 64, 1, 0, 0, 0, 14, 68, 1, 0, 0, 0, 16, 72, 1, 0, 0, 0, 18,
		76, 1, 0, 0, 0, 20, 80, 1, 0, 0, 0, 22, 95, 1, 0, 0, 0, 24, 97, 1, 0, 0,
		0, 26, 102, 1, 0, 0, 0, 28, 108, 1, 0, 0, 0, 30, 110, 1, 0, 0, 0, 32, 116,
		1, 0, 0, 0, 34, 130, 1, 0, 0, 0, 36, 132, 1, 0, 0, 0, 38, 39, 5, 1, 0,
		0, 39, 40, 5, 18, 0, 0, 40, 41, 5, 20, 0, 0, 41, 1, 1, 0, 0, 0, 42, 43,
		5, 2, 0, 0, 43, 44, 5, 18, 0, 0, 44, 45, 5, 21, 0, 0, 45, 3, 1, 0, 0, 0,
		46, 47, 5, 3, 0, 0, 47, 48, 5, 18, 0, 0, 48, 49, 5, 20, 0, 0, 49, 50, 5,
		18, 0, 0, 50, 51, 3, 24, 12, 0, 51, 5, 1, 0, 0, 0, 52, 53, 5, 4, 0, 0,
		53, 54, 5, 18, 0, 0, 54, 55, 3, 24, 12, 0, 55, 7, 1, 0, 0, 0, 56, 57, 5,
		5, 0, 0, 57, 58, 5, 18, 0, 0, 58, 59, 5, 23, 0, 0, 59, 9, 1, 0, 0, 0, 60,
		61, 5, 6, 0, 0, 61, 62, 5, 18, 0, 0, 62, 63, 5, 23, 0, 0, 63, 11, 1, 0,
		0, 0, 64, 65, 5, 7, 0, 0, 65, 66, 5, 18, 0, 0, 66, 67, 5, 23, 0, 0, 67,
		13, 1, 0, 0, 0, 68, 69, 5, 8, 0, 0, 69, 70, 5, 18, 0, 0, 70, 71, 5, 23,
		0, 0, 71, 15, 1, 0, 0, 0, 72, 73, 5, 9, 0, 0, 73, 74, 5, 18, 0, 0, 74,
		75, 5, 23, 0, 0, 75, 17, 1, 0, 0, 0, 76, 77, 5, 10, 0, 0, 77, 78, 5, 18,
		0, 0, 78, 79, 5, 23, 0, 0, 79, 19, 1, 0, 0, 0, 80, 81, 5, 11, 0, 0, 81,
		82, 5, 18, 0, 0, 82, 83, 5, 23, 0, 0, 83, 21, 1, 0, 0, 0, 84, 96, 3, 4,
		2, 0, 85, 96, 3, 0, 0, 0, 86, 96, 3, 2, 1, 0, 87, 96, 3, 6, 3, 0, 88, 96,
		3, 12, 6, 0, 89, 96, 3, 16, 8, 0, 90, 96, 3, 14, 7, 0, 91, 96, 3, 10, 5,
		0, 92, 96, 3, 18, 9, 0, 93, 96, 3, 20, 10, 0, 94, 96, 3, 8, 4, 0, 95, 84,
		1, 0, 0, 0, 95, 85, 1, 0, 0, 0, 95, 86, 1, 0, 0, 0, 95, 87, 1, 0, 0, 0,
		95, 88, 1, 0, 0, 0, 95, 89, 1, 0, 0, 0, 95, 90, 1, 0, 0, 0, 95, 91, 1,
		0, 0, 0, 95, 92, 1, 0, 0, 0, 95, 93, 1, 0, 0, 0, 95, 94, 1, 0, 0, 0, 96,
		23, 1, 0, 0, 0, 97, 98, 5, 22, 0, 0, 98, 99, 5, 12, 0, 0, 99, 100, 5, 22,
		0, 0, 100, 25, 1, 0, 0, 0, 101, 103, 5, 17, 0, 0, 102, 101, 1, 0, 0, 0,
		102, 103, 1, 0, 0, 0, 103, 104, 1, 0, 0, 0, 104, 105, 3, 22, 11, 0, 105,
		27, 1, 0, 0, 0, 106, 109, 3, 26, 13, 0, 107, 109, 5, 23, 0, 0, 108, 106,
		1, 0, 0, 0, 108, 107, 1, 0, 0, 0, 109, 29, 1, 0, 0, 0, 110, 111, 5, 13,
		0, 0, 111, 112, 3, 32, 16, 0, 112, 113, 5, 14, 0, 0, 113, 31, 1, 0, 0,
		0, 114, 117, 3, 28, 14, 0, 115, 117, 3, 30, 15, 0, 116, 114, 1, 0, 0, 0,
		116, 115, 1, 0, 0, 0, 117, 127, 1, 0, 0, 0, 118, 120, 3, 34, 17, 0, 119,
		118, 1, 0, 0, 0, 119, 120, 1, 0, 0, 0, 120, 123, 1, 0, 0, 0, 121, 124,
		3, 28, 14, 0, 122, 124, 3, 30, 15, 0, 123, 121, 1, 0, 0, 0, 123, 122, 1,
		0, 0, 0, 124, 126, 1, 0, 0, 0, 125, 119, 1, 0, 0, 0, 126, 129, 1, 0, 0,
		0, 127, 125, 1, 0, 0, 0, 127, 128, 1, 0, 0, 0, 128, 33, 1, 0, 0, 0, 129,
		127, 1, 0, 0, 0, 130, 131, 7, 0, 0, 0, 131, 35, 1, 0, 0, 0, 132, 133, 3,
		32, 16, 0, 133, 134, 5, 0, 0, 1, 134, 37, 1, 0, 0, 0, 7, 95, 102, 108,
		116, 119, 123, 127,
	}
	deserializer := antlr.NewATNDeserializer(nil)
	staticData.atn = deserializer.Deserialize(staticData.serializedATN)
	atn := staticData.atn
	staticData.decisionToDFA = make([]*antlr.DFA, len(atn.DecisionToState))
	decisionToDFA := staticData.decisionToDFA
	for index, state := range atn.DecisionToState {
		decisionToDFA[index] = antlr.NewDFA(state, index)
	}
}

// FeatureSearchParserInit initializes any static state used to implement FeatureSearchParser. By default the
// static state used to implement the parser is lazily initialized during the first call to
// NewFeatureSearchParser(). You can call this function if you wish to initialize the static state ahead
// of time.
func FeatureSearchParserInit() {
	staticData := &FeatureSearchParserStaticData
	staticData.once.Do(featuresearchParserInit)
}

// NewFeatureSearchParser produces a new parser instance for the optional input antlr.TokenStream.
func NewFeatureSearchParser(input antlr.TokenStream) *FeatureSearchParser {
	FeatureSearchParserInit()
	this := new(FeatureSearchParser)
	this.BaseParser = antlr.NewBaseParser(input)
	staticData := &FeatureSearchParserStaticData
	this.Interpreter = antlr.NewParserATNSimulator(this, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	this.RuleNames = staticData.RuleNames
	this.LiteralNames = staticData.LiteralNames
	this.SymbolicNames = staticData.SymbolicNames
	this.GrammarFileName = "FeatureSearch.g4"

	return this
}

// FeatureSearchParser tokens.
const (
	FeatureSearchParserEOF             = antlr.TokenEOF
	FeatureSearchParserT__0            = 1
	FeatureSearchParserT__1            = 2
	FeatureSearchParserT__2            = 3
	FeatureSearchParserT__3            = 4
	FeatureSearchParserT__4            = 5
	FeatureSearchParserT__5            = 6
	FeatureSearchParserT__6            = 7
	FeatureSearchParserT__7            = 8
	FeatureSearchParserT__8            = 9
	FeatureSearchParserT__9            = 10
	FeatureSearchParserT__10           = 11
	FeatureSearchParserT__11           = 12
	FeatureSearchParserT__12           = 13
	FeatureSearchParserT__13           = 14
	FeatureSearchParserAND             = 15
	FeatureSearchParserOR              = 16
	FeatureSearchParserNOT             = 17
	FeatureSearchParserCOLON           = 18
	FeatureSearchParserWS              = 19
	FeatureSearchParserBROWSER_NAME    = 20
	FeatureSearchParserBASELINE_STATUS = 21
	FeatureSearchParserDATE            = 22
	FeatureSearchParserANY_VALUE       = 23
)

// FeatureSearchParser rules.
const (
	FeatureSearchParserRULE_available_on_term        = 0
	FeatureSearchParserRULE_baseline_status_term     = 1
	FeatureSearchParserRULE_available_date_term      = 2
	FeatureSearchParserRULE_baseline_date_term       = 3
	FeatureSearchParserRULE_name_term                = 4
	FeatureSearchParserRULE_description_term         = 5
	FeatureSearchParserRULE_group_term               = 6
	FeatureSearchParserRULE_snapshot_term            = 7
	FeatureSearchParserRULE_id_term                  = 8
	FeatureSearchParserRULE_saved_term               = 9
	FeatureSearchParserRULE_hotlist_term             = 10
	FeatureSearchParserRULE_term                     = 11
	FeatureSearchParserRULE_date_range_query         = 12
	FeatureSearchParserRULE_generic_search_term      = 13
	FeatureSearchParserRULE_search_criteria          = 14
	FeatureSearchParserRULE_parenthesizedCriteria    = 15
	FeatureSearchParserRULE_combined_search_criteria = 16
	FeatureSearchParserRULE_operator                 = 17
	FeatureSearchParserRULE_query                    = 18
)

// IAvailable_on_termContext is an interface to support dynamic dispatch.
type IAvailable_on_termContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	COLON() antlr.TerminalNode
	BROWSER_NAME() antlr.TerminalNode

	// IsAvailable_on_termContext differentiates from other interfaces.
	IsAvailable_on_termContext()
}

type Available_on_termContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyAvailable_on_termContext() *Available_on_termContext {
	var p = new(Available_on_termContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_available_on_term
	return p
}

func InitEmptyAvailable_on_termContext(p *Available_on_termContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_available_on_term
}

func (*Available_on_termContext) IsAvailable_on_termContext() {}

func NewAvailable_on_termContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Available_on_termContext {
	var p = new(Available_on_termContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = FeatureSearchParserRULE_available_on_term

	return p
}

func (s *Available_on_termContext) GetParser() antlr.Parser { return s.parser }

func (s *Available_on_termContext) COLON() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserCOLON, 0)
}

func (s *Available_on_termContext) BROWSER_NAME() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserBROWSER_NAME, 0)
}

func (s *Available_on_termContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Available_on_termContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Available_on_termContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case FeatureSearchVisitor:
		return t.VisitAvailable_on_term(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *FeatureSearchParser) Available_on_term() (localctx IAvailable_on_termContext) {
	localctx = NewAvailable_on_termContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 0, FeatureSearchParserRULE_available_on_term)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(38)
		p.Match(FeatureSearchParserT__0)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(39)
		p.Match(FeatureSearchParserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(40)
		p.Match(FeatureSearchParserBROWSER_NAME)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IBaseline_status_termContext is an interface to support dynamic dispatch.
type IBaseline_status_termContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	COLON() antlr.TerminalNode
	BASELINE_STATUS() antlr.TerminalNode

	// IsBaseline_status_termContext differentiates from other interfaces.
	IsBaseline_status_termContext()
}

type Baseline_status_termContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyBaseline_status_termContext() *Baseline_status_termContext {
	var p = new(Baseline_status_termContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_baseline_status_term
	return p
}

func InitEmptyBaseline_status_termContext(p *Baseline_status_termContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_baseline_status_term
}

func (*Baseline_status_termContext) IsBaseline_status_termContext() {}

func NewBaseline_status_termContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Baseline_status_termContext {
	var p = new(Baseline_status_termContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = FeatureSearchParserRULE_baseline_status_term

	return p
}

func (s *Baseline_status_termContext) GetParser() antlr.Parser { return s.parser }

func (s *Baseline_status_termContext) COLON() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserCOLON, 0)
}

func (s *Baseline_status_termContext) BASELINE_STATUS() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserBASELINE_STATUS, 0)
}

func (s *Baseline_status_termContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Baseline_status_termContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Baseline_status_termContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case FeatureSearchVisitor:
		return t.VisitBaseline_status_term(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *FeatureSearchParser) Baseline_status_term() (localctx IBaseline_status_termContext) {
	localctx = NewBaseline_status_termContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 2, FeatureSearchParserRULE_baseline_status_term)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(42)
		p.Match(FeatureSearchParserT__1)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(43)
		p.Match(FeatureSearchParserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(44)
		p.Match(FeatureSearchParserBASELINE_STATUS)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IAvailable_date_termContext is an interface to support dynamic dispatch.
type IAvailable_date_termContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllCOLON() []antlr.TerminalNode
	COLON(i int) antlr.TerminalNode
	BROWSER_NAME() antlr.TerminalNode
	Date_range_query() IDate_range_queryContext

	// IsAvailable_date_termContext differentiates from other interfaces.
	IsAvailable_date_termContext()
}

type Available_date_termContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyAvailable_date_termContext() *Available_date_termContext {
	var p = new(Available_date_termContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_available_date_term
	return p
}

func InitEmptyAvailable_date_termContext(p *Available_date_termContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_available_date_term
}

func (*Available_date_termContext) IsAvailable_date_termContext() {}

func NewAvailable_date_termContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Available_date_termContext {
	var p = new(Available_date_termContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = FeatureSearchParserRULE_available_date_term

	return p
}

func (s *Available_date_termContext) GetParser() antlr.Parser { return s.parser }

func (s *Available_date_termContext) AllCOLON() []antlr.TerminalNode {
	return s.GetTokens(FeatureSearchParserCOLON)
}

func (s *Available_date_termContext) COLON(i int) antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserCOLON, i)
}

func (s *Available_date_termContext) BROWSER_NAME() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserBROWSER_NAME, 0)
}

func (s *Available_date_termContext) Date_range_query() IDate_range_queryContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDate_range_queryContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IDate_range_queryContext)
}

func (s *Available_date_termContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Available_date_termContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Available_date_termContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case FeatureSearchVisitor:
		return t.VisitAvailable_date_term(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *FeatureSearchParser) Available_date_term() (localctx IAvailable_date_termContext) {
	localctx = NewAvailable_date_termContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 4, FeatureSearchParserRULE_available_date_term)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(46)
		p.Match(FeatureSearchParserT__2)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(47)
		p.Match(FeatureSearchParserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(48)
		p.Match(FeatureSearchParserBROWSER_NAME)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(49)
		p.Match(FeatureSearchParserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

	{
		p.SetState(50)
		p.Date_range_query()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IBaseline_date_termContext is an interface to support dynamic dispatch.
type IBaseline_date_termContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	COLON() antlr.TerminalNode
	Date_range_query() IDate_range_queryContext

	// IsBaseline_date_termContext differentiates from other interfaces.
	IsBaseline_date_termContext()
}

type Baseline_date_termContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyBaseline_date_termContext() *Baseline_date_termContext {
	var p = new(Baseline_date_termContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_baseline_date_term
	return p
}

func InitEmptyBaseline_date_termContext(p *Baseline_date_termContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_baseline_date_term
}

func (*Baseline_date_termContext) IsBaseline_date_termContext() {}

func NewBaseline_date_termContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Baseline_date_termContext {
	var p = new(Baseline_date_termContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = FeatureSearchParserRULE_baseline_date_term

	return p
}

func (s *Baseline_date_termContext) GetParser() antlr.Parser { return s.parser }

func (s *Baseline_date_termContext) COLON() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserCOLON, 0)
}

func (s *Baseline_date_termContext) Date_range_query() IDate_range_queryContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDate_range_queryContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IDate_range_queryContext)
}

func (s *Baseline_date_termContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Baseline_date_termContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Baseline_date_termContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case FeatureSearchVisitor:
		return t.VisitBaseline_date_term(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *FeatureSearchParser) Baseline_date_term() (localctx IBaseline_date_termContext) {
	localctx = NewBaseline_date_termContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 6, FeatureSearchParserRULE_baseline_date_term)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(52)
		p.Match(FeatureSearchParserT__3)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(53)
		p.Match(FeatureSearchParserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

	{
		p.SetState(54)
		p.Date_range_query()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IName_termContext is an interface to support dynamic dispatch.
type IName_termContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	COLON() antlr.TerminalNode
	ANY_VALUE() antlr.TerminalNode

	// IsName_termContext differentiates from other interfaces.
	IsName_termContext()
}

type Name_termContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyName_termContext() *Name_termContext {
	var p = new(Name_termContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_name_term
	return p
}

func InitEmptyName_termContext(p *Name_termContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_name_term
}

func (*Name_termContext) IsName_termContext() {}

func NewName_termContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Name_termContext {
	var p = new(Name_termContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = FeatureSearchParserRULE_name_term

	return p
}

func (s *Name_termContext) GetParser() antlr.Parser { return s.parser }

func (s *Name_termContext) COLON() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserCOLON, 0)
}

func (s *Name_termContext) ANY_VALUE() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserANY_VALUE, 0)
}

func (s *Name_termContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Name_termContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Name_termContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case FeatureSearchVisitor:
		return t.VisitName_term(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *FeatureSearchParser) Name_term() (localctx IName_termContext) {
	localctx = NewName_termContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 8, FeatureSearchParserRULE_name_term)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(56)
		p.Match(FeatureSearchParserT__4)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(57)
		p.Match(FeatureSearchParserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(58)
		p.Match(FeatureSearchParserANY_VALUE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IDescription_termContext is an interface to support dynamic dispatch.
type IDescription_termContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	COLON() antlr.TerminalNode
	ANY_VALUE() antlr.TerminalNode

	// IsDescription_termContext differentiates from other interfaces.
	IsDescription_termContext()
}

type Description_termContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyDescription_termContext() *Description_termContext {
	var p = new(Description_termContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_description_term
	return p
}

func InitEmptyDescription_termContext(p *Description_termContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_description_term
}

func (*Description_termContext) IsDescription_termContext() {}

func NewDescription_termContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Description_termContext {
	var p = new(Description_termContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = FeatureSearchParserRULE_description_term

	return p
}

func (s *Description_termContext) GetParser() antlr.Parser { return s.parser }

func (s *Description_termContext) COLON() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserCOLON, 0)
}

func (s *Description_termContext) ANY_VALUE() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserANY_VALUE, 0)
}

func (s *Description_termContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Description_termContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Description_termContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case FeatureSearchVisitor:
		return t.VisitDescription_term(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *FeatureSearchParser) Description_term() (localctx IDescription_termContext) {
	localctx = NewDescription_termContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 10, FeatureSearchParserRULE_description_term)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(60)
		p.Match(FeatureSearchParserT__5)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(61)
		p.Match(FeatureSearchParserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(62)
		p.Match(FeatureSearchParserANY_VALUE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IGroup_termContext is an interface to support dynamic dispatch.
type IGroup_termContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	COLON() antlr.TerminalNode
	ANY_VALUE() antlr.TerminalNode

	// IsGroup_termContext differentiates from other interfaces.
	IsGroup_termContext()
}

type Group_termContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyGroup_termContext() *Group_termContext {
	var p = new(Group_termContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_group_term
	return p
}

func InitEmptyGroup_termContext(p *Group_termContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_group_term
}

func (*Group_termContext) IsGroup_termContext() {}

func NewGroup_termContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Group_termContext {
	var p = new(Group_termContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = FeatureSearchParserRULE_group_term

	return p
}

func (s *Group_termContext) GetParser() antlr.Parser { return s.parser }

func (s *Group_termContext) COLON() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserCOLON, 0)
}

func (s *Group_termContext) ANY_VALUE() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserANY_VALUE, 0)
}

func (s *Group_termContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Group_termContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Group_termContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case FeatureSearchVisitor:
		return t.VisitGroup_term(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *FeatureSearchParser) Group_term() (localctx IGroup_termContext) {
	localctx = NewGroup_termContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 12, FeatureSearchParserRULE_group_term)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(64)
		p.Match(FeatureSearchParserT__6)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(65)
		p.Match(FeatureSearchParserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(66)
		p.Match(FeatureSearchParserANY_VALUE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ISnapshot_termContext is an interface to support dynamic dispatch.
type ISnapshot_termContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	COLON() antlr.TerminalNode
	ANY_VALUE() antlr.TerminalNode

	// IsSnapshot_termContext differentiates from other interfaces.
	IsSnapshot_termContext()
}

type Snapshot_termContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySnapshot_termContext() *Snapshot_termContext {
	var p = new(Snapshot_termContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_snapshot_term
	return p
}

func InitEmptySnapshot_termContext(p *Snapshot_termContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_snapshot_term
}

func (*Snapshot_termContext) IsSnapshot_termContext() {}

func NewSnapshot_termContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Snapshot_termContext {
	var p = new(Snapshot_termContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = FeatureSearchParserRULE_snapshot_term

	return p
}

func (s *Snapshot_termContext) GetParser() antlr.Parser { return s.parser }

func (s *Snapshot_termContext) COLON() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserCOLON, 0)
}

func (s *Snapshot_termContext) ANY_VALUE() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserANY_VALUE, 0)
}

func (s *Snapshot_termContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Snapshot_termContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Snapshot_termContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case FeatureSearchVisitor:
		return t.VisitSnapshot_term(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *FeatureSearchParser) Snapshot_term() (localctx ISnapshot_termContext) {
	localctx = NewSnapshot_termContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 14, FeatureSearchParserRULE_snapshot_term)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(68)
		p.Match(FeatureSearchParserT__7)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(69)
		p.Match(FeatureSearchParserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(70)
		p.Match(FeatureSearchParserANY_VALUE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IId_termContext is an interface to support dynamic dispatch.
type IId_termContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	COLON() antlr.TerminalNode
	ANY_VALUE() antlr.TerminalNode

	// IsId_termContext differentiates from other interfaces.
	IsId_termContext()
}

type Id_termContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyId_termContext() *Id_termContext {
	var p = new(Id_termContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_id_term
	return p
}

func InitEmptyId_termContext(p *Id_termContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_id_term
}

func (*Id_termContext) IsId_termContext() {}

func NewId_termContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Id_termContext {
	var p = new(Id_termContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = FeatureSearchParserRULE_id_term

	return p
}

func (s *Id_termContext) GetParser() antlr.Parser { return s.parser }

func (s *Id_termContext) COLON() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserCOLON, 0)
}

func (s *Id_termContext) ANY_VALUE() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserANY_VALUE, 0)
}

func (s *Id_termContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Id_termContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Id_termContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case FeatureSearchVisitor:
		return t.VisitId_term(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *FeatureSearchParser) Id_term() (localctx IId_termContext) {
	localctx = NewId_termContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 16, FeatureSearchParserRULE_id_term)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(72)
		p.Match(FeatureSearchParserT__8)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(73)
		p.Match(FeatureSearchParserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(74)
		p.Match(FeatureSearchParserANY_VALUE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ISaved_termContext is an interface to support dynamic dispatch.
type ISaved_termContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	COLON() antlr.TerminalNode
	ANY_VALUE() antlr.TerminalNode

	// IsSaved_termContext differentiates from other interfaces.
	IsSaved_termContext()
}

type Saved_termContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySaved_termContext() *Saved_termContext {
	var p = new(Saved_termContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_saved_term
	return p
}

func InitEmptySaved_termContext(p *Saved_termContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_saved_term
}

func (*Saved_termContext) IsSaved_termContext() {}

func NewSaved_termContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Saved_termContext {
	var p = new(Saved_termContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = FeatureSearchParserRULE_saved_term

	return p
}

func (s *Saved_termContext) GetParser() antlr.Parser { return s.parser }

func (s *Saved_termContext) COLON() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserCOLON, 0)
}

func (s *Saved_termContext) ANY_VALUE() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserANY_VALUE, 0)
}

func (s *Saved_termContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Saved_termContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Saved_termContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case FeatureSearchVisitor:
		return t.VisitSaved_term(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *FeatureSearchParser) Saved_term() (localctx ISaved_termContext) {
	localctx = NewSaved_termContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 18, FeatureSearchParserRULE_saved_term)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(76)
		p.Match(FeatureSearchParserT__9)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(77)
		p.Match(FeatureSearchParserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(78)
		p.Match(FeatureSearchParserANY_VALUE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IHotlist_termContext is an interface to support dynamic dispatch.
type IHotlist_termContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	COLON() antlr.TerminalNode
	ANY_VALUE() antlr.TerminalNode

	// IsHotlist_termContext differentiates from other interfaces.
	IsHotlist_termContext()
}

type Hotlist_termContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyHotlist_termContext() *Hotlist_termContext {
	var p = new(Hotlist_termContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_hotlist_term
	return p
}

func InitEmptyHotlist_termContext(p *Hotlist_termContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_hotlist_term
}

func (*Hotlist_termContext) IsHotlist_termContext() {}

func NewHotlist_termContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Hotlist_termContext {
	var p = new(Hotlist_termContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = FeatureSearchParserRULE_hotlist_term

	return p
}

func (s *Hotlist_termContext) GetParser() antlr.Parser { return s.parser }

func (s *Hotlist_termContext) COLON() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserCOLON, 0)
}

func (s *Hotlist_termContext) ANY_VALUE() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserANY_VALUE, 0)
}

func (s *Hotlist_termContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Hotlist_termContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Hotlist_termContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case FeatureSearchVisitor:
		return t.VisitHotlist_term(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *FeatureSearchParser) Hotlist_term() (localctx IHotlist_termContext) {
	localctx = NewHotlist_termContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 20, FeatureSearchParserRULE_hotlist_term)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(80)
		p.Match(FeatureSearchParserT__10)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(81)
		p.Match(FeatureSearchParserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(82)
		p.Match(FeatureSearchParserANY_VALUE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ITermContext is an interface to support dynamic dispatch.
type ITermContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Available_date_term() IAvailable_date_termContext
	Available_on_term() IAvailable_on_termContext
	Baseline_status_term() IBaseline_status_termContext
	Baseline_date_term() IBaseline_date_termContext
	Group_term() IGroup_termContext
	Id_term() IId_termContext
	Snapshot_term() ISnapshot_termContext
	Description_term() IDescription_termContext
	Saved_term() ISaved_termContext
	Hotlist_term() IHotlist_termContext
	Name_term() IName_termContext

	// IsTermContext differentiates from other interfaces.
	IsTermContext()
}

type TermContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyTermContext() *TermContext {
	var p = new(TermContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_term
	return p
}

func InitEmptyTermContext(p *TermContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_term
}

func (*TermContext) IsTermContext() {}

func NewTermContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *TermContext {
	var p = new(TermContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = FeatureSearchParserRULE_term

	return p
}

func (s *TermContext) GetParser() antlr.Parser { return s.parser }

func (s *TermContext) Available_date_term() IAvailable_date_termContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IAvailable_date_termContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IAvailable_date_termContext)
}

func (s *TermContext) Available_on_term() IAvailable_on_termContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IAvailable_on_termContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IAvailable_on_termContext)
}

func (s *TermContext) Baseline_status_term() IBaseline_status_termContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBaseline_status_termContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBaseline_status_termContext)
}

func (s *TermContext) Baseline_date_term() IBaseline_date_termContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBaseline_date_termContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBaseline_date_termContext)
}

func (s *TermContext) Group_term() IGroup_termContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IGroup_termContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IGroup_termContext)
}

func (s *TermContext) Id_term() IId_termContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IId_termContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IId_termContext)
}

func (s *TermContext) Snapshot_term() ISnapshot_termContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISnapshot_termContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISnapshot_termContext)
}

func (s *TermContext) Description_term() IDescription_termContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDescription_termContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IDescription_termContext)
}

func (s *TermContext) Saved_term() ISaved_termContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISaved_termContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISaved_termContext)
}

func (s *TermContext) Hotlist_term() IHotlist_termContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IHotlist_termContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IHotlist_termContext)
}

func (s *TermContext) Name_term() IName_termContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IName_termContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IName_termContext)
}

func (s *TermContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *TermContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *TermContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case FeatureSearchVisitor:
		return t.VisitTerm(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *FeatureSearchParser) Term() (localctx ITermContext) {
	localctx = NewTermContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 22, FeatureSearchParserRULE_term)
	p.SetState(95)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case FeatureSearchParserT__2:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(84)
			p.Available_date_term()
		}

	case FeatureSearchParserT__0:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(85)
			p.Available_on_term()
		}

	case FeatureSearchParserT__1:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(86)
			p.Baseline_status_term()
		}

	case FeatureSearchParserT__3:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(87)
			p.Baseline_date_term()
		}

	case FeatureSearchParserT__6:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(88)
			p.Group_term()
		}

	case FeatureSearchParserT__8:
		p.EnterOuterAlt(localctx, 6)
		{
			p.SetState(89)
			p.Id_term()
		}

	case FeatureSearchParserT__7:
		p.EnterOuterAlt(localctx, 7)
		{
			p.SetState(90)
			p.Snapshot_term()
		}

	case FeatureSearchParserT__5:
		p.EnterOuterAlt(localctx, 8)
		{
			p.SetState(91)
			p.Description_term()
		}

	case FeatureSearchParserT__9:
		p.EnterOuterAlt(localctx, 9)
		{
			p.SetState(92)
			p.Saved_term()
		}

	case FeatureSearchParserT__10:
		p.EnterOuterAlt(localctx, 10)
		{
			p.SetState(93)
			p.Hotlist_term()
		}

	case FeatureSearchParserT__4:
		p.EnterOuterAlt(localctx, 11)
		{
			p.SetState(94)
			p.Name_term()
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IDate_range_queryContext is an interface to support dynamic dispatch.
type IDate_range_queryContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// GetStartDate returns the startDate token.
	GetStartDate() antlr.Token

	// GetEndDate returns the endDate token.
	GetEndDate() antlr.Token

	// SetStartDate sets the startDate token.
	SetStartDate(antlr.Token)

	// SetEndDate sets the endDate token.
	SetEndDate(antlr.Token)

	// Getter signatures
	AllDATE() []antlr.TerminalNode
	DATE(i int) antlr.TerminalNode

	// IsDate_range_queryContext differentiates from other interfaces.
	IsDate_range_queryContext()
}

type Date_range_queryContext struct {
	antlr.BaseParserRuleContext
	parser    antlr.Parser
	startDate antlr.Token
	endDate   antlr.Token
}

func NewEmptyDate_range_queryContext() *Date_range_queryContext {
	var p = new(Date_range_queryContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_date_range_query
	return p
}

func InitEmptyDate_range_queryContext(p *Date_range_queryContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_date_range_query
}

func (*Date_range_queryContext) IsDate_range_queryContext() {}

func NewDate_range_queryContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Date_range_queryContext {
	var p = new(Date_range_queryContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = FeatureSearchParserRULE_date_range_query

	return p
}

func (s *Date_range_queryContext) GetParser() antlr.Parser { return s.parser }

func (s *Date_range_queryContext) GetStartDate() antlr.Token { return s.startDate }

func (s *Date_range_queryContext) GetEndDate() antlr.Token { return s.endDate }

func (s *Date_range_queryContext) SetStartDate(v antlr.Token) { s.startDate = v }

func (s *Date_range_queryContext) SetEndDate(v antlr.Token) { s.endDate = v }

func (s *Date_range_queryContext) AllDATE() []antlr.TerminalNode {
	return s.GetTokens(FeatureSearchParserDATE)
}

func (s *Date_range_queryContext) DATE(i int) antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserDATE, i)
}

func (s *Date_range_queryContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Date_range_queryContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Date_range_queryContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case FeatureSearchVisitor:
		return t.VisitDate_range_query(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *FeatureSearchParser) Date_range_query() (localctx IDate_range_queryContext) {
	localctx = NewDate_range_queryContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 24, FeatureSearchParserRULE_date_range_query)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(97)

		var _m = p.Match(FeatureSearchParserDATE)

		localctx.(*Date_range_queryContext).startDate = _m
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(98)
		p.Match(FeatureSearchParserT__11)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(99)

		var _m = p.Match(FeatureSearchParserDATE)

		localctx.(*Date_range_queryContext).endDate = _m
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IGeneric_search_termContext is an interface to support dynamic dispatch.
type IGeneric_search_termContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Term() ITermContext
	NOT() antlr.TerminalNode

	// IsGeneric_search_termContext differentiates from other interfaces.
	IsGeneric_search_termContext()
}

type Generic_search_termContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyGeneric_search_termContext() *Generic_search_termContext {
	var p = new(Generic_search_termContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_generic_search_term
	return p
}

func InitEmptyGeneric_search_termContext(p *Generic_search_termContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_generic_search_term
}

func (*Generic_search_termContext) IsGeneric_search_termContext() {}

func NewGeneric_search_termContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Generic_search_termContext {
	var p = new(Generic_search_termContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = FeatureSearchParserRULE_generic_search_term

	return p
}

func (s *Generic_search_termContext) GetParser() antlr.Parser { return s.parser }

func (s *Generic_search_termContext) Term() ITermContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITermContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITermContext)
}

func (s *Generic_search_termContext) NOT() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserNOT, 0)
}

func (s *Generic_search_termContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Generic_search_termContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Generic_search_termContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case FeatureSearchVisitor:
		return t.VisitGeneric_search_term(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *FeatureSearchParser) Generic_search_term() (localctx IGeneric_search_termContext) {
	localctx = NewGeneric_search_termContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 26, FeatureSearchParserRULE_generic_search_term)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(102)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == FeatureSearchParserNOT {
		{
			p.SetState(101)
			p.Match(FeatureSearchParserNOT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	{
		p.SetState(104)
		p.Term()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ISearch_criteriaContext is an interface to support dynamic dispatch.
type ISearch_criteriaContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Generic_search_term() IGeneric_search_termContext
	ANY_VALUE() antlr.TerminalNode

	// IsSearch_criteriaContext differentiates from other interfaces.
	IsSearch_criteriaContext()
}

type Search_criteriaContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySearch_criteriaContext() *Search_criteriaContext {
	var p = new(Search_criteriaContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_search_criteria
	return p
}

func InitEmptySearch_criteriaContext(p *Search_criteriaContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_search_criteria
}

func (*Search_criteriaContext) IsSearch_criteriaContext() {}

func NewSearch_criteriaContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Search_criteriaContext {
	var p = new(Search_criteriaContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = FeatureSearchParserRULE_search_criteria

	return p
}

func (s *Search_criteriaContext) GetParser() antlr.Parser { return s.parser }

func (s *Search_criteriaContext) Generic_search_term() IGeneric_search_termContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IGeneric_search_termContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IGeneric_search_termContext)
}

func (s *Search_criteriaContext) ANY_VALUE() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserANY_VALUE, 0)
}

func (s *Search_criteriaContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Search_criteriaContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Search_criteriaContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case FeatureSearchVisitor:
		return t.VisitSearch_criteria(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *FeatureSearchParser) Search_criteria() (localctx ISearch_criteriaContext) {
	localctx = NewSearch_criteriaContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 28, FeatureSearchParserRULE_search_criteria)
	p.SetState(108)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case FeatureSearchParserT__0, FeatureSearchParserT__1, FeatureSearchParserT__2, FeatureSearchParserT__3, FeatureSearchParserT__4, FeatureSearchParserT__5, FeatureSearchParserT__6, FeatureSearchParserT__7, FeatureSearchParserT__8, FeatureSearchParserT__9, FeatureSearchParserT__10, FeatureSearchParserNOT:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(106)
			p.Generic_search_term()
		}

	case FeatureSearchParserANY_VALUE:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(107)
			p.Match(FeatureSearchParserANY_VALUE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IParenthesizedCriteriaContext is an interface to support dynamic dispatch.
type IParenthesizedCriteriaContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Combined_search_criteria() ICombined_search_criteriaContext

	// IsParenthesizedCriteriaContext differentiates from other interfaces.
	IsParenthesizedCriteriaContext()
}

type ParenthesizedCriteriaContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyParenthesizedCriteriaContext() *ParenthesizedCriteriaContext {
	var p = new(ParenthesizedCriteriaContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_parenthesizedCriteria
	return p
}

func InitEmptyParenthesizedCriteriaContext(p *ParenthesizedCriteriaContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_parenthesizedCriteria
}

func (*ParenthesizedCriteriaContext) IsParenthesizedCriteriaContext() {}

func NewParenthesizedCriteriaContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ParenthesizedCriteriaContext {
	var p = new(ParenthesizedCriteriaContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = FeatureSearchParserRULE_parenthesizedCriteria

	return p
}

func (s *ParenthesizedCriteriaContext) GetParser() antlr.Parser { return s.parser }

func (s *ParenthesizedCriteriaContext) Combined_search_criteria() ICombined_search_criteriaContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICombined_search_criteriaContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICombined_search_criteriaContext)
}

func (s *ParenthesizedCriteriaContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ParenthesizedCriteriaContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ParenthesizedCriteriaContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case FeatureSearchVisitor:
		return t.VisitParenthesizedCriteria(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *FeatureSearchParser) ParenthesizedCriteria() (localctx IParenthesizedCriteriaContext) {
	localctx = NewParenthesizedCriteriaContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 30, FeatureSearchParserRULE_parenthesizedCriteria)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(110)
		p.Match(FeatureSearchParserT__12)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(111)
		p.Combined_search_criteria()
	}
	{
		p.SetState(112)
		p.Match(FeatureSearchParserT__13)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ICombined_search_criteriaContext is an interface to support dynamic dispatch.
type ICombined_search_criteriaContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllSearch_criteria() []ISearch_criteriaContext
	Search_criteria(i int) ISearch_criteriaContext
	AllParenthesizedCriteria() []IParenthesizedCriteriaContext
	ParenthesizedCriteria(i int) IParenthesizedCriteriaContext
	AllOperator() []IOperatorContext
	Operator(i int) IOperatorContext

	// IsCombined_search_criteriaContext differentiates from other interfaces.
	IsCombined_search_criteriaContext()
}

type Combined_search_criteriaContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyCombined_search_criteriaContext() *Combined_search_criteriaContext {
	var p = new(Combined_search_criteriaContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_combined_search_criteria
	return p
}

func InitEmptyCombined_search_criteriaContext(p *Combined_search_criteriaContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_combined_search_criteria
}

func (*Combined_search_criteriaContext) IsCombined_search_criteriaContext() {}

func NewCombined_search_criteriaContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Combined_search_criteriaContext {
	var p = new(Combined_search_criteriaContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = FeatureSearchParserRULE_combined_search_criteria

	return p
}

func (s *Combined_search_criteriaContext) GetParser() antlr.Parser { return s.parser }

func (s *Combined_search_criteriaContext) AllSearch_criteria() []ISearch_criteriaContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ISearch_criteriaContext); ok {
			len++
		}
	}

	tst := make([]ISearch_criteriaContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ISearch_criteriaContext); ok {
			tst[i] = t.(ISearch_criteriaContext)
			i++
		}
	}

	return tst
}

func (s *Combined_search_criteriaContext) Search_criteria(i int) ISearch_criteriaContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISearch_criteriaContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISearch_criteriaContext)
}

func (s *Combined_search_criteriaContext) AllParenthesizedCriteria() []IParenthesizedCriteriaContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IParenthesizedCriteriaContext); ok {
			len++
		}
	}

	tst := make([]IParenthesizedCriteriaContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IParenthesizedCriteriaContext); ok {
			tst[i] = t.(IParenthesizedCriteriaContext)
			i++
		}
	}

	return tst
}

func (s *Combined_search_criteriaContext) ParenthesizedCriteria(i int) IParenthesizedCriteriaContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IParenthesizedCriteriaContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IParenthesizedCriteriaContext)
}

func (s *Combined_search_criteriaContext) AllOperator() []IOperatorContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IOperatorContext); ok {
			len++
		}
	}

	tst := make([]IOperatorContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IOperatorContext); ok {
			tst[i] = t.(IOperatorContext)
			i++
		}
	}

	return tst
}

func (s *Combined_search_criteriaContext) Operator(i int) IOperatorContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IOperatorContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IOperatorContext)
}

func (s *Combined_search_criteriaContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Combined_search_criteriaContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Combined_search_criteriaContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case FeatureSearchVisitor:
		return t.VisitCombined_search_criteria(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *FeatureSearchParser) Combined_search_criteria() (localctx ICombined_search_criteriaContext) {
	localctx = NewCombined_search_criteriaContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 32, FeatureSearchParserRULE_combined_search_criteria)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(116)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case FeatureSearchParserT__0, FeatureSearchParserT__1, FeatureSearchParserT__2, FeatureSearchParserT__3, FeatureSearchParserT__4, FeatureSearchParserT__5, FeatureSearchParserT__6, FeatureSearchParserT__7, FeatureSearchParserT__8, FeatureSearchParserT__9, FeatureSearchParserT__10, FeatureSearchParserNOT, FeatureSearchParserANY_VALUE:
		{
			p.SetState(114)
			p.Search_criteria()
		}

	case FeatureSearchParserT__12:
		{
			p.SetState(115)
			p.ParenthesizedCriteria()
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}
	p.SetState(127)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&8630270) != 0 {
		p.SetState(119)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == FeatureSearchParserAND || _la == FeatureSearchParserOR {
			{
				p.SetState(118)
				p.Operator()
			}

		}
		p.SetState(123)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case FeatureSearchParserT__0, FeatureSearchParserT__1, FeatureSearchParserT__2, FeatureSearchParserT__3, FeatureSearchParserT__4, FeatureSearchParserT__5, FeatureSearchParserT__6, FeatureSearchParserT__7, FeatureSearchParserT__8, FeatureSearchParserT__9, FeatureSearchParserT__10, FeatureSearchParserNOT, FeatureSearchParserANY_VALUE:
			{
				p.SetState(121)
				p.Search_criteria()
			}

		case FeatureSearchParserT__12:
			{
				p.SetState(122)
				p.ParenthesizedCriteria()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}

		p.SetState(129)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IOperatorContext is an interface to support dynamic dispatch.
type IOperatorContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AND() antlr.TerminalNode
	OR() antlr.TerminalNode

	// IsOperatorContext differentiates from other interfaces.
	IsOperatorContext()
}

type OperatorContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyOperatorContext() *OperatorContext {
	var p = new(OperatorContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_operator
	return p
}

func InitEmptyOperatorContext(p *OperatorContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_operator
}

func (*OperatorContext) IsOperatorContext() {}

func NewOperatorContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *OperatorContext {
	var p = new(OperatorContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = FeatureSearchParserRULE_operator

	return p
}

func (s *OperatorContext) GetParser() antlr.Parser { return s.parser }

func (s *OperatorContext) AND() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserAND, 0)
}

func (s *OperatorContext) OR() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserOR, 0)
}

func (s *OperatorContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *OperatorContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *OperatorContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case FeatureSearchVisitor:
		return t.VisitOperator(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *FeatureSearchParser) Operator() (localctx IOperatorContext) {
	localctx = NewOperatorContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 34, FeatureSearchParserRULE_operator)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(130)
		_la = p.GetTokenStream().LA(1)

		if !(_la == FeatureSearchParserAND || _la == FeatureSearchParserOR) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IQueryContext is an interface to support dynamic dispatch.
type IQueryContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Combined_search_criteria() ICombined_search_criteriaContext
	EOF() antlr.TerminalNode

	// IsQueryContext differentiates from other interfaces.
	IsQueryContext()
}

type QueryContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyQueryContext() *QueryContext {
	var p = new(QueryContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_query
	return p
}

func InitEmptyQueryContext(p *QueryContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = FeatureSearchParserRULE_query
}

func (*QueryContext) IsQueryContext() {}

func NewQueryContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *QueryContext {
	var p = new(QueryContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = FeatureSearchParserRULE_query

	return p
}

func (s *QueryContext) GetParser() antlr.Parser { return s.parser }

func (s *QueryContext) Combined_search_criteria() ICombined_search_criteriaContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICombined_search_criteriaContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICombined_search_criteriaContext)
}

func (s *QueryContext) EOF() antlr.TerminalNode {
	return s.GetToken(FeatureSearchParserEOF, 0)
}

func (s *QueryContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *QueryContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *QueryContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case FeatureSearchVisitor:
		return t.VisitQuery(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *FeatureSearchParser) Query() (localctx IQueryContext) {
	localctx = NewQueryContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 36, FeatureSearchParserRULE_query)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(132)
		p.Combined_search_criteria()
	}
	{
		p.SetState(133)
		p.Match(FeatureSearchParserEOF)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}
