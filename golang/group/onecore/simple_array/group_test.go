package simple_array

import "testing"

func TestGroupByOsAndSumByPopularity(t *testing.T) {
	GroupByOsAndSumByPopularity()
}

func BenchmarkGroupByOsAndSumByPopularity(b *testing.B) {
	for n := 0; n < b.N; n++ {
		GroupByOsAndSumByPopularity()
	}
}
