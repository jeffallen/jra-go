package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"code.google.com/p/jra-go/stl"
	"code.google.com/p/jra-go/debugreader"
)

var debug = flag.Bool("debug", false, "debug I/O")
var dump = flag.Bool("dump", false, "dump all triangles")
var check = flag.Bool("check", false, "check STL file for errors")

func main() {
	flag.Parse()

	var r io.Reader
	r = os.Stdin
	if *debug {
		r = debugreader.NewReader(r)
	}

	s, _ := stl.Decode(r)
	fmt.Printf("Num triangles: %d\n", len(s.Triangles))
	fmt.Printf("From: %v\n", s.Bounds.Min)
	fmt.Printf("  To: %v\n", s.Bounds.Max)

	for n,t := range s.Triangles {
if *check {
	if !t.Normal.IsNormal() {
		fmt.Printf("Triangle %d normal vector: abs(%v) != 1\n", t.Normal)
	}
}

if *dump {
		fmt.Printf("Triangle %d:\n", n)
		fmt.Printf("  %v\n", t.Normal)
		fmt.Printf("  %v\n", t.Vertex[0])
		fmt.Printf("  %v\n", t.Vertex[1])
		fmt.Printf("  %v\n", t.Vertex[2])
		fmt.Printf("  %#x\n", t.Attr)
}
	}
}
