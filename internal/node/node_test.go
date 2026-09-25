// Copyright (c) 2026, Peter Ohler, All rights reserved.

package node_test

import (
	"reflect"
	"sort"
	"testing"

	"github.com/ohler55/ojg"
	"github.com/ohler55/ojg/gen"
	"github.com/ohler55/ojg/internal/node"
	"github.com/ohler55/ojg/internal/node/nodetest"
	"github.com/ohler55/ojg/tt"
)

func TestInspectContract(t *testing.T) {
	for _, f := range nodetest.Fixtures() {
		t.Run(f.Name, func(t *testing.T) {
			n := node.Inspect(f.Value)
			tt.Equal(t, f.Kind, n.Kind, "kind mismatch for %#v", f.Value)
		})
	}
}

func TestNodeArrayAccess(t *testing.T) {
	for _, data := range []any{
		[]any{1, "a", nil},
		gen.Array{gen.Int(1), gen.String("a"), nil},
	} {
		n := node.Inspect(data)
		tt.Equal(t, node.Array, n.Kind)
		tt.Equal(t, 3, n.Len())
		tt.NotNil(t, n.At(0))
		tt.Nil(t, n.At(2))
	}
	n := node.Inspect(nil)
	tt.Equal(t, 0, n.Len())
	tt.Nil(t, n.At(0))
}

func TestNodeObjectAccess(t *testing.T) {
	for _, data := range []any{
		map[string]any{"b": 2, "a": 1},
		gen.Object{"b": gen.Int(2), "a": gen.Int(1)},
	} {
		n := node.Inspect(data)
		tt.Equal(t, node.Object, n.Kind)
		tt.Equal(t, 2, n.Len())
		tt.NotNil(t, n.Value("a"))
		tt.Nil(t, n.Value("zzz"))

		count := 0
		n.Each(func(key string, val any) {
			count++
			if key != "a" && key != "b" {
				t.Fatalf("unexpected key %q", key)
			}
		})
		tt.Equal(t, 2, count)

		var sorted []string
		n.EachSorted(func(key string, val any) {
			sorted = append(sorted, key)
			tt.NotNil(t, val)
		})
		tt.Equal(t, []string{"a", "b"}, sorted)
		tt.Equal(t, []string{"a", "b"}, n.SortedKeys())
	}
}

func TestNodeUnwrap(t *testing.T) {
	n := node.Inspect(gen.Int(3))
	tt.Equal(t, node.Simplify, n.Kind)
	tt.Equal(t, int64(3), n.Unwrap())

	n = node.Inspect(nodetest.Simplifier{X: 7})
	tt.Equal(t, node.Simplify, n.Kind)
	tt.Equal(t, map[string]any{"x": int64(7)}, n.Unwrap())

	n = node.Inspect(3)
	tt.Equal(t, 3, n.Unwrap(), "non-simplify nodes unwrap to themselves")
}

func TestInt64Widths(t *testing.T) {
	for i, v := range []any{
		int(-7), int8(-7), int16(-7), int32(-7), int64(-7),
		uint(7), uint8(7), uint16(7), uint32(7), uint64(7),
	} {
		want := int64(-7)
		if 5 <= i {
			want = 7
		}
		tt.Equal(t, want, node.Int64(v), "case %d (%T)", i, v)
	}
	tt.Equal(t, int64(-9223372036854775808), node.Int64(uint64(1)<<63))
	tt.Equal(t, uint64(18446744073709551615), node.Uint64(int8(-1)))
	tt.Equal(t, int64(0), node.Int64("not an int"))
}

func TestGuardCycle(t *testing.T) {
	var g node.Guard
	m := map[string]any{}
	rv := reflect.ValueOf(m)
	tt.Nil(t, g.Enter(rv))
	err := g.Enter(rv)
	tt.NotNil(t, err)
	if _, ok := err.(*ojg.CycleError); !ok {
		t.Fatalf("expected *ojg.CycleError, got %T", err)
	}
	tt.Equal(t, "cyclic reference detected at $", err.Error())
	g.Leave(rv)
	tt.Nil(t, g.Enter(rv), "after Leave the same map can be entered again")
}

func TestGuardPath(t *testing.T) {
	var g node.Guard
	p := reflect.ValueOf(&g)
	tt.Nil(t, g.Enter(p))
	g.Push("a")
	g.Push(3)
	err := g.Enter(p)
	tt.NotNil(t, err)
	tt.Equal(t, "cyclic reference detected at a[3]", err.Error())
	g.Pop()
	g.Pop()
	g.Leave(p)
}

func TestGuardTypedNil(t *testing.T) {
	var g node.Guard
	var p *int
	tt.Nil(t, g.Enter(reflect.ValueOf(p)), "typed nil pointer is not a cycle")
	var m map[string]any
	tt.Nil(t, g.Enter(reflect.ValueOf(m)), "typed nil map is not a cycle")
	var s []any
	tt.Nil(t, g.Enter(reflect.ValueOf(s)), "typed nil slice is not a cycle")
}

func TestGuardSharedBackingArray(t *testing.T) {
	var g node.Guard
	s := []any{1, 2, 3}
	tt.Nil(t, g.Enter(reflect.ValueOf(s)))
	// A sub-slice shares the backing array pointer but has a different
	// length and must not be reported as a cycle.
	tt.Nil(t, g.Enter(reflect.ValueOf(s[:2])))
	g.Leave(reflect.ValueOf(s[:2]))
	g.Leave(reflect.ValueOf(s))
}

func TestGuardSharedReferenceNotCycle(t *testing.T) {
	var g node.Guard
	m := map[string]any{}
	rv := reflect.ValueOf(m)
	tt.Nil(t, g.Enter(rv))
	g.Leave(rv)
	tt.Nil(t, g.Enter(rv), "revisiting from a sibling branch is not a cycle")
	g.Leave(rv)
}

func TestEachSortedDeterministic(t *testing.T) {
	data := map[string]any{}
	for _, k := range []string{"z", "y", "x", "w", "v"} {
		data[k] = 1
	}
	n := node.Inspect(data)
	var prev []string
	for i := 0; i < 10; i++ {
		var keys []string
		n.EachSorted(func(key string, _ any) {
			keys = append(keys, key)
		})
		if !sort.StringsAreSorted(keys) {
			t.Fatalf("keys not sorted: %v", keys)
		}
		if prev != nil {
			tt.Equal(t, prev, keys, "iteration order changed between runs")
		}
		prev = keys
	}
}

func TestGuardOverflowToMap(t *testing.T) {
	var g node.Guard
	maps := make([]map[string]any, 12)
	for i := range maps {
		maps[i] = map[string]any{"i": i}
		tt.Nil(t, g.Enter(reflect.ValueOf(maps[i])), "enter %d", i)
		g.Push(i)
	}
	// The first container is inline, the rest overflowed to the map. A
	// duplicate of either kind must be detected.
	err := g.Enter(reflect.ValueOf(maps[0]))
	tt.NotNil(t, err, "duplicate of an inline visit must be detected")
	err = g.Enter(reflect.ValueOf(maps[10]))
	tt.NotNil(t, err, "duplicate of an overflow visit must be detected")
	for i := len(maps) - 1; 0 <= i; i-- {
		g.Leave(reflect.ValueOf(maps[i]))
		g.Pop()
	}
	// After all leaves the same containers can be entered again.
	tt.Nil(t, g.Enter(reflect.ValueOf(maps[0])))
	tt.Nil(t, g.Enter(reflect.ValueOf(maps[10])))
}
