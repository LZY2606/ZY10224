// Copyright (c) 2021, Peter Ohler, All rights reserved.

package alt

import (
	"fmt"
	"reflect"
	"sort"
	"time"
	"unsafe"

	"github.com/ohler55/ojg/gen"
	"github.com/ohler55/ojg/internal/node"
)

// TimeTolerance is the tolerance when comparing time elements
var TimeTolerance = time.Millisecond

// Path is a list of keys that can be either a string, int, or nil. Strings
// are used for keys in a map, ints are for indexes to a slice/array, and nil
// is a wildcard that matches either.
type Path []any

// String representation of the Path.
func (p Path) String() string {
	var b []byte

	for i, a := range p {
		switch ta := a.(type) {
		case int:
			b = fmt.Appendf(b, "[%d]", ta)
		case string:
			if 0 < i {
				b = append(b, '.')
			}
			b = append(b, ta...)
		}
	}
	return string(b)
}

// Diff returns the paths to the differences between two values. Any ignore
// paths are ignored in the comparison. Differences across map members are
// reported in sorted key order so the result does not depend on map
// iteration order.
//
// Cyclic data can not be compared fully. Where a cycle is detected the
// cyclic value is treated as nil instead of recursing until the stack
// overflows.
func Diff(v0, v1 any, ignores ...Path) (diffs []Path) {
	var s0, s1 node.Session
	return diff(v0, v1, false, &s0, &s1, node.Seg{}, ignores...)
}

// Compare returns a path to the first difference encountered between two
// values. Any ignore paths are ignored in the comparison. Map members are
// visited in sorted key order so the result does not depend on map
// iteration order.
func Compare(v0, v1 any, ignores ...Path) Path {
	var s0, s1 node.Session
	if diffs := diff(v0, v1, true, &s0, &s1, node.Seg{}, ignores...); 0 < len(diffs) {
		return diffs[0]
	}
	return nil
}

// Match returns true if all elements in the fingerprint match those in
// target. Fields in target but not in the fingerprint are ignored. An
// explicit nil in the fingerprint will match either a nil in the target or a
// missing value in the target.
func Match(fingerprint, target any) bool {
	var s0, s1 node.Session
	return matchValue(fingerprint, target, &s0, &s1, node.Seg{})
}

// enterContainers enters both values on their respective sessions when the
// values are containers. The returned flags are true for the values that
// complete a cycle. Both values are left again by leaveContainers.
func enterContainers(s0, s1 *node.Session, v0, v1 any, seg node.Seg) (c0, c1 bool) {
	e0 := s0.Enter(v0, seg)
	e1 := s1.Enter(v1, seg)
	if !e0 || !e1 {
		if e0 {
			s0.Leave(v0)
		}
		if e1 {
			s1.Leave(v1)
		}
		return !e0, !e1
	}
	return false, false
}

func leaveContainers(s0, s1 *node.Session, v0, v1 any) {
	s0.Leave(v0)
	s1.Leave(v1)
}

func matchValue(fingerprint, target any, s0, s1 *node.Session, seg node.Seg) bool {
	switch fp := fingerprint.(type) {
	case nil:
		if target != nil {
			return false
		}
	case bool:
		if t1, ok := target.(bool); !ok || fp != t1 {
			return false
		}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		i0, _ := asInt(fp)
		if i1, ok := asInt(target); !ok || i0 != i1 {
			return false
		}
	case float32, float64:
		f0, _ := asFloat(fp)
		if f1, ok := asFloat(target); !ok || f0 != f1 {
			return false
		}
	case string:
		if t1, ok := target.(string); !ok || fp != t1 {
			return false
		}
	case time.Time:
		if t1, ok := target.(time.Time); !ok || !fp.Round(TimeTolerance).Equal(t1.Round(TimeTolerance)) {
			return false
		}
	case []any:
		t1, ok := target.([]any)
		if !ok || len(fp) != len(t1) {
			return false
		}
		if c0, c1 := enterContainers(s0, s1, fingerprint, target, seg); c0 || c1 {
			// A cyclic value is treated as nil; both must be cyclic to
			// match.
			return c0 && c1
		}
		for i, v := range fp {
			if !matchValue(v, t1[i], s0, s1, node.IndexSeg(i)) {
				leaveContainers(s0, s1, fingerprint, target)
				return false
			}
		}
		leaveContainers(s0, s1, fingerprint, target)
		return true
	case map[string]any:
		t1, ok := target.(map[string]any)
		if !ok {
			return false
		}
		if c0, c1 := enterContainers(s0, s1, fingerprint, target, seg); c0 || c1 {
			return c0 && c1
		}
		for k, v := range fp {
			if !matchValue(v, t1[k], s0, s1, node.KeySeg(k)) {
				leaveContainers(s0, s1, fingerprint, target)
				return false
			}
		}
		leaveContainers(s0, s1, fingerprint, target)
		return true
	default:
		vt0 := (*[2]uintptr)(unsafe.Pointer(&fingerprint))[0]
		vt1 := (*[2]uintptr)(unsafe.Pointer(&target))[0]
		if vt0 == vt1 {
			if simp0, _ := fingerprint.(Simplifier); simp0 != nil {
				if simp1, _ := target.(Simplifier); simp1 != nil {
					return matchValue(simp0.Simplify(), simp1.Simplify(), s0, s1, seg)
				}
			}
			opt := &Options{}
			fingerprint = reflectValue(s0, reflect.ValueOf(fingerprint), fingerprint, opt, seg)
			target = reflectValue(s1, reflect.ValueOf(target), target, opt, seg)
			if fingerprint != nil && target != nil {
				return matchValue(fingerprint, target, s0, s1, seg)
			}
		}
		return false
	}
	return true
}

