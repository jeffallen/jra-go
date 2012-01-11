//	Copyright 2009 Jeff R. Allen. All rights reserved.
//	Use of this source code is governed by a BSD-style
//	license that can be found in the LICENSE file of the Go
//	distribution.

package main

import (
	"os"
	"flag"
	"fmt"
	"syscall"
	"unsafe"
	"log"
	"net"
	"gob"
)

type connCmd int

const (
	Listen connCmd = iota
	Write
	Take
	Steal
	Close
)

type connReq struct {
	Cmd   connCmd
	Name  string
	Input []byte
}

type connReplyCode int

const (
	Ok connReplyCode = iota
	Err
	ReadWrite
	ReadOnly
	Data
)

type connReply struct {
	Code connReplyCode
	Err  string
	Data []byte
}

func debug(arg ...interface{}) {
	if *debugFlag {
		log.Print(arg...)
		buf := []byte{'\r', '\n'}
		os.Stdout.Write(buf[:])
	}
}

func take(enc *gob.Encoder, ts connCmd) {
	req := &connReq{Cmd: ts}
	// we ignore the error because there's not much to do; the
	// main loop will figure it out
	_ = enc.Encode(req)
}

var debugFlag *bool = flag.Bool("debug", false, "show debug messages")
var readwrite bool
var cons string

func main() {
	flag.Parse()
	if flag.NArg() > 0 {
		cons = flag.Arg(0)
	} else {
		log.Fatal("Must give a console name as the last argument.")
	}

	oldterm, err := getTermios(os.Stdin)
	if err != nil {
		log.Fatal("get termios: ", err)
	}

	// put it back the way we found it when we exit
	defer func() {
		_ = setTermios(os.Stdin, oldterm)
		// clean up cursor position
		fmt.Print("\r")
	}()

	err = tty_raw(os.Stdin, oldterm)
	if err != nil {
		log.Print("tty raw: ", err)
		return
	}

	c, err := net.Dial("tcp", "localhost:1234")
	if err != nil {
		log.Print("Could not connect: ", err)
		return
	}
	enc := gob.NewEncoder(c)

	req := new(connReq)
	req.Cmd = Listen
	req.Name = cons
	err = enc.Encode(req)
	if err != nil {
		log.Print("Could not write request: ", err)
		return
	}
	take(enc, Take)

	go readinput(c, enc)

	dec := gob.NewDecoder(c)
	reply := new(connReply)
L:
	for {
		err := dec.Decode(reply)
		if err != nil {
			if err != os.EOF {
				log.Print("recv error: ", err)
			}
			break
		}

		debug("Reply code: ", reply.Code)
		switch reply.Code {
		default:
			log.Print("Bad reply from server: ", reply.Code, "\r")
			break
		case Ok:
			// nothing
		case ReadOnly:
			fmt.Fprintf(os.Stdout, " [read-only] ")
			readwrite = false
		case ReadWrite:
			fmt.Fprintf(os.Stdout, " [read-write] ")
			readwrite = true
		case Err:
			log.Print("Error from server: ", reply.Err, "\r")
			break L
		case Data:
			os.Stdout.Write(reply.Data[:])
		}
	}
	log.Print("Disconnecting.\r")
	c.Close()
}

func readinput(c net.Conn, enc *gob.Encoder) {
	var buf [1]byte
	for {
		n, err := os.Stdin.Read(buf[:])
		if err != nil {
			log.Print("Failed to read stdin.")
			return
		}
		if n != 1 {
			// short read? why?
			continue
		}

		if buf[0] == 0x01 { // ctrl-a
			ctrla := escape(c, enc)
			if !ctrla {
				continue
			}
		}

		if readwrite {
			req := &connReq{Cmd: Write, Input: buf[:]}
			err = enc.Encode(req)
			if err != nil {
				log.Print("Failed to write on connection.")
				// just keep trying
			}
		}
	}
}

func escape(c net.Conn, enc *gob.Encoder) (ctrla bool) {
	var buf [1]byte
	quit := false

Loop:
	fmt.Print("\r\nYes? ")

	n, err := os.Stdin.Read(buf[0:1])
	if err != nil || n != 1 {
		// error? tell caller we want to exit
		return true
	}

	switch buf[0] {
	default:
		fmt.Print(" [canceled]\r\n")
	case '?':
		fallthrough
	case 'H':
		fallthrough
	case 'h':
		r := "read-only"
		if readwrite {
			r = "read-write"
		}
		fmt.Print("\r\nConnected to: ", cons, " (", r, ")")
		fmt.Print("\r\n\tq: quit\th: help\tt: take\ts: steal\r\n\t<others>: continue")
		goto Loop
	case 'q':
		fallthrough
	case 'Q':
		quit = true
	case 't':
		fallthrough
	case 'T':
		take(enc, Take)
	case 's':
		fallthrough
	case 'S':
		take(enc, Steal)
	case 0x01: // ctrl-a
		ctrla = true
	}

	if quit {
		fmt.Print(" [quitting]\r\n")
		// gross hack
		c.Close()
	}
	return
}

// from http://go.pastie.org/813153, then hacked up to work like I want

// termios types
type cc_t byte
type speed_t uint
type tcflag_t uint

// termios constants
const (
	BRKINT = tcflag_t(0000002)
	ICRNL  = tcflag_t(0000400)
	INPCK  = tcflag_t(0000020)
	ISTRIP = tcflag_t(0000040)
	IXON   = tcflag_t(0002000)
	OPOST  = tcflag_t(0000001)
	CS8    = tcflag_t(0000060)
	ECHO   = tcflag_t(0000010)
	ICANON = tcflag_t(0000002)
	IEXTEN = tcflag_t(0100000)
	ISIG   = tcflag_t(0000001)
	VTIME  = tcflag_t(5)
	VMIN   = tcflag_t(6)
)

const NCCS = 32

type termios struct {
	c_iflag, c_oflag, c_cflag, c_lflag tcflag_t
	c_line                             cc_t
	c_cc                               [NCCS]cc_t
	c_ispeed, c_ospeed                 speed_t
}

// ioctl constants
const (
	TCGETS = 0x5401
	TCSETS = 0x5402
)

func getTermios(tty *os.File) (termios, os.Error) {
	t := termios{}
	r1, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(tty.Fd()), uintptr(TCGETS),
		uintptr(unsafe.Pointer(&t)))

	if errno != 0 {
		return termios{}, os.NewSyscallError("SYS_IOCTL", int(errno))
	} else if r1 != 0 {
		return termios{}, os.NewError(fmt.Sprintf("SYS_IOCTL returned %d", r1))
	}
	return t, nil
}

func setTermios(tty *os.File, src termios) os.Error {
	r1, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(tty.Fd()), uintptr(TCSETS),
		uintptr(unsafe.Pointer(&src)))

	if errno != 0 {
		return os.NewSyscallError("SYS_IOCTL", int(errno))
	} else if r1 != 0 {
		return os.NewError(fmt.Sprintf("SYS_IOCTL returned %d", r1))
	}

	return nil
}

func tty_raw(tty *os.File, current termios) os.Error {
	raw := current

	raw.c_iflag &= ^(BRKINT | ICRNL | INPCK | ISTRIP | IXON)
	raw.c_oflag &= ^(OPOST)
	raw.c_cflag |= (CS8)
	raw.c_lflag &= ^(ECHO | ICANON | IEXTEN | ISIG)

	raw.c_cc[VMIN] = 1
	raw.c_cc[VTIME] = 0

	if err := setTermios(tty, raw); err != nil {
		return err
	}

	return nil
}

// vim:ts=2
