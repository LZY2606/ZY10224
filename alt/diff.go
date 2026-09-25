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
// paths are ignored in the comparison. Differences in maps are reported in
// sorted key order so that Go map iteration order can not leak into the
// result.
func Diff(v0, v1 any, ignores ...Path) (diffs []Path) {
	return diff(v0, v1, false, ignores...)
}

// Compare returns a path to the first difference encountered between two
// values. Any ignore paths are ignored in the comparison. Map keys are
// visited in sorted order so the result is deterministic.
func Compare(v0, v1 any, ignores ...Path) Path {
	if diffs := diff(v0, v1, true, ignores...); 0 < len(diffs) {
		return diffs[0]
	}
	return nil
}

// Match returns true if all elements in the fingerprint match those in
// target. Fields in target but not in the fingerprint are ignored. An
// explicit nil in the fingerprint will match either a nil in the target or
// a missing value in the target.
func Match(fingerprint, target any) bool {
	n := node.Inspect(fingerprint)
	switch n.Kind {
	case node.Null:
		if target != nil {
			return false
		}
	case node.Bool:
		fp, _ := fingerprint.(bool)
		if t1, ok := target.(bool); !ok || fp != t1 {
			return false
		}
	case node.Int:
		i0, _ := asInt(fingerprint)
		if i1, ok := asInt(target); !ok || i0 != i1 {
			return false
		}
	case node.Float:
		f0, _ := asFloat(fingerprint)
		if f1, ok := asFloat(target); !ok || f0 != f1 {
			return false
		}
	case node.String:
		fp, _ := fingerprint.(string)
		if t1, ok := target.(string); !ok || fp != t1 {
			return false
		}
	case node.Time:
		fp, _ := fingerprint.(time.Time)
		if t1, ok := target.(time.Time); !ok || !fp.Round(TimeTolerance).Equal(t1.Round(TimeTolerance)) {
			return false
		}
	case node.Array:
		fp, ok := fingerprint.([]any)
		if !ok {
			return matchOther(fingerprint, target)
		}
		if t1, ok := target.([]any); ok && len(fp) == len(t1) {
			for i, v := range fp {
				if !Match(v, t1[i]) {
					return false
				}
			}
			return true
		}
		return false
	case node.Object:
		fp, ok := fingerprint.(map[string]any)
		if !ok {
			return matchOther(fingerprint, target)
		}
		if t1, ok := target.(map[string]any); ok {
			for k, v := range fp {
				if !Match(v, t1[k]) {
					return false
				}
			}
			return true
		}
		return false
	default:
		return matchOther(fingerprint, target)
	}
	return true
}

func matchOther(fingerprint, target any) bool {
	vt0 := (*[2]uintptr)(unsafe.Pointer(&fingerprint))[0]
	vt1 := (*[2]uintptr)(unsafe.Pointer(&target))[0]
	if vt0 == vt1 {
		if s0, _ := fingerprint.(Simplifier); s0 != nil {
			if s1, _ := target.(Simplifier); s1 != nil {
				return Match(s0.Simplify(), s1.Simplify())
			}
		}
		opt := &Options{}
		fingerprint = reflectValue(reflect.ValueOf(fingerprint), fingerprint, opt)
		target = reflectValue(reflect.ValueOf(target), target, opt)
		if fingerprint != nil && target != nil {
			return Match(fingerprint, target)
		}
	}
	return false
}

func diff(v0, v1 any, one bool, ignores ...Path) (diffs []Path) {
	n := node.Inspect(v0)
	switch n.Kind {
	case node.Null:
		if v1 != nil {
			diffs = append(diffs, Path{nil})
		}
	case node.Bool:
		t0, _ := v0.(bool)
		if t1, ok := v1.(bool); !ok || t0 != t1 {
			diffs = append(diffs, Path{nil})
		}
	case node.Int:
		i0, _ := asInt(v0)
		if i1, ok := asInt(v1); !ok || i0 != i1 {
			diffs = append(diffs, Path{nil})
		}
	case node.Float:
		f0, _ := asFloat(v0)
		if f1, ok := asFloat(v1); !ok || f0 != f1 {
			diffs = append(diffs, Path{nil})
		}
	case node.String:
		t0, _ := v0.(string)
		if t1, ok := v1.(string); !ok || t0 != t1 {
			diffs = append(diffs, Path{nil})
		}
	case node.Time:
		t0, _ := v0.(time.Time)
		if t1, ok := v1.(time.Time); !ok || !t0.Round(TimeTolerance).Equal(t1.Round(TimeTolerance)) {
			diffs = append(diffs, Path{nil})
		}
	case node.Array:
		// Only []any is compared as an array. A gen.Array keeps the
		// historical behavior of being compared through its simplified
		// form.
		t0, ok := v0.([]any)
		if !ok {
			return diffOther(v0, v1, one, ignores)
		}
		t1, ok := v1.([]any)
		if !ok {
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
				return
			}
			var ds []Path
			if ii == i || ii < 0 {
				ds = diff(m1, t1[i], one, childIgnores...)
			} else {
				ds = diff(m1, t1[i], one)
			}
			for _, d := range ds {
				if len(d) == 1 && d[0] == nil {
					d[0] = i
				} else {
					d = append(Path{i}, d...)
				}
				diffs = append(diffs, d)
				if one {
					return
				}
			}
		}
		if len(t0) != len(t1) && !ignoreIndex(len(t0), ignores) {
			diffs = append(diffs, Path{len(t0)})
		}
	case node.Object:
		// Only map[string]any is compared as an object. A gen.Object
		// keeps the historical behavior of being compared through its
		// simplified form.
		t0, ok := v0.(map[string]any)
		if !ok {
			return diffOther(v0, v1, one, ignores)
		}
		t1, ok := v1.(map[string]any)
		if !ok {
			diffs = append(diffs, Path{nil})
			break
		}
		keySet := map[string]bool{}
		for k := range t0 {
			keySet[k] = true
		}
		for k := range t1 {
			keySet[k] = true
		}
		keys := make([]string, 0, len(keySet))
		for k := range keySet {
			keys = append(keys, k)
		}
		// Keys are sorted so that Go map iteration order can not leak
		// into the returned diffs.
		sort.Strings(keys)
		for _, k := range keys {
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
				ds = diff(t0[k], t1[k], one, childIgnores...)
			} else {
				ds = diff(t0[k], t1[k], one)
			}
			for _, d := range ds {
				if len(d) == 1 && d[0] == nil {
					d[0] = k
				} else {
					d = append(Path{k}, d...)
				}
				diffs = append(diffs, d)
				if one {
					return
				}
			}
		}
	default:
		return diffOther(v0, v1, one, ignores)
	}
	return
}

func diffOther(v0, v1 any, one bool, ignores []Path) (diffs []Path) {
	vt0 := (*[2]uintptr)(unsafe.Pointer(&v0))[0]
	vt1 := (*[2]uintptr)(unsafe.Pointer(&v1))[0]
	if vt0 == vt1 {
		if s0, _ := v0.(Simplifier); s0 != nil {
			if s1, _ := v1.(Simplifier); s1 != nil {
				return diff(s0.Simplify(), s1.Simplify(), one, ignores...)
			}
		}
		opt := &Options{}
		// TBD optimize by a more direct compare of fields
		v0 = reflectValue(reflect.ValueOf(v0), v0, opt)
		v1 = reflectValue(reflect.ValueOf(v1), v1, opt)
		if v0 != nil && v1 != nil {
			return diff(v0, v1, one, ignores...)
		}
	}
	diffs = append(diffs, Path{nil})
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
