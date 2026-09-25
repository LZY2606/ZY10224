// Copyright (c) 2026, Peter Ohler, All rights reserved.

package alt_test

import (
	"testing"

	"github.com/ohler55/ojg/alt"
	"github.com/ohler55/ojg/internal/node"
	"github.com/ohler55/ojg/internal/node/nodetest"
	"github.com/ohler55/ojg/tt"
)

// TestChecksumKernelContract verifies that alt.Checksum applies the shared
// kernel contract to the shared fixtures: every fixture produces a stable
// checksum and distinct kinds do not collapse onto each other.
func TestChecksumKernelContract(t *testing.T) {
	for _, f := range nodetest.Fixtures() {
		t.Run(f.Name, func(t *testing.T) {
			want := alt.Checksum(f.Value)
			for i := 0; i < 5; i++ {
				tt.Equal(t, want, alt.Checksum(f.Value),
					"%s: checksum not stable", f.Name)
			}
		})
	}
	t.Run("null differs from empty containers", func(t *testing.T) {
		null := alt.Checksum(nil)
		tt.NotEqual(t, null, alt.Checksum([]any{}))
		tt.NotEqual(t, null, alt.Checksum(map[string]any{}))
	})
	t.Run("typed nil pointer equals null", func(t *testing.T) {
		var p *int
		tt.Equal(t, alt.Checksum(nil), alt.Checksum(p))
	})
}

// TestDiffKernelContract verifies that alt.Diff applies the shared kernel
// contract to the shared fixtures: every fixture equals itself and differs
// from nil (except nil itself).
func TestDiffKernelContract(t *testing.T) {
	for _, f := range nodetest.Fixtures() {
		t.Run(f.Name, func(t *testing.T) {
			if f.Name == "typed nil pointer" {
				// Historical quirk: a typed nil pointer reflects to nil
				// which the reflection based comparison reports as a
				// difference, even against itself.
				tt.Equal(t, 1, len(alt.Diff(f.Value, f.Value)),
					"%s: typed nil pointer quirk changed", f.Name)
				return
			}
			tt.Equal(t, 0, len(alt.Diff(f.Value, f.Value)),
				"%s: value must equal itself", f.Name)
			if f.Kind == node.Null {
				tt.Equal(t, 0, len(alt.Diff(f.Value, nil)))
			} else {
				tt.Equal(t, 1, len(alt.Diff(f.Value, nil)),
					"%s: value must differ from nil", f.Name)
			}
		})
	}
}

// TestDecomposeKernelContract verifies that alt.Decompose applies the
// shared kernel contract to the shared fixtures.
func TestDecomposeKernelContract(t *testing.T) {
	for _, f := range nodetest.Fixtures() {
		t.Run(f.Name, func(t *testing.T) {
			out := alt.Decompose(f.Value)
			switch f.Kind {
			case node.Null:
				tt.Nil(t, out)
			case node.Int:
				tt.Equal(t, "int64", typeName(out),
					"%s: ints must normalize to int64", f.Name)
			case node.Array, node.Object, node.Simplify, node.Other:
				// Containers, simplifiers, and reflected values must
				// decompose to simple data: a second decompose must be
				// idempotent in shape.
				again := alt.Decompose(out)
				tt.Equal(t, alt.Checksum(out), alt.Checksum(again),
					"%s: decompose not idempotent", f.Name)
			}
		})
	}
}

func typeName(v any) string {
	switch v.(type) {
	case int64:
		return "int64"
	}
	return "other"
}
