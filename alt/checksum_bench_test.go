// Copyright (c) 2026, Peter Ohler, All rights reserved.

package alt_test

import (
	"fmt"
	"testing"

	"github.com/ohler55/ojg/alt"
)

func checksumBenchData(size int) map[string]any {
	data := map[string]any{}
	for i := 0; i < size; i++ {
		var arr []any
		for j := 0; j < 16; j++ {
			arr = append(arr, map[string]any{
				"ix":    j,
				"name":  fmt.Sprintf("name-%d-%d", i, j),
				"flag":  j%2 == 0,
				"score": float64(j) * 1.5,
			})
		}
		data[fmt.Sprintf("key-%04d", i)] = arr
	}
	return data
}

func BenchmarkChecksumLarge(b *testing.B) {
	data := checksumBenchData(64)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = alt.Checksum(data)
	}
}

func BenchmarkDiffLarge(b *testing.B) {
	d0 := checksumBenchData(64)
	d1 := checksumBenchData(64)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = alt.Diff(d0, d1)
	}
}
