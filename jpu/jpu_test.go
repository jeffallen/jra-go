package jpu

import (
	"bytes"
	"testing"
)

type logit struct {
	t *testing.T
}

func (l logit)Log(msg string) {
	l.t.Log(msg)
}

func TestIO(t *testing.T) {
	var expected byte = 99
	var got byte

	program := []byte{
		byte(InsNop),
		byte(InsImmReg), 0, 11, 2,
		byte(InsImmReg), 0, 12, 3,
		byte(InsMemReg), 2, 1,
		byte(InsRegMem), 1, 3,
		byte(InsHalt),
	}

	it := NewProcessor(1000)
	it.LoadMem(program, 100)
	it.RegisterIn(func(where Address) byte { return expected }, 11)
	it.RegisterOut(func(where Address, what byte) { got = what }, 12)

	it.Trace("cpu", logit{t:t})
	it.Reg[0] = 100
	for {
		if !it.Step() {
			break
		}
	}
	if got != expected {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestHello(t *testing.T) {
	var buf bytes.Buffer

	program := []byte{
		byte(InsImmReg), 0, 0, 2,
		byte(InsImmReg), 0, 72, 1,
		byte(InsRegMem), 1, 2,
		byte(InsImmReg), 0, 69, 1,
		byte(InsRegMem), 1, 2,
		byte(InsImmReg), 0, 76, 1,
		byte(InsRegMem), 1, 2,
		byte(InsImmReg), 0, 76, 1,
		byte(InsRegMem), 1, 2,
		byte(InsImmReg), 0, 79, 1,
		byte(InsRegMem), 1, 2,
		byte(InsHalt),
	}

	it := NewProcessor(1000)
	it.Trace("cpu", logit{t:t})
	it.RegisterOut(func(where Address, what byte) { buf.WriteByte(what) }, 0)

	it.LoadMem(program, 100)
	it.Reg[0] = 100 // set initial PC

	for it.Step() {
	}
	t.Logf("result: %v", string(buf.Bytes()))
	if !bytes.Equal(buf.Bytes(), []byte("HELLO")) {
		t.Fatal("not hello")
	}
}

func TestAssemble(t *testing.T) {
	input := `
# this is a test
	org 100
top: immreg top 0
	regmem 0 0
	memreg 0 0
	nop
	gotoIfEqual stop 0x80 0
stop:
	halt
	raw 1 2 3
`
	expected := []byte{0x3, 0x0, 0x0, 0x2, 0x0, 0x0, 0x1, 0x9, 0x0, 0x70, 0x80, 0x0, 0x0, 0x0, 0x1, 0x2, 0x3}

	got, org := Assemble(input)
	if !bytes.Equal(got, expected) {
		t.Fatalf("Got %#v, expected %#v:", got, expected)
	}
	if org != 100 {
		t.Fatal("Org is wrong.")
	}
}
