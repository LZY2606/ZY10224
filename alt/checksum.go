// Copyright (c) 2026, Peter Ohler, All rights reserved.

package alt

import (
	"hash/crc64"
	"math"
	"sort"

	"github.com/ohler55/ojg/internal/node"
)

var emcaTable = crc64.MakeTable(crc64.ECMA)

// Checksum of the provided data using a custom encoding and checksum
// routine. The functions is most efficient with simple data.
//
// Map keys are sorted before encoding so the checksum does not depend on
// map iteration order. Where a cycle is detected the cyclic value is
// encoded as null instead of recursing until the stack overflows.
func Checksum(v any) uint64 {
	var sess node.Session
	return crc64.Checksum(checksumAppend(nil, v, &sess, node.Seg{}), emcaTable)
}

func checksumAppend(b []byte, v any, sess *node.Session, seg node.Seg) []byte {
top:
	if simp, _ := v.(Simplifier); simp != nil {
		v = simp.Simplify()
		goto top
	}
	switch nv := node.Of(v); nv.Kind() {
	case node.Null:
		b = append(b, 0)
	case node.Bool:
		if nv.Bool() {
			b = append(b, "true"...)
		} else {
			b = append(b, "false"...)
		}
	case node.Int:
		b = appendUint64(b, uint64(nv.Int()))
	case node.Uint:
		b = appendUint64(b, nv.Uint())
	case node.Float:
		b = appendUint64(b, math.Float64bits(nv.Float()))
	case node.String:
		b = append(b, nv.String()...)
	case node.Bytes:
		b = append(b, nv.Bytes()...)
	case node.Time:
		tv := nv.Time()
		b = appendUint64(b, uint64(tv.UnixNano()))
		_, zone := tv.Zone()
		b = appendUint64(b, uint64(zone))
	case node.Array:
		if !sess.Enter(v, seg) {
			return append(b, 0)
		}
		b = append(b, '[')
		for i := 0; i < nv.Len(); i++ {
			b = checksumAppend(b, nv.Index(i).Raw(), sess, node.IndexSeg(i))
			b = append(b, ',')
		}
		b = append(b, ']')
		sess.Leave(v)
	case node.Object:
		if !sess.Enter(v, seg) {
			return append(b, 0)
		}
		if m, ok := v.(map[string]any); ok {
			// Fast path for the common generic map that avoids building an
			// entry list.
			var buf [16]string
			keys := buf[:0]
			for k := range m {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			b = append(b, '{')
			for _, k := range keys {
				b = append(b, k...)
				b = append(b, ':')
				b = checksumAppend(b, m[k], sess, node.KeySeg(k))
				b = append(b, ',')
			}
		} else {
			entries := nv.Entries()
			sort.Slice(entries, func(i, j int) bool { return entries[i].Key < entries[j].Key })
			b = append(b, '{')
			for _, e := range entries {
				b = append(b, e.Key...)
				b = append(b, ':')
				b = checksumAppend(b, e.Value.Raw(), sess, node.KeySeg(e.Key))
				b = append(b, ',')
			}
		}
		b = append(b, '}')
		sess.Leave(v)
	case node.Other:
		// Values outside the generic data model are decomposed with the
		// default options, matching the original behavior.
		b = checksumAppend(b, decompose(v, &DefaultOptions, sess, seg), sess, seg)
	}
	return b
}

func appendUint64(b []byte, v uint64) []byte {
	return append(b,
		byte(v>>56),
		byte(v>>48),
		byte(v>>40),
		byte(v>>32),
		byte(v>>24),
		byte(v>>16),
		byte(v>>8),
		byte(v),
	)
}
