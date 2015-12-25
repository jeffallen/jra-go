package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

var insecure = flag.Bool("insecure", false, "allow insecure TLS connections?")

// A DialTLS that returns error if the server won't agree to do HTTP/2.
func dialTLSonlyHTTP2(network, addr string) (net.Conn, error) {
	cfg := &tls.Config{
		InsecureSkipVerify: *insecure,
		NextProtos:         []string{"h2"},
	}

	c, err := tls.Dial(network, addr, cfg)
	if err != nil {
		return nil, err
	}
	st := c.ConnectionState()
	if st.NegotiatedProtocol != "h2" {
		return nil, errors.New("server does not support HTTP/2")
	}
	return c, nil
}

func main() {
	flag.Parse()

	url := flag.Arg(0)
	log.Print("fetching ", url)

	in, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	body := bytes.NewBuffer(in)

	client := &http.Client{Transport: &http.Transport{DialTLS: dialTLSonlyHTTP2}}
	resp, err := client.Post(url, "application/binary", body)
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Server was:", resp.Header["Server"])
	log.Printf("tls.ConnectionState was: %#v", resp.TLS)

	if resp.Proto != "HTTP/2.0" {
		log.Fatalf("Expecting proto HTTP/2.0, got %#v", resp.Proto)
	}

	if resp.StatusCode != 200 {
		in, _ := ioutil.ReadAll(resp.Body)
		log.Fatalf("status: %v (%v)", resp.StatusCode, strings.Replace(string(in[0:100]), "\n", "", -1))
	}
	io.Copy(os.Stdout, resp.Body)
	resp.Body.Close()
}
