// Copyright (c) 2026, Peter Ohler, All rights reserved.

package node_test

import (
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/ohler55/ojg/gen"
	"github.com/ohler55/ojg/internal/node"
	"github.com/ohler55/ojg/tt"
)

type aliasInt int
type aliasUint uint
type aliasFloat float32
type aliasString string
type aliasBool bool
type aliasBytes []byte
type aliasSlice []int
type aliasMap map[int]string
type aliasStruct struct{ X int }

func TestOfKinds(t *testing.T) {
	for i, tc := range []struct {
		value any
		kind  node.Kind
	}{
		{nil, node.Null},
		{true, node.Bool},
		{int(1), node.Int},
		{int8(1), node.Int},
		{int16(1), node.Int},
		{int32(1), node.Int},
		{int64(1), node.Int},
		{uint(1), node.Uint},
		{uint8(1), node.Uint},
		{uint16(1), node.Uint},
		{uint32(1), node.Uint},
		{uint64(1), node.Uint},
		{float32(1), node.Float},
		{float64(1), node.Float},
		{"x", node.String},
		{[]byte("x"), node.Bytes},
		{time.Now(), node.Time},
		{[]any{}, node.Array},
		{gen.Array{}, node.Array},
		{map[string]any{}, node.Object},
		{gen.Object{}, node.Object},
		{gen.Int(1), node.Other},
		{aliasInt(1), node.Other},
		{aliasStruct{}, node.Other},
	} {
		tt.Equal(t, tc.kind, node.Of(tc.value).Kind(), "case %d: %T", i, tc.value)
	}
}

func TestReflectKinds(t *testing.T) {
	var nilPtr *int
	var nilSlice []int
	var nilMap map[int]string
	for i, tc := range []struct {
		value any
		kind  node.Kind
	}{
		{nilPtr, node.Null},
		{nilSlice, node.Array},
		{nilMap, node.Object},
		{aliasInt(3), node.Int},
		{aliasUint(3), node.Uint},
		{aliasFloat(1.5), node.Float},
		{aliasString("x"), node.String},
		{aliasBool(true), node.Bool},
		{aliasBytes("x"), node.Array},
		{aliasSlice{1}, node.Array},
		{aliasMap{1: "x"}, node.Object},
		{aliasStruct{}, node.Other},
		{complex(1, 2), node.Other},
		{func() {}, node.Other},
		{make(chan int), node.Other},
		{time.Now(), node.Other},
		{&[]int{1}, node.Array},
	} {
		tt.Equal(t, tc.kind, node.Reflect(reflect.ValueOf(tc.value), tc.value).Kind(), "case %d: %T", i, tc.value)
	}
}

func TestReflectScalarAccessors(t *testing.T) {
	tt.Equal(t, int64(-3), node.Reflect(reflect.ValueOf(aliasInt(-3)), nil).Int(), "alias int")
	tt.Equal(t, uint64(3), node.Reflect(reflect.ValueOf(aliasUint(3)), nil).Uint(), "alias uint")
	tt.Equal(t, float64(float32(1.5)), node.Reflect(reflect.ValueOf(aliasFloat(1.5)), nil).Float(), "alias float")
	tt.Equal(t, "x", node.Reflect(reflect.ValueOf(aliasString("x")), nil).String(), "alias string")
	tt.Equal(t, true, node.Reflect(reflect.ValueOf(aliasBool(true)), nil).Bool(), "alias bool")
}

func TestConcreteScalarAccessors(t *testing.T) {
	tt.Equal(t, int64(-3), node.Of(int8(-3)).Int(), "int8")
	tt.Equal(t, uint64(3), node.Of(uint8(3)).Uint(), "uint8")
	tt.Equal(t, 1.5, node.Of(float64(1.5)).Float(), "float64")
	tt.Equal(t, "x", node.Of("x").String(), "string")
	tt.Equal(t, []byte("x"), node.Of([]byte("x")).Bytes(), "bytes")
	tm := time.Unix(1700000000, 0).UTC()
	tt.Equal(t, tm, node.Of(tm).Time(), "time")
}

func TestArrayAccess(t *testing.T) {
	v := node.Of([]any{1, "x", nil})
	tt.Equal(t, 3, v.Len(), "len")
	tt.Equal(t, node.Int, v.Index(0).Kind(), "element kind")
	tt.Equal(t, 1, v.Index(0).Raw(), "element raw")

	rv := node.Reflect(reflect.ValueOf([]int{4, 5}), nil)
	tt.Equal(t, 2, rv.Len(), "reflected len")
	tt.Equal(t, 5, rv.Index(1).Raw(), "reflected element")

	gv := node.Of(gen.Array{gen.Int(7)})
	tt.Equal(t, 1, gv.Len(), "gen len")
	tt.Equal(t, gen.Int(7), gv.Index(0).Raw(), "gen element")
}

