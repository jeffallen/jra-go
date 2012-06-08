package main

import (
	"bufio"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"

	"code.google.com/p/jra-go/jpu"
)

const intro = `
It's a moonless night, you're lucky. Cutting your way through 
the barbed wire fence, you've made your way into the compound.
Your target up ahead, just a 10 meter sprint across open ground:
a service door and the control panel next to it with its slowly
blinking red LED. You concentrate on the red dot and make a
run for it.

Kneeling beneath the control panel, you unscrew the cover and
find it right where they told you to look for it, the DIAG_IO
port. You hook up Tx, Rx, and ground then boot up your portable
terminal. Propping your back against the wall, you settle in
for a little hacking...
`

const help = `
You have successfully connected your terminal to the DIAG_IO port
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
	go
		Run until halt
	init
		Reinitialize (same as exit and run the program again)
	exit
		Disconnect the leads and scurry back through the fence.

As a shortcut, you can type just the first letter of any command.
`

const dead = `
The red LED stops blinking. Nothing you do changes anything, the
door controller seems to be completely hung. No sense waiting around
here. As you start to pack up your gear you hear the barking
of approaching guard dogs from around the corner.

You make a break for it, but the last thing you see as a
dog sinks his teeth into your forearm is the blinding
flashlight of the guard.

GAME OVER.
`

type logger struct{}

func (l logger) Log(msg string) {
	fmt.Println(msg)
}

const includeEncode = true
const includeAssemble = true

var restart *bool = flag.Bool("restart", false, "for internal use only")
var havingFun *bool = flag.Bool("havingFun", false, "set this to false if you're not having fun anymore")
var assemble *string
var encode *string

func init() {
	if includeEncode {
		encode = flag.String("encode", "", "string to hide")
	}
	if includeAssemble {
		assemble = flag.String("assemble", "", "program to assemble")
	}
}

