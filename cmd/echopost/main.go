package main

import (
	"io/ioutil"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/echopost", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "bad method", 500)
			return
		}

		in, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read error", 500)
			return
		}
		r.Body.Close()

		ct := r.Header["Content-Type"]
		if len(ct) == 0 {
			ct = []string{"text/plain"}
		}
		w.Header().Set("Content-Type", ct[0])
		n, err := w.Write(in)
		if err != nil {
			panic(err)
		}
		if n != len(in) {
			panic("short write?")
		}
		log.Printf("echoed %v bytes", len(in))
	})

	log.Fatal(http.ListenAndServe(":8000", nil))
}
