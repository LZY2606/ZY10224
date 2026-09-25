// Copyright (c) 2026, Peter Ohler, All rights reserved.

package alt

import (
	"hash/crc64"
	"math"
	"time"

	"github.com/ohler55/ojg/internal/node"
)

var emcaTable = crc64.MakeTable(crc64.ECMA)

// Checksum of the provided data using a custom encoding and checksum
// routine. The functions is most efficient with simple data.
func Checksum(v any) uint64 {
	return crc64.Checksum(checksumAppend(nil, v), emcaTable)
}

func checksumAppend(b []byte, v any) []byte {
	n := node.Inspect(v)
	switch n.Kind {
	case node.Null:
		b = append(b, 0)
	case node.Bool:
		if tv, _ := v.(bool); tv {
			b = append(b, "true"...)
		} else {
			b = append(b, "false"...)
		}
	case node.Int:
		b = appendUint64(b, node.Uint64(v))
	case node.Float:
		if tv, ok := v.(float32); ok {
			b = appendUint64(b, math.Float64bits(float64(tv)))
		} else {
			tv, _ := v.(float64)
			b = appendUint64(b, math.Float64bits(tv))
		}
	case node.String:
		b = append(b, v.(string)...)
	case node.Bytes:
		b = append(b, v.([]byte)...)
	case node.Time:
		tv, _ := v.(time.Time)
		b = appendUint64(b, uint64(tv.UnixNano()))
		_, zone := tv.Zone()
		b = appendUint64(b, uint64(zone))
	case node.Array:
		b = append(b, '[')
		if a, g := n.UnpackArray(); a != nil {
			for _, v2 := range a {
				b = checksumAppend(b, v2)
				b = append(b, ',')
			}
		} else {
			for _, v2 := range g {
				b = checksumAppend(b, v2)
				b = append(b, ',')
			}
		}
		b = append(b, ']')
	case node.Object:
		// Keys are sorted so that Go map iteration order can not leak
		// into the checksum.
		b = append(b, '{')
		for _, k := range n.SortedKeys() {
			b = append(b, k...)
			b = append(b, ':')
			b = checksumAppend(b, n.Value(k))
			b = append(b, ',')
		}
		b = append(b, '}')
	case node.Simplify:
		b = checksumAppend(b, n.Unwrap())
	default:
		b = checksumAppend(b, Decompose(v))
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
