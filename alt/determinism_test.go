// Copyright (c) 2026, Peter Ohler, All rights reserved.

package alt_test

import (
	"testing"

	"github.com/ohler55/ojg/alt"
	"github.com/ohler55/ojg/tt"
)

// Go map iteration order must not leak into diff results. Before the
// migration to the shared kernel the order of map diffs and the result of
// Compare depended on map iteration order.

func TestDiffMapOrderDeterministic(t *testing.T) {
	v0 := map[string]any{"z": 1, "y": 2, "x": 3, "w": 4}
	v1 := map[string]any{}
	want := alt.Diff(v0, v1)
	var wantStr []string
	for _, d := range want {
		wantStr = append(wantStr, d.String())
	}
	tt.Equal(t, []string{"w", "x", "y", "z"}, wantStr,
		"map diffs must be reported in sorted key order")
	for i := 0; i < 50; i++ {
		diffs := alt.Diff(v0, v1)
		var got []string
		for _, d := range diffs {
			got = append(got, d.String())
		}
		tt.Equal(t, wantStr, got, "diff order changed between runs")
	}
}

func TestCompareMapOrderDeterministic(t *testing.T) {
	v0 := map[string]any{"z": 1, "y": 2, "x": 3, "w": 4}
	v1 := map[string]any{}
	first := alt.Compare(v0, v1)
	tt.Equal(t, "w", first.String(), "compare must return the first key in sorted order")
	for i := 0; i < 50; i++ {
		tt.Equal(t, first.String(), alt.Compare(v0, v1).String(),
			"compare result changed between runs")
	}
}
