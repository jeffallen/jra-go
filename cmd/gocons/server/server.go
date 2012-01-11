//	Copyright 2009 Jeff R. Allen. All rights reserved.
//	Use of this source code is governed by a BSD-style
//	license that can be found in the LICENSE file of the Go
//	distribution.

package main

import (
	"os"
	"net"
	"log"
	"time"
	"gob"
	"flag"
	"runtime"
	"json"
)

var managers map[string]*consoleManager

var fake *bool = flag.Bool("fake", false, "run a fake server")
var config *string = flag.String("config", "", "the JSON config file")

func init() {
	flag.Parse()
}

func addConsole(cons string, dial string) {
	m, ok := managers[cons]
	if !ok {
		log.Printf("Adding console %v on %v.", cons, dial)
		m = NewConsoleManager(cons, dial)
		managers[cons] = m
		go m.run()
	} else {
		log.Print("Duplicate console %v, ignoring it.")
	}
}

func main() {
	managers = make(map[string]*consoleManager)

	if *config == "" {
		log.Fatal("usage: goconserver -config config.js")
	}
	r, err := os.OpenFile(*config, os.O_RDONLY, 0)
	if err != nil {
		log.Fatalf("Cannot read config file %v: %v", *config, err)
	}
	dec := json.NewDecoder(r)
	var conf interface{}
	err = dec.Decode(&conf)
	if err != nil {
		log.Fatal("JSON decode: ", err)
	}
	hash, ok := conf.(map[string]interface{})
	if !ok {
		log.Fatal("JSON format error: got %T", conf)
	}
	consoles, ok := hash["consoles"]
	if !ok {
		log.Fatal("JSON format error: key consoles not found")
	}
	c2, ok := consoles.(map[string]interface{})
	if !ok {
		log.Fatalf("JSON format error: consoles key wrong type, %T", consoles)
	}
	for k, v := range c2 {
		s, ok := v.(string)
		if ok {
			addConsole(k, s)
		} else {
			log.Fatal("Dial string for console %v is not a string.", k)
		}
	}

	if *fake {
		ready := make(chan bool)
		go fakeserver(ready)
		_ = <-ready
		addConsole("fake", ":2070")
	}

	l, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Fatal("Cannot start server: ", err)
	}
	log.Print("Goconserver listening on ", l.Addr())

	for {
		rw, e := l.Accept()
		if e != nil {
			log.Print("accept: ", e)
			continue
		}
		c := newConn(rw)
		go c.run()
	}
}

func fakeserver(ready chan bool) {
	l, err := net.Listen("tcp", ":2070")
	if err != nil {
		log.Fatal("Cannot start fake server: ", err)
	}
	log.Print("Fake server listening on ", l.Addr())

	ready <- true

	for {
		rw, e := l.Accept()
		if e != nil {
			log.Print("(fake) accept: ", e)
			continue
		}
		go func() {
			for {
				var buf [1]byte
				n, err := rw.Read(buf[:])
				if err != nil || n != 1 {
					log.Print("(fake) read: ", e)
					break
				}
				// just echo their typing back
				rw.Write(buf[:])
			}
			rw.Close()
		}()
	}
}

type connection struct {
	rw net.Conn
}

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

func newConn(rw net.Conn) (c *connection) {
	c = new(connection)
	c.rw = rw
	return
}

func (c *connection) decoder(ch chan connReq) {
	dec := gob.NewDecoder(c.rw)
	for {
		req := new(connReq)
		err := dec.Decode(req)
		if err != nil {
			if err != os.EOF {
				log.Print("connection ", c.rw, ", could not decode: ", err)
			}
			break
		}
		ch <- *req
	}
	close(ch)
}

