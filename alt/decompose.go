// Copyright (c) 2020, Peter Ohler, All rights reserved.

package alt

import (
	"encoding/base64"
	"math"
	"reflect"
	"time"

	"github.com/ohler55/ojg"
	"github.com/ohler55/ojg/gen"
	"github.com/ohler55/ojg/internal/node"
)

// 23 for fraction in IEEE 754 which amounts to 7 significant digits. Use base
// 10 so that numbers look correct when displayed in base 10.
const fracMax = 10000000.0

// decCtx carries the shared state of a single Decompose or Alter
// traversal: the cycle detection guard. The guard is armed once the
// traversal passes through a reflected value. Pure simple data ([]any and
// map[string]any) is not guarded to keep the simple fast path free of
// bookkeeping; cycles in pure simple data remain unsupported as
// documented.
//
// The options are passed as a separate parameter and not stored in decCtx
// on purpose: the guard bookkeeping escapes to the heap by design and a
// shared context struct would drag the caller's options along with it.
type decCtx struct {
	guard node.Guard
	armed bool
}

// enter guards a container against cycles once the traversal is armed.
func (ctx *decCtx) enter(v any) {
	if !ctx.armed {
		return
	}
	if err := ctx.guard.EnterValue(v); err != nil {
		panic(err)
	}
}

// leave balances enter.
func (ctx *decCtx) leave(v any) {
	if ctx.armed {
		ctx.guard.LeaveValue(v)
	}
}

func decompose(v any, opt *Options) any {
	ctx := decCtx{}
	return ctx.decomposeNode(v, node.Inspect(v), opt)
}

// decomposeKeyed decomposes a map or field value. The key is pushed onto
// the guard path only when v can recurse (containers and reflection
// candidates). Scalar values can not be part of a cycle and are not
// tracked which keeps the path bookkeeping and the key boxing off the
// scalar hot path.
func (ctx *decCtx) decomposeKeyed(v any, key string, opt *Options) (out any) {
	n := node.Inspect(v)
	switch n.Kind {
	case node.Array, node.Object, node.Simplify, node.Other:
		ctx.guard.Push(key)
		out = ctx.decomposeNode(v, n, opt)
		ctx.guard.Pop()
	default:
		out = ctx.decomposeNode(v, n, opt)
	}
	return
}

// decomposeIndexed decomposes an array element. See decomposeKeyed.
func (ctx *decCtx) decomposeIndexed(v any, index int, opt *Options) (out any) {
	n := node.Inspect(v)
	switch n.Kind {
	case node.Array, node.Object, node.Simplify, node.Other:
		ctx.guard.Push(index)
		out = ctx.decomposeNode(v, n, opt)
		ctx.guard.Pop()
	default:
		out = ctx.decomposeNode(v, n, opt)
	}
	return
}

func (ctx *decCtx) decomposeNode(v any, n node.Node, opt *Options) any {
	switch n.Kind {
	case node.Null, node.Bool, node.String:
		// already simple
	case node.Int:
		v = node.Int64(v)
	case node.Float:
		if tv, ok := v.(float32); ok {
			// This small rounding makes the conversion from 32 bit to 64
			// bit display nicer.
			f, i := math.Frexp(float64(tv))
			f = float64(int64(f*fracMax)) / fracMax
			v = math.Ldexp(f, i)
		}
	case node.Array:
		ctx.enter(v)
		a := make([]any, n.Len())
		for i := range a {
			a[i] = ctx.decomposeIndexed(n.At(i), i, opt)
		}
		ctx.leave(v)
		v = a
	case node.Object:
		ctx.enter(v)
		o := map[string]any{}
		// The entry loops are kept per concrete type so that no closure
		// escapes to the heap on this hot path.
		switch t := v.(type) {
		case map[string]any:
			for k, m := range t {
				condMapSet(o, k, ctx.decomposeKeyed(m, k, opt), opt)
			}
		case gen.Object:
			for k, m := range t {
				condMapSet(o, k, ctx.decomposeKeyed(m, k, opt), opt)
			}
		}
		ctx.leave(v)
		v = o
	case node.Bytes:
		tv, _ := v.([]byte)
		switch opt.BytesAs {
		case ojg.BytesAsBase64:
			v = base64.StdEncoding.EncodeToString(tv)
		case ojg.BytesAsArray:
			a := make([]any, len(tv))
			for i, m := range tv {
				a[i] = ctx.decomposeNode(m, node.Inspect(m), opt)
			}
			v = a
		default:
			v = string(tv)
		}
	case node.Time:
		tv, _ := v.(time.Time)
		v = opt.DecomposeTime(tv)
	default:
		if n.Kind == node.Simplify {
			if simp, _ := v.(Simplifier); simp != nil {
				sv := simp.Simplify()
				return ctx.decomposeNode(sv, node.Inspect(sv), opt)
			}
		}
		return ctx.reflectValue(reflect.ValueOf(v), v, opt)
	}
	return v
}

