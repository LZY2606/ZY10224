// Copyright (c) 2020, Peter Ohler, All rights reserved.

package jp

import (
	"github.com/ohler55/ojg/alt"
	"github.com/ohler55/ojg/internal/node"
)

// Walk data and call the cb callback for each node in the data. The path is
// reused in each call so if the path needs to be save it should be copied.
//
// Map and gen.Object members are visited in the native order of the
// underlying map which is not specified. Values that implement
// alt.Simplifier are simplified before descending. Any other value outside
// the generic data model is reported as a leaf as is.
func Walk(data any, cb func(path Expr, value any), justLeaves ...bool) {
	path := Expr{Root('$')}
	walk(path, node.Of(data), cb, 0 < len(justLeaves) && justLeaves[0])
}

func walk(path Expr, nv node.Value, cb func(path Expr, value any), justLeaves bool) {
top:
	switch nv.Kind() {
	case node.Null, node.Bool, node.Int, node.Uint, node.Float, node.String, node.Bytes, node.Time:
		// leaf node
		cb(path, nv.Raw())
	case node.Array:
		if !justLeaves {
			cb(path, nv.Raw())
		}
		pi := len(path)
		path = append(path, nil)
		for i := 0; i < nv.Len(); i++ {
			path[pi] = Nth(i)
			walk(path, nv.Index(i), cb, justLeaves)
		}
	case node.Object:
		if !justLeaves {
			cb(path, nv.Raw())
		}
		pi := len(path)
		path = append(path, nil)
		nv.Each(func(key string, child node.Value) bool {
			path[pi] = Child(key)
			walk(path, child, cb, justLeaves)
			return true
		})
	case node.Other:
		if simp, ok := nv.Raw().(alt.Simplifier); ok {
			nv = node.Of(simp.Simplify())
			goto top
		}
		cb(path, nv.Raw())
	}
}

// Walk the matching elements in the data and call cb on the matches. The path
// passed to the cb function is the normalized path to the current location
// while the nodes are the chain of elements up to and including the current
// element.
func (x Expr) Walk(data any, cb func(path Expr, nodes []any)) {
	if 0 < len(x) {
		x[0].Walk(x[1:], Expr{}, []any{data}, cb)
	}
}
