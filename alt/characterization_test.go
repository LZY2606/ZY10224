// Copyright (c) 2026, Peter Ohler, All rights reserved.

package alt_test

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/ohler55/ojg/alt"
	"github.com/ohler55/ojg/gen"
	"github.com/ohler55/ojg/tt"
)

// The tests in this file are characterization tests. They pin the existing
// externally visible behavior of the alt package entry points that are
// migrated to the shared internal structural access kernel
// (internal/node). They must pass unchanged before and after the migration
// except where a behavior change is explicitly documented in CHANGELOG.md.

type charTime time.Time

type charText string

func TestCharacterizeNullVsEmpty(t *testing.T) {
	t.Run("decompose nil slice becomes empty slice", func(t *testing.T) {
		out := alt.Decompose([]any(nil))
		a, ok := out.([]any)
		tt.Equal(t, true, ok, "expected []any, got %T", out)
		tt.Equal(t, true, a != nil, "expected non-nil empty slice, got %#v", out)
		tt.Equal(t, 0, len(a))
	})
	t.Run("alter nil slice stays nil", func(t *testing.T) {
		out := alt.Alter([]any(nil))
		a, ok := out.([]any)
		tt.Equal(t, true, ok, "expected []any, got %T", out)
		tt.Equal(t, true, a == nil, "expected nil slice, got %#v", out)
	})
	t.Run("decompose typed nil map becomes empty object", func(t *testing.T) {
		out := alt.Decompose(map[string]int(nil))
		m, ok := out.(map[string]any)
		tt.Equal(t, true, ok, "expected map[string]any, got %T", out)
		tt.Equal(t, 0, len(m))
	})
	t.Run("decompose typed nil slice becomes empty array", func(t *testing.T) {
		out := alt.Decompose([]int(nil))
		a, ok := out.([]any)
		tt.Equal(t, true, ok, "expected []any, got %T", out)
		tt.Equal(t, 0, len(a))
	})
	t.Run("decompose typed nil pointer becomes nil", func(t *testing.T) {
		var p *int
		tt.Nil(t, alt.Decompose(p))
	})
	t.Run("typed nil pointer checksums like JSON null", func(t *testing.T) {
		var p *int
		tt.Equal(t, alt.Checksum(nil), alt.Checksum(p))
	})
	t.Run("empty slice does not checksum like null", func(t *testing.T) {
		tt.Equal(t, uint64(2895511793288787539), alt.Checksum([]any(nil)))
		tt.Equal(t, alt.Checksum([]any{}), alt.Checksum([]any(nil)))
		if alt.Checksum([]any{}) == alt.Checksum(nil) {
			t.Fatalf("checksum of empty slice must differ from null")
		}
	})
	t.Run("diff reports nil against empty containers", func(t *testing.T) {
		tt.Equal(t, 1, len(alt.Diff([]any(nil), nil)))
		tt.Equal(t, 1, len(alt.Diff([]any{}, nil)))
		tt.Equal(t, 1, len(alt.Diff(nil, []any{})))
		tt.Equal(t, 1, len(alt.Diff(nil, map[string]any{})))
	})
	t.Run("match nil against typed nil pointer fails", func(t *testing.T) {
		var p *int
		tt.Equal(t, false, alt.Match(nil, p))
		tt.Equal(t, false, alt.Match(p, nil))
	})
}

func TestCharacterizeIntWidths(t *testing.T) {
	t.Run("decompose normalizes all widths to int64", func(t *testing.T) {
		for i, v := range []any{
			int(-7), int8(-7), int16(-7), int32(-7), int64(-7),
			uint(7), uint8(7), uint16(7), uint32(7), uint64(7),
		} {
			out := alt.Decompose(v)
			tt.Equal(t, "int64", fmt.Sprintf("%T", out), "case %d: expected int64, got %T", i, out)
		}
	})
	t.Run("decompose wraps large uint64", func(t *testing.T) {
		tt.Equal(t, int64(-9223372036854775808), alt.Decompose(uint64(1)<<63))
	})
	t.Run("checksum is width independent", func(t *testing.T) {
		want := alt.Checksum(7)
		for _, v := range []any{
			int8(7), int16(7), int32(7), int64(7),
			uint(7), uint8(7), uint16(7), uint32(7), uint64(7),
		} {
			tt.Equal(t, want, alt.Checksum(v), "checksum of %T(%v) differs from int(7)", v, v)
		}
	})
	t.Run("checksum treats negative like unsigned bit pattern", func(t *testing.T) {
		tt.Equal(t, alt.Checksum(uint64(18446744073709551615)), alt.Checksum(int8(-1)))
	})
	t.Run("diff equates ints across widths", func(t *testing.T) {
		tt.Equal(t, 0, len(alt.Diff(int(7), int64(7))))
		tt.Equal(t, 0, len(alt.Diff(uint64(1)<<63, int64(-9223372036854775808))))
	})
	t.Run("diff gen.Int asymmetry", func(t *testing.T) {
		// asInt understands gen.Int but the gen.Int in the first position
		// falls through to the reflection based comparison. Pin the
		// existing asymmetry.
		tt.Equal(t, 0, len(alt.Diff(int64(1), gen.Int(1))))
		tt.Equal(t, 1, len(alt.Diff(gen.Int(1), int64(1))))
		tt.Equal(t, 0, len(alt.Diff(gen.Int(1), gen.Int(1))))
		tt.Equal(t, 1, len(alt.Diff(gen.Int(1), gen.Int(2))))
	})
}

