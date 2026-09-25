// Copyright (c) 2020, Peter Ohler, All rights reserved.

package jp

import (
	"github.com/ohler55/ojg/internal/node"
)

// Walk data and call the cb callback for each node in the data. The path is
// reused in each call so if the path needs to be save it should be copied.
func Walk(data any, cb func(path Expr, value any), justLeaves ...bool) {
	path := Expr{Root('$')}
	walk(path, data, cb, 0 < len(justLeaves) && justLeaves[0])
}

func walk(path Expr, data any, cb func(path Expr, value any), justLeaves bool) {
top:
	n := node.Inspect(data)
	switch n.Kind {
	case node.Null, node.Bool, node.Int, node.Float, node.String, node.Bytes, node.Time:
		// leaf node
		cb(path, data)
	case node.Array:
		if !justLeaves {
			cb(path, data)
		}
		pi := len(path)
		path = append(path, nil)
		// Unpack once so the loop ranges without per element dispatch.
		if a, g := n.UnpackArray(); a != nil {
			for i, v := range a {
				path[pi] = Nth(i)
				walk(path, v, cb, justLeaves)
			}
		} else {
			for i, v := range g {
				path[pi] = Nth(i)
				walk(path, v, cb, justLeaves)
			}
		}
	case node.Object:
		if !justLeaves {
			cb(path, data)
		}
		pi := len(path)
		path = append(path, nil)
		// Unpack once so the loop ranges without per entry dispatch.
		if m, g := n.UnpackObject(); m != nil {
			for k, v := range m {
				path[pi] = Child(k)
				walk(path, v, cb, justLeaves)
			}
		} else {
			for k, v := range g {
				path[pi] = Child(k)
				walk(path, v, cb, justLeaves)
			}
		}
	case node.Simplify:
		data = n.Unwrap()
		goto top
	default:
		cb(path, data)
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