func alter(v any, opt *Options) any {
	ctx := decCtx{}
	return ctx.alterNode(v, node.Inspect(v), opt)
}

// alterKeyed alters a map value. See decomposeKeyed for the key tracking
// contract.
func (ctx *decCtx) alterKeyed(v any, key string, opt *Options) (out any) {
	n := node.Inspect(v)
	switch n.Kind {
	case node.Array, node.Object, node.Simplify, node.Other:
		ctx.guard.Push(key)
		out = ctx.alterNode(v, n, opt)
		ctx.guard.Pop()
	default:
		out = ctx.alterNode(v, n, opt)
	}
	return
}

// alterIndexed alters an array element. See decomposeKeyed.
func (ctx *decCtx) alterIndexed(v any, index int, opt *Options) (out any) {
	n := node.Inspect(v)
	switch n.Kind {
	case node.Array, node.Object, node.Simplify, node.Other:
		ctx.guard.Push(index)
		out = ctx.alterNode(v, n, opt)
		ctx.guard.Pop()
	default:
		out = ctx.alterNode(v, n, opt)
	}
	return
}

func (ctx *decCtx) alterNode(v any, n node.Node, opt *Options) any {
	switch n.Kind {
	case node.Null, node.Bool, node.String, node.Time:
		// already simple
	case node.Int:
		v = node.Int64(v)
	case node.Float:
		if tv, ok := v.(float32); ok {
			// This small rounding makes the conversion from 32 bit to 64
			// bit display nicer.
			f, i := math.Frexp(float64(tv))
			f = float64(int64(f*fracMax)) / fracMax
			v = math.Ldexp(f, i)
		}
	case node.Array:
		// Only []any is altered in place. A gen.Array keeps the
		// historical behavior of being replaced by its simplified form.
		if ta, ok := v.([]any); ok {
			ctx.enter(v)
			for i, m := range ta {
				ta[i] = ctx.alterIndexed(m, i, opt)
			}
			ctx.leave(v)
			return ta
		}
		return ctx.alterOther(v, opt)
	case node.Object:
		// Only map[string]any is altered in place. A gen.Object keeps
		// the historical behavior of being replaced by its simplified
		// form.
		if tm, ok := v.(map[string]any); ok {
			ctx.enter(v)
			for k, m := range tm {
				mv := ctx.alterKeyed(m, k, opt)
				switch tmv := mv.(type) {
				case nil:
					if opt.OmitNil || opt.OmitEmpty {
						delete(tm, k)
						continue
					}
				case string:
					if opt.OmitEmpty && len(tmv) == 0 {
						delete(tm, k)
						continue
					}
				case []any:
					if opt.OmitEmpty && len(tmv) == 0 {
						delete(tm, k)
						continue
					}
				case map[string]any:
					if opt.OmitEmpty && len(tmv) == 0 {
						delete(tm, k)
						continue
					}
				case bool:
					if opt.OmitEmpty && !tmv {
						delete(tm, k)
						continue
					}
				case int64:
					if opt.OmitEmpty && tmv == 0 {
						delete(tm, k)
						continue
					}
				}
				tm[k] = mv
			}
			ctx.leave(v)
			return tm
		}
		return ctx.alterOther(v, opt)
	case node.Bytes:
		tv, _ := v.([]byte)
		switch opt.BytesAs {
		case ojg.BytesAsBase64:
			v = base64.StdEncoding.EncodeToString(tv)
		case ojg.BytesAsArray:
			a := make([]any, len(tv))
			for i, m := range tv {
				a[i] = ctx.decomposeNode(m, node.Inspect(m), opt)
			}
			v = a
		default:
			v = string(tv)
		}
	default:
		return ctx.alterOther(v, opt)
	}
	return v
}

func (ctx *decCtx) alterOther(v any, opt *Options) any {
	if simp, _ := v.(Simplifier); simp != nil {
		sv := simp.Simplify()
		return ctx.alterNode(sv, node.Inspect(sv), opt)
	}
	return ctx.reflectValue(reflect.ValueOf(v), v, opt)
}

func reflectValue(rv reflect.Value, val any, opt *Options) any {
	ctx := decCtx{}
	return ctx.reflectValue(rv, val, opt)
}

