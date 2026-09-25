// Copyright (c) 2026, Peter Ohler, All rights reserved.

package alt_test

import (
	"testing"

	"github.com/ohler55/ojg"
	"github.com/ohler55/ojg/alt"
	"github.com/ohler55/ojg/tt"
)

// Cyclic references through reflected values must raise a path carrying
// *ojg.CycleError instead of overflowing the stack.

type cycleNode struct {
	Next *cycleNode `json:"next"`
}

type cycleMap map[string]cycleMap

type cycleSlice []any

func TestDecomposePointerCycle(t *testing.T) {
	a := &cycleNode{}
	b := &cycleNode{}
	a.Next = b
	b.Next = a

	out, err := alt.DecomposeSafe(a)
	tt.Nil(t, out)
	tt.NotNil(t, err)
	ce, ok := err.(*ojg.CycleError)
	tt.Equal(t, true, ok, "expected *ojg.CycleError, got %T", err)
	tt.Equal(t, "cyclic reference detected at next.next", ce.Error())
	tt.Equal(t, []any{"next", "next"}, ce.Path)
}

func TestDecomposeSelfPointerCycle(t *testing.T) {
	a := &cycleNode{}
	a.Next = a
	_, err := alt.DecomposeSafe(a)
	tt.NotNil(t, err)
	tt.Equal(t, "cyclic reference detected at next", err.Error())
}

func TestDecomposeMapCycle(t *testing.T) {
	m := cycleMap{}
	m["self"] = m
	_, err := alt.DecomposeSafe(m)
	tt.NotNil(t, err)
	tt.Equal(t, "cyclic reference detected at self", err.Error())
}

func TestDecomposeSliceCycle(t *testing.T) {
	s := cycleSlice{nil}
	s[0] = s
	_, err := alt.DecomposeSafe(s)
	tt.NotNil(t, err)
	tt.Equal(t, "cyclic reference detected at [0]", err.Error())
}

func TestDecomposeNestedPathInCycleError(t *testing.T) {
	a := &cycleNode{}
	a.Next = a
	data := map[string]any{"deep": []any{1, a}}
	_, err := alt.DecomposeSafe(data)
	tt.NotNil(t, err)
	tt.Equal(t, "cyclic reference detected at deep[1].next", err.Error())
}

func TestDecomposePanicsOnCycle(t *testing.T) {
	a := &cycleNode{}
	a.Next = a
	defer func() {
		r := recover()
		tt.NotNil(t, r)
		if _, ok := r.(*ojg.CycleError); !ok {
			t.Fatalf("expected *ojg.CycleError panic, got %T", r)
		}
	}()
	_ = alt.Decompose(a)
}

func TestAlterSafeCycle(t *testing.T) {
	a := &cycleNode{}
	a.Next = a
	out, err := alt.AlterSafe(a)
	tt.Nil(t, out)
	tt.NotNil(t, err)
	tt.Equal(t, "cyclic reference detected at next", err.Error())
}

func TestDecomposeSharedReferenceIsNotACycle(t *testing.T) {
	// The same pointer reachable from two sibling branches is shared, not
	// cyclic, and must decompose without error.
	shared := &cycleNode{}
	data := map[string]any{"a": shared, "b": shared}
	out, err := alt.DecomposeSafe(data)
	tt.Nil(t, err)
	m, ok := out.(map[string]any)
	tt.Equal(t, true, ok, "expected map[string]any, got %T", out)
	tt.Equal(t, 2, len(m))
}

func TestDecomposeSharedSliceBackingArray(t *testing.T) {
	// Two slices over one backing array are not a cycle.
	s := []any{int64(1), int64(2)}
	data := []any{s, s[:1]}
	out, err := alt.DecomposeSafe(data)
	tt.Nil(t, err)
	a, ok := out.([]any)
	tt.Equal(t, true, ok, "expected []any, got %T", out)
	tt.Equal(t, 2, len(a))
}

func TestChecksumCyclePanics(t *testing.T) {
	// Checksum decomposes reflected values and inherits the cycle
	// detection.
	a := &cycleNode{}
	a.Next = a
	defer func() {
		r := recover()
		tt.NotNil(t, r)
		if _, ok := r.(*ojg.CycleError); !ok {
			t.Fatalf("expected *ojg.CycleError panic, got %T", r)
		}
	}()
	_ = alt.Checksum(a)
}

func TestDiffCyclePanics(t *testing.T) {
	a := &cycleNode{}
	a.Next = a
	b := &cycleNode{}
	b.Next = b
	defer func() {
		r := recover()
		tt.NotNil(t, r)
		if _, ok := r.(*ojg.CycleError); !ok {
			t.Fatalf("expected *ojg.CycleError panic, got %T", r)
		}
	}()
	_ = alt.Diff(a, b)
}

func TestDecomposeDeepCycle(t *testing.T) {
	// Deeper than the guard inline depth to exercise the map overflow.
	head := &cycleNode{}
	tail := head
	for i := 0; i < 12; i++ {
		tail.Next = &cycleNode{}
		tail = tail.Next
	}
	tail.Next = head
	_, err := alt.DecomposeSafe(head)
	tt.NotNil(t, err)
	ce, ok := err.(*ojg.CycleError)
	tt.Equal(t, true, ok)
	tt.Equal(t, 13, len(ce.Path))
	for i, p := range ce.Path {
		tt.Equal(t, "next", p, "path element %d", i)
	}
}

func TestDecomposeVeryDeepCyclePathTruncated(t *testing.T) {
	// Deeper than the guard path cap: the reported path is truncated.
	head := &cycleNode{}
	tail := head
	for i := 0; i < 20; i++ {
		tail.Next = &cycleNode{}
		tail = tail.Next
	}
	tail.Next = head
	_, err := alt.DecomposeSafe(head)
	tt.NotNil(t, err)
	ce, ok := err.(*ojg.CycleError)
	tt.Equal(t, true, ok)
	tt.Equal(t, 16, len(ce.Path), "path should be truncated to the cap")
}
