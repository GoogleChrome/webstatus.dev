// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package searchtypes

import (
	"fmt"

	parser "github.com/GoogleChrome/webstatus.dev/lib/gen/featuresearch/parser/antlr"
	"github.com/antlr4-go/antlr/v4"
)

type FeaturesSearchQueryParser struct{}

func (f FeaturesSearchQueryParser) Parse(in string) (*SearchNode, error) {
	is := antlr.NewInputStream(in)

	// Create the Lexer
	lexer := parser.NewFeatureSearchLexer(is)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	// Create the Parser
	p := parser.NewFeatureSearchParser(stream)
	p.RemoveErrorListeners()
	lexer.RemoveErrorListeners()

	p.BuildParseTrees = true
	p.AddErrorListener(antlr.NewDiagnosticErrorListener(true))

	visitor := FeaturesSearchVisitor{
		BaseFeatureSearchVisitor: parser.BaseFeatureSearchVisitor{
			BaseParseTreeVisitor: &antlr.BaseParseTreeVisitor{},
		},
		err: nil,
	}
	lexer.AddErrorListener(&visitor)
	p.AddErrorListener(&visitor)

	query := p.Query()

	ret := query.Accept((parser.FeatureSearchVisitor)(&visitor))
	if visitor.err != nil {
		return nil, visitor.err
	}
	if node, ok := ret.(*SearchNode); ok {
		return node, nil
	}

	return nil, fmt.Errorf("parse returned unexpected type")
}