func (c *connection) run() {
	var mgr *consoleManager
	evCh := make(chan consoleEvent, 1000)

	var inCh chan []byte
	reqCh := make(chan connReq)
	go c.decoder(reqCh)

	enc := gob.NewEncoder(c.rw)

L:
	for {
		reply := new(connReply)
		select {
		case req, closed := <-reqCh:
			if !closed {
				log.Print("connection ", c, " terminated")
				break L
			}
			switch req.Cmd {
			default:
				log.Print("connection ", c.rw, " bad command")
				break L
			case Listen:
				log.Print("connection ", c.rw, " wants to listen to ", req.Name)

				var ok bool
				mgr, ok = managers[req.Name]
				if !ok {
					reply.Code = Err
					reply.Err = "unknown console name"
					enc.Encode(reply)
				} else {
					listen := consoleRequest{cmd: conListen, ch: evCh}
					mgr.reqCh <- listen

					reply.Code = Ok
					err := enc.Encode(reply)
					if err != nil {
						log.Print("connection ", c.rw, " failed to send reply: ", err)
						break L
					}
				}
			case Take:
				fallthrough
			case Steal:
				if mgr == nil {
					reply.Code = Err
					reply.Err = "not connected to a console yet"
				} else {
					// talk to the consoleManager asynchronously to ask to take ownership
					inCh = make(chan []byte, 10)
					okCh := make(chan bool)

					cmd := conTake
					if req.Cmd == Steal {
						cmd = conSteal
					}

					r := consoleRequest{cmd: cmd, inch: inCh, ok: okCh}
					mgr.reqCh <- r
					ok := <-okCh
					if ok {
						reply.Code = ReadWrite
					} else {
						reply.Code = ReadOnly
						reply.Err = "another user already has the console for read/write"
						close(inCh)
						inCh = nil
					}
				}
				err := enc.Encode(reply)
				if err != nil {
					log.Print("connection ", c.rw, " failed to send reply: ", err)
					break L
				}
			case Write:
				//if inCh == nil || closed(inCh) {
				if inCh == nil {
					// ignore write when we are not owner
					log.Print("Write when we don't own the console.")
				} else {
					log.Print("Writing ", req.Input, " into channel ", inCh)
					inCh <- req.Input
					runtime.Gosched()
				}
			case Close:
				break L
			}
		case ev := <-evCh:
			reply.Code = Data
			reply.Data = ev.data
			err := enc.Encode(reply)
			if err != nil {
				log.Print("connection ", c.rw, ", failed to send data: ", err)
				break L
			}
		}
	}

	// tell the consoleManager we aren't listening now
	close(evCh)

	// tell the consoleManager we won't be writing any more
	//if inCh != nil && !closed(inCh) {
	if inCh != nil {
		close(inCh)
	}

	log.Print("connection ", c.rw, " closing")
	c.rw.Close()
}

type consoleEvent struct {
	data []byte
}

type consoleCmd int

const (
	conListen consoleCmd = iota
	conTake
	conSteal
)

type consoleRequest struct {
	cmd  consoleCmd
	ch   chan consoleEvent
	inch chan []byte
	ok   chan bool // for asynchronous reply
}

type inputReq struct {
	data []byte
	raw  bool
}

type consoleManager struct {
	cons, addr string
	c          net.Conn
	quitCh     chan bool
	dataCh     chan []byte
	inputCh    chan inputReq // from the consoleManager to the sender
	owner      chan []byte
	reqCh      chan consoleRequest
}

type listener struct {
	ch   chan consoleEvent
	next *listener
}

func NewConsoleManager(cons, addr string) (m *consoleManager) {
	m = new(consoleManager)
	m.cons = cons
	m.addr = addr
	m.quitCh = make(chan bool)
	m.dataCh = make(chan []byte, 10)
	m.inputCh = make(chan inputReq, 10)
	m.reqCh = make(chan consoleRequest, 10)
	return
}

