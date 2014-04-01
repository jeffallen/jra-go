package insert

import (
	"math/rand"
	"sort"
	"testing"
)

func BenchmarkIntList(b *testing.B) {
	var l IntList

	// ensure that the same random insertions are done for this test
	// and the other one
	rand.Seed(1)

	for i := 0; i < b.N; i++ {
		x := rand.Int()
		l.Insert(x)
	}

	b.StopTimer()
	if !sort.IntsAreSorted(l.list) {
		b.Fatal("not sorted")
	}
}

func BenchmarkContainerList(b *testing.B) {
  var l ContainerList

	// ensure that the same random insertions are done for this test
	// and the other one
	rand.Seed(1)

	for i := 0; i < b.N; i++ {
		x := rand.Int()
		l.Insert(x)
	}
}

// ex: ts=2
