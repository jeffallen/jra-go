package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"code.google.com/p/jra-go/property"
)

func main() {
	for i, a := range os.Args {
		if i == 0 {
			continue
		}
		f, err := os.Open(a)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v, %v\n", a, err)
			return
		}
		defer f.Close()

		// because the files are in ISO-8859-1 and the HTML parser wants UTF8
		utf := toUtf8(f)
		r := strings.NewReader(utf)

		p, err := property.Read(r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
			return
		}
		if len(p.Owners) >= 1 && p.Owners[0] != "Information indisponible !" {
			fmt.Printf("%v\t%.0f\t%v\n", p.Id, p.Surface, strings.Join(p.Owners, ", "))
		}
	}
}

func toUtf8(r io.Reader) string {
	all, _ := ioutil.ReadAll(r)
	buf := make([]rune, len(all))
	for i, b := range all {
		buf[i] = rune(b)
	}
	return string(buf)
}
