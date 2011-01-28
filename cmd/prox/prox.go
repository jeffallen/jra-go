package main

import (
	"http"
	"io"
	"os"
	"log"
	"strings"
)

type Proxy struct {
}

func NewProxy() *Proxy { return &Proxy{} }

func loghit(r *http.Request, code int, hit bool) {
	log.Printf("%v %v %v %v", r.Method, r.RawURL, code, hit)
}

func (p *Proxy) ServeHTTP(wr http.ResponseWriter, r *http.Request) {
	if r.Method == "CONNECT" {
//CONNECT gmail.com:443 HTTP/1.1
//User-Agent: 
//Proxy-Connection: keep-alive
//Host: gmail.com
//HTTP/1.1 200 Connection established.
//Proxy-Connection: close
		log.Printf("connect %s", r.Host)
		wr.WriteHeader(http.StatusInternalServerError)
		loghit(r, http.StatusInternalServerError, false)
	}

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
		log.Printf("getting %v", r.RawURL)
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
	io.Copy(wr, resp.Body)
	resp.Body.Close()
	loghit(r, resp.StatusCode, false)
}

func safeGetCT(r1 *http.Request, r2 *http.Response, def string) (ct string) {
	var ok bool
	if r1 != nil {
		ct, ok = r1.Header["Content-Type"]
		log.Print("req ct ", ct)
	} else {
		ct, ok = r2.Header["Content-Type"]
		log.Print("resp ct ", ct)
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