func TestCharacterizeMapKeys(t *testing.T) {
	t.Run("non-string map keys are stringified", func(t *testing.T) {
		tt.Equal(t, map[string]any{"10": "a", "2": "b"}, alt.Decompose(map[int]string{2: "b", 10: "a"}))
		tt.Equal(t, map[string]any{"1.5": 1}, alt.Decompose(map[float64]int{1.5: 1}))
	})
	t.Run("checksum of non-string keys is stable", func(t *testing.T) {
		tt.Equal(t, uint64(6418600635885679347), alt.Checksum(map[int]string{2: "b", 10: "a"}))
	})
	t.Run("checksum is independent of map iteration order", func(t *testing.T) {
		base := map[string]any{"a": 1, "b": 2, "c": 3, "d": 4, "e": 5}
		want := alt.Checksum(base)
		for i := 0; i < 20; i++ {
			tt.Equal(t, want, alt.Checksum(base), "checksum changed between iterations")
		}
		rebuilt := map[string]any{}
		for _, k := range []string{"e", "d", "c", "b", "a"} {
			rebuilt[k] = base[k]
		}
		tt.Equal(t, want, alt.Checksum(rebuilt))
	})
}

func TestCharacterizeTimeLike(t *testing.T) {
	t.Run("time alias decomposes to type tagged object", func(t *testing.T) {
		out := alt.Decompose(charTime(time.Unix(0, 0).UTC()))
		tt.Equal(t, map[string]any{"type": "charTime"}, out)
	})
	t.Run("time alias checksum is stable", func(t *testing.T) {
		tt.Equal(t, uint64(208527574596582875), alt.Checksum(charTime(time.Unix(0, 0).UTC())))
	})
	t.Run("time alias diff equal to itself", func(t *testing.T) {
		tt.Equal(t, 0, len(alt.Diff(
			charTime(time.Unix(0, 0).UTC()),
			charTime(time.Unix(0, 0).UTC()))))
	})
	t.Run("string alias decomposes to string", func(t *testing.T) {
		tt.Equal(t, "hi", alt.Decompose(charText("hi")))
	})
}

func TestCharacterizeGenTypes(t *testing.T) {
	t.Run("gen.Object decomposes to map", func(t *testing.T) {
		tt.Equal(t, map[string]any{"a": int64(1)}, alt.Decompose(gen.Object{"a": gen.Int(1)}))
	})
	t.Run("gen types checksum like simplified values", func(t *testing.T) {
		tt.Equal(t,
			alt.Checksum(map[string]any{"a": int64(1), "b": []any{"x"}}),
			alt.Checksum(gen.Object{"a": gen.Int(1), "b": gen.Array{gen.String("x")}}))
	})
}

func TestCharacterizeDiffResults(t *testing.T) {
	t.Run("map diffs as a set", func(t *testing.T) {
		// The order of map diffs is not part of the pinned contract. The
		// set of paths is.
		diffs := alt.Diff(
			map[string]any{"a": 1, "b": 2, "c": 3},
			map[string]any{"a": 1, "b": 9})
		got := make([]string, 0, len(diffs))
		for _, d := range diffs {
			got = append(got, d.String())
		}
		sort.Strings(got)
		tt.Equal(t, []string{"b", "c"}, got)
	})
	t.Run("array diff paths", func(t *testing.T) {
		diffs := alt.Diff([]any{1, 2, 3}, []any{1, 9})
		got := make([]string, 0, len(diffs))
		for _, d := range diffs {
			got = append(got, d.String())
		}
		tt.Equal(t, []string{"[1]", "[2]"}, got)
	})
	t.Run("typed nil pointer diffs against nil", func(t *testing.T) {
		var p *int
		tt.Equal(t, 1, len(alt.Diff(p, nil)))
	})
}
