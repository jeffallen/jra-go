package main

import (
	"code.google.com/p/jra-go/insert"
	"flag"
	"fmt"
	"math/rand"
	"time"
)

var linear *bool = flag.Bool("linear", false, "do linear insert")
var linked *bool = flag.Bool("linked", false, "do linked list insert")

func do(f func(int)) {
}

func main() {
	flag.Parse()

  var l insert.Inserter
	if *linear {
		l = insert.IntList{}
	}
	if *linked {
		l = insert.ContainerList{}
	}

	elapsed := time.Duration(0)

	for i := 0; elapsed < 200*time.Millisecond; i += 10000 {
		start := time.Now()
		BenchmarkList(i, l)
		elapsed = time.Since(start)
		fmt.Printf("%v %d\n", i, elapsed/time.Millisecond)
	}
}

func BenchmarkList(n int, i insert.Inserter) {
	// ensure that the same random insertions are done for every test
	rand.Seed(1)

	for j := 0; j < n; j++ {
		x := rand.Int()
		i.Insert(x)
	}
}

// ex: ts=2
