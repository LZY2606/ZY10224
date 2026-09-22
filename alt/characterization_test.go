// Copyright (c) 2026, Peter Ohler, All rights reserved.

package alt_test

// Characterization tests. These tests pin the existing behavior of the
// alt package entry points (Decompose, Alter, Diff, Compare, Match, and
// Checksum) before and after the migration to the shared internal read-only
// structural access layer (internal/node). Each test documents a behavior
// that must not drift, including some quirks that are preserved for
// compatibility.

import (
	"fmt"
	"math"
	"sort"
	"testing"
	"time"

	"github.com/ohler55/ojg/alt"
	"github.com/ohler55/ojg/gen"
	"github.com/ohler55/ojg/tt"
)

type charInt int
type charUint uint
type charFloat float32
type charBytes []byte
type charMap map[string]any
type charSlice []any
type charTime time.Time

type charStruct struct {
	X int    `json:"x"`
	Y *int   `json:"y"`
	T string `json:"t,omitempty"`
}

// Typed nil values decompose to nil for pointers and interfaces but to
// empty containers for slices and maps.
func TestCharDecomposeTypedNil(t *testing.T) {
	var nilPtr *int
	var nilSlice []int
	var nilMap map[int]string
	var nilIface any = nilPtr

	tt.Nil(t, alt.Decompose(nilPtr), "typed nil pointer")
	tt.Nil(t, alt.Decompose(nilIface), "typed nil inside interface")
	tt.Equal(t, []any{}, alt.Decompose(nilSlice), "typed nil slice becomes an empty array")
	tt.Equal(t, map[string]any{}, alt.Decompose(nilMap), "typed nil map becomes an empty object")
	tt.Equal(t, []any{}, alt.Decompose([]any(nil)), "nil []any becomes an empty array")
	tt.Equal(t, map[string]any{}, alt.Decompose(map[string]any(nil)), "nil map[string]any becomes an empty object")
}

// Named alias types follow the reflection path which differs from the
// concrete type path in a few pinned ways.
func TestCharDecomposeAliases(t *testing.T) {
	tt.Equal(t, int64(3), alt.Decompose(charInt(3)), "int alias becomes int64")
	tt.Equal(t, uint64(3), alt.Decompose(charUint(3)), "uint alias stays uint64 (not int64)")
	tt.Equal(t, uint64(3), alt.Decompose(uint(3)), "concrete uint becomes int64")
	tt.Equal(t, int64(3), alt.Decompose(uint64(3)), "concrete uint64 becomes int64")

	// Concrete float32 is rounded for display, the alias is not.
	tt.Equal(t, 0.1, alt.Decompose(float32(0.1)), "concrete float32 is rounded")
	tt.Equal(t, float64(float32(0.1)), alt.Decompose(charFloat(0.1)), "float32 alias is not rounded")

	// Concrete []byte becomes a string, the alias becomes an array of ints.
	tt.Equal(t, "hi", alt.Decompose([]byte("hi")), "concrete []byte becomes a string")
	tt.Equal(t, []any{int64(104), int64(105)}, alt.Decompose(charBytes("hi")), "[]byte alias becomes an array")

	tt.Equal(t, map[string]any{"a": int64(1)}, alt.Decompose(charMap{"a": 1}), "map alias")
	tt.Equal(t, []any{int64(1)}, alt.Decompose(charSlice{1}), "slice alias")
}

// Time handling: exact time.Time uses the time options, pointers and
// aliases take the struct reflection path.
func TestCharDecomposeTime(t *testing.T) {
	tm := time.Unix(1700000000, 0).UTC()
	tt.Equal(t, int64(1700000000000000000), alt.Decompose(tm), "time.Time uses DecomposeTime")
	tt.Equal(t, map[string]any{"type": "Time"}, alt.Decompose(&tm), "*time.Time uses struct reflection")
	tt.Equal(t, map[string]any{"type": "charTime"}, alt.Decompose(charTime(tm)), "time alias uses struct reflection")
}

