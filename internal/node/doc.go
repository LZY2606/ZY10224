// Copyright (c) 2026, Peter Ohler, All rights reserved.

// Package node is the shared read-only structural access kernel for the
// ojg packages. It exists so that the JSONPath walker (jp.Walk), the alt
// converters and diff (alt.Decompose, alt.Alter, alt.Diff, alt.Compare,
// alt.Match), and the stable checksum (alt.Checksum) classify and traverse
// Go values through one code path instead of drifting copies of the same
// type switches.
//
// # Semantics
//
// Inspect classifies a value into a small set of structural kinds: Null,
// Bool, Int (any integer width), Float, String, Bytes, Time, Array, Object,
// Simplify (a value with a Simplify() any method), and Other for values
// that have no direct structural representation (pointers, interfaces,
// structs, maps with non-string keys, channels, functions, and other
// reflected-only shapes). Classification never allocates and never copies
// the inspected value; Node.Raw always holds the original value.
//
// Container access is lazy. Len, At, Value, Each, and EachSorted read from
// the original []any, gen.Array, map[string]any, or gen.Object in place.
// Nothing is copied into an intermediate tree representation. Each reports
// entries in the native Go map iteration order which is deliberately
// unstable; callers that need a deterministic order (alt.Checksum,
// alt.Diff) use EachSorted or SortedKeys so map order can not leak into
// their output. Callers that must preserve a user defined order (jp.Walk,
// alt.Decompose) use Each and keep the native order of the value.
//
// Reflection based traversal is guarded. Guard detects cyclic references
// through reflected pointers, maps, and slices and reports them as
// *ojg.CycleError carrying the path to the cycle. Typed nil pointers,
// interfaces, and slices are reported as nil by the guard (they can not be
// part of a cycle) and slices are identified by pointer and length so
// shared backing arrays are not mistaken for cycles.
//
// # Complexity
//
// Inspect is O(1). Len, At, and Value are O(1). Each is O(n) in the number
// of entries with no allocation. EachSorted and SortedKeys are
// O(n log n) with a single O(n) allocation for the key slice, the same
// cost the previous ad-hoc implementations had. Guard.Enter and
// Guard.Leave are O(1) (a short inline scan, spilling to a map only for
// traversals deeper than 8 reflected containers) and are only applied to
// reflected containers, never to the simple fast paths.
//
// # Compatibility
//
// The kernel is internal. Public behavior of the migrated entry points is
// unchanged except where documented in CHANGELOG.md: alt.Diff and
// alt.Compare now return map differences in sorted key order instead of
// map iteration order, and cyclic references in reflected values raise
// *ojg.CycleError instead of overflowing the stack.
package node
