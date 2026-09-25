// Copyright (c) 2026, Peter Ohler, All rights reserved.

package node

import (
	"reflect"

	"github.com/ohler55/ojg"
)

// visit identifies a reflected container on the current traversal path.
// Slices are identified by pointer and length so that two slices sharing a
// backing array are not mistaken for a cycle.
type visit struct {
	typ reflect.Type
	ptr uintptr
	len int
}

const (
	// guardInline is the number of visits held inline before spilling to
	// a map. Traversals are rarely deeper than this so the common case
	// allocates nothing.
	guardInline = 8

	// guardPathCap is the maximum number of path elements tracked for the
	// cycle error report. Deeper cycles are reported with a truncated
	// path. The path is kept inline so that tracking it never allocates.
	guardPathCap = 16
)

// Guard detects cyclic references during structural traversal. It tracks
// the containers on the current path only, so shared (but not cyclic)
// references visited from different branches are not reported as cycles.
// Enter and Leave must be balanced in LIFO order, the natural order of a
// depth first traversal.
//
// The zero value is ready to use.
type Guard struct {
	inline [guardInline]visit
	n      int
	seen   map[visit]struct{}
	path   [guardPathCap]any
	pn     int
}

// Push appends a path element, a string object key or an int array index,
// to the current path. It is used to build the path reported by a
// *ojg.CycleError. Elements beyond guardPathCap are counted but not
// stored.
func (g *Guard) Push(key any) {
	if g.pn < guardPathCap {
		g.path[g.pn] = key
	}
	g.pn++
}

// Pop removes the most recently pushed path element.
func (g *Guard) Pop() {
	g.pn--
	if g.pn < guardPathCap {
		g.path[g.pn] = nil
	}
}

// Path returns the current path, truncated to guardPathCap elements.
func (g *Guard) Path() []any {
	n := g.pn
	if guardPathCap < n {
		n = guardPathCap
	}
	// Copy element-wise. Slicing g.path would take the address of the
	// inline array and force the guard onto the heap.
	out := make([]any, n)
	for i := 0; i < n; i++ {
		out[i] = g.path[i]
	}
	return out
}

// keyOf returns the identity of a reflected container. Typed nil maps,
// pointers, and slices can not be part of a cycle and report false.
func keyOf(rv reflect.Value) (visit, bool) {
	switch rv.Kind() {
	case reflect.Map, reflect.Pointer:
		if rv.IsNil() {
			return visit{}, false
		}
		return visit{typ: rv.Type(), ptr: rv.Pointer()}, true
	case reflect.Slice:
		if rv.IsNil() {
			return visit{}, false
		}
		return visit{typ: rv.Type(), ptr: rv.Pointer(), len: rv.Len()}, true
	}
	return visit{}, false
}

// Enter marks a reflected container as on the current path. It returns a
// *ojg.CycleError carrying the path to the container if the container is
// already on the path, and nil otherwise. Every successful Enter must be
// balanced by a Leave.
func (g *Guard) Enter(rv reflect.Value) error {
	v, ok := keyOf(rv)
	if !ok {
		return nil
	}
	for i := 0; i < g.n; i++ {
		if g.inline[i] == v {
			return &ojg.CycleError{Path: g.Path()}
		}
	}
	if g.seen != nil {
		if _, dup := g.seen[v]; dup {
			return &ojg.CycleError{Path: g.Path()}
		}
		g.seen[v] = struct{}{}
		return nil
	}
	if g.n < guardInline {
		g.inline[g.n] = v
		g.n++
		return nil
	}
	// Deeper than the inline stack: visits beyond guardInline overflow
	// into a map. The inline visits stay inline so the guard itself never
	// escapes.
	g.seen = make(map[visit]struct{}, 8)
	g.seen[v] = struct{}{}
	return nil
}

// Leave removes a reflected container from the current path.
func (g *Guard) Leave(rv reflect.Value) {
	v, ok := keyOf(rv)
	if !ok {
		return
	}
	if g.seen != nil {
		if _, ok := g.seen[v]; ok {
			delete(g.seen, v)
			return
		}
	}
	// Enters and leaves are balanced in LIFO order so the matching visit
	// is the top of the inline stack.
	if 0 < g.n && g.inline[g.n-1] == v {
		g.n--
		g.inline[g.n] = visit{}
		return
	}
	// Defensive: remove a matching visit anywhere in the stack.
	for i := g.n - 1; 0 <= i; i-- {
		if g.inline[i] == v {
			for j := i; j+1 < g.n; j++ {
				g.inline[j] = g.inline[j+1]
			}
			g.n--
			g.inline[g.n] = visit{}
			return
		}
	}
}

// EnterValue is Enter for a value that is not already reflected.
func (g *Guard) EnterValue(v any) error {
	return g.Enter(reflect.ValueOf(v))
}

// LeaveValue is Leave for a value that is not already reflected.
func (g *Guard) LeaveValue(v any) {
	g.Leave(reflect.ValueOf(v))
}
