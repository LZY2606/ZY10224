// Copyright (c) 2026, Peter Ohler, All rights reserved.

package jp_test

import (
	"fmt"
	"testing"

	"github.com/ohler55/ojg/gen"
	"github.com/ohler55/ojg/jp"
)

// bigWalkData builds a wide and deep object graph for walk benchmarks.
func bigWalkData(depth, width int) any {
	if depth == 0 {
		return map[string]any{"leaf": 1}
	}
	obj := map[string]any{}
	for i := 0; i < width; i++ {
		obj[fmt.Sprintf("k%d", i)] = bigWalkData(depth-1, width)
	}
	return obj
}

func bigWalkGenData(depth, width int) gen.Node {
	if depth == 0 {
		return gen.Object{"leaf": gen.Int(1)}
	}
	obj := gen.Object{}
	for i := 0; i < width; i++ {
		obj[fmt.Sprintf("k%d", i)] = bigWalkGenData(depth-1, width)
	}
	return obj
}

func BenchmarkWalkBigObject(b *testing.B) {
	data := bigWalkData(6, 5)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jp.Walk(data, func(path jp.Expr, value any) {})
	}
}

func BenchmarkWalkBigGenObject(b *testing.B) {
	data := bigWalkGenData(6, 5)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jp.Walk(data, func(path jp.Expr, value any) {})
	}
}

func BenchmarkWalkBigArray(b *testing.B) {
	var build func(depth int) any
	build = func(depth int) any {
		if depth == 0 {
			return 1
		}
		a := make([]any, 5)
		for i := range a {
			a[i] = build(depth - 1)
		}
		return a
	}
	data := build(8)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jp.Walk(data, func(path jp.Expr, value any) {})
	}
}