func main() {
	// panic handler
/*
	defer func() {
		if r := recover(); r != nil {
			if r == "would deadlock" {
				fmt.Print(dead)
				os.Exit(1)
			}
			fmt.Println("Error:", r)
		}
	}()
*/

	// override the normal usage to prevent them from easily seeing
	// the havingFun flag (still might see it with strings, good on 'em)
	flag.Usage = func() { fmt.Fprintf(os.Stderr, "Solve this clue. Duh.\n") }
	flag.Parse()

	if includeAssemble && *assemble != "" {
		fmt.Printf("load 100 ")
		res := jpu.Assemble(strings.NewReader(*assemble))
		for _, x := range res {
			fmt.Printf("%d ", x)
		}
		fmt.Println("")
		return
	}

	// this feature is only compiled in when includeEncode is set
	// it creates an encrypted version of the answer using a one-time-pad
	// so that strings on the binary can't find the answer.
	// the answer is decrypted before it is loaded into jpu1's RAM
	if includeEncode && *encode != "" {
		in := []byte(*encode)
		mask := make([]byte, len(in))
		text := make([]byte, len(in))

		_, _ = rand.Read(mask)
		for i, _ := range in {
			for {
				text[i] = mask[i] ^ in[i]
				// do not allow zeros in the encrypted results
				if text[i] == 0 {
					mask[i]++
					// go try again
				} else {
					break
				}
			}
		}

		// add terminators
		mask = append(mask, 0)
		text = append(text, 0)

		fmt.Printf("mask := %#v\n", mask)
		fmt.Printf("text := %#v\n", text)
		for i, _ := range in {
			text[i] = mask[i] ^ text[i]
		}
		fmt.Printf("//check: %s\n", string(text))
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
		byte(jpu.InsImmReg), 0, 1, 7, // r7 holds address of the output port
		byte(jpu.InsImmReg), 0, 2, 2, // write to 2 to signal the other guy
		byte(jpu.InsImmReg), 0, 200, 3, // r3 = start of message
		byte(jpu.InsImmReg), 0, 0, 6, // r6 = previous letter
		// top: wait until signaled to send
		byte(jpu.InsWait),
		// read next byte of message (*r3)
		byte(jpu.InsMemReg), 3, 4,
		// if byte just read is zero, go direct to write
		byte(jpu.InsImmReg), 0, 0, 8,
		byte(jpu.InsGotoIfEqual), 0, 151, 4, 8,
		// encrypt: r4-=64, r4->r5(to use next time), r4*=3, r4+=r6(last one), r5=r6
		byte(jpu.InsImmReg), 0, 64, 8,
		byte(jpu.InsSubReg), 8, 4,
		// remember it for next char
		byte(jpu.InsMovReg), 4, 5,
		// r4*=3
		byte(jpu.InsAddReg), 5, 4,
		byte(jpu.InsAddReg), 5, 4,
		// add last letter to this one
		byte(jpu.InsAddReg), 6, 4,
		// remember it for next time
		byte(jpu.InsMovReg), 4, 6,
		// write to output, incr, signal
		byte(jpu.InsRegMem), 4, 7,	// 151
		// r3++
		byte(jpu.InsImmReg), 0, 1, 8,
		byte(jpu.InsAddReg), 8, 3,
		byte(jpu.InsRegMem), 2, 2, // signal jpu2
		// if we just wrote zero, loop with reinit
		byte(jpu.InsImmReg), 0, 0, 8,
		byte(jpu.InsGotoIfEqual), 0, 108, 4, 8,
		// loop without reset
		byte(jpu.InsImmReg), 0, 116, 0,
	}

	logger := logger{}
	jpu1 := jpu.NewProcessor(1000)
	jpu2 := jpu.NewProcessor(300)

	// this is for the deadlock-preemption hack
	jpu1.Peer = jpu2
	jpu2.Peer = jpu1

	if !*havingFun {
		jpu1.Trace(" BIGMAC", logger)
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

	// compile with includeEncode and use ./clue -encode="XXX" to make these
	mask := []byte{0xb0, 0xa6, 0xcb, 0xa9, 0x14, 0x2a, 0x3f, 0x64, 0x65, 0xeb, 0x1d, 0x80, 0x9a, 0xb0, 0x26, 0x7b, 0x5, 0xce, 0xb3, 0x11, 0xcf, 0xbf, 0xe9, 0x9d, 0x7c, 0xf5, 0x3a, 0xa5, 0x29, 0x9f, 0xb4, 0xf3, 0x76, 0x9c, 0x3, 0x7e, 0x2a, 0x79, 0x8a, 0xaa, 0xb7, 0x83, 0xb3, 0x48, 0x9c, 0xd5, 0xde, 0x99, 0x20, 0x65, 0x91, 0x3, 0xdf, 0x51, 0xa6, 0x1f, 0xdd, 0x1e, 0x1f, 0x3b, 0xb5, 0x18, 0x5f, 0xe9, 0x85, 0xe2, 0x73, 0xc9, 0x49, 0xc7, 0xb, 0x39, 0x95, 0xbb, 0xc, 0x21, 0x2c, 0x3e, 0x11, 0xf7, 0x72, 0x9f, 0xb0, 0xa3, 0xd, 0x84, 0xa7, 0x7a, 0xec, 0xb5, 0xe1, 0x5e, 0x86, 0xeb, 0xd6, 0x12, 0xba, 0x83, 0x43, 0x87, 0x36, 0xb, 0x96, 0x81, 0xcd, 0x5, 0x51, 0xdc, 0x95, 0xd9, 0xd3, 0x2b, 0x82, 0xe1, 0xda, 0xac, 0x1e, 0xdf, 0xa1, 0x1c, 0xde, 0xd8, 0x53, 0x68, 0x60, 0x1, 0xa7, 0x4a, 0x47, 0x38, 0xed, 0xcd, 0xc7, 0x53, 0x94, 0x21, 0x3, 0x57, 0xd8, 0x30, 0xac, 0x10, 0xe4, 0xe2, 0x97, 0xe1, 0x4f, 0x3b, 0xa2, 0x21, 0x38, 0x3e, 0x63, 0xa0, 0xc0, 0x1a, 0x6b, 0xdc, 0x4e, 0xdb, 0x6b, 0x13, 0xc, 0xb6, 0x17, 0x57, 0x47, 0x2d, 0x89, 0x58, 0x5d, 0x17, 0xb7, 0x68, 0x4d, 0x6a, 0x5e, 0xf, 0x60, 0x1c, 0xf8, 0xfc, 0x70, 0x82, 0x2f, 0x8d, 0x67, 0x73, 0xb6, 0x4d, 0x40, 0x60, 0xc4, 0xca, 0x13, 0x7f, 0x9, 0xb2, 0x55, 0x8b, 0xef, 0xbc, 0x41, 0x34, 0x6f, 0xeb, 0xbd, 0x9e, 0xe9, 0xc0, 0xb5, 0x27, 0}
	text := []byte{0xf3, 0xc9, 0xa5, 0xce, 0x66, 0x4b, 0x4b, 0x11, 0x9, 0x8a, 0x69, 0xe9, 0xf5, 0xde, 0x55, 0x57, 0x25, 0xb7, 0xdc, 0x64, 0xef, 0xd7, 0x88, 0xeb, 0x19, 0xd5, 0x52, 0xc4, 0x4a, 0xf4, 0xd1, 0x97, 0x56, 0xf5, 0x6d, 0xa, 0x45, 0x59, 0xfe, 0xc2, 0xd2, 0xa3, 0xde, 0x29, 0xf5, 0xbb, 0xb8, 0xeb, 0x41, 0x8, 0xf4, 0x23, 0xbe, 0x3f, 0xc2, 0x3f, 0xb3, 0x71, 0x68, 0x1b, 0xcc, 0x77, 0x2a, 0xc9, 0xe4, 0x90, 0x16, 0xe9, 0x28, 0xa5, 0x64, 0x4c, 0xe1, 0x9b, 0x78, 0x4e, 0xc, 0x4c, 0x74, 0x94, 0x17, 0xf6, 0xc6, 0xc6, 0x2d, 0xf0, 0xcf, 0x1f, 0xcc, 0xd8, 0x80, 0x39, 0xef, 0x88, 0xf6, 0x79, 0xdf, 0xfa, 0x63, 0xf3, 0x59, 0x2b, 0xe6, 0xf3, 0xa2, 0x73, 0x34, 0xfc, 0xe1, 0xb6, 0xf3, 0x52, 0xed, 0x94, 0xa8, 0x8c, 0x7c, 0xba, 0xcf, 0x79, 0xa8, 0xb7, 0x3f, 0xd, 0xe, 0x75, 0x87, 0x2e, 0x2e, 0x5b, 0x99, 0xac, 0xb3, 0x3c, 0xe6, 0x1, 0x65, 0x38, 0xaa, 0x10, 0xc0, 0x79, 0x82, 0x87, 0xbb, 0xc1, 0xb, 0x5e, 0xc7, 0x40, 0x56, 0x50, 0x4f, 0x80, 0xb4, 0x72, 0xa, 0xa8, 0x6e, 0xa2, 0x4, 0x66, 0x2b, 0xc0, 0x72, 0x77, 0x34, 0x42, 0xe5, 0x2e, 0x38, 0x73, 0x97, 0x1c, 0x25, 0xf, 0x7e, 0x6c, 0xc, 0x69, 0x9d, 0xc6, 0x50, 0xd1, 0x7e, 0xd8, 0x2e, 0x37, 0x96, 0xf, 0x12, 0x25, 0x85, 0x81, 0x55, 0x3e, 0x5a, 0xe6, 0x7b, 0xab, 0xbb, 0xd4, 0x20, 0x40, 0x4f, 0x82, 0xce, 0xbe, 0x88, 0xac, 0xd9, 0x9, 0}

	// decrypt the text before putting it in the RAM of the sending machine
	for i := 0; i < len(text); i++ {
		text[i] = text[i] ^ mask[i]
	}
	jpu1.LoadMem(text, 200)

	// start backend machine running indefinitely
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Error in BIGMAC:", r)
				os.Exit(0)
			}
		}()

		for jpu1.Step() {
		}
	}()

	if !*restart {
		fmt.Print(intro)
		fmt.Print(help)
	}

	// process commands
	r := bufio.NewReader(os.Stdin)