func (ctx *decCtx) reflectValue(rv reflect.Value, val any, opt *Options) (v any) {
	// Arm cycle detection for the rest of the traversal. Reflected
	// containers are always guarded; simple containers reached from here
	// are guarded by the armed decCtx as well.
	ctx.armed = true
	switch rv.Kind() {
	case reflect.Invalid, reflect.Uintptr, reflect.UnsafePointer, reflect.Chan, reflect.Func, reflect.Interface:
		v = nil
	case reflect.Complex64, reflect.Complex128:
		v = ctx.reflectComplex(rv, opt)
	case reflect.Map:
		if err := ctx.guard.Enter(rv); err != nil {
			panic(err)
		}
		v = ctx.reflectMap(rv, opt)
		ctx.guard.Leave(rv)
	case reflect.Pointer:
		if err := ctx.guard.Enter(rv); err != nil {
			panic(err)
		}
		elem := rv.Elem()
		if elem.IsValid() && elem.CanInterface() {
			v = ctx.reflectValue(elem, elem.Interface(), opt)
		} else {
			v = nil
		}
		ctx.guard.Leave(rv)
	case reflect.Slice:
		if err := ctx.guard.Enter(rv); err != nil {
			panic(err)
		}
		v = ctx.reflectArray(rv, opt)
		ctx.guard.Leave(rv)
	case reflect.Array:
		v = ctx.reflectArray(rv, opt)
	case reflect.Struct:
		v = ctx.reflectStruct(rv, val, opt)
	case reflect.String:
		v = rv.String()
	case reflect.Bool:
		v = rv.Bool()
	case reflect.Float32, reflect.Float64:
		v = rv.Float()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v = rv.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v = rv.Uint()
	}
	return
}

func (ctx *decCtx) reflectStruct(rv reflect.Value, val any, opt *Options) any {
	if !rv.CanAddr() {
		return ctx.reflectEmbed(rv, val, opt)
	}
	obj := map[string]any{}
	si := getSinfo(val, opt.OmitEmpty)
	t := si.rt
	if 0 < len(opt.CreateKey) {
		if opt.FullTypePath {
			obj[opt.CreateKey] = t.PkgPath() + "/" + t.Name()
		} else {
			obj[opt.CreateKey] = t.Name()
		}
	}
	fields := si.getFields(opt)
	addr := rv.UnsafeAddr()
	for _, fi := range fields {
		if v, fv, omit := fi.value(fi, rv, addr); !omit {
			if fv.IsValid() {
				if opt.NestEmbed && fv.Kind() == reflect.Struct {
					ctx.guard.Push(fi.key)
					v = ctx.reflectEmbed(fv, v, opt)
					ctx.guard.Pop()
				} else {
					v = ctx.decomposeKeyed(v, fi.key, opt)
				}
			}
			condMapSet(obj, fi.key, v, opt)
		}
	}
	return obj
}

func (ctx *decCtx) reflectEmbed(rv reflect.Value, val any, opt *Options) any {
	obj := map[string]any{}
	si := getSinfo(val, opt.OmitEmpty)
	t := si.rt
	if 0 < len(opt.CreateKey) {
		if opt.FullTypePath {
			obj[opt.CreateKey] = t.PkgPath() + "/" + t.Name()
		} else {
			obj[opt.CreateKey] = t.Name()
		}
	}
	fields := si.getFields(opt)
	for _, fi := range fields {
		if v, fv, omit := fi.ivalue(fi, rv, 0); !omit {
			if fv.IsValid() {
				if opt.NestEmbed && fv.Kind() == reflect.Struct {
					ctx.guard.Push(fi.key)
					v = ctx.reflectEmbed(fv, v, opt)
					ctx.guard.Pop()
				} else {
					v = ctx.decomposeKeyed(v, fi.key, opt)
				}
			}
			condMapSet(obj, fi.key, v, opt)
		}
	}
	return obj
}

func (ctx *decCtx) reflectComplex(rv reflect.Value, opt *Options) any {
	c := rv.Complex()
	obj := map[string]any{
		"real": real(c),
		"imag": imag(c),
	}
	if 0 < len(opt.CreateKey) {
		obj[opt.CreateKey] = "complex"
	}
	return obj
}

func (ctx *decCtx) reflectMap(rv reflect.Value, opt *Options) any {
	obj := map[string]any{}
	it := rv.MapRange()
	for it.Next() {
		var g any
		vv := it.Value()
		key := ojg.KeyString(it.Key())
		if !isNil(vv) {
			g = ctx.decomposeKeyed(vv.Interface(), key, opt)
		}
		condMapSet(obj, key, g, opt)
	}
	return obj
}

func (ctx *decCtx) reflectArray(rv reflect.Value, opt *Options) any {
	size := rv.Len()
	a := make([]any, size)
	for i := size - 1; 0 <= i; i-- {
		a[i] = ctx.decomposeIndexed(rv.Index(i).Interface(), i, opt)
	}
	return a
}

func isNil(rv reflect.Value) bool {
	switch rv.Kind() {
	case reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	}
	return false
}

func condMapSet(m map[string]any, key string, value any, opt *Options) {
	switch tv := value.(type) {
	case nil:
		if opt.OmitNil || opt.OmitEmpty {
			return
		}
	case string:
		if opt.OmitEmpty && len(tv) == 0 {
			return
		}
	case []any:
		if opt.OmitEmpty && len(tv) == 0 {
			return
		}
	case map[string]any:
		if opt.OmitEmpty && len(tv) == 0 {
			return
		}
	case bool:
		if opt.OmitEmpty && !tv {
			return
		}
	case int64:
		if opt.OmitEmpty && tv == 0 {
			return
		}
	}
	m[key] = value
}
