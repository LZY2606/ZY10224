// Copyright (c) 2026, Peter Ohler, All rights reserved.

package node

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ohler55/ojg/gen"
)

// CycleError is recorded by a Session when a cyclic reference is detected
// during a traversal. It carries the path to the value that completed the
// cycle.
type CycleError struct {
	// Path to the cyclic value. String elements are object keys and int
	// elements are array indexes.
	Path []any
}

// Error returns a description of the cycle including the path to the
// cyclic value.
func (e *CycleError) Error() string {
	var b strings.Builder

	b.WriteString("cyclic reference detected")
	if 0 < len(e.Path) {
		b.WriteString(" at ")
		for i, seg := range e.Path {
			switch ts := seg.(type) {
			case string:
				if 0 < i {
					b.WriteByte('.')
				}
				b.WriteString(ts)
			case int:
				_, _ = fmt.Fprintf(&b, "[%d]", ts)
			}
		}
	}
	return b.String()
}

// Seg is a path segment identifying how a value was reached from its
// parent: either an object key or an array index. It is a value type so
// that tracking paths does not allocate.
type Seg struct {
	Key   string
	Index int
	IsKey bool
}

// KeySeg returns a key path segment.
func KeySeg(key string) Seg {
	return Seg{Key: key, IsKey: true}
}

// IndexSeg returns an array index path segment.
func IndexSeg(index int) Seg {
	return Seg{Index: index}
}

// frame is a single entry on the traversal stack. The ptr and typ identify
// a container with identity; ptr is zero for frames that only provide a
// path segment.
type frame struct {
	ptr uintptr
	typ reflect.Type
	seg Seg
}

// Session tracks a single traversal so that cyclic references can be
// detected instead of recursing until the stack overflows. A Session is
// not safe for concurrent use.
//
// Enter is called when a traversal descends into a value and Leave when
// the traversal ascends back out of the same value. Values that can not be
// part of a cycle (scalars, empty containers, and nil-able nils) are
// ignored so the common case stays cheap.
type Session struct {
	frames []frame
	err    error
}

// NewSession creates a Session for a single traversal.
func NewSession() *Session {
	return &Session{}
}

var (
	mapStringAnyType = reflect.TypeOf(map[string]any(nil))
	sliceAnyType     = reflect.TypeOf([]any(nil))
	genObjectType    = reflect.TypeOf(gen.Object(nil))
	genArrayType     = reflect.TypeOf(gen.Array(nil))
)

// frameOf returns the frame for a value and whether the value is tracked
// on the traversal stack.
func frameOf(v any, seg Seg) (f frame, ok bool) {
	switch tv := v.(type) {
	case map[string]any:
		if 0 < len(tv) {
			return frame{ptr: reflect.ValueOf(v).Pointer(), typ: mapStringAnyType, seg: seg}, true
		}
	case []any:
		if 0 < len(tv) {
			return frame{ptr: reflect.ValueOf(v).Pointer(), typ: sliceAnyType, seg: seg}, true
		}
	case gen.Object:
		if 0 < len(tv) {
			return frame{ptr: reflect.ValueOf(v).Pointer(), typ: genObjectType, seg: seg}, true
		}
	case gen.Array:
		if 0 < len(tv) {
			return frame{ptr: reflect.ValueOf(v).Pointer(), typ: genArrayType, seg: seg}, true
		}
	default:
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Pointer, reflect.Map:
			if !rv.IsNil() {
				return frame{ptr: rv.Pointer(), typ: rv.Type(), seg: seg}, true
			}
		case reflect.Slice:
			if 0 < rv.Len() {
				return frame{ptr: rv.Pointer(), typ: rv.Type(), seg: seg}, true
			}
		case reflect.Struct:
			// Structs can not be part of a cycle but a frame keeps the
			// path segments complete for cycle error reporting.
			return frame{typ: rv.Type(), seg: seg}, true
		}
	}
	return
}

// Enter registers a descent into v reached through the path segment seg.
// It returns false and records a CycleError if v is already on the
// traversal stack. Every Enter must be paired with a Leave of the same
// value.
func (s *Session) Enter(v any, seg Seg) bool {
	f, ok := frameOf(v, seg)
	if !ok {
		return true
	}
	if f.ptr != 0 {
		for _, pf := range s.frames {
			if pf.ptr == f.ptr && pf.typ == f.typ {
				if s.err == nil {
					s.err = &CycleError{Path: s.pathTo(seg)}
				}
				return false
			}
		}
	}
	s.frames = append(s.frames, f)
	return true
}

// Leave registers an ascent back out of a value previously passed to
// Enter.
func (s *Session) Leave(v any) {
	if _, ok := frameOf(v, Seg{}); ok && 0 < len(s.frames) {
		s.frames = s.frames[:len(s.frames)-1]
	}
}

// Err returns the first CycleError recorded during the traversal, if any.
func (s *Session) Err() error {
	return s.err
}

// pathTo builds the path to the value being entered through seg.
func (s *Session) pathTo(seg Seg) []any {
	path := make([]any, 0, len(s.frames)+1)
	for _, f := range s.frames[1:] {
		path = append(path, f.seg.any())
	}
	return append(path, seg.any())
}

// any returns the segment as a string for keys and an int for indexes.
func (seg Seg) any() any {
	if seg.IsKey {
		return seg.Key
	}
	return seg.Index
}
