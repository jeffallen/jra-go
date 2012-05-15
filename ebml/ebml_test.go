package ebml

import (
	"bytes"
	"io"
	"testing"
)

type test struct {
	bytes  []byte
	expect uint64
	err    error
}

var thetests = []test{
	{[]byte{0x3a, 0x41, 0xfe}, 0x1a41fe, nil},
	{[]byte{0x80}, 0x0, nil},
	{[]byte{0x80, 0xff, 0xff}, 0x0, nil}, // extra bytes ok
	{[]byte{0xff}, 0x7f, nil},
	{[]byte{0x41, 0x00}, 0x100, nil},
	{[]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x1f}, 0x1f, nil},
	{[]byte{0x01, 0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff}, 0xff0000000000ff, nil},
	{[]byte{0x41}, 0x100, io.EOF},
	{[]byte{0x00}, 0, BadFormat},
}

func TestVint(t *testing.T) {
	for _, x := range thetests {
		t.Logf("Expecting %v", x.expect)
		r := bytes.NewReader(x.bytes)
		v, err := readVint(r)
		if err != x.err {
			t.Fatal(err)
		} else {
			if uint64(v) != x.expect {
				t.Fatalf("Got %v exepected %v", v, x.expect)
			}
		}
	}
}

// ex: ts=2
