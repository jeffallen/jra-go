package what3words

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// Results can come back with lat/long quoted or not. To make the JSON
// decoder happy, we use this regexp to fix them.
// Example broken one:
// {"type":"oneword","lat":"51.498262","lng":"-0.218548","text":"*johnoffice","info":false}

var fixLL = regexp.MustCompile("\"(lat|lng)\":\"([0-9.-]+)\"")

type Location struct {
	Type  string
	Lat   float32
	Lng   float32
	Text  string
	Info  bool
	Error string
}

// a template for decoding this JSON that comes back from a lng/lat query
// {"words":["codes","championed","dumbly"],"position":[51.933512,-8.597174]}
type position struct {
	Words    []string
	Position []float32
	Error    string
}

type Session struct {
	client *http.Client
}

var defaultSession = NewSession(http.DefaultClient)

func NewSession(client *http.Client) *Session {
	return &Session{client: client}
}

func (s *Session) SearchPosition(lat, lng float32) (loc Location, err error) {
	ll := fmt.Sprintf("%0.6f,%0.6f", lat, lng)
	url := "http://a.what3words.com/calls/position/" + ll

	resp, err := s.client.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	d := json.NewDecoder(resp.Body)

	var pos position
	err = d.Decode(&pos)
	if err != nil {
		return
	}
	if err = getError(pos.Error); err != nil {
		return
	}

	if len(pos.Position) != 2 {
		err = ErrBadPosition
		return
	}

	return Location{
		Type: "position",
		Lat:  pos.Position[0],
		Lng:  pos.Position[1],
		Text: strings.Join(pos.Words, "."),
		Info: false,
	}, nil
}

func (s *Session) SearchWords(words string) ([]Location, error) {
	url := "http://a.what3words.com/calls/search/" + words
	resp, err := s.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b := &bytes.Buffer{}
	io.Copy(b, resp.Body)
	b = bytes.NewBuffer(fixLL.ReplaceAll(b.Bytes(), []byte("\"$1\":$2")))

	d := json.NewDecoder(b)

	var loc Location
	err = d.Decode(&loc)
	if err != nil {
		return nil, err
	}

	if err = getError(loc.Error); err != nil {
		return nil, err
	}

	return []Location{loc}, nil
}

func SearchWords(words string) ([]Location, error) {
	return defaultSession.SearchWords(words)
}

func SearchPosition(lat, lng float32) (Location, error) {
	return defaultSession.SearchPosition(lat, lng)
}

var ErrNotFound = errors.New("not found")
var ErrInvalidPosition = errors.New("invalid position string")
var ErrBadPosition = errors.New("result had incorrect position array length")

func getError(in string) error {
	switch in {
	case "":
		return nil
	case "not found":
		return ErrNotFound
	case "Invalid position: invalid position string":
		return ErrInvalidPosition
	case "Invalid position: Position not found":
		return ErrInvalidPosition
	default:
		return errors.New(in)
	}
}
