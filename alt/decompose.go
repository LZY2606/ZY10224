// Copyright (c) 2020, Peter Ohler, All rights reserved.

package alt

import (
	"encoding/base64"
	"math"
	"reflect"
	"time"

	"github.com/ohler55/ojg"
	"github.com/ohler55/ojg/internal/node"
)

// 23 for fraction in IEEE 754 which amounts to 7 significant digits. Use base
// 10 so that numbers look correct when displayed in base 10.
const fracMax = 10000000.0

func decompose(v any, opt *Options, sess *node.Session, seg node.Seg) any {
	switch tv := v.(type) {
	case nil, bool, int64, float64, string:
	case int:
		v = int64(tv)
	case int8:
		v = int64(tv)
	case int16:
		v = int64(tv)
	case int32:
		v = int64(tv)
	case uint:
		v = int64(tv)
	case uint8:
		v = int64(tv)
	case uint16:
		v = int64(tv)
	case uint32:
		v = int64(tv)
	case uint64:
		v = int64(tv)
	case float32:
		// This small rounding makes the conversion from 32 bit to 64 bit
		// display nicer.
		f, i := math.Frexp(float64(tv))
		f = float64(int64(f*fracMax)) / fracMax
		v = math.Ldexp(f, i)
	case []any:
		if !sess.Enter(v, seg) {
			return nil
		}
		a := make([]any, len(tv))
		for i, m := range tv {
			a[i] = decompose(m, opt, sess, node.IndexSeg(i))
		}
		sess.Leave(v)
		v = a
	case map[string]any:
		if !sess.Enter(v, seg) {
			return nil
		}
		o := map[string]any{}
		for k, m := range tv {
			condMapSet(o, k, decompose(m, opt, sess, node.KeySeg(k)), opt)
		}
		sess.Leave(v)
		v = o
	case []byte:
		switch opt.BytesAs {
		case ojg.BytesAsBase64:
			v = base64.StdEncoding.EncodeToString(tv)
		case ojg.BytesAsArray:
			a := make([]any, len(tv))
			for i, m := range tv {
				a[i] = decompose(m, opt, sess, node.IndexSeg(i))
			}
			v = a
		default:
			v = string(tv)
		}
	case time.Time:
		v = opt.DecomposeTime(tv)
	default:
		if simp, _ := v.(Simplifier); simp != nil {
			return decompose(simp.Simplify(), opt, sess, seg)
		}
		return reflectValue(sess, reflect.ValueOf(v), v, opt, seg)
	}
	return v
}

func alter(v any, opt *Options, sess *node.Session, seg node.Seg) any {
	switch tv := v.(type) {
	case bool, nil, int64, float64, string, time.Time:
	case int:
		v = int64(tv)
	case int8:
		v = int64(tv)
	case int16:
		v = int64(tv)
	case int32:
		v = int64(tv)
	case uint:
		v = int64(tv)
	case uint8:
		v = int64(tv)
	case uint16:
		v = int64(tv)
	case uint32:
		v = int64(tv)
	case uint64:
		v = int64(tv)
	case float32:
		// This small rounding makes the conversion from 32 bit to 64 bit
		// display nicer.
		f, i := math.Frexp(float64(tv))
		f = float64(int64(f*fracMax)) / fracMax
		v = math.Ldexp(f, i)
	case []any:
		if !sess.Enter(v, seg) {
			return nil
		}
		for i, m := range tv {
			tv[i] = alter(m, opt, sess, node.IndexSeg(i))
		}
		sess.Leave(v)
	case map[string]any:
		if !sess.Enter(v, seg) {
			return nil
		}
		for k, m := range tv {
			mv := alter(m, opt, sess, node.KeySeg(k))
			switch tmv := mv.(type) {
			case nil:
				if opt.OmitNil || opt.OmitEmpty {
					delete(tv, k)
					continue
				}
			case string:
				if opt.OmitEmpty && len(tmv) == 0 {
					delete(tv, k)
					continue
				}
			case []any:
				if opt.OmitEmpty && len(tmv) == 0 {
					delete(tv, k)
					continue
				}
			case map[string]any:
				if opt.OmitEmpty && len(tmv) == 0 {
					delete(tv, k)
					continue
				}
			case bool:
				if opt.OmitEmpty && !tmv {
					delete(tv, k)
					continue
				}
			case int64:
				if opt.OmitEmpty && tmv == 0 {
					delete(tv, k)
					continue
				}
			}
			tv[k] = mv
		}
		sess.Leave(v)
	case []byte:
		switch opt.BytesAs {
		case ojg.BytesAsBase64:
			v = base64.StdEncoding.EncodeToString(tv)
		case ojg.BytesAsArray:
			a := make([]any, len(tv))
			for i, m := range tv {
				a[i] = decompose(m, opt, sess, node.IndexSeg(i))
			}
			v = a
		default:
			v = string(tv)
		}
	default:
		if simp, _ := v.(Simplifier); simp != nil {
			return alter(simp.Simplify(), opt, sess, seg)
		}
		return reflectValue(sess, reflect.ValueOf(v), v, opt, seg)
	}
	return v
}

