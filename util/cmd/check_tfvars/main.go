// Copyright 2026 Google LLC
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

// Package main implements a Terraform .tfvars validator.
//
// WHY THIS EXISTS:
// Standard `terraform validate` has a known gap: it performs syntactic and internal consistency checks
// on .tf files, but it does NOT validate that root module variables provided in .tfvars files
// actually match the type constraints (e.g., object({ ... })) defined in variables.tf.
//
// This gap often leads to deployment failures where a variable is missing a required field,
// but `terraform validate` passes because it treats root variables as "unknown" during validation.
//
// This tool provides a deterministic, offline, and non-authenticated check to ensure all
// .tfvars files are fully compliant with the variable definitions.
package main

import (
	"flag"
	"fmt"
	"log"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// TypeSchema defines the expected structure of a Terraform variable type.
type TypeSchema struct {
	Kind      string                 // "object", "map", "list", "primitive"
	Fields    map[string]*TypeSchema // for "object"
	ValueType *TypeSchema            // for "map" and "list"
}

// VarSchema defines the schema for a single Terraform variable.
type VarSchema struct {
	Name        string
	Type        *TypeSchema
	IsMandatory bool
}

func main() {
	infraDir := flag.String("dir", "infra", "Path to the infra directory")
	varsPaths := flag.String("vars", "", "Comma-separated paths to variable files (for coverage check)")
	backendVarsPaths := flag.String(
		"backend-vars",
		"",
		"Comma-separated paths to backend variable files (partial check)",
	)
	flag.Parse()

	schemas, err := parseVariables(*infraDir)
	if err != nil {
		log.Fatalf("failed to parse variables.tf: %v", err)
	}

	allAttrs := make(map[string]*hcl.Attribute)
	allFiles := append(splitComma(*varsPaths), splitComma(*backendVarsPaths)...)

	// 1. Gather all attributes from all provided files
	for _, path := range allFiles {
		fullPath := resolvePath(*infraDir, path)
		attrs, err := loadAttributes(fullPath)
		if err != nil {
			log.Fatalf("failed to load %s: %v", fullPath, err)
		}
		maps.Copy(allAttrs, attrs)
	}

	// 2. Validate all gathered attributes against schemas
	failed := false
	for varName, attr := range allAttrs {
		schema, ok := schemas[varName]
		if !ok {
			continue
		}
		val, diag := attr.Expr.Value(nil)
		if diag.HasErrors() {
			continue // Skip non-evaluable values
		}
		if err := validateValue(val, schema.Type, varName, attr.Range.Filename); err != nil {
			fmt.Printf("ERROR in %s: %v\n", attr.Range.Filename, err)
			failed = true
		}
	}

	// 3. Check for mandatory variable coverage across ALL provided files
	// (only if we have at least one file in -vars)
	if *varsPaths != "" {
		if err := checkMandatoryVars(allAttrs, schemas); err != nil {
			fmt.Printf("ERROR: %v\n", err)
			failed = true
		}
	}

	if failed {
		os.Exit(1)
	}
	fmt.Println("Validation successful.")
}

func splitComma(s string) []string {
	if s == "" {
		return nil
	}

	return strings.Split(s, ",")
}

func resolvePath(infraDir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(infraDir, path)
}

func loadAttributes(fullPath string) (map[string]*hcl.Attribute, error) {
	parser := hclparse.NewParser()
	f, diag := parser.ParseHCLFile(fullPath)
	if diag.HasErrors() {
		return nil, fmt.Errorf("failed to parse %s: %w", fullPath, diag)
	}
	attrs, diag := f.Body.JustAttributes()
	if diag.HasErrors() {
		return nil, fmt.Errorf("failed to get attributes from %s: %w", fullPath, diag)
	}

	return attrs, nil
}

func parseVariables(infraDir string) (map[string]*VarSchema, error) {
	varsFile := filepath.Join(infraDir, "variables.tf")
	src, err := os.ReadFile(varsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", varsFile, err)
	}

	//nolint:exhaustruct
	f, diag := hclsyntax.ParseConfig(src, varsFile, hcl.Pos{Line: 1, Column: 1})
	if diag.HasErrors() {
		return nil, fmt.Errorf("failed to parse %s: %w", varsFile, diag)
	}

	var body *hclsyntax.Body
	switch b := f.Body.(type) {
	case *hclsyntax.Body:
		body = b
	default:
		panic(fmt.Sprintf("unexpected body type %T", f.Body))
	}

	schemas := make(map[string]*VarSchema)
	for _, block := range body.Blocks {
		if block.Type != "variable" {
			continue
		}
		varName := block.Labels[0]
		//nolint:exhaustruct
		v := &VarSchema{
			Name:        varName,
			IsMandatory: true,
		}
		if typeAttr, ok := block.Body.Attributes["type"]; ok {
			v.Type = parseTypeExpr(typeAttr.Expr)
		}
		if _, ok := block.Body.Attributes["default"]; ok {
			v.IsMandatory = false
		}
		schemas[varName] = v
	}

	return schemas, nil
}

func checkMandatoryVars(attrs map[string]*hcl.Attribute, schemas map[string]*VarSchema) error {
	for varName, schema := range schemas {
		if !schema.IsMandatory || varName == "env_id" {
			continue
		}
		if _, ok := attrs[varName]; !ok {
			return fmt.Errorf("missing mandatory variable '%s' across provided files", varName)
		}
	}

	return nil
}

func parseTypeExpr(expr hcl.Expression) *TypeSchema {
	switch e := expr.(type) {
	case *hclsyntax.FunctionCallExpr:
		return parseFunctionCallType(e)
	case *hclsyntax.ScopeTraversalExpr:
		//nolint:exhaustruct
		return &TypeSchema{Kind: "primitive"}
	default:
		panic(fmt.Sprintf("unsupported type expression %T", expr))
	}
}

func parseFunctionCallType(e *hclsyntax.FunctionCallExpr) *TypeSchema {
	if len(e.Args) != 1 {
		return nil
	}
	switch e.Name {
	case "object":
		return parseObjectCons(e.Args[0])
	case "map":
		//nolint:exhaustruct
		return &TypeSchema{Kind: "map", ValueType: parseTypeExpr(e.Args[0])}
	case "list", "set":
		//nolint:exhaustruct
		return &TypeSchema{Kind: "list", ValueType: parseTypeExpr(e.Args[0])}
	}

	return nil
}

func parseObjectCons(expr hcl.Expression) *TypeSchema {
	var objCons *hclsyntax.ObjectConsExpr
	switch e := expr.(type) {
	case *hclsyntax.ObjectConsExpr:
		objCons = e
	default:
		panic(fmt.Sprintf("expected ObjectConsExpr, got %T", expr))
	}

	fields := make(map[string]*TypeSchema)
	for _, item := range objCons.Items {
		name := extractKey(item.KeyExpr)
		if name != "" {
			fields[name] = parseTypeExpr(item.ValueExpr)
		}
	}
	//nolint:exhaustruct
	return &TypeSchema{Kind: "object", Fields: fields}
}

func extractKey(expr hcl.Expression) string {
	switch k := expr.(type) {
	case *hclsyntax.ObjectConsKeyExpr:
		return extractKey(k.Wrapped)
	case *hclsyntax.ScopeTraversalExpr:
		return k.Traversal.RootName()
	case *hclsyntax.LiteralValueExpr:
		if k.Val.Type() == cty.String {
			return k.Val.AsString()
		}
		panic(fmt.Sprintf("unsupported literal type for key: %s", k.Val.Type().FriendlyName()))
	default:
		panic(fmt.Sprintf("unsupported key expression %T", expr))
	}
}

func validateValue(val cty.Value, schema *TypeSchema, path, filename string) error {
	if schema == nil {
		return nil
	}
	switch schema.Kind {
	case "object":
		return validateObjectValue(val, schema, path, filename)
	case "map":
		return validateMapValue(val, schema, path, filename)
	case "list":
		return validateListValue(val, schema, path, filename)
	case "primitive":
		return nil
	default:
		panic(fmt.Sprintf("unsupported schema kind %s", schema.Kind))
	}
}

func validateObjectValue(val cty.Value, schema *TypeSchema, path, filename string) error {
	if !val.Type().IsObjectType() && !val.Type().IsMapType() {
		return fmt.Errorf("variable '%s' in %s: expected object/map, got %s", path, filename, val.Type().FriendlyName())
	}
	actual := make(map[string]cty.Value)
	if val.Type().IsObjectType() {
		actual = val.AsValueMap()
	} else {
		for it := val.ElementIterator(); it.Next(); {
			k, v := it.Element()
			actual[k.AsString()] = v
		}
	}
	for fieldName, fieldSchema := range schema.Fields {
		v, ok := actual[fieldName]
		if !ok {
			return fmt.Errorf("variable '%s' in %s: missing field '%s'", path, filename, fieldName)
		}
		if err := validateValue(v, fieldSchema, path+"."+fieldName, filename); err != nil {
			return err
		}
	}

	return nil
}

func validateMapValue(val cty.Value, schema *TypeSchema, path, filename string) error {
	if !val.Type().IsMapType() && !val.Type().IsObjectType() {
		return fmt.Errorf("variable '%s' in %s: expected map/object, got %s", path, filename, val.Type().FriendlyName())
	}
	var it cty.ElementIterator
	if val.Type().IsMapType() {
		it = val.ElementIterator()
	} else {
		m := val.AsValueMap()
		val = cty.MapVal(m)
		it = val.ElementIterator()
	}
	for it.Next() {
		k, v := it.Element()
		if err := validateValue(v, schema.ValueType, path+"["+k.AsString()+"]", filename); err != nil {
			return err
		}
	}

	return nil
}

func validateListValue(val cty.Value, schema *TypeSchema, path, filename string) error {
	if !val.Type().IsListType() && !val.Type().IsSetType() && !val.Type().IsTupleType() {
		return fmt.Errorf(
			"variable '%s' in %s: expected list/set/tuple, got %s",
			path,
			filename,
			val.Type().FriendlyName(),
		)
	}
	i := 0
	for it := val.ElementIterator(); it.Next(); {
		_, v := it.Element()
		if err := validateValue(v, schema.ValueType, fmt.Sprintf("%s[%d]", path, i), filename); err != nil {
			return err
		}
		i++
	}

	return nil
}
