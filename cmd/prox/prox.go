package main

import (
	"fmt"
	"http"
	"io"
	"jra-go.googlecode.com/hg/linkio"
	"log"
	"net"
	"strings"
	"url"
)

var gLink *linkio.Link

func init() {
	gLink = linkio.NewLink(56 /* kbps */ )
}

func loghit(r *http.Request, code int) {
	log.Printf("%v %v %v", r.Method, r.RawURL, code)
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
	oreq := new(http.Request)
	oreq.ProtoMajor = 1
	oreq.ProtoMinor = 1
	oreq.Close = true
	oreq.Header = creq.Header
	oreq.Method = creq.Method

	ourl, err := url.Parse(creq.RawURL)
	if err != nil {
		http.Error(cwr, fmt.Sprint("Malformed request", err),
			http.StatusNotImplemented)
		loghit(creq, http.StatusNotImplemented)
		return
	}
	// don't set RawURL, or else the request will be written with only
	// it, instead of using the URL.Path as we want (see (* http.Request)Write)
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
		http.Error(cwr, err.String(), http.StatusGatewayTimeout)
		loghit(creq, http.StatusGatewayTimeout)
		return
	}
	c.SetReadTimeout(3 * 1e9)
	cc := http.NewClientConn(c, nil)

	// debug
	//dbg, err := http.DumpRequest(oreq, true)
	//log.Print("Dump request to origin server:\n", string(dbg))

	err = cc.Write(oreq)
	if err != nil {
		http.Error(cwr, err.String(), http.StatusGatewayTimeout)
		loghit(creq, http.StatusGatewayTimeout)
		return
	}

	oresp, err := cc.Read(oreq)
	if err != nil && err != http.ErrPersistEOF {
		http.Error(cwr, err.String(), http.StatusGatewayTimeout)
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

func main() {
	proxy := NewProxy()
	err := http.ListenAndServe("[::]:12345", proxy)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.String())
	}
}

// ex: ts=2
