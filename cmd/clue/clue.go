package main

import (
	"bufio"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

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
	quiet
		Turn off instruction logging
	init
		Reinitialize (same as exit and run the program again)
	exit
		Disconnect the leads and scurry back through the fence.

As a shortcut, you can type just the first letter of any command.
`

type logger struct{}

func (l logger) Log(msg string) {
	fmt.Println(msg)
}

const includeEncode = false
const includeAssemble = true

var restart *bool = flag.Bool("restart", false,
	"for internal use only")
var havingFun *bool = flag.Bool("havingFun", true,
	"set this to false if you're not having fun anymore")
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
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Error:", r)
		}
	}()

	// override the normal usage to prevent them from easily seeing
	// the havingFun flag (still might see it with strings, good on 'em)
	flag.Usage = func() { fmt.Fprintf(os.Stderr, "Solve this clue. Duh.\n") }
	flag.Parse()

	if includeAssemble && *assemble != "" {
		f, err := os.Open(*assemble)
		if err != nil {
			fmt.Print("Error: ", err)
			os.Exit(1)
		}
		all, err := ioutil.ReadAll(f)
		if err != nil {
			fmt.Print("Error: ", err)
			os.Exit(1)
		}
		res, org := jpu.Assemble(string(all))

		if *assemble == "bigmac.src" {
			fmt.Printf("%#v\n", res)
		} else {
			fmt.Printf("load %d ", org)
			for _, x := range res {
				fmt.Printf("%d ", x)
			}
			fmt.Println()
		}

		fmt.Printf("reg 0 %d\n", org)
		fmt.Println("quiet")
		fmt.Println("go")
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

	logger := logger{}
	jpu1 := jpu.NewProcessor(1000)
	jpu2 := jpu.NewProcessor(300)

	if !*havingFun {
		jpu1.Trace(" BIGMAC", logger)
	}
	jpu2.Trace("DoorCtl", logger)

	var theBuffer byte

	f := func(where jpu.Address, what byte) { theBuffer = what }
	jpu1.RegisterOut(f, 1)
	jpu2.RegisterOut(f, 1)

	g := func(where jpu.Address) byte { return theBuffer }
	jpu1.RegisterIn(g, 1)
	jpu2.RegisterIn(g, 1)

	// an output port for jpu2
	jpu2.RegisterOut(func(where jpu.Address, what byte) { fmt.Fprintf(os.Stderr, "%c", what) }, 0)

	// from assembling bigmac.src
	prog := []byte{0x4, 0x0, 0x1, 0x1, 0x4, 0x0, 0xc8, 0x3, 0x4, 0x0, 0x0, 0x6, 0x2, 0x3, 0x4, 0x4, 0x0, 0x0, 0x8, 0x9, 0x0, 0x92, 0x4, 0x8, 0x4, 0x0, 0x40, 0x8, 0x7, 0x8, 0x4, 0x5, 0x4, 0x5, 0x6, 0x5, 0x4, 0x6, 0x5, 0x4, 0x6, 0x6, 0x4, 0x5, 0x5, 0x6, 0x3, 0x4, 0x1, 0x4, 0x0, 0x1, 0x8, 0x6, 0x8, 0x3, 0x4, 0x0, 0xff, 0x8, 0x2, 0x1, 0x9, 0xa, 0x0, 0xa0, 0x9, 0x8, 0x4, 0x0, 0x0, 0x8, 0x9, 0x0, 0x68, 0x4, 0x8, 0x4, 0x0, 0x70, 0x0}
	jpu1.LoadMem(prog, 100)
	jpu1.Reg[0] = 100

	// compile with includeEncode and use ./clue -encode="XXX" to make these
	mask := []byte{0xc2, 0x87, 0x9c, 0x2a, 0xdf, 0xd2, 0x83, 0x4c, 0xa2, 0x2f, 0xd7, 0x1c, 0xa2, 0xb5, 0x9, 0x8e, 0x52, 0xf, 0xf6, 0x4f, 0xd4, 0x77, 0xa, 0xa5, 0x16, 0x8b, 0x20, 0x36, 0x21, 0xc5, 0x74, 0x85, 0x95, 0xa2, 0xcb, 0xa1, 0x48, 0xcf, 0x9b, 0x0}
	text := []byte{0x96, 0xcf, 0xd9, 0x6a, 0x9c, 0x9d, 0xc7, 0x9, 0xe2, 0x7b, 0x98, 0x5c, 0xed, 0xe5, 0x4c, 0xc0, 0x12, 0x5b, 0xbe, 0xa, 0x94, 0x33, 0x45, 0xea, 0x44, 0xcb, 0x69, 0x65, 0x61, 0x88, 0x3d, 0xd6, 0xc6, 0xf2, 0x82, 0xe6, 0xf, 0x96, 0xdb, 0x0}

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
			time.Sleep(10 * time.Millisecond)
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
		case "q":
			fallthrough
		case "quiet":
			jpu1.Trace("", nil)
			jpu2.Trace("", nil)
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
			runtime.Gosched()
			if !running {
				fmt.Println("Halted.")
			}
		case "g":
			fallthrough
		case "go":
			for jpu2.Step() {
				// make the door run 1/5 as fast, so that BIGMAC beats it to
				// the first spin loop
				time.Sleep(50 * time.Millisecond)
				runtime.Gosched()
			}
			fmt.Println("Halted.")
		}
	}
}

// ex:ts=2