// reflectValue converts a value outside the generic data model into simple
// types using the shared node adapter for pointers, interfaces, aliases,
// slices, and maps. Structs and complex numbers keep the alt specific
// handling. The value is entered on the session for cycle detection and
// left again before returning.
func reflectValue(sess *node.Session, rv reflect.Value, val any, opt *Options, seg node.Seg) (v any) {
	if !sess.Enter(val, seg) {
		return nil
	}
	nv := node.Reflect(rv, val)
	switch nv.Kind() {
	case node.Null:
		v = nil
	case node.Bool:
		v = nv.Bool()
	case node.Int:
		v = nv.Int()
	case node.Uint:
		v = nv.Uint()
	case node.Float:
		v = nv.Float()
	case node.String:
		v = nv.String()
	case node.Array:
		size := nv.Len()
		a := make([]any, size)
		for i := size - 1; 0 <= i; i-- {
			a[i] = decompose(nv.Index(i).Raw(), opt, sess, node.IndexSeg(i))
		}
		v = a
	case node.Object:
		obj := map[string]any{}
		nv.Each(func(key string, child node.Value) bool {
			var g any
			if !child.IsNil() {
				g = decompose(child.Raw(), opt, sess, node.KeySeg(key))
			}
			condMapSet(obj, key, g, opt)
			return true
		})
		v = obj
	case node.Other:
		switch urv := nv.ReflectValue(); urv.Kind() {
		case reflect.Struct:
			if !urv.CanInterface() {
				v = nil
			} else if urv.CanAddr() {
				v = reflectStruct(sess, urv, urv.Interface(), opt)
			} else {
				v = reflectEmbed(sess, urv, urv.Interface(), opt)
			}
		case reflect.Complex64, reflect.Complex128:
			v = reflectComplex(urv, opt)
		default:
			// Channels, functions, and pointers without a target can not
			// be represented.
			v = nil
		}
	}
	sess.Leave(val)
	return
}

func reflectStruct(sess *node.Session, rv reflect.Value, val any, opt *Options) any {
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
					v = reflectEmbed(sess, fv, v, opt)
				} else {
					v = decompose(v, opt, sess, node.KeySeg(fi.key))
				}
			}
			condMapSet(obj, fi.key, v, opt)
		}
	}
	return obj
}

func reflectEmbed(sess *node.Session, rv reflect.Value, val any, opt *Options) any {
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
					v = reflectEmbed(sess, fv, v, opt)
				} else {
					v = decompose(v, opt, sess, node.KeySeg(fi.key))
				}
			}
			condMapSet(obj, fi.key, v, opt)
		}
	}
	return obj
}

func reflectComplex(rv reflect.Value, opt *Options) any {
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
