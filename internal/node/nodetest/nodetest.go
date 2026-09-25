// Copyright (c) 2026, Peter Ohler, All rights reserved.

// Package nodetest provides the shared contract fixtures used by the
// tests of every entry point migrated to the internal/node structural
// access kernel. Each migrated entry (jp.Walk, alt.Checksum, alt.Diff and
// family, alt.Decompose and alt.Alter) runs the same fixtures and asserts
// its entry specific view of the shared classification.
package nodetest

import (
	"time"

	"github.com/ohler55/ojg/gen"
	"github.com/ohler55/ojg/internal/node"
)

// Fixture is a single shared contract case.
type Fixture struct {
	// Name identifies the fixture in test failure output.
	Name string
	// Value is the fixture value.
	Value any
	// Kind is the kernel classification the value must receive.
	Kind node.Kind
}

// Simplifier is a fixture type with a Simplify method.
type Simplifier struct {
	X int
}

// Simplify returns a simple representation of the fixture.
func (s Simplifier) Simplify() any {
	return map[string]any{"x": int64(s.X)}
}

// Text is a named string fixture type.
type Text string

// Point is a struct fixture type.
type Point struct {
	X int
	Y int
}

// Fixtures returns the shared contract fixtures. The fixtures cover null,
// every scalar width, the container types, gen types, simplifiers, and
// values that only reflection can represent.
func Fixtures() []Fixture {
	var nilPtr *int
	return []Fixture{
		{Name: "null", Value: nil, Kind: node.Null},
		{Name: "bool", Value: true, Kind: node.Bool},
		{Name: "int", Value: int(3), Kind: node.Int},
		{Name: "int8", Value: int8(3), Kind: node.Int},
		{Name: "int16", Value: int16(3), Kind: node.Int},
		{Name: "int32", Value: int32(3), Kind: node.Int},
		{Name: "int64", Value: int64(3), Kind: node.Int},
		{Name: "uint", Value: uint(3), Kind: node.Int},
		{Name: "uint8", Value: uint8(3), Kind: node.Int},
		{Name: "uint16", Value: uint16(3), Kind: node.Int},
		{Name: "uint32", Value: uint32(3), Kind: node.Int},
		{Name: "uint64", Value: uint64(3), Kind: node.Int},
		{Name: "float32", Value: float32(1.5), Kind: node.Float},
		{Name: "float64", Value: float64(1.5), Kind: node.Float},
		{Name: "string", Value: "x", Kind: node.String},
		{Name: "bytes", Value: []byte("x"), Kind: node.Bytes},
		{Name: "time", Value: time.Unix(0, 0).UTC(), Kind: node.Time},
		{Name: "array", Value: []any{1, "a"}, Kind: node.Array},
		{Name: "nil array", Value: []any(nil), Kind: node.Array},
		{Name: "object", Value: map[string]any{"a": 1}, Kind: node.Object},
		{Name: "nil object", Value: map[string]any(nil), Kind: node.Object},
		{Name: "gen array", Value: gen.Array{gen.Int(1)}, Kind: node.Array},
		{Name: "gen object", Value: gen.Object{"a": gen.Int(1)}, Kind: node.Object},
		{Name: "gen int", Value: gen.Int(3), Kind: node.Simplify},
		{Name: "gen string", Value: gen.String("x"), Kind: node.Simplify},
		{Name: "simplifier", Value: Simplifier{X: 7}, Kind: node.Simplify},
		{Name: "typed nil pointer", Value: nilPtr, Kind: node.Other},
		{Name: "named string", Value: Text("x"), Kind: node.Other},
		{Name: "non-string key map", Value: map[int]int{1: 2}, Kind: node.Other},
		{Name: "struct", Value: Point{X: 1, Y: 2}, Kind: node.Other},
	}
}
