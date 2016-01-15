package main

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/http2"
)

var insecure = flag.Bool("insecure", false, "allow insecure TLS connections?")

func main() {
	flag.Parse()

	url := flag.Arg(0)
	log.Print("fetching ", url)

	in, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	body := bytes.NewBuffer(in)

	client := &http.Client{Transport: &http2.Transport{}}
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
