package main

import (
	"fmt"
	"io"
	"os"

	"code.google.com/p/jra-go/ebml"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: dumpmk filename")
		os.Exit(1)
	}
	filename := os.Args[1]

	f, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open %v: %v", filename, err)
		os.Exit(1)
	}

	d := ebml.NewMaster(f)
	dumpAll(d, "")
}

func dumpAll(d ebml.Master, ind string) {
	var err error
	for err == nil {
		var element interface{}
		element, err = d.Next()
		if err != nil {
			break
		}

		switch t := element.(type) {
		case ebml.Header:
			fmt.Printf("%sHeader (DocType: %s)", ind, t.DocType)
		case ebml.Segment:
			fmt.Printf("%sSegment", ind)
			dumpAll(t.Master, ind+"  ")
		case ebml.MetaSeek:
			fmt.Printf("%sMetaSeek (%d seeks)", ind, len(t.Seeks))
		case ebml.Unknown:
			fmt.Printf("%sUnknown EBML element: %x", ind, t.Id)
		default:
			fmt.Printf("%sUnknown element type: %T", ind, element)
		}
	}
	if err != io.EOF {
		fmt.Fprintf(os.Stderr, "Could not read next: %v", err)
	}
}

// ex:ts=2
