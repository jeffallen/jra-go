package jpu

import (
	"bytes"
	"strings"
	"testing"
)

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

	it.Trace("cpu", t)
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
	it.Trace("cpu", t)
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

func TestLoop(t *testing.T) {
	var buf bytes.Buffer

	program := []byte{
		byte(InsImmReg), 0, 0, 9,
		//   r1 = start of data
		byte(InsImmReg), 0, 200, 1,
		// top: load character at r1 into r2		(108)
		byte(InsMemReg), 1, 2,
		//   if 0 (r9), goto done
		byte(InsGotoIfEqual), 0, 125, 2, 9,
		//   print r2
		byte(InsRegMem), 2, 9,
		//   increment r1
		byte(InsIncReg), 1,
		//   goto top
		byte(InsImmReg), 0, 108, 0,
		// done: halt
		byte(InsHalt),
	}

	it := NewProcessor(1000)
	it.Trace("cpu", t)
	it.RegisterOut(func(where Address, what byte) { buf.WriteByte(what) }, 0)

	it.LoadMem(program, 100)
	it.LoadMem([]byte("This is a test.\000"), 200)

	it.Reg[0] = 100 // set initial PC

	for it.Step() {
	}
	t.Logf("result: %v", string(buf.Bytes()))
	if !bytes.Equal(buf.Bytes(), []byte("This is a test.")) {
		t.Fatal("not equal")
	}
}

func TestCall(t *testing.T) {
	program := []byte{
		byte(InsImmReg), 1, 50, 9, // set up stack up at 256+50 = 306
		byte(InsCall), 0, 108, // call it to test call mechanism
		byte(InsHalt),
		byte(InsImmReg), 0, 99, 1, // subroutiune: set r1 = 99
		byte(InsReturn),
	}

	it := NewProcessor(1000)
	it.Trace("cpu", t)

	it.LoadMem(program, 100)
	it.Reg[0] = 100 // set initial PC
	for it.Step() {
	}

	if it.Reg[0] != 107 {
		t.Fatal("ip is", it.Reg[0])
	}
	if it.Reg[1] != 99 {
		t.Fatal("reg1 is", it.Reg[1])
	}
}

func TestCommunicate(t *testing.T) {
	var buf bytes.Buffer

	// two machines, connected by an 8-bit buffer (both
	// see it as location 1). When the front machine (jpu2)
	// writes anything to location 2, the back end is awoken from
	// an InsWait instruction, and is clear to send.
	// When jpu1 writes into location 2, the front end is awoken and
	// is clear to read.
	// jpu2's location 0 is a normal output port for writing the results

	p1 := []byte{
		// init:
		byte(InsImmReg), 0, 1, 5, // r5 holds address of the output port
		byte(InsImmReg), 0, 2, 2, // write to 2 to signal the other guy
		byte(InsImmReg), 0, 0, 8, // for compare to end of string
		byte(InsImmReg), 0, 200, 3, // r3 = start of message
		// top: wait until signaled to send
		byte(InsWait),
		// read next byte of message (*r3)
		byte(InsMemReg), 3, 4,
		// write to output, incr, signal
		byte(InsRegMem), 4, 5,
		byte(InsIncReg), 3,
		byte(InsRegMem), 2, 2, // signal jpu2
		// if byte just sent is zero, reset to start of string (top - 4)
		byte(InsGotoIfEqual), 0, 112, 4, 8,
		// otherwise loop without reset
		byte(InsImmReg), 0, 116, 0,
	}
	p2 := []byte{
		byte(InsImmReg), 0, 1, 1,
		byte(InsImmReg), 0, 2, 2,
		byte(InsImmReg), 0, 0, 8,
		byte(InsRegMem), 2, 2, // signal jpu1
		byte(InsWait),
		byte(InsMemReg), 1, 3, // bring in a byte
		byte(InsGotoIfEqual), 0, 132, 3, 8, // check for zero: exit
		byte(InsRegMem), 3, 8, // output the byte
		byte(InsImmReg), 0, 112, 0, // loop
		byte(InsHalt),
	}

	jpu1 := NewProcessor(1000)
	jpu2 := NewProcessor(1000)
	jpu1.Trace("jpu1", t)
	jpu2.Trace("jpu2", t)

	// the data buffer between them
	ch := make(chan byte, 1)
	var theOldOne byte
	jpu1.RegisterOut(func(where Address, what byte) {
		// non-blocking send of the byte to jpu2
		select {
		case ch <- what:
			// nothing to do
		default:
		}
	}, 1)
	jpu2.RegisterIn(func(where Address) byte {
		// read a byte from the channel, if available return it
		// otherwise, return the old one.
		select {
		case theOldOne = <-ch:
		default:
		}
		return theOldOne
	}, 1)

	// writing to jpu1's addr 2 sends a signal to jpu2 (and vice versa)
	jpu2.RegisterOut(func(where Address, what byte) { jpu1.Signal() }, 2)
	jpu1.RegisterOut(func(where Address, what byte) { jpu2.Signal() }, 2)

	// an output port for jpu2
	jpu2.RegisterOut(func(where Address, what byte) { buf.WriteByte(what) }, 0)

	jpu1.LoadMem(p1, 100)
	jpu1.Reg[0] = 100
	jpu1.LoadMem([]byte("HELLO\000"), 200)
	jpu2.LoadMem(p2, 100)
	jpu2.Reg[0] = 100

	done := make(chan bool, 2)

	go func() {
		jpu1.StepN(1000)
	}()
	go func() {
		jpu2.StepN(1000)
		done <- true
	}()

	// wait for jpu2 to halt
	_ = <-done

	got := string(buf.Bytes())
	expected := "HELLO"
	if got != expected {
		t.Fatal("got %v, expected %v", got, expected)
	}
}

func TestAssemble(t *testing.T) {
	input := `
# this is a test
	org 100
top: immreg top 0
	regmem 0 0
	memreg 0 0
	wait
	nop
	gotoIfEqual top 0x80 0
	halt
`
	expected := []byte{4, 0, 100, 0, 3, 0, 0, 2, 0, 0, 9,
		1, 6, 0, 100, 128, 0, 0}

	got := Assemble(strings.NewReader(input))
	if !bytes.Equal(got, expected) {
		t.Fatalf("Got %v, expected %v:", got, expected)
	}
}