loop:
	for {
		// prompt and get input
		fmt.Fprintf(os.Stdout, "> ")
		cmd, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("Error:", err)
		}

		// process it
		cmd = strings.ToLower(cmd)
		tok := strings.Fields(cmd)

		if len(tok) < 1 {
			fmt.Print(help)
			continue
		}

		switch tok[0] {
		default:
			fmt.Println("Command not recognized. Type help for a reminder.")
		case "h":
			fallthrough
		case "help":
			fmt.Print(help)
		case "i":
			fallthrough
		case "init":
			fmt.Println("Time warp: ...you settle in for a little hacking...")
			syscall.Exec(os.Args[0], []string{os.Args[0], "-restart"}, nil)
			panic("exec failed")
		case "e":
			fallthrough
		case "exit":
			break loop
		case "d":
			fallthrough
		case "dump":
			row := 15
			for i := 0; i < jpu2.Top/row; i++ {
				fmt.Printf("%3d: ", i*row)
				for j := 0; j < row; j++ {
					fmt.Printf("%3d ", jpu2.Peek(jpu.Address(i*row+j)))
				}
				fmt.Println("")
			}
		case "l":
			fallthrough
		case "load":
			if len(tok) < 2 {
				fmt.Println("Load needs at least one arg.")
			} else {
				a, _ := strconv.ParseInt(tok[1], 0, 16)
				addr := jpu.Address(a)
				for i := 2; i < len(tok); i++ {
					val, err := strconv.ParseInt(tok[i], 0, 16)
					if err == nil {
						jpu2.Poke(addr, byte(val))
						addr++
					} else {
						fmt.Println(err)
					}
				}
			}
		case "r":
			fallthrough
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
		case "s":
			fallthrough
		case "step":
			running := jpu2.Step()
			if !running {
				fmt.Println("Halted.")
			}
		case "g":
			fallthrough
		case "go":
			for jpu2.Step() {
			}
			fmt.Println("Halted.")
		}
	}
}

// ex:ts=2