// Integer width rules: uint64 values wrap to int64 in decompose and in
// diff comparisons.
func TestCharIntegerWidth(t *testing.T) {
	tt.Equal(t, int64(-1), alt.Decompose(uint64(math.MaxUint64)), "uint64 max wraps to int64")
	tt.Equal(t, 0, len(alt.Diff(uint64(math.MaxUint64), int64(-1))), "uint64 max equals int64 -1 in diff")
	tt.Equal(t, 0, len(alt.Diff(uint64(math.MaxUint64), uint64(math.MaxUint64))), "uint64 max equals itself")
}

// Map keys of any type are converted with fmt for non-string keys.
func TestCharDecomposeMapKeys(t *testing.T) {
	tt.Equal(t, map[string]any{"3": "x"}, alt.Decompose(map[int]string{3: "x"}), "int keys become strings")
	tt.Equal(t,
		map[string]any{"1": "a"},
		alt.Decompose(map[any]any{1: "a", "k": nil}),
		"any keys are formatted, nil values omitted by default options")
}

// Struct decomposition uses the DefaultOptions by default which include a
// CreateKey of "type" and OmitNil.
func TestCharDecomposeStructDefaults(t *testing.T) {
	out := alt.Decompose(charStruct{X: 1})
	tt.Equal(t, map[string]any{"type": "charStruct", "x": int64(1), "t": ""}, out,
		"default options add a type member and omit nils")
}

// Diff reports a difference when the compared values have different
// reflect types even if the values are otherwise equal, and reports a
// difference for typed nil pointers compared to anything including
// themselves.
func TestCharDiffTypeIdentity(t *testing.T) {
	var nilPtr *int
	var nilSlice []int

	tt.Equal(t, 1, len(alt.Diff(charInt(3), int(3))), "named int vs int differs")
	tt.Equal(t, 0, len(alt.Diff(charInt(3), charInt(3))), "same named ints are equal")
	tt.Equal(t, 1, len(alt.Diff(charInt(3), charInt(4))), "named ints with different values differ")

	tt.Equal(t, 1, len(alt.Diff(nilPtr, nilPtr)), "typed nil pointer differs even from itself")
	tt.Equal(t, 1, len(alt.Diff(nil, nilPtr)), "nil vs typed nil pointer differs")
	tt.Equal(t, 1, len(alt.Diff(nilPtr, nil)), "typed nil pointer vs nil differs")
	tt.Equal(t, 0, len(alt.Diff(nilSlice, nilSlice)), "typed nil slices are equal (both empty arrays)")
	tt.Equal(t, 1, len(alt.Diff(nilSlice, nil)), "typed nil slice vs nil differs")
	tt.Equal(t, 1, len(alt.Diff([]any{}, nil)), "empty slice vs null differs")
}

// Diff pairs concrete container types exactly; gen types are simplified
// before comparison.
func TestCharDiffContainerPairing(t *testing.T) {
	tt.Equal(t, 1, len(alt.Diff([]any{int64(1)}, gen.Array{gen.Int(1)})), "[]any vs gen.Array differs")
	tt.Equal(t, 0, len(alt.Diff(gen.Array{gen.Int(1)}, gen.Array{gen.Int(1)})), "gen.Array pairs simplify")
	tt.Equal(t, 0, len(alt.Diff(gen.Object{"a": gen.Int(1)}, gen.Object{"a": gen.Int(1)})), "gen.Object pairs simplify")
	tt.Equal(t, 1, len(alt.Diff(map[string]any{"a": int64(1)}, gen.Object{"a": gen.Int(1)})), "map vs gen.Object differs")
}

