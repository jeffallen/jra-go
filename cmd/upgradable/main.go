// Copyright 2013 Jeff R. Allen. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"
)

var verbose *bool = flag.Bool("verbose", false, "log every transaction")
var listenFD *int = flag.Int("listenFD", 0, "the already-open fd to listen on (internal use only)")

func init() {
	log.SetPrefix(fmt.Sprintf("[%5d] ", syscall.Getpid()))
}

type counter struct {
	m sync.Mutex
	c int
}

func (c counter)get() (ct int) {
	c.m.Lock()
	ct = c.c
	c.m.Unlock()
	return
}

var connCount counter

type watchedConn struct {
	net.Conn
}

func (w watchedConn) Close() error {
	if *verbose {
		log.Printf("close on conn to %v", w.RemoteAddr())
	}

	connCount.m.Lock()
	connCount.c--
	connCount.m.Unlock()

	return w.Conn.Close()
}

type signal struct{}

type stoppableListener struct {
	net.Listener
	stop    chan signal
	stopped bool
}

var theStoppable *stoppableListener

func newStoppable(l net.Listener) (sl *stoppableListener) {
	sl = &stoppableListener{Listener: l, stop: make(chan signal, 1)}

	// this goroutine monitors the channel. Can't do this in
	// Accept (below) because once it enters sl.Listener.Accept()
	// it blocks. We unblock it by closing the fd it is trying to
	// accept(2) on.
	go func() {
		_ = <-sl.stop
		sl.stopped = true
		sl.Listener.Close()
	}()
	return
}

func (sl *stoppableListener) Accept() (c net.Conn, err error) {
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
	fmt.Fprintf(w, "hello world\n")
}

func UpgradeServer(w http.ResponseWriter, req *http.Request) {
	logreq(req)
	var sig signal

	tl := theStoppable.Listener.(*net.TCPListener)
	fl, err := tl.File()
	if err != nil {
		log.Fatal(err)
	}
	fd := fl.Fd()

	// net/fd.go marks all sockets as close on exec, so we need to undo
	// that before we start the child, so that the listen FD survives
	// the fork/exec
	noCloseOnExec(fd)

	cmd := exec.Command("./upgradable",
		fmt.Sprintf("-verbose=%v", *verbose),
		fmt.Sprintf("-listenFD=%d", fd))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Print("starting cmd: ", cmd.Args)
	if err := cmd.Start(); err != nil {
		log.Print("error:", err)
		return
	}

	// no error, the new one must have started. Arrange to
	// stop ourselves.
	theStoppable.stop <- sig
}

func main() {
	flag.Parse()

	http.HandleFunc("/hello", HelloServer)
	http.HandleFunc("/upgrade", UpgradeServer)

	var err error
	var l net.Listener
	server := &http.Server{Addr: ":8000"}
	if *listenFD != 0 {
		log.Print("Listening to existing fd ", *listenFD)
		f := os.NewFile(uintptr(*listenFD), "listen socket")
		l, err = net.FileListener(f)
	} else {
		log.Print("Listening on a new fd")
		l, err = net.Listen("tcp", server.Addr)
	}
	if err != nil {
		log.Fatal(err)
	}

	theStoppable = newStoppable(l)

	log.Print("Serving on http://localhost:8000/")
	err = server.Serve(theStoppable)

	// did we get here due to a legitimate stop signal or an err?
	log.Print("not longer serving...")
	if theStoppable.stopped {
		for i := 0; i < 10; i++ {
			if connCount.get() == 0 {
				continue
			}
			log.Print("waiting for clients...")
			time.Sleep(1 * time.Second)
		}

		if connCount.get() == 0 {
			log.Print("server gracefully stopped.")
			os.Exit(0)
		} else {
			log.Fatalf("server stopped after 10 seconds with %d clients still connected.", connCount.get())
		}
	}
	if err != nil {
		log.Fatal(err)
	}
}

// These are here because there is no API in syscall for turning OFF
// close-on-exec (yet).

// from syscall/zsyscall_linux_386.go, but it seems like it might work
// for other platforms too.
func fcntl(fd int, cmd int, arg int) (val int, err error) {
	if runtime.GOOS != "linux" {
		log.Fatal("Function fcntl has not been tested on other platforms than linux.")
	}

	r0, _, e1 := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), uintptr(cmd), uintptr(arg))
	val = int(r0)
	if e1 != 0 {
		err = e1
	}
	return
}

func noCloseOnExec(fd uintptr) {
	fcntl(int(fd), syscall.F_SETFD, ^syscall.FD_CLOEXEC)
}