func TestObjectEachNativeOrder(t *testing.T) {
	// map order is unspecified; collect and sort.
	v := node.Of(map[string]any{"a": 1, "b": 2})
	var keys []string
	v.Each(func(key string, child node.Value) bool {
		keys = append(keys, key)
		return true
	})
	sort.Strings(keys)
	tt.Equal(t, []string{"a", "b"}, keys, "each keys")

	// early stop
	count := 0
	v.Each(func(key string, child node.Value) bool {
		count++
		return false
	})
	tt.Equal(t, 1, count, "each stops when fn returns false")
}

func TestObjectEntriesReflectedKeys(t *testing.T) {
	v := node.Reflect(reflect.ValueOf(map[int]string{3: "x", 1: "y"}), nil)
	entries := v.Entries()
	tt.Equal(t, 2, len(entries), "entry count")
	keys := []string{entries[0].Key, entries[1].Key}
	sort.Strings(keys)
	tt.Equal(t, []string{"1", "3"}, keys, "reflected keys as strings")
}

func TestObjectEntriesDuplicateKeysLastWins(t *testing.T) {
	// Keys 1 (int) and "1" (string) both convert to "1".
	v := node.Reflect(reflect.ValueOf(map[any]any{1: "a", "1": "b"}), nil)
	entries := v.Entries()
	tt.Equal(t, 1, len(entries), "duplicate converted keys collapse to one entry")
	tt.Equal(t, "1", entries[0].Key, "converted key")
}

func TestIsNil(t *testing.T) {
	var nilPtr *int
	var nilSlice []int
	tt.Equal(t, true, node.Of(nil).IsNil(), "nil")
	tt.Equal(t, true, node.Of(nilPtr).IsNil(), "typed nil pointer")
	tt.Equal(t, true, node.Of(nilSlice).IsNil(), "typed nil slice")
	tt.Equal(t, false, node.Of(0).IsNil(), "zero int")
	tt.Equal(t, false, node.Of("").IsNil(), "empty string")
}

func TestSessionCycleDetection(t *testing.T) {
	type recur struct {
		Name string `json:"name"`
		Next *recur `json:"next"`
	}
	r := &recur{Name: "a"}
	r.Next = r

	sess := node.NewSession()
	tt.Equal(t, true, sess.Enter(r, node.Seg{}), "root enter")
	tt.Equal(t, false, sess.Enter(r, node.KeySeg("next")), "cycle detected")
	tt.NotNil(t, sess.Err(), "cycle error recorded")
	cycle, ok := sess.Err().(*node.CycleError)
	tt.Equal(t, true, ok, "error is a CycleError")
	tt.Equal(t, []any{"next"}, cycle.Path, "cycle path")
	tt.Equal(t, "cyclic reference detected at next", cycle.Error(), "error text")
}

func TestSessionSharedSubValueIsNotACycle(t *testing.T) {
	shared := []any{1}
	data := map[string]any{"a": shared, "b": shared}
	sess := node.NewSession()
	tt.Equal(t, true, sess.Enter(data, node.Seg{}), "root")
	tt.Equal(t, true, sess.Enter(shared, node.KeySeg("a")), "first use")
	sess.Leave(shared)
	tt.Equal(t, true, sess.Enter(shared, node.KeySeg("b")), "second use is not a cycle")
	sess.Leave(shared)
	sess.Leave(data)
	tt.Nil(t, sess.Err(), "no error")
}

func TestSessionLeaveBalance(t *testing.T) {
	sess := node.NewSession()
	// scalars and empty containers are not tracked
	tt.Equal(t, true, sess.Enter(3, node.Seg{}), "scalar enter")
	sess.Leave(3)
	tt.Equal(t, true, sess.Enter([]any{}, node.Seg{}), "empty slice enter")
	sess.Leave([]any{})
	m := map[string]any{"a": 1}
	tt.Equal(t, true, sess.Enter(m, node.Seg{}), "map enter")
	sess.Leave(m)
	tt.Equal(t, true, sess.Enter(m, node.Seg{}), "map can be entered again after leave")
	sess.Leave(m)
	tt.Nil(t, sess.Err(), "no error")
}