func (m *consoleManager) run() {
	var listeners *listener

	reconnect := true

	// keep trying to reconnect
	for reconnect {
		c, err := net.Dial("tcp", m.addr)
		if err != nil {
			log.Print("Failed to connect to ", m.cons, ": ", err)
			time.Sleep(10 * 1e9)
			continue
		}
		log.Print("Connected to ", m.cons, " on ", m.addr)
		m.c = c

		connDead := make(chan bool)
		go func() { m.send(); close(connDead) }()
		go func() { m.recv(); close(connDead) }()

		// big event loop for consoleManager
	L:
		for {
			select {
			case _ = <-m.quitCh:
				reconnect = false
				break L
			case _ = <-connDead:
				break L
			case req, closed := <-m.reqCh:
				if !closed {
					break L
				}
				if req.cmd == conListen {
					l := new(listener)
					l.ch = req.ch
					l.next = listeners
					listeners = l
				} else if req.cmd == conTake || req.cmd == conSteal {
					if req.cmd == conTake && m.owner != nil {
						// not available for taking
						req.ok <- false
						log.Print("Console not taken")
					} else {
						log.Print("Console taken by ", req.inch)
						m.setOwner(req.inch)
						req.ok <- true
					}
				} else {
					log.Print("Ignoring unknown request command: ", req.cmd)
				}
			case data, closed := <-m.dataCh:
				if !closed {
					break L
				}
				// multicast the data to the listeners
				for l := listeners; l != nil; l = l.next {
					//if closed(l.ch) {
					// TODO: need to remove this node from the list, not just mark nil
					//	l.ch = nil
					//	log.Print("Marking listener ", l, " no longer active.")
					//}
					if l.ch != nil {
						select {
						case l.ch <- consoleEvent{data}:
						default:
							log.Print("Listener ", l, " lost an event.")
						}
					}
				}
			}
		}
		c.Close()
	}

	return
}

func (m *consoleManager) setOwner(ch chan []byte) {
	if m.owner != nil {
		// tell the current owner he's been pwned
		close(m.owner)
	}
	m.owner = ch

	// a pump to move things into the console (because
	// when we try to do this in the same select as everything
	// else, we never see anything on ch
	go func(ch chan []byte) {
		log.Print("pumper starting for chan ", ch)
		for in := range ch {
			log.Print("bytes arrived:", len(in))
			m.inputCh <- inputReq{data: in, raw: false}
		}
		m.owner = nil
		log.Print("pumper done")
	}(ch)
}

type iacState int

const (
	None iacState = iota
	IAC
	IAC2
	IACWill
)

func (m *consoleManager) recv() {
	var st iacState = None

	file, err := os.OpenFile(m.cons, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		log.Printf("Error opening log file %s: %v", file, err)
		return
	}

	for {
		var buf [1]byte

		n, err := m.c.Read(buf[:])
		if err != nil {
			if err != os.EOF {
				log.Print("Error reading from net: ", err)
			}
			break
		}

		switch st {
		default:
			// huh?
			log.Print("Unexpected IAC state", st)
			st = None
		case None:
			if buf[0] == 0xff {
				st = IAC
				continue
			}
			// else: keep going below, process byte normally
		case IAC:
			switch buf[0] {
			// don't, do, won't
			case 254, 253, 252:
				st = IAC2
				continue
			// will
			case 251:
				st = IACWill
				continue
			default:
				// IAC IAC, so back to none and handle like a normal char (no continue)
				st = None
			}
		case IAC2:
			// we don't care, just eat the byte
			st = None
			continue
		case IACWill:
			var iacbuf [3]byte
			iacbuf[0] = 255 // IAC
			iacbuf[1] = 253 // DO
			// we only want to reply "do" for 1 (8-bit) and 3 (sga)
			if buf[0] == 1 || buf[0] == 3 {
				iacbuf[2] = buf[0]
				m.inputCh <- inputReq{raw: true, data: iacbuf[:]}
			}
			st = None
			continue
		}

		x, err := file.Write(buf[0:n])
		if err != nil {
			log.Print("Error writing to file: ", err)
			return
		}
		if x != n {
			log.Print("Error writing to file: short write", n, "!=", x)
			return
		}

		// send the stuff up to the manager
		m.dataCh <- buf[0:n]
	}
	log.Print("Connection to ", m.cons, " closed.")
}

func (m *consoleManager) send() {
	var buf [512]byte

	m.c.SetWriteTimeout(1e9)

	for req := range m.inputCh {
		// for each write, out starts as an empty slice of the buffer,
		// and append grows it if necessary, including growing
		// the underlying buffer maybe
		out := buf[0:0]

		for i := 0; i < len(req.data); i++ {
			if !req.raw && req.data[i] == 0xff {
				out = append(out, 0xff)
			}
			out = append(out, req.data[i])
		}

		x, err := m.c.Write(out[:])
		if err != nil {
			log.Print("Error writing to connection: ", err)
			return
		}
		if x != len(out) {
			log.Print("Error writing to connection: short write", x, "!=", len(out))
			return
		}
	}
}

// vim:ts=2
