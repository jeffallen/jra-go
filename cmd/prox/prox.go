package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"code.google.com/p/jra-go/linkio"
)

func loghit(r *http.Request, code int) {
	log.Printf("%v %v %v", r.Method, r.RequestURI, code)
}

// Given a string of the form "host", "host:port", or "[ipv6::address]:port",
// return true if the string includes a port.
func hasPort(s string) bool { return strings.LastIndex(s, ":") > strings.LastIndex(s, "]") }

var doNotCopy = map[string]bool{
	"Connection":        true,
	"Transfer-Encoding": true,
	"Trailer":           true,
}

type Proxy struct{}

func NewProxy() *Proxy { return &Proxy{} }

func (p *Proxy) ServeHTTP(cwr http.ResponseWriter, creq *http.Request) {
	// c = things towards the client of the proxy
	// o = things towards origin server

	if creq.Method == "CONNECT" {
		rc, err := net.Dial("tcp", creq.URL.Host)
		if err != nil {
			http.Error(cwr, err.Error(), http.StatusGatewayTimeout)
			loghit(creq, http.StatusGatewayTimeout)
			return
		}
		remote := bufio.NewReadWriter(bufio.NewReader(rc),
			bufio.NewWriter(rc))

		cwr.WriteHeader(http.StatusOK)
		loghit(creq, http.StatusOK)

		hj, ok := cwr.(http.Hijacker)
		if !ok {
			panic("not hijackable")
		}
		wc, client, err := hj.Hijack()

		done := make(chan int)

		f := func(from, to *bufio.ReadWriter) {
			var err error
			n := 0
			for err == nil {
				var c byte
				c, err = from.ReadByte()
				n++
				if err == nil {
					err = to.WriteByte(c)
					to.Flush()
				}
			}
			done <- n
		}
		go f(remote, client)
		go f(client, remote)

		// wait for one side to finish and close both sides
		tot := <-done
		wc.Close()
		rc.Close()
		tot += <-done

		log.Print("CONNECT finished, ", tot, " bytes")
		return
	}

	oreq := new(http.Request)
	oreq.ProtoMajor = 1
	oreq.ProtoMinor = 1
	oreq.Close = true
	oreq.Header = creq.Header
	oreq.Method = creq.Method

	ourl, err := url.Parse(creq.RequestURI)
	if err != nil {
		http.Error(cwr, fmt.Sprint("Malformed request", err),
			http.StatusNotImplemented)
		loghit(creq, http.StatusNotImplemented)
		return
	}
	oreq.URL = ourl

	if oreq.URL.Scheme != "http" {
		http.Error(cwr, "I only proxy http", http.StatusNotImplemented)
		loghit(creq, http.StatusNotImplemented)
		return
	}

	if oreq.Method != "GET" && oreq.Method != "POST" {
		log.Print("Cannot handle method ", creq.Method)
		http.Error(cwr, "I only handle GET and POST", http.StatusNotImplemented)
		return
	}

	if oreq.Method == "POST" {
		oreq.Method = "POST"
		if _, ok := oreq.Header["Content-Type"]; !ok {
			oreq.Header.Set("Content-Type", "multipart/form-data")
		}
		oreq.ContentLength = creq.ContentLength
		oreq.Body = creq.Body
	}

	addr := oreq.URL.Host
	if !hasPort(addr) {
		addr += ":" + oreq.URL.Scheme
	}
	c, err := net.Dial("tcp", addr)
	if err != nil {
		http.Error(cwr, err.Error(), http.StatusGatewayTimeout)
		loghit(creq, http.StatusGatewayTimeout)
		return
	}
	c.SetReadDeadline(time.Now().Add(3 * time.Second))
	cc := httputil.NewProxyClientConn(c, nil)

	// debug
	//dbg, err := http.DumpRequest(oreq, true)
	//log.Print("Dump request to origin server:\n", string(dbg))

	err = cc.Write(oreq)
	if err != nil {
		http.Error(cwr, err.Error(), http.StatusGatewayTimeout)
		loghit(creq, http.StatusGatewayTimeout)
		return
	}

	oresp, err := cc.Read(oreq)
	if err != nil && err != httputil.ErrPersistEOF {
		http.Error(cwr, err.Error(), http.StatusGatewayTimeout)
		loghit(creq, http.StatusGatewayTimeout)
		return
	}

	//dbg, err = http.DumpResponse(oresp, true)
	//log.Print("Dump response from origin server:\n", string(dbg))

	for hdr, val := range oresp.Header {
		if !doNotCopy[hdr] {
			h := cwr.Header()
			h[hdr] = val
		}
	}
	cwr.WriteHeader(oresp.StatusCode)

	// simulate it coming in over gLink, a shared rate-limited link
	io.Copy(cwr, gLink.NewLinkReader(oresp.Body))

	cc.Close()
	c.Close()
	loghit(creq, oresp.StatusCode)
}

var speed = flag.Int("speed", 56, "speed of simulated link in kbps")
var gLink = linkio.NewLink(56 /* kbps */)

func main() {
	flag.Parse()
	gLink.SetSpeed(*speed)

	proxy := NewProxy()
	err := http.ListenAndServe("[::]:12345", proxy)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}

// ex: ts=2
