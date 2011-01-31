package main

import (
	"http"
	"io"
	"os"
	"log"
	"strings"
	"jra-go.googlecode.com/hg/linkio"
)

var gLink *linkio.Link

func init() {
	gLink = linkio.NewLink(56 /* kbps */)
}

type Proxy struct {
}

func NewProxy() *Proxy { return &Proxy{} }

func loghit(r *http.Request, code int, hit bool) {
	log.Printf("%v %v %v %v", r.Method, r.RawURL, code, hit)
}

func (p *Proxy) ServeHTTP(wr http.ResponseWriter, r *http.Request) {
	if ! strings.HasPrefix(r.RawURL, "http://") {
		http.Error(wr, "501 I only proxy http", http.StatusNotImplemented)
		loghit(r, http.StatusNotImplemented, false)
		return
	}

	var resp *http.Response
	var err os.Error

	switch r.Method {
	default: {
		log.Print("Cannot handle method ", r.Method)
		http.Error(wr, "501 I only handle GET and POST", http.StatusNotImplemented)
		return
	}
	case "GET": {
		resp, _, err = http.Get(r.RawURL)
	}
	case "POST": {
		resp, err = http.Post(r.RawURL, safeGetCT(r, nil, "multipart/form-data"), r.Body)
		r.Body.Close()
	}
	}

	// combined for GET/POST
	if err != nil {
		http.Error(wr, err.String(), http.StatusInternalServerError)
		loghit(r, http.StatusInternalServerError, false)
		return
	}
	wr.SetHeader("Content-Type", safeGetCT(nil, resp, "text/plain"))
	wr.WriteHeader(resp.StatusCode)

	// simulate it coming in over gLink, a shared rate-limited link
	io.Copy(wr, linkio.NewLinkReader(resp.Body, gLink))

	resp.Body.Close()
	loghit(r, resp.StatusCode, false)
}

func safeGetCT(r1 *http.Request, r2 *http.Response, def string) (ct string) {
	var ok bool
	if r1 != nil {
		ct, ok = r1.Header["Content-Type"]
	} else {
		ct, ok = r2.Header["Content-Type"]
	}
	if ! ok {
		ct = def
	}
	return
}

func main() {
	proxy := NewProxy()
	err := http.ListenAndServe(":12345", proxy)
	if err != nil {
		log.Exit("ListenAndServe: ", err.String())
	}
}

// ex: ts=2
