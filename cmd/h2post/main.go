package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func main() {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	flag.Parse()
	url := flag.Arg(0)
	log.Print("fetching ", url)

	in, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	body := bytes.NewBuffer(in)
	resp, err := client.Post(url, "application/binary", body)
	if err != nil {
		log.Print(err)
		return
	}
	if resp.StatusCode != 200 {
		in, _ := ioutil.ReadAll(resp.Body)
		log.Fatalf("status: %v (%v)", resp.StatusCode, string(in))
	}
	io.Copy(os.Stdout, resp.Body)
	resp.Body.Close()
}