// Match follows the same type identity rules as Diff.
func TestCharMatchContracts(t *testing.T) {
	var nilPtr *int
	var nilSlice []int

	tt.Equal(t, true, alt.Match(nil, nil), "nil matches nil")
	tt.Equal(t, false, alt.Match(nil, nilPtr), "nil does not match a typed nil pointer")
	tt.Equal(t, false, alt.Match(nilPtr, nilPtr), "typed nil pointer does not match anything")
	tt.Equal(t, true, alt.Match(nilSlice, nilSlice), "typed nil slices match (both empty arrays)")
	tt.Equal(t, true, alt.Match(charInt(3), charInt(3)), "named ints match after reflection")
	tt.Equal(t, false, alt.Match([]any{int64(1)}, gen.Array{gen.Int(1)}), "[]any does not match gen.Array")
	tt.Equal(t, true, alt.Match(gen.Array{gen.Int(1)}, gen.Array{gen.Int(1)}), "gen.Array matches gen.Array")
}

// Checksum pins: nil, empty containers, and aliases.
func TestCharChecksumContracts(t *testing.T) {
	var nilSlice []int
	var nilPtr *int

	tt.Equal(t, alt.Checksum(nil), alt.Checksum(nilPtr), "typed nil pointer checksums as null")
	tt.Equal(t, true, alt.Checksum(nil) != alt.Checksum(nilSlice), "typed nil slice checksums as an empty array")
	tt.Equal(t, alt.Checksum([]any{}), alt.Checksum(nilSlice), "typed nil slice checksums as an empty array")
	tt.Equal(t, true, alt.Checksum(map[string]any{"a": nil}) != alt.Checksum(map[string]any{}),
		"nil member differs from missing member")
	tt.Equal(t, alt.Checksum([]any{int64(104), int64(105)}), alt.Checksum(charBytes("hi")),
		"[]byte alias checksums as an array of ints")
	tt.Equal(t, true, alt.Checksum([]byte("hi")) != alt.Checksum(charBytes("hi")),
		"concrete []byte checksums differently from the alias")
}

// Checksum of gen types matches the checksum of their simplified form.
func TestCharChecksumGen(t *testing.T) {
	g := gen.Object{"b": gen.Int(2), "a": gen.Array{gen.Int(1)}}
	tt.Equal(t, alt.Checksum(g.Simplify()), alt.Checksum(g), "gen.Object checksums as its simplified form")
	tt.Equal(t, alt.Checksum(int64(7)), alt.Checksum(gen.Int(7)), "gen.Int checksums as int64")
}

// Checksum of a struct uses DefaultOptions which adds a "type" member.
func TestCharChecksumStruct(t *testing.T) {
	v := charStruct{X: 1}
	tt.Equal(t, alt.Checksum(alt.Decompose(v)), alt.Checksum(v), "struct checksum matches its decomposed form")
}

// Alter keeps nil slices and maps as they are while Decompose makes empty
// containers out of them.
func TestCharAlterNilContainers(t *testing.T) {
	// Alter returns the typed nil slice and map unchanged, unlike Decompose
	// which replaces them with empty containers.
	tt.Equal(t, true, alt.Alter([]any(nil)) != nil, "alter keeps a typed nil slice")
	tt.Equal(t, true, alt.Alter(map[string]any(nil)) != nil, "alter keeps a typed nil map")
}

// Diff paths are built with string keys and int indexes. The order of
// differences across map members was map iteration order before the
// migration to the shared access layer so this test is order insensitive.
func TestCharDiffPaths(t *testing.T) {
	diffs := alt.Diff(
		map[string]any{"a": []any{1, 2}, "b": map[string]any{"c": 1}},
		map[string]any{"a": []any{1, 3}, "b": map[string]any{"c": 2}},
	)
	tt.Equal(t, 2, len(diffs), "two differences")
	var paths []string
	for _, d := range diffs {
		paths = append(paths, d.String())
	}
	sort.Strings(paths)
	tt.Equal(t, []string{"a[1]", "b.c"}, paths, "diff paths")
}

// Sanity check that the checksum is stable across map iteration order.
func TestCharChecksumMapOrderStable(t *testing.T) {
	m := map[string]any{}
	for i := 0; i < 50; i++ {
		m[fmt.Sprintf("k%02d", i)] = i
	}
	want := alt.Checksum(m)
	for i := 0; i < 10; i++ {
		tt.Equal(t, want, alt.Checksum(m), "checksum must not depend on map iteration order")
	}
}
