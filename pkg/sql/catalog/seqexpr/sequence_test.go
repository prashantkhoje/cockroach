// Copyright 2021 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package seqexpr_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/cockroachdb/cockroach/pkg/sql/catalog/seqexpr"
	"github.com/cockroachdb/cockroach/pkg/sql/parser"
	_ "github.com/cockroachdb/cockroach/pkg/sql/sem/builtins" // register all builtins in builtins:init() for seqexpr package
	"github.com/cockroachdb/cockroach/pkg/sql/sem/builtins/builtinsregistry"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroach/pkg/sql/types"
)

func TestGetSequenceFromFunc(t *testing.T) {
	testData := []struct {
		expr     string
		expected *seqexpr.SeqIdentifier
	}{
		{`nextval('seq')`, &seqexpr.SeqIdentifier{SeqName: "seq"}},
		{`nextval(123::REGCLASS)`, &seqexpr.SeqIdentifier{SeqID: 123}},
		{`nextval(123)`, &seqexpr.SeqIdentifier{SeqID: 123}},
		{`nextval(123::OID::REGCLASS)`, &seqexpr.SeqIdentifier{SeqID: 123}},
		{`nextval(123::OID)`, &seqexpr.SeqIdentifier{SeqID: 123}},
	}

	ctx := context.Background()
	for i, test := range testData {
		t.Run(fmt.Sprintf("%d %s", i, test.expr), func(t *testing.T) {
			parsedExpr, err := parser.ParseExpr(test.expr)
			if err != nil {
				t.Fatal(err)
			}
			semaCtx := tree.MakeSemaContext()
			typedExpr, err := tree.TypeCheck(ctx, parsedExpr, &semaCtx, types.Any)
			if err != nil {
				t.Fatal(err)
			}
			funcExpr, ok := typedExpr.(*tree.FuncExpr)
			if !ok {
				t.Fatal("Expr is not a FuncExpr")
			}
			identifier, err := seqexpr.GetSequenceFromFunc(funcExpr, builtinsregistry.GetBuiltinProperties)
			if err != nil {
				t.Fatal(err)
			}
			if identifier.IsByID() {
				if identifier.SeqID != test.expected.SeqID {
					t.Fatalf("expected %d, got %d", test.expected.SeqID, identifier.SeqID)
				}
			} else {
				if identifier.SeqName != test.expected.SeqName {
					t.Fatalf("expected %s, got %s", test.expected.SeqName, identifier.SeqName)
				}
			}
		})
	}
}

func TestGetUsedSequences(t *testing.T) {
	testData := []struct {
		expr     string
		expected []seqexpr.SeqIdentifier
	}{
		{`nextval('seq')`, []seqexpr.SeqIdentifier{
			{SeqName: "seq"},
		}},
		{`nextval(123::REGCLASS)`, []seqexpr.SeqIdentifier{
			{SeqID: 123},
		}},
		{`nextval(123::REGCLASS) + nextval('seq')`, []seqexpr.SeqIdentifier{
			{SeqID: 123},
			{SeqName: "seq"},
		}},
	}

	ctx := context.Background()
	for i, test := range testData {
		t.Run(fmt.Sprintf("%d %s", i, test.expr), func(t *testing.T) {
			parsedExpr, err := parser.ParseExpr(test.expr)
			if err != nil {
				t.Fatal(err)
			}
			semaCtx := tree.MakeSemaContext()
			typedExpr, err := tree.TypeCheck(ctx, parsedExpr, &semaCtx, types.Any)
			if err != nil {
				t.Fatal(err)
			}
			identifiers, err := seqexpr.GetUsedSequences(typedExpr, builtinsregistry.GetBuiltinProperties)
			if err != nil {
				t.Fatal(err)
			}

			if len(identifiers) != len(test.expected) {
				t.Fatalf("expected %d identifiers, got %d", len(test.expected), len(identifiers))
			}

			for i, identifier := range identifiers {
				if identifier.IsByID() {
					if identifier.SeqID != test.expected[i].SeqID {
						t.Fatalf("expected %d, got %d", test.expected[i].SeqID, identifier.SeqID)
					}
				} else {
					if identifier.SeqName != test.expected[i].SeqName {
						t.Fatalf("expected %s, got %s", test.expected[i].SeqName, identifier.SeqName)
					}
				}
			}
		})
	}
}

func TestReplaceSequenceNamesWithIDs(t *testing.T) {
	namesToID := map[string]int64{
		"seq": 123,
	}

	testData := []struct {
		expr     string
		expected string
	}{
		{`nextval('seq')`, `nextval(123:::REGCLASS)`},
		{`nextval('non_existent')`, `nextval('non_existent')`},
		{`nextval(123::REGCLASS)`, `nextval(123::REGCLASS)`},
		{`nextval(123)`, `nextval(123)`},
	}

	ctx := context.Background()
	for i, test := range testData {
		t.Run(fmt.Sprintf("%d %s", i, test.expr), func(t *testing.T) {
			parsedExpr, err := parser.ParseExpr(test.expr)
			if err != nil {
				t.Fatal(err)
			}
			semaCtx := tree.MakeSemaContext()
			typedExpr, err := tree.TypeCheck(ctx, parsedExpr, &semaCtx, types.Any)
			if err != nil {
				t.Fatal(err)
			}
			newExpr, err := seqexpr.ReplaceSequenceNamesWithIDs(typedExpr, namesToID, builtinsregistry.GetBuiltinProperties)
			if err != nil {
				t.Fatal(err)
			}
			if newExpr.String() != test.expected {
				t.Fatalf("expected %s, got %s", test.expected, newExpr.String())
			}
		})
	}
}
