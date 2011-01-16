package main

import (
	"os"
	"net"
	"log"
	"time"
	"gob"
	"flag"
	"runtime"
)

var managers map[string] *consoleManager

var fake *bool = flag.Bool("fake", false, "run a fake server")

func init() {
	flag.Parse()
}

func main() {
	managers = make(map[string] *consoleManager)

	var cons, addr string
	if (*fake) {
		ready := make(chan bool)
		go fakeserver(ready)
		_ = <-ready
		cons, addr = "fake", ":2070"
	} else {
		cons, addr = "msi-switch", "msi-rt3:2070"
	}
	m := NewConsoleManager(cons, addr)
	managers[cons] = m
	go m.run()

	l, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Exit("Cannot start server: ", err)
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
    log.Exit("Cannot start fake server: ", err)
  }
  log.Print("Fake server listening on ", l.Addr())

	ready <- true

 L:
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
	cmd connCmd
	name string
	input []byte
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
  code connReplyCode
  err string
  data []byte
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
		case req := <-reqCh:
			if closed(reqCh) {
				log.Print("connection ", c, " terminated")
				break L
			}
			switch req.cmd {
			default:
				log.Print("connection ", c.rw, " bad command")
				break L
			case Listen:
				log.Print("connection ", c.rw, " wants to listen to ", req.name)

				var ok bool
				mgr, ok = managers[req.name]
				if !ok {
					reply.code = Err
					reply.err = "unknown console name"
					enc.Encode(reply)
				} else {
					listen := consoleRequest{ cmd: conListen, ch: evCh }
					mgr.reqCh <- listen

					reply.code = Ok
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
					reply.code = Err
					reply.err = "not connected to a console yet"
				} else {
					// talk to the consoleManager asynchronously to ask to take ownership
					inCh = make(chan []byte, 10)
					okCh := make(chan bool)

					cmd := conTake
					if req.cmd == Steal {
						cmd = conSteal
					}

					r := consoleRequest{ cmd: cmd, inch: inCh, ok: okCh }
					mgr.reqCh <- r
					ok := <- okCh
					if (ok) {
						reply.code = ReadWrite
					} else {
						reply.code = ReadOnly
						reply.err = "another user already has the console for read/write"
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
				if inCh == nil || closed(inCh) {
					// ignore write when we are not owner
					log.Print("Write when we don't own the console.")
				} else {
					log.Print("Writing ", req.input, " into channel ", inCh)
					inCh<-req.input
					runtime.Gosched()
				}
			case Close:
				break L
			}
/* this doesn't work: once it is closed, it triggers the select repeatedly.
	Need to be notified a different way. */
/*		case _ = <-inCh:
			if closed(inCh) {
				reply.code = ReadOnly
				reply.err = "another user has stolen the console from you"

				err := enc.Encode(reply)
				if (err != nil) {
					log.Print("connection ", c.rw, ", failed to send lost reply: ", err)
					break L
				}
			}
*/
		case ev := <-evCh:
			reply.code = Data
			reply.data = ev.data
			err := enc.Encode(reply)
			if (err != nil) {
				log.Print("connection ", c.rw, ", failed to send data: ", err)
				break L
			}
		}
	}

	// tell the consoleManager we aren't listening now
	close(evCh)

	// tell the consoleManager we won't be writing any more
	if inCh != nil && !closed(inCh) {
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
	cmd consoleCmd
	ch chan consoleEvent
	inch chan []byte
	ok chan bool			// for asynchronous reply
}

type inputReq struct {
	data []byte
	raw bool
}

type consoleManager struct {
	cons, addr string
  c net.Conn
	quitCh chan bool
	dataCh chan []byte
	inputCh chan inputReq				// from the consoleManager to the sender
	owner chan []byte
	reqCh chan consoleRequest
}

type listener struct {
	ch chan consoleEvent
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
		c, err := net.Dial("tcp", "", m.addr)
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
			case req := <-m.reqCh:
				if closed(m.reqCh) {
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
			case data := <-m.dataCh:
				if closed(m.dataCh) {
					break L
				}
				// multicast the data to the listeners
				for l := listeners; l != nil; l = l.next {
					if closed(l.ch) {
						// TODO: need to remove this node from the list, not just mark nil
						l.ch = nil
						log.Print("Marking listener ", l, " no longer active.")
					}
					if l.ch != nil {
						ok := l.ch <- consoleEvent{data}
						if !ok {
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

func (m *consoleManager)setOwner(ch chan []byte) {
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
			m.inputCh <- inputReq{ data: in, raw: false }
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

  file, err := os.Open(m.cons, os.O_WRONLY | os.O_CREAT | os.O_APPEND, 0600)
	if (err != nil) {
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
			iacbuf[0] = 255		// IAC
			iacbuf[1] = 253		// DO
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
