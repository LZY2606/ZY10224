// Copyright (c) 2026, Peter Ohler, All rights reserved.

package jp_test

import (
	"fmt"
	"testing"

	"github.com/ohler55/ojg/gen"
	"github.com/ohler55/ojg/jp"
)

// walkBenchData builds a large nested data tree with the given number of
// top level members. Each member is an array of 16 nested maps.
func walkBenchData(size int) map[string]any {
	data := map[string]any{}
	for i := 0; i < size; i++ {
		var arr []any
		for j := 0; j < 16; j++ {
			arr = append(arr, map[string]any{
				"ix":    j,
				"name":  fmt.Sprintf("name-%d-%d", i, j),
				"flag":  j%2 == 0,
				"score": float64(j) * 1.5,
				"nil":   nil,
			})
		}
		data[fmt.Sprintf("key-%04d", i)] = arr
	}
	return data
}

func BenchmarkWalkLarge(b *testing.B) {
	data := walkBenchData(64)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cnt := 0
		jp.Walk(data, func(path jp.Expr, value any) { cnt++ })
	}
}

func BenchmarkWalkLargeLeaves(b *testing.B) {
	data := walkBenchData(64)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cnt := 0
		jp.Walk(data, func(path jp.Expr, value any) { cnt++ }, true)
	}
}

func BenchmarkWalkGenLarge(b *testing.B) {
	simple := walkBenchData(64)
	data := gen.Object{}
	for k, v := range simple {
		var arr gen.Array
		for range v.([]any) {
			arr = append(arr, gen.Object{"x": gen.Int(1), "s": gen.String("abc")})
		}
		data[k] = arr
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cnt := 0
		jp.Walk(data, func(path jp.Expr, value any) { cnt++ })
	}
}