func diff(v0, v1 any, one bool, s0, s1 *node.Session, seg node.Seg, ignores ...Path) (diffs []Path) {
	switch t0 := v0.(type) {
	case nil:
		if v1 != nil {
			diffs = append(diffs, Path{nil})
		}
	case bool:
		if t1, ok := v1.(bool); !ok || t0 != t1 {
			diffs = append(diffs, Path{nil})
		}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		i0, _ := asInt(v0)
		if i1, ok := asInt(v1); !ok || i0 != i1 {
			diffs = append(diffs, Path{nil})
		}
	case float32, float64:
		f0, _ := asFloat(v0)
		if f1, ok := asFloat(v1); !ok || f0 != f1 {
			diffs = append(diffs, Path{nil})
		}
	case string:
		if t1, ok := v1.(string); !ok || t0 != t1 {
			diffs = append(diffs, Path{nil})
		}
	case time.Time:
		if t1, ok := v1.(time.Time); !ok || !t0.Round(TimeTolerance).Equal(t1.Round(TimeTolerance)) {
			diffs = append(diffs, Path{nil})
		}
	case []any:
		t1, ok := v1.([]any)
		if !ok {
			diffs = append(diffs, Path{nil})
			break
		}
		if c0, c1 := enterContainers(s0, s1, v0, v1, seg); c0 || c1 {
			// A cyclic value is treated as nil; it differs from a
			// non-cyclic value.
			if c0 && c1 {
				break
			}
			diffs = append(diffs, Path{nil})
			break
		}
		var childIgnores []Path
		ii := -1
		for _, ign := range ignores {
			if 1 < len(ign) {
				switch ti := ign[0].(type) {
				case nil:
					childIgnores = append(childIgnores, ign[1:])
				case int:
					ii = ti
					childIgnores = append(childIgnores, ign[1:])
				}
			}
		}
		for i, m1 := range t0 {
			if ignoreIndex(i, ignores) {
				continue
			}
			if len(t1) <= i {
				diffs = append(diffs, Path{i})
				leaveContainers(s0, s1, v0, v1)
				return
			}
			var ds []Path
			if ii == i || ii < 0 {
				ds = diff(m1, t1[i], one, s0, s1, node.IndexSeg(i), childIgnores...)
			} else {
				ds = diff(m1, t1[i], one, s0, s1, node.IndexSeg(i))
			}
			for _, d := range ds {
				if len(d) == 1 && d[0] == nil {
					d[0] = i
				} else {
					d = append(Path{i}, d...)
				}
				diffs = append(diffs, d)
				if one {
					leaveContainers(s0, s1, v0, v1)
					return
				}
			}
		}
		if len(t0) != len(t1) && !ignoreIndex(len(t0), ignores) {
			diffs = append(diffs, Path{len(t0)})
		}
		leaveContainers(s0, s1, v0, v1)
	case map[string]any:
		t1, ok := v1.(map[string]any)
		if !ok {
			diffs = append(diffs, Path{nil})
			break
		}
		if c0, c1 := enterContainers(s0, s1, v0, v1, seg); c0 || c1 {
			if c0 && c1 {
				break
			}
			diffs = append(diffs, Path{nil})
			break
		}
		// Keys of the first map are visited in sorted order followed by
		// the sorted keys unique to the second map so the result does not
		// depend on map iteration order. Small key sets are collected on
		// the stack.
		var keyBuf [16]string
		keys := keyBuf[:0]
		for k := range t0 {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var extraBuf [16]string
		extra := extraBuf[:0]
		for k := range t1 {
			if _, ok := t0[k]; !ok {
				extra = append(extra, k)
			}
		}
		sort.Strings(extra)
		for _, k := range append(keys, extra...) {
			if ignoreKey(k, ignores) {
				continue
			}
			var ds []Path
			if 0 < len(ignores) {
				var childIgnores []Path
				for _, ign := range ignores {
					if 1 < len(ign) {
						switch ti := ign[0].(type) {
						case nil:
							childIgnores = append(childIgnores, ign[1:])
						case string:
							if k == ti {
								childIgnores = append(childIgnores, ign[1:])
							}
						}
					}
				}
				ds = diff(t0[k], t1[k], one, s0, s1, node.KeySeg(k), childIgnores...)
			} else {
				ds = diff(t0[k], t1[k], one, s0, s1, node.KeySeg(k))
			}
			for _, d := range ds {
				if len(d) == 1 && d[0] == nil {
					d[0] = k
				} else {
					d = append(Path{k}, d...)
				}
				diffs = append(diffs, d)
				if one {
					leaveContainers(s0, s1, v0, v1)
					return
				}
			}
		}
		leaveContainers(s0, s1, v0, v1)
	default:
		vt0 := (*[2]uintptr)(unsafe.Pointer(&v0))[0]
		vt1 := (*[2]uintptr)(unsafe.Pointer(&v1))[0]
		if vt0 == vt1 {
			if simp0, _ := v0.(Simplifier); simp0 != nil {
				if simp1, _ := v1.(Simplifier); simp1 != nil {
					return diff(simp0.Simplify(), simp1.Simplify(), one, s0, s1, seg, ignores...)
				}
			}
			opt := &Options{}
			// TBD optimize by a more direct compare of fields
			v0 = reflectValue(s0, reflect.ValueOf(v0), v0, opt, seg)
			v1 = reflectValue(s1, reflect.ValueOf(v1), v1, opt, seg)
			if v0 != nil && v1 != nil {
				return diff(v0, v1, one, s0, s1, seg, ignores...)
			}
		}
		diffs = append(diffs, Path{nil})
		return
	}
	return
}

func asInt(v any) (i int64, ok bool) {
	ok = true
	switch tv := v.(type) {
	case int64:
		i = tv
	case int:
		i = int64(tv)
	case int8:
		i = int64(tv)
	case int16:
		i = int64(tv)
	case int32:
		i = int64(tv)
	case uint:
		i = int64(tv)
	case uint8:
		i = int64(tv)
	case uint16:
		i = int64(tv)
	case uint32:
		i = int64(tv)
	case uint64:
		i = int64(tv)
	case float32:
		i = int64(tv)
		if float32(int64(tv)) != tv {
			ok = false
		}
	case float64:
		i = int64(tv)
		if float64(int64(tv)) != tv {
			ok = false
		}
	case gen.Int:
		i = int64(tv)
	case gen.Float:
		i = int64(tv)
		if float64(int64(tv)) != float64(tv) {
			ok = false
		}
	default:
		ok = false
	}
	return
}

func asFloat(v any) (f float64, ok bool) {
	ok = true
	switch tv := v.(type) {
	case float64:
		f = tv
	case float32:
		f = float64(tv)
	case gen.Float:
		f = float64(tv)
	case int64:
		f = float64(tv)
	case int:
		f = float64(tv)
	case int8:
		f = float64(tv)
	case int16:
		f = float64(tv)
	case int32:
		f = float64(tv)
	case uint:
		f = float64(tv)
	case uint8:
		f = float64(tv)
	case uint16:
		f = float64(tv)
	case uint32:
		f = float64(tv)
	case uint64:
		f = float64(tv)
	case gen.Int:
		f = float64(tv)
	default:
		ok = false
	}
	return
}

func ignoreIndex(i int, ignores []Path) bool {
	for _, ign := range ignores {
		if len(ign) == 1 {
			switch ii := ign[0].(type) {
			case nil: // wildcard, matches any index
				return true
			case int:
				if i == ii {
					return true
				}
			}
		}
	}
	return false
}

func ignoreKey(k string, ignores []Path) bool {
	for _, ign := range ignores {
		if len(ign) == 1 {
			switch ik := ign[0].(type) {
			case nil: // wildcard, matches any index
				return true
			case string:
				if k == ik {
					return true
				}
			}
		}
	}
	return false
}
