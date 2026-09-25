// Copyright (c) 2026, Peter Ohler, All rights reserved.

package jp_test

import (
	"testing"

	"github.com/ohler55/ojg/internal/node"
	"github.com/ohler55/ojg/internal/node/nodetest"
	"github.com/ohler55/ojg/jp"
	"github.com/ohler55/ojg/tt"
)

// TestWalkKernelContract verifies that jp.Walk applies the shared kernel
// contract to the shared fixtures: scalar kinds and unrepresentable values
// are leaves, containers are traversed, and simplifiers are unwrapped.
func TestWalkKernelContract(t *testing.T) {
	for _, f := range nodetest.Fixtures() {
		t.Run(f.Name, func(t *testing.T) {
			var paths []string
			jp.Walk(f.Value, func(path jp.Expr, value any) {
				paths = append(paths, path.String())
			})
			switch f.Kind {
			case node.Null, node.Bool, node.Int, node.Float,
				node.String, node.Bytes, node.Time, node.Other:
				tt.Equal(t, []string{"$"}, paths,
					"%s: expected a single leaf visit", f.Name)
			case node.Array:
				tt.Equal(t, 1+node.Inspect(f.Value).Len(), len(paths),
					"%s: expected the array and each element", f.Name)
			case node.Object:
				tt.Equal(t, 1+node.Inspect(f.Value).Len(), len(paths),
					"%s: expected the object and each entry", f.Name)
			case node.Simplify:
				// A simplifier must be walked exactly as its unwrapped
				// value would be.
				var want []string
				jp.Walk(node.Inspect(f.Value).Unwrap(), func(path jp.Expr, value any) {
					want = append(want, path.String())
				})
				tt.Equal(t, want, paths,
					"%s: walk differs from walking the unwrapped value", f.Name)
			}
		})
	}
}

// TestWalkContractLeaves verifies the justLeaves form against the shared
// fixtures.
func TestWalkContractLeaves(t *testing.T) {
	for _, f := range nodetest.Fixtures() {
		t.Run(f.Name, func(t *testing.T) {
			var leaves int
			jp.Walk(f.Value, func(path jp.Expr, value any) {
				leaves++
			}, true)
			switch f.Kind {
			case node.Array, node.Object:
				// Containers are not leaves. The fixture containers hold
				// only leaves so the leaf count equals the length.
				tt.Equal(t, node.Inspect(f.Value).Len(), leaves,
					"%s: leaf count mismatch", f.Name)
			default:
				if leaves < 1 {
					t.Fatalf("%s: expected at least one leaf", f.Name)
				}
			}
		})
	}
}
