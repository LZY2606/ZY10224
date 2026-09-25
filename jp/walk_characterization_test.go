// Copyright (c) 2026, Peter Ohler, All rights reserved.

package jp_test

import (
	"sort"
	"testing"

	"github.com/ohler55/ojg/alt"
	"github.com/ohler55/ojg/gen"
	"github.com/ohler55/ojg/jp"
	"github.com/ohler55/ojg/tt"
)

// The tests in this file are characterization tests. They pin the existing
// externally visible behavior of jp.Walk before and after the migration to
// the shared internal structural access kernel (internal/node).

// walkPaths collects the string form of every path passed to the callback.
func walkPaths(data any, justLeaves ...bool) []string {
	var paths []string
	jp.Walk(data, func(path jp.Expr, value any) {
		paths = append(paths, path.String())
	}, justLeaves...)
	return paths
}

func sorted(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func TestWalkCharacterizeLeaves(t *testing.T) {
	t.Run("typed nil pointer is a leaf", func(t *testing.T) {
		var p *int
		var got []any
		jp.Walk(p, func(path jp.Expr, value any) { got = append(got, value) })
		tt.Equal(t, 1, len(got))
		if got[0] != p {
			t.Fatalf("expected the typed nil pointer itself as a leaf, got %#v", got[0])
		}
	})
	t.Run("non-string keyed map is a leaf", func(t *testing.T) {
		tt.Equal(t, []string{"$"}, walkPaths(map[int]int{1: 2}))
	})
	t.Run("unknown named type is a leaf", func(t *testing.T) {
		tt.Equal(t, []string{"$"}, walkPaths(other(5)))
	})
	t.Run("nil is a leaf", func(t *testing.T) {
		tt.Equal(t, []string{"$"}, walkPaths(nil))
	})
	t.Run("nil slice is an empty array node", func(t *testing.T) {
		tt.Equal(t, []string{"$"}, walkPaths([]any(nil)))
		tt.Equal(t, 0, len(walkPaths([]any(nil), true)),
			"a nil slice has no leaves")
	})
	t.Run("scalar kinds are leaves", func(t *testing.T) {
		for _, v := range []any{
			true, int(1), int8(1), int16(1), int32(1), int64(1),
			uint(1), uint8(1), uint16(1), uint32(1), uint64(1),
			float32(1), float64(1), "x", []byte("x"),
		} {
			tt.Equal(t, []string{"$"}, walkPaths(v), "value %#v", v)
		}
	})
}

func TestWalkCharacterizeContainers(t *testing.T) {
	t.Run("gen types are traversed", func(t *testing.T) {
		data := gen.Object{"a": gen.Array{gen.Int(1)}, "b": gen.Object{"c": gen.Int(2)}}
		tt.Equal(t,
			[]string{"$", "$.a", "$.a[0]", "$.b", "$.b.c"},
			sorted(walkPaths(data)))
	})
	t.Run("simplifier is unwrapped", func(t *testing.T) {
		tt.Equal(t,
			[]string{"$", "$.x"},
			sorted(walkPaths(simple(3))))
	})
	t.Run("mixed simple data", func(t *testing.T) {
		data := map[string]any{
			"a": []any{1, map[string]any{"z": true}},
			"b": nil,
		}
		tt.Equal(t,
			[]string{"$", "$.a", "$.a[0]", "$.a[1]", "$.a[1].z", "$.b"},
			sorted(walkPaths(data)))
	})
	t.Run("just leaves", func(t *testing.T) {
		data := map[string]any{"a": []any{1, 2}, "b": map[string]any{"c": 3}}
		tt.Equal(t,
			[]string{"$.a[0]", "$.a[1]", "$.b.c"},
			sorted(walkPaths(data, true)))
	})
	t.Run("array order is preserved", func(t *testing.T) {
		// Array elements must be visited in index order without sorting.
		var order []string
		jp.Walk([]any{[]any{1}, "x", []any{2}}, func(path jp.Expr, value any) {
			order = append(order, path.String())
		})
		tt.Equal(t, []string{"$", "$[0]", "$[0][0]", "$[1]", "$[2]", "$[2][0]"}, order)
	})
}

func TestWalkCharacterizeSimplifierDepth(t *testing.T) {
	t.Run("nested simplifier", func(t *testing.T) {
		tt.Equal(t,
			[]string{"$", "$.x"},
			sorted(walkPaths(alt.Simplifier(simple(1)))))
	})
}
