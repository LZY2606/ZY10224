// Copyright (c) 2026, Peter Ohler, All rights reserved.

package jp_test

// Characterization tests for jp.Walk. These pin the existing walk behavior
// before and after the migration to the shared internal read-only
// structural access layer (internal/node): which values are leaves, what
// the callback receives for each value, and how containers are descended.

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/ohler55/ojg/gen"
	"github.com/ohler55/ojg/jp"
	"github.com/ohler55/ojg/tt"
)

type walkCharInt int

type walkCharSimp int

func (s walkCharSimp) Simplify() any { return map[string]any{"v": int64(s)} }

// collectWalk walks data and returns "path => type" strings sorted by path
// along with a map of path to the value passed to the callback.
func collectWalk(data any, justLeaves ...bool) (lines []string, values map[string]any) {
	values = map[string]any{}
	jp.Walk(data, func(path jp.Expr, value any) {
		key := path.String()
		lines = append(lines, fmt.Sprintf("%s => %T", key, value))
		values[key] = value
	}, justLeaves...)
	sort.Strings(lines)
	return
}

// The callback receives leaves for scalars, []byte, and time.Time while
// gen containers and generic containers are descended. gen scalars and
// Simplifier values are simplified before the callback.
func TestCharWalkNodeKinds(t *testing.T) {
	var nilPtr *int
	tm := time.Unix(1700000000, 0).UTC()
	data := map[string]any{
		"nil":      nil,
		"nilPtr":   nilPtr,
		"bytes":    []byte("hi"),
		"time":     tm,
		"genInt":   gen.Int(7),
		"genObj":   gen.Object{"x": gen.Int(1)},
		"genArr":   gen.Array{gen.Int(2)},
		"genTime":  gen.Time(tm),
		"myInt":    walkCharInt(3),
		"simp":     walkCharSimp(4),
		"emptyArr": []any{},
		"emptyMap": map[string]any{},
		"u64":      uint64(1<<64 - 1),
		"f32":      float32(1.5),
	}
	lines, values := collectWalk(data)
	tt.Equal(t, []string{
		`$ => map[string]interface {}`,
		`$.bytes => []uint8`,
		`$.emptyArr => []interface {}`,
		`$.emptyMap => map[string]interface {}`,
		`$.f32 => float32`,
		`$.genArr => gen.Array`,
		`$.genArr[0] => int64`,
		`$.genInt => int64`,
		`$.genObj => gen.Object`,
		`$.genObj.x => int64`,
		`$.genTime => time.Time`,
		`$.myInt => jp_test.walkCharInt`,
		`$.nil => <nil>`,
		`$.nilPtr => *int`,
		`$.simp => map[string]interface {}`,
		`$.simp.v => int64`,
		`$.time => time.Time`,
		`$.u64 => uint64`,
	}, lines, "walk node kinds")

	tt.Equal(t, uint64(1<<64-1), values["$.u64"], "uint64 leaf value")
	tt.Equal(t, float32(1.5), values["$.f32"], "float32 leaf value")
	tt.Equal(t, []byte("hi"), values["$.bytes"], "[]byte leaf value")
	tt.Equal(t, tm, values["$.time"], "time leaf value")
	tt.Equal(t, int64(7), values["$.genInt"], "gen.Int is simplified to int64")
	tt.Equal(t, walkCharInt(3), values["$.myInt"], "unhandled types are leaves passed through as is")
	tt.Equal(t, nilPtr, values["$.nilPtr"], "typed nil pointer is a leaf passed through as is")
}

// With justLeaves only leaf values are reported.
func TestCharWalkJustLeaves(t *testing.T) {
	data := map[string]any{
		"a":        []any{1, 2},
		"b":        map[string]any{"c": 3},
		"emptyArr": []any{},
		"emptyMap": map[string]any{},
		"gen":      gen.Array{gen.Int(4)},
	}
	lines, _ := collectWalk(data, true)
	tt.Equal(t, []string{
		`$.a[0] => int`,
		`$.a[1] => int`,
		`$.b.c => int`,
		`$.gen[0] => int64`,
	}, lines, "only leaves are reported, empty containers are not leaves")
}

// Map iteration order is not specified for walk; paths are collected and
// sorted by the tests. gen.Object member order is likewise unspecified.
func TestCharWalkGenObject(t *testing.T) {
	data := gen.Object{"a": gen.Array{gen.Int(1)}, "b": nil}
	lines, _ := collectWalk(data)
	tt.Equal(t, []string{
		`$ => gen.Object`,
		`$.a => gen.Array`,
		`$.a[0] => int64`,
		`$.b => <nil>`,
	}, lines, "gen.Object walk")
}
