// Copyright (c) 2026, Peter Ohler, All rights reserved.

package node

import (
	"sort"
	"time"

	"github.com/ohler55/ojg/gen"
)

// Kind is the structural classification of a value.
type Kind int8

const (
	// Other marks values that have no direct structural representation and
	// need entry specific handling, usually reflection.
	Other Kind = iota
	// Null is a nil value.
	Null
	// Bool is a bool value.
	Bool
	// Int is any of the signed or unsigned integer widths.
	Int
	// Float is a float32 or float64 value.
	Float
	// String is a string value.
	String
	// Bytes is a []byte value.
	Bytes
	// Time is a time.Time value.
	Time
	// Array is a []any or gen.Array value.
	Array
	// Object is a map[string]any or gen.Object value.
	Object
	// Simplify is a value with a Simplify() any method such as the gen
	// scalar types or alt.Simplifier implementations.
	Simplify
)

// simplifier matches alt.Simplifier and gen.Node without importing alt
// which would create an import cycle.
type simplifier interface {
	Simplify() any
}

// Node is a read-only structural view of a value. It is a classification
// only; Raw always holds the original value and no part of the value is
// copied.
type Node struct {
	Kind Kind
	Raw  any
}

// Inspect classifies v without allocating or copying. The case order is
// significant: concrete container types are matched before the simplifier
// interface so gen.Array and gen.Object keep their container identity.
func Inspect(v any) (n Node) {
	switch tv := v.(type) {
	case []any, gen.Array:
		n = Node{Kind: Array, Raw: v}
	case map[string]any, gen.Object:
		n = Node{Kind: Object, Raw: v}
	case nil:
		n.Kind = Null
	case bool:
		n = Node{Kind: Bool, Raw: v}
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		n = Node{Kind: Int, Raw: v}
	case float32, float64:
		n = Node{Kind: Float, Raw: v}
	case string:
		n = Node{Kind: String, Raw: v}
	case []byte:
		n = Node{Kind: Bytes, Raw: v}
	case time.Time:
		n = Node{Kind: Time, Raw: v}
	case simplifier:
		n = Node{Kind: Simplify, Raw: tv}
	default:
		n = Node{Kind: Other, Raw: v}
	}
	return
}

// Len returns the number of elements of an Array or Object node. It
// returns 0 for any other kind.
func (n Node) Len() int {
	switch t := n.Raw.(type) {
	case []any:
		return len(t)
	case gen.Array:
		return len(t)
	case map[string]any:
		return len(t)
	case gen.Object:
		return len(t)
	}
	return 0
}

// At returns the i-th element of an Array node. It returns nil for any
// other kind.
func (n Node) At(i int) any {
	switch t := n.Raw.(type) {
	case []any:
		return t[i]
	case gen.Array:
		return t[i]
	}
	return nil
}

// Value returns the entry of an Object node for key. It returns nil for
// any other kind or if the key is not present.
func (n Node) Value(key string) any {
	switch t := n.Raw.(type) {
	case map[string]any:
		return t[key]
	case gen.Object:
		return t[key]
	}
	return nil
}

// UnpackArray returns the concrete array of an Array node as either a
// []any or a gen.Array. It lets hot loops range directly over the
// underlying slice without a per element type dispatch.
func (n Node) UnpackArray() ([]any, gen.Array) {
	switch t := n.Raw.(type) {
	case []any:
		return t, nil
	case gen.Array:
		return nil, t
	}
	return nil, nil
}

// UnpackObject returns the concrete object of an Object node as either a
// map[string]any or a gen.Object. It lets hot loops range directly over
// the underlying map without a per entry type dispatch.
func (n Node) UnpackObject() (map[string]any, gen.Object) {
	switch t := n.Raw.(type) {
	case map[string]any:
		return t, nil
	case gen.Object:
		return nil, t
	}
	return nil, nil
}

// Each calls fn for every entry of an Object node in the native Go map
// iteration order. That order is deliberately unstable; use EachSorted
// when a deterministic order is required.
func (n Node) Each(fn func(key string, val any)) {
	switch t := n.Raw.(type) {
	case map[string]any:
		for k, v := range t {
			fn(k, v)
		}
	case gen.Object:
		for k, v := range t {
			fn(k, v)
		}
	}
}

// EachSorted calls fn for every entry of an Object node in sorted key
// order so that Go map iteration order can not leak into the caller's
// output.
func (n Node) EachSorted(fn func(key string, val any)) {
	for _, k := range n.SortedKeys() {
		fn(k, n.Value(k))
	}
}

// SortedKeys returns the sorted keys of an Object node or nil for any
// other kind.
func (n Node) SortedKeys() []string {
	keys := make([]string, 0, n.Len())
	n.Each(func(key string, _ any) {
		keys = append(keys, key)
	})
	sort.Strings(keys)
	return keys
}

// Unwrap returns the simplified value of a Simplify node. For any other
// kind it returns the raw value unchanged.
func (n Node) Unwrap() any {
	if s, ok := n.Raw.(simplifier); ok {
		return s.Simplify()
	}
	return n.Raw
}

// Int64 returns the value of any Int kind value as an int64. Widths
// narrower than 64 bits are sign or zero extended and a uint64 value
// greater than math.MaxInt64 wraps, matching the historical behavior of
// the alt converters.
func Int64(v any) int64 {
	switch tv := v.(type) {
	case int:
		return int64(tv)
	case int8:
		return int64(tv)
	case int16:
		return int64(tv)
	case int32:
		return int64(tv)
	case int64:
		return tv
	case uint:
		return int64(tv)
	case uint8:
		return int64(tv)
	case uint16:
		return int64(tv)
	case uint32:
		return int64(tv)
	case uint64:
		return int64(tv)
	}
	return 0
}

// Uint64 returns the raw 64 bit pattern of any Int kind value. Negative
// signed values are sign extended into the high bits so int8(-1) and
// uint64(math.MaxUint64) share the same pattern, matching the historical
// behavior of alt.Checksum.
func Uint64(v any) uint64 {
	return uint64(Int64(v))
}
