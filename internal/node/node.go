// Copyright (c) 2026, Peter Ohler, All rights reserved.

// Package node provides a shared read-only structural access layer over
// arbitrary Go values. It is the single place that classifies values into
// scalars, object entries, array elements, nulls, and unrepresentable
// values so that packages such as alt and jp do not each maintain their
// own type dispatch for walking data.
//
// A Value is a read-only view of a node in a value graph. It never copies
// the graph; children are adapted lazily as they are requested through
// Len, Index, Each, or Entries.
//
// The adapter handles the generic data model (nil, bool, the integer and
// float widths, string, []byte, time.Time, []any, map[string]any,
// gen.Array, and gen.Object) directly. Reflection based adaptation covers
// pointers, interfaces, typed nils, named alias types, slices and arrays
// of any element type, and maps with any key type. Values that have no
// representation in the generic data model (structs, complex numbers,
// channels, functions, and unsafe pointers) report a Kind of Other and are
// left to the caller which keeps its own policy for them.
package node

import (
	"reflect"
	"time"

	"github.com/ohler55/ojg"
	"github.com/ohler55/ojg/gen"
)

// Kind identifies the structural kind of a Value.
type Kind int8

const (
	// Other is a value that does not map to the generic data model such as
	// a struct, complex number, channel, or function. The caller decides
	// how to handle these values.
	Other Kind = iota
	// Null is an untyped nil or a typed nil pointer, interface, or other
	// nil-able value that the generic data model represents as null.
	Null
	// Bool is a boolean value.
	Bool
	// Int is a signed integer of any width.
	Int
	// Uint is an unsigned integer of any width.
	Uint
	// Float is a float32 or float64.
	Float
	// String is a string.
	String
	// Bytes is a []byte.
	Bytes
	// Time is a time.Time.
	Time
	// Array is an ordered collection of elements.
	Array
	// Object is a collection of string keyed entries.
	Object
)

// Value is a read-only view of a single node in a value graph.
type Value struct {
	raw  any
	rv   reflect.Value
	kind Kind
}

// Of adapts a value of the generic data model. Values outside the model
// report a Kind of Other; use Reflect to adapt those.
func Of(v any) Value {
	switch v.(type) {
	case nil:
		return Value{kind: Null}
	case bool:
		return Value{kind: Bool, raw: v}
	case int, int8, int16, int32, int64:
		return Value{kind: Int, raw: v}
	case uint, uint8, uint16, uint32, uint64:
		return Value{kind: Uint, raw: v}
	case float32, float64:
		return Value{kind: Float, raw: v}
	case string:
		return Value{kind: String, raw: v}
	case []byte:
		return Value{kind: Bytes, raw: v}
	case time.Time:
		return Value{kind: Time, raw: v}
	case []any, gen.Array:
		return Value{kind: Array, raw: v}
	case map[string]any, gen.Object:
		return Value{kind: Object, raw: v}
	}
	return Value{kind: Other, raw: v}
}

// Reflect adapts a value using reflection. Pointers and interfaces are
// unwrapped, typed nil pointers and interfaces report Null, named alias
// types report the kind of their underlying type, slices and arrays of any
// element type report Array, and maps with any key type report Object.
// Structs, complex numbers, channels, functions, and unsafe pointers
// report Other.
func Reflect(rv reflect.Value, raw any) Value {
	for {
		switch rv.Kind() {
		case reflect.Pointer, reflect.Interface:
			if rv.IsNil() {
				return Value{kind: Null, raw: raw}
			}
			rv = rv.Elem()
		default:
			goto done
		}
	}
done:
	switch rv.Kind() {
	case reflect.Bool:
		return Value{kind: Bool, raw: raw, rv: rv}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return Value{kind: Int, raw: raw, rv: rv}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return Value{kind: Uint, raw: raw, rv: rv}
	case reflect.Float32, reflect.Float64:
		return Value{kind: Float, raw: raw, rv: rv}
	case reflect.String:
		return Value{kind: String, raw: raw, rv: rv}
	case reflect.Slice, reflect.Array:
		return Value{kind: Array, raw: raw, rv: rv}
	case reflect.Map:
		return Value{kind: Object, raw: raw, rv: rv}
	case reflect.Invalid:
		return Value{kind: Null, raw: raw}
	}
	return Value{kind: Other, raw: raw, rv: rv}
}

// Kind returns the structural kind of the value.
func (v Value) Kind() Kind {
	return v.kind
}

// Raw returns the underlying value as provided to Of or Reflect.
func (v Value) Raw() any {
	return v.raw
}

// ReflectValue returns the reflected value after unwrapping pointers and
// interfaces. It is only valid for values adapted with Reflect.
func (v Value) ReflectValue() reflect.Value {
	return v.rv
}

