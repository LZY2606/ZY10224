// Copyright (c) 2026, Peter Ohler, All rights reserved.

package alt_test

// Contract tests for the alt entry points migrated to the shared internal
// read-only structural access layer (internal/node). These tests pin the
// shared contract of the layer: cyclic references terminate with a defined
// result instead of a stack overflow, map iteration order does not leak
// into checksums or diffs, and null, typed nil, and empty container
// semantics are consistent across entries.

import (
	"testing"

	"github.com/ohler55/ojg"
	"github.com/ohler55/ojg/alt"
	"github.com/ohler55/ojg/tt"
)

type contractNode struct {
	Name string        `json:"name"`
	Next *contractNode `json:"next"`
}

var plainOptions = &ojg.Options{}

// A cyclic reference through a pointer decomposes to nil at the cycle
// point instead of overflowing the stack.
func TestContractDecomposeCyclePointer(t *testing.T) {
	n := &contractNode{Name: "a"}
	n.Next = n
	out := alt.Decompose(n, plainOptions)
	tt.Equal(t, map[string]any{"name": "a", "next": nil}, out,
		"cyclic pointer decomposes to nil at the cycle point")
}

// A cyclic map decomposes to nil at the cycle point.
func TestContractDecomposeCycleMap(t *testing.T) {
	m := map[string]any{"a": 1}
	m["self"] = m
	out := alt.Decompose(m, plainOptions)
	tt.Equal(t, map[string]any{"a": int64(1), "self": nil}, out,
		"cyclic map member decomposes to nil")
}

// A cyclic slice decomposes to nil at the cycle point.
func TestContractDecomposeCycleSlice(t *testing.T) {
	a := []any{1, 2}
	b := []any{a, a}
	b[0] = b
	out := alt.Decompose(b, plainOptions)
	tt.Equal(t, []any{nil, []any{int64(1), int64(2)}}, out,
		"cyclic slice element decomposes to nil")
}

// A cycle through a reflected map with non-string keys also terminates.
func TestContractDecomposeCycleReflectedMap(t *testing.T) {
	m := map[int]any{1: "x"}
	m[2] = m
	out := alt.Decompose(m, plainOptions)
	tt.Equal(t, map[string]any{"1": "x", "2": nil}, out,
		"cyclic reflected map member decomposes to nil")
}

// Checksum of cyclic data is defined and matches the checksum of the
// equivalent data with nil at the cycle point.
func TestContractChecksumCycle(t *testing.T) {
	m := map[string]any{"a": 1}
	m["self"] = m
	equiv := map[string]any{"a": 1, "self": nil}
	tt.Equal(t, alt.Checksum(equiv), alt.Checksum(m),
		"cyclic map checksums as if the cycle point were null")

	n := &contractNode{Name: "a"}
	n.Next = n
	tt.Equal(t, alt.Checksum(alt.Decompose(n)), alt.Checksum(n),
		"cyclic struct checksums as its decomposed form")
}

// Diff of two identically cyclic maps reports no difference; a cyclic map
// differs from a non-cyclic one.
func TestContractDiffCycle(t *testing.T) {
	m0 := map[string]any{"a": 1}
	m0["self"] = m0
	m1 := map[string]any{"a": 1}
	m1["self"] = m1

	tt.Equal(t, 0, len(alt.Diff(m0, m1)), "identically cyclic maps are equal")
	tt.Equal(t, 1, len(alt.Diff(m0, map[string]any{"a": 1, "self": 2})),
		"cyclic map differs from a non-cyclic map")
	tt.Equal(t, true, alt.Match(m0, m1), "cyclic maps match")
	tt.Equal(t, false, alt.Match(m0, map[string]any{"a": 1, "self": 2}),
		"cyclic map does not match a non-cyclic map")
}

// Differences across map members are reported in sorted key order so the
// result does not depend on map iteration order.
func TestContractDiffDeterministicOrder(t *testing.T) {
	d0 := map[string]any{"z": 1, "a": 2, "m": 3}
	d1 := map[string]any{"z": 9, "a": 9, "m": 9}
	var paths []string
	for _, d := range alt.Diff(d0, d1) {
		paths = append(paths, d.String())
	}
	tt.Equal(t, []string{"a", "m", "z"}, paths, "diffs are reported in sorted key order")
	for i := 0; i < 10; i++ {
		tt.Equal(t, "a", alt.Compare(d0, d1).String(), "compare returns the first sorted difference")
	}
}

// Null, typed nil, and empty container semantics are consistent across the
// migrated entries.
func TestContractNullSemantics(t *testing.T) {
	var nilPtr *int
	var nilSlice []int

	// A typed nil pointer is null for decompose and checksum but remains a
	// distinct leaf for diff.
	tt.Nil(t, alt.Decompose(nilPtr, plainOptions), "typed nil pointer decomposes to nil")
	tt.Equal(t, alt.Checksum(nil), alt.Checksum(nilPtr), "typed nil pointer checksums as null")
	tt.Equal(t, 1, len(alt.Diff(nil, nilPtr)), "nil and a typed nil pointer differ")

	// A typed nil slice is an empty array, not null.
	tt.Equal(t, []any{}, alt.Decompose(nilSlice, plainOptions), "typed nil slice decomposes to an empty array")
	tt.Equal(t, alt.Checksum([]any{}), alt.Checksum(nilSlice), "typed nil slice checksums as an empty array")
	tt.Equal(t, 1, len(alt.Diff(nil, nilSlice)), "null and a typed nil slice differ")
	tt.Equal(t, 1, len(alt.Diff([]any{}, nil)), "empty slice and null differ")
}

// Shared sub-values that are not cyclic are fully decomposed.
func TestContractSharedSubValues(t *testing.T) {
	shared := []any{1, 2}
	data := map[string]any{"a": shared, "b": shared}
	out := alt.Decompose(data, plainOptions)
	tt.Equal(t,
		map[string]any{"a": []any{int64(1), int64(2)}, "b": []any{int64(1), int64(2)}},
		out,
		"shared sub-values are not cycles")
}
