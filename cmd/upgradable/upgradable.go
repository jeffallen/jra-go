package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

var verbose *bool = flag.Bool("verbose", false, "log every transaction")
var listenFD *int = flag.Int("listenFD", 0, "the already-open fd to listen on (internal use only)")

func init() {
	log.SetPrefix(fmt.Sprintf("[%5d] ", syscall.Getpid()))
}

var connCount struct {
	m sync.Mutex
	c int
}

type watchedConn struct {
	net.Conn
}

func (w watchedConn) Close() error {
	//log.Printf("close on conn to %v", w.RemoteAddr())

	connCount.m.Lock()
	connCount.c--
	connCount.m.Unlock()

	return w.Conn.Close()
}

type signal struct{}

type stoppableListener struct {
	net.Listener
	stop chan signal
}

var theStoppable *stoppableListener
var theListener net.Listener
var stopped = errors.New("listener stopped")

func newStoppable(l net.Listener) *stoppableListener {
	return &stoppableListener{Listener: l, stop: make(chan signal, 1)}
}

func (sl *stoppableListener) Accept() (c net.Conn, err error) {
	// non-blocking read on the stop channel
	select {
	default:
		// nothing
	case <-sl.stop:
		return nil, stopped
	}

	// if we got here, we have not been asked to stop, so call
	// Accept on the underlying listener.

	c, err = sl.Listener.Accept()
	if err != nil {
		return
	}

	// Wrap the returned connection, so that we can observe when
	// it is closed.
	c = watchedConn{Conn: c}

	// Count it
	connCount.m.Lock()
	connCount.c++
	connCount.m.Unlock()

	return
}

func logreq(req *http.Request) {
	if *verbose {
		log.Printf("%v %v from %v", req.Method, req.URL, req.RemoteAddr)
	}
}

func HelloServer(w http.ResponseWriter, req *http.Request) {
	logreq(req)
	io.WriteString(w, "hello, world!\n")
}

func UpgradeServer(w http.ResponseWriter, req *http.Request) {
	logreq(req)
	var sig signal

	tl := theListener.(*net.TCPListener)
	fd := tl.GetFD()

	// net/fd.go marks all sockets as close on exec, so we need to undo
	// that before we start the child, so that the listen FD survives
	// the fork/exec
	syscall.NoCloseOnExec(fd)

	cmd := exec.Command("./upgradable", "-listenFD", fmt.Sprintf("%d", fd))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Print("starting cmd: ", cmd.Args)
	if err := cmd.Start(); err != nil {
		log.Print("error:", err)
		return
	}

	// no error, the new one must have started. Arrange to
	// stop ourselves before the *next* call to Accept().
	// The current blocked call to Accept() needs to finish, meaning
	// we will process one more transaction after the upgrade one.
	theStoppable.stop <- sig
}

func main() {
	flag.Parse()

	http.HandleFunc("/hello", HelloServer)
	http.HandleFunc("/upgrade", UpgradeServer)

	var err error
	server := &http.Server{Addr: ":8000"}
	if *listenFD != 0 {
		log.Print("Listening to existing fd ", *listenFD)
		theListener, err = net.NewTCPListener(*listenFD)
	} else {
		log.Print("Listening on a new fd")
		theListener, err = net.Listen("tcp", server.Addr)
	}
	if err != nil {
		log.Fatal(err)
	}

	theStoppable = newStoppable(theListener)

	err = server.Serve(theStoppable)
	if err == stopped {
		for i, done := 10, false; !done && i > 0; i-- {
			connCount.m.Lock()
			if connCount.c == 0 {
				done = true
				continue
			}
			connCount.m.Unlock()
			time.Sleep(1e9)
		}
		log.Fatal("server gracefully stopped")
	}
	if err != nil {
		log.Fatal(err)
	}
}