// Bool returns the value as a bool. The Kind must be Bool.
func (v Value) Bool() bool {
	if b, ok := v.raw.(bool); ok {
		return b
	}
	return v.rv.Bool()
}

// Int returns the value as an int64. The Kind must be Int.
func (v Value) Int() int64 {
	switch t := v.raw.(type) {
	case int:
		return int64(t)
	case int8:
		return int64(t)
	case int16:
		return int64(t)
	case int32:
		return int64(t)
	case int64:
		return t
	}
	return v.rv.Int()
}

// Uint returns the value as a uint64. The Kind must be Uint.
func (v Value) Uint() uint64 {
	switch t := v.raw.(type) {
	case uint:
		return uint64(t)
	case uint8:
		return uint64(t)
	case uint16:
		return uint64(t)
	case uint32:
		return uint64(t)
	case uint64:
		return t
	}
	return v.rv.Uint()
}

// Float returns the value as a float64. The Kind must be Float.
func (v Value) Float() float64 {
	switch t := v.raw.(type) {
	case float32:
		return float64(t)
	case float64:
		return t
	}
	return v.rv.Float()
}

// String returns the value as a string. The Kind must be String.
func (v Value) String() string {
	if s, ok := v.raw.(string); ok {
		return s
	}
	return v.rv.String()
}

// Bytes returns the value as a []byte. The Kind must be Bytes.
func (v Value) Bytes() []byte {
	return v.raw.([]byte)
}

// Time returns the value as a time.Time. The Kind must be Time.
func (v Value) Time() time.Time {
	return v.raw.(time.Time)
}

// IsNil returns true if the value is nil or a typed nil pointer, map,
// slice, interface, channel, or function.
func (v Value) IsNil() bool {
	if v.raw == nil {
		return true
	}
	rv := reflect.ValueOf(v.raw)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface, reflect.Chan, reflect.Func:
		return rv.IsNil()
	}
	return false
}

// Len returns the number of elements of an Array or entries of an Object.
func (v Value) Len() int {
	switch t := v.raw.(type) {
	case []any:
		return len(t)
	case gen.Array:
		return len(t)
	case map[string]any:
		return len(t)
	case gen.Object:
		return len(t)
	}
	return v.rv.Len()
}

// Index returns the element at index i of an Array. The child is adapted
// with Of; use Reflect on the child Raw value for reflection based
// adaptation.
func (v Value) Index(i int) Value {
	switch t := v.raw.(type) {
	case []any:
		return Of(t[i])
	case gen.Array:
		return Of(t[i])
	}
	return Of(v.rv.Index(i).Interface())
}

// Each calls fn for every entry of an Object in the native order of the
// underlying collection. Iteration stops when fn returns false. Keys of
// reflected maps with non-string key types are converted with
// ojg.KeyString and are not deduplicated; use Entries for a deduplicated
// list.
func (v Value) Each(fn func(key string, child Value) bool) {
	switch t := v.raw.(type) {
	case map[string]any:
		for k, m := range t {
			if !fn(k, Of(m)) {
				return
			}
		}
	case gen.Object:
		for k, m := range t {
			if !fn(k, Of(m)) {
				return
			}
		}
	default:
		it := v.rv.MapRange()
		for it.Next() {
			if !fn(ojg.KeyString(it.Key()), Of(it.Value().Interface())) {
				return
			}
		}
	}
}

// Entry is a single object entry.
type Entry struct {
	Key   string
	Value Value
}

// Entries returns the entries of an Object in the native order of the
// underlying collection. When a reflected map has keys that convert to the
// same string with ojg.KeyString the last entry wins, matching the
// behavior of building a map[string]any from the reflected map.
func (v Value) Entries() []Entry {
	switch t := v.raw.(type) {
	case map[string]any:
		entries := make([]Entry, 0, len(t))
		for k, m := range t {
			entries = append(entries, Entry{Key: k, Value: Of(m)})
		}
		return entries
	case gen.Object:
		entries := make([]Entry, 0, len(t))
		for k, m := range t {
			entries = append(entries, Entry{Key: k, Value: Of(m)})
		}
		return entries
	}
	entries := make([]Entry, 0, v.rv.Len())
	index := map[string]int{}
	it := v.rv.MapRange()
	for it.Next() {
		k := ojg.KeyString(it.Key())
		if i, dup := index[k]; dup {
			entries[i].Value = Of(it.Value().Interface())
			continue
		}
		index[k] = len(entries)
		entries = append(entries, Entry{Key: k, Value: Of(it.Value().Interface())})
	}
	return entries
}
