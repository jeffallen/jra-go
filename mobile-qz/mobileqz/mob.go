package mobileqz

import (
	"time"
	"bytes"
	"appengine"
	"appengine/urlfetch"
	"appengine/memcache"
	"encoding/json"
	"fmt"
	"strings"
	"strconv"
	"io/ioutil"
	"log"
	"net/http"
	"text/template"
)

var page *template.Template

// A Page is passed into the main HTML page template.
type Page struct {
	Title string
	Body string
}

type Response struct {
	Results []Result
	// also available:
	// Recent
	// Header
}

type Author struct {
	Name string
	Twitter string
	Url	string
	Username	string
}

type Byline struct {
	BylineString string
	Authors []Author
}

type Result struct {
	Id int
	Title     string
	Permalink string
	Summary   string
	Content   string
	Byline	Byline
	Taxonomies Taxonomy
}

type Taxonomy struct {
	Kicker []Tag
	Tags []Tag
}

type Tag struct {
	Name	string
	Tag	string
}

func (r Result)byline() string {
	if r.Byline.BylineString != "" {
		return r.Byline.BylineString
	}

	lines := make([]string, len(r.Byline.Authors))
	for i,a := range r.Byline.Authors {
		lines[i] = fmt.Sprintf("<a href=\"%v\">%v</a>", a.Url, a.Name)
	}
	return strings.Join(lines, ", ")
}

func (r Result)tags() string {
	t := r.Taxonomies.Kicker
	t = append(t, r.Taxonomies.Tags...)

	t2 := make([]string, len(t))
	for i,tag := range t {
		t2[i] = tag.Name
	}
	return strings.Join(t2, ", ")
}

func init() {
	http.HandleFunc("/", frontPage)
	http.HandleFunc("/article/", articlePage)

	page = template.Must(template.New("page").Parse(`<!DOCTYPE html>
<html>
<head>
  <title>{{html .Title}}</title>
</head>
<body>
  {{/* Body should already be HTML escaped */}}
  {{ printf "%s" .Body }}
</body>
</html>`))
}

func getQuartz(ctx appengine.Context) (*Response, error) {
	if r, err := getQuartzFromCache(ctx); err == nil {
		return r, err
	}
	return getQuartzFromNet(ctx)
}

func getQuartzFromCache(ctx appengine.Context) (*Response, error) {
	var res Response
	_, err := memcache.Gob.Get(ctx, "api/top", &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func getQuartzFromNet(ctx appengine.Context) (*Response, error) {
	resp, err := urlfetch.Client(ctx).Get("http://qz.com/api/top")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var r Response
	err = json.Unmarshal(b, &r)
	if err != nil {
		return nil, err
	}

	i := &memcache.Item{
		Key: "api/top",
		Object: &r,
		Expiration: 300 * time.Second,
	}
	memcache.Gob.Set(ctx, i)

	return &r, nil
}

func articlePage(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	q, err := getQuartz(c)
	if err != nil {
		fmt.Fprintf(w, "Error: %v", err)
		log.Fatal(err)
		return
	}

	artstr := strings.Trim(r.URL.Path, "/article/")
	wanted, err := strconv.ParseUint(artstr, 10, 64)
	if err != nil {
		log.Fatal(err)
		return
	}

	var art Result
	for _,r := range q.Results {
		if r.Id == int(wanted) {
			art = r
		}
	}

	b := &bytes.Buffer{}
	fmt.Fprintf(b, "<h1> %s </h1>\n", art.Title)
	fmt.Fprintf(b, "<p>By: %v\n", art.byline())
	fmt.Fprintf(b, "<small><i><p>%v</p></i></small>\n", art.tags())
	fmt.Fprintf(b, "%v", art.Content)

	p := Page{ Title: "Mobile Quartz: Front Page", Body: string(b.Bytes()) }
	page.Execute(w, p)
}

func frontPage(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	q, err := getQuartz(c)
	if err != nil {
		fmt.Fprintf(w, "Error: %v", err)
		log.Fatal(err)
		return
	}

	b := &bytes.Buffer{}
	fmt.Fprintf(b, "<dl>")
	for _, r := range q.Results {
		fmt.Fprintf(b, "<dt><a href=\"/article/%v\">%v</a></dt>\n", r.Id, r.Title)
		fmt.Fprintf(b, "<dd>%v</dd>\n", r.Summary)
	}
	fmt.Fprintf(b, "</ul>")

	p := Page{ Title: "Mobile Quartz: Front Page", Body: string(b.Bytes()) }
	page.Execute(w, p)
}
