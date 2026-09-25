// Copyright (c) 2026, Peter Ohler, All rights reserved.

package ojg

import (
	"fmt"
	"strings"
)

// CycleError describes a cyclic reference detected while traversing or
// converting a Go value. Structural traversals can not terminate on cyclic
// data so the cycle is reported instead of recursing until the stack
// overflows.
//
// The Path holds the keys and indexes from the root of the traversed value
// to the node that closes the cycle. A string element is an object key and
// an int element is an array index, the same convention used by alt.Path.
type CycleError struct {
	Path []any
}

// Error returns a string representation of the error.
func (e *CycleError) Error() string {
	var b strings.Builder

	b.WriteString("cyclic reference detected at ")
	if len(e.Path) == 0 {
		b.WriteByte('$')
	}
	for i, p := range e.Path {
		switch tp := p.(type) {
		case int:
			_, _ = fmt.Fprintf(&b, "[%d]", tp)
		case string:
			if 0 < i {
				b.WriteByte('.')
			}
			b.WriteString(tp)
		default:
			_, _ = fmt.Fprintf(&b, "[%v]", tp)
		}
	}
	return b.String()
}
