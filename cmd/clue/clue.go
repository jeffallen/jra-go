package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"code.google.com/p/jra-go/jpu"
)

const intro = `
It's a moonless night, you're lucky. Cutting your way through 
the barbed wire fence, you've made your way into the compound.
The front is brightly lit, but you've already noticed that the
rear is much darker. Your target entry point is up ahead,
just a 10 meter sprint across open ground: a door and the control
panel next to it with its slowly brinking red LED. You concentrate
on the red dot and make a run for it.

Kneeling beneath the control panel, you unscrew the cover and
find it right where they told you to look for it, the DIAG_IN
port. You hook up TX, RX, and ground then boot up your portable
terminal. Propping your back against the wall, you settle in
for a little hacking...
`

const help = `
You have successfully connected your teletype to the DIAG_IN port
of the door control computer. You can send the following commands
to its debugger:

	load address byte byte byte ...
		Load the bytes into memory, starting at address.
	dump
		Dump memory
	reg
		See the registers
	reg x y
		Set register x to y
	step
		Run one step
	run
		Run until halt
	exit
		Disconnect the leads and scurry back through the fence.
`

type logger struct{}

func (l logger) Log(args ...interface{}) {
	fmt.Println(args)
}

var havingFun *bool = flag.Bool("havingFun", true, "set this if you're not having fun anymore")
var assemble *string = flag.String("assemble", "", "program to assemble")

func main() {
	// override the normal usage to prevent them from easily seeing
	// the havingFun flag (still might see it with strings, good on 'em)
	flag.Usage = func () { fmt.Fprintf(os.Stderr, "Solve this clue. Duh.\n") }
	flag.Parse()

	if *assemble != "" {
		fmt.Printf("%#v\n", jpu.Assemble(strings.NewReader(*assemble)))
		return
	}

	// two machines, connected by an 8-bit buffer (both
	// see it as location 1). When the front machine (jpu2)
	// writes anything to location 2, the back end is awoken from
	// an InsWait instruction, and is clear to send.
	// When jpu1 writes into location 2, the front end is awoken and
	// is clear to read.
	// jpu2's location 0 is a normal output port for writing the results

	p1 := []byte{
	// init:
	/*
		byte(InsImmReg), 0, 1, 5, // r5 holds address of the output port
		byte(InsImmReg), 0, 2, 2,	// write to 2 to signal the other guy
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
	*/
	}

	logger := logger{}
	jpu1 := jpu.NewProcessor(300)
	jpu2 := jpu.NewProcessor(300)
	if !*havingFun {
		jpu1.Trace("  BigMC", logger)
	}
	jpu2.Trace("DoorCtl", logger)

	// the data buffer between them
	ch := make(chan byte, 1)
	var theOldOne byte
	jpu1.RegisterOut(func(where jpu.Address, what byte) {
		// non-blocking send of the byte to jpu2
		select {
		case ch <- what:
			// nothing to do
		default:
		}
	}, 1)
	jpu2.RegisterIn(func(where jpu.Address) byte {
		// read a byte from the channel, if available return it
		// otherwise, return the old one.
		select {
		case theOldOne = <-ch:
		default:
		}
		return theOldOne
	}, 1)

	// writing to jpu1's addr 2 sends a signal to jpu2 (and vice versa)
	jpu2.RegisterOut(func(where jpu.Address, what byte) { jpu1.Signal() }, 2)
	jpu1.RegisterOut(func(where jpu.Address, what byte) { jpu2.Signal() }, 2)

	// an output port for jpu2
	jpu2.RegisterOut(func(where jpu.Address, what byte) { fmt.Fprintf(os.Stderr, "%c", what) }, 0)

	jpu1.LoadMem(p1, 100)
	jpu1.Reg[0] = 100
	jpu1.LoadMem([]byte("HELLO\000"), 200)

	// start backend machine running indefinitely
	go func() {
		for jpu1.Step() {
		}
	}()

	fmt.Print(intro)
	fmt.Print(help)

	// process commands
	r := bufio.NewReader(os.Stdin)
loop:
	for {
		fmt.Fprintf(os.Stdout, "> ")
		cmd, err := r.ReadString('\n')

		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("Error:", err)
		}
		cmd = strings.TrimRight(cmd, "\n\r")
		cmd = strings.ToLower(cmd)
		cmd = strings.Replace(cmd, "\t", " ", -1)
		tok := strings.Split(cmd, " ")

		switch tok[0] {
		default:
			// print help for all unrecognized commands
			fmt.Print(help)
		case "exit":
			break loop
		case "dump":
			row := 15
			for i := 0; i < jpu2.Top/row; i++ {
				fmt.Printf("%3d: ", i*row)
				for j := 0; j < row; j++ {
					fmt.Printf("%3d ", jpu2.Peek(jpu.Address(i*row+j)))
				}
				fmt.Println("")
			}
		case "load":
			if len(tok) < 2 {
				fmt.Println("Load needs at least one arg.")
			} else {
				a, _ := strconv.ParseInt(tok[1], 0, 16)
				addr := jpu.Address(a)
				for i := 2; i < len(tok); i++ {
					val, err := strconv.ParseInt(tok[i], 0, 8)
					if err == nil {
						jpu2.Poke(addr, byte(val))
						addr++
					} else {
						fmt.Println(err)
					}
				}
			}
		case "reg":
			if len(tok) > 1 {
				if len(tok) != 3 {
					fmt.Println("Expected 2 args.")
				} else {
					reg, _ := strconv.ParseInt(tok[1], 0, 16)
					val, _ := strconv.ParseInt(tok[2], 0, 16)
					if int(reg) < len(jpu2.Reg) {
						jpu2.Reg[int(reg)] = jpu.Address(val)
					} else {
						fmt.Println("No such register.")
					}
				}
			} else {
				// print all
				for j := 0; j < len(jpu2.Reg); j++ {
					fmt.Printf("%3d ", j)
				}
				fmt.Println("")
				for j := 0; j < len(jpu2.Reg); j++ {
					fmt.Printf("%3d ", jpu2.Reg[j])
				}
				fmt.Println("")
			}
		case "step":
			running := jpu2.Step()
			if !running {
				fmt.Println("Halted.")
			}
		case "run":
			for jpu2.Step() {
			}
			fmt.Println("Halted.")
		}
	}
}
