// Package property parses property records downloaded from
// the Canton of Vaud:
// http://www.geoplanet.vd.ch/ and
// http://www.rfinfo.vd.ch/rfinfo.php?no_commune=XXX&no_immeuble=XXX
package property

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"code.google.com/p/go.net/html"
)

const debug = false

// A Property holds all the data parsed out of one record.
// (There is other data available in the records, but it is not currently
// parsed.)
type Property struct {
	Commune string
	Id      int
	Surface float64
	Owners  []string
}

// skip forward to the next text, and return it as a string
func next(z *html.Tokenizer) string {
	for tt := z.Next(); true; tt = z.Next() {
		if tt == html.TextToken {
			res := string(z.Text())
			if debug {
				fmt.Printf("next: %q\n", res)
			}
			return res
		}
		if tt == html.ErrorToken {
			return ""
		}
		if debug {
			fmt.Println("skipping: ", tt)
		}
	}
	return ""
}

// Read parses one record off of the given io.Reader. The input
// must be UTF-8 HTML. The rinfo.vd.ch website gives output in
// IOS-8859-1, so it needs to be converted to UTF-8 before parsing.
func Read(r io.Reader) (p *Property, err error) {
	p = &Property{}
	z := html.NewTokenizer(r)
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			if z.Err() != io.EOF {
				err = z.Err()
			}
			return
		case html.TextToken:
			t := string(z.Text())
			switch t {
			case "Commune : ":
				p.Commune = next(z)
			case "Bien-fonds : ":
				p.Id, _ = strconv.Atoi(next(z))
			case "Surface : ":
				next(z)
				p.Surface, _ = strconv.ParseFloat(trimM(next(z)), 64)
			case "Propriétaire(s) : ":
				next(z)
				for owner := next(z); owner != "\n"; owner = next(z) {
					p.Owners = append(p.Owners, owner)
					next(z)
				}
			default:
				if debug {
					fmt.Printf("text: %q\n", t)
				}
			}
		default:
			if debug {
				fmt.Printf("other: %v\n", tt)
			}
		}
	}
	return
}

// removes " m" on the end of a string ("801 m" -> "801")
// removes ' in the middle ("8'001 m" -> "8001")
func trimM(in string) string {
	in = strings.Replace(in, "'", "", -1)
	if strings.HasSuffix(in, " m") {
		return in[:len(in)-2]
	}
	return in
}
