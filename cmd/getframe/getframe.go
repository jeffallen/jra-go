package main

import (
	"code.google.com/p/jra-go/ebml"
	//"code.google.com/p/vp8-go/vp8"
	"io"
	"log"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: getframe filename")
	}
	filename := os.Args[1]

	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Could not open %v: %v", filename, err)
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
			log.Printf("%sHeader (DocType: %s)", ind, t.DocType)
		case ebml.Segment:
			log.Printf("%sSegment", ind)
			dumpAll(t.Master, ind + "  ")
		case ebml.MetaSeek:
			log.Printf("%sMetaSeek (%d seeks)", ind, len(t.Seeks))
		case ebml.Unknown:
			log.Printf("%sUnknown EBML element: %x", ind, t.Id)
		default:
			log.Printf("%sUnknown element type: %T", ind, element)
		}
	}
	if err != io.EOF {
		log.Printf("Could not read next: %v", err)
	}
}

/*
	d := vp8.NewDecoder()
	d.Init(f, int(sz))

	framenum := 0
	for framenum < 100 {
		fh, err := d.DecodeFrameHeader()
		if err != nil {
			log.Fatalf("Could not decode frame header %v: %v", framenum, err)
		}
		log.Printf("Num: %v Key: %v Show: %v", framenum, fh.KeyFrame, fh.ShowFrame)

		_, err = d.DecodeFrame()
		if err != nil {
			log.Fatalf("Could not decode frame %v: %v", framenum, err)
		}
		framenum++
	}
*/

// ex:ts=2
