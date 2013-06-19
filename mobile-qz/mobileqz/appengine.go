// +build appengine

package mobileqz

import (
	"appengine"
	"appengine/memcache"
	"appengine/urlfetch"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"
)

var page *template.Template

// A Page is passed into the main HTML page template.
type Page struct {
	Title string
	Body  string
}

type Response struct {
	Results []Result
	// also available:
	// Recent
	// Header
}

type Author struct {
	Name     string
	Twitter  string
	Url      string
	Username string
}

type Byline struct {
	BylineString string
	Authors      []Author
}

type Result struct {
	Id         int
	Title      string
	Permalink  string
	Summary    string
	Content    string
	Byline     Byline
	Taxonomies Taxonomy
}

type Taxonomy struct {
	Kicker []Tag
	Tags   []Tag
}

type Tag struct {
	Name string
	Tag  string
}

func (r Result) byline() string {
	if r.Byline.BylineString != "" {
		return r.Byline.BylineString
	}

	lines := make([]string, len(r.Byline.Authors))
	for i, a := range r.Byline.Authors {
		lines[i] = fmt.Sprintf("<a href=\"%v\">%v</a>", a.Url, a.Name)
	}
	return strings.Join(lines, ", ")
}

func (r Result) tags() string {
	t := r.Taxonomies.Kicker
	t = append(t, r.Taxonomies.Tags...)

	t2 := make([]string, len(t))
	for i, tag := range t {
		t2[i] = tag.Name
	}
	return strings.Join(t2, ", ")
}

func init() {
	http.HandleFunc("/", frontPage)
	http.HandleFunc("/article/", articlePage)
	http.HandleFunc("/about", aboutPage)

	page = template.Must(template.New("page").Parse(`<!DOCTYPE html>
<html>
<head>
  <title>{{.Title}}</title>
</head>
<body>
  {{/* Body should already be HTML escaped */}}
  {{ printf "%s" .Body }}
<script src="/scripts/sz.js"></script>
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
	return &res, err
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
		Key:        "api/top",
		Object:     &r,
		Expiration: 300 * time.Second,
	}
	memcache.Gob.Set(ctx, i)

	// fix stuff I don't like about their content...
	for i := range r.Results {
		r.Results[i].Title = fixHeadline(r.Results[i].Title)
	}

	return &r, nil
}

func aboutPage(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	fmt.Fprintf(w, "<h1>%v</h1>", appengine.DefaultVersionHostname(c))

	token, expire, _ := appengine.AccessToken(c, "test")
	fmt.Fprintf(w, "<p>AccessToken: %v %v", token, expire)

	fmt.Fprintf(w, "<p>AppID: %v", appengine.AppID(c))
	fmt.Fprintf(w, "<p>FQAppID: %v", c.FullyQualifiedAppID())
	fmt.Fprintf(w, "<p>Go version: %v", runtime.Version())
	fmt.Fprintf(w, "<p>Datacenter: %v", appengine.Datacenter())
	fmt.Fprintf(w, "<p>InstanceID: %v", appengine.InstanceID())
	fmt.Fprintf(w, "<p>IsDevAppServer: %v", appengine.IsDevAppServer())
	fmt.Fprintf(w, "<p>RequestID: %v", appengine.RequestID(c))
	fmt.Fprintf(w, "<p>ServerSoftware: %v", appengine.ServerSoftware())

	sa, _ := appengine.ServiceAccount(c)
	fmt.Fprintf(w, "<p>ServiceAccount: %v", sa)

	keyname, signed, _ := appengine.SignBytes(c, []byte("test"))
	fmt.Fprintf(w, "<p>SignBytes: %v %v", keyname, signed)
	fmt.Fprintf(w, "<p>VersionID: %v", appengine.VersionID(c))

	fmt.Fprintf(w, "<p>Request: %v", r)
	r2 := c.Request()
	fmt.Fprintf(w, "<p>Context Request type/value: %T %v", r2, r2)
}

func articlePage(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	q, err := getQuartz(c)
	if err != nil {
		fmt.Fprintf(w, "Error: %v", err)
		c.Errorf("%v", err)
		return
	}

	artstr := strings.Trim(r.URL.Path, "/article/")
	wanted, err := strconv.ParseUint(artstr, 10, 64)
	if err != nil {
		c.Errorf("%v", err)
		return
	}

	var art Result
	for _, r := range q.Results {
		if r.Id == int(wanted) {
			art = r
		}
	}

	// if they told us their screen size, then adapt images to it
	if ck, err := r.Cookie("w"); err == nil {
		if w, err := strconv.Atoi(ck.Value); err == nil {
			//c.Debugf("got w cookie with w=%v", w)
			// Android cell phone reports width 800 even
			// when zoomed to make text readable.
			if w == 800 {
				art.Content = fixImages(art.Content, 400)
			}
		}
	}

	b := &bytes.Buffer{}
	fmt.Fprintf(b, "<h1> %s </h1>\n", art.Title)
	fmt.Fprintf(b, "<p>By: %v\n", art.byline())
	fmt.Fprintf(b, "<small><i><p>%v</p></i></small>\n", art.tags())
	fmt.Fprintf(b, "%v", art.Content)

	p := Page{Title: art.Title, Body: string(b.Bytes())}
	page.Execute(w, p)
}

func frontPage(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	q, err := getQuartz(c)
	if err != nil {
		fmt.Fprintf(w, "Error: %v", err)
		c.Errorf("%v", err)
		return
	}

	b := &bytes.Buffer{}
	fmt.Fprintf(b, "<dl>")
	for _, r := range q.Results {
		fmt.Fprintf(b, "<dt><a href=\"/article/%v\">%v</a></dt>\n", r.Id, r.Title)
		fmt.Fprintf(b, "<dd>%v</dd>\n", r.Summary)
	}
	fmt.Fprintf(b, "</ul>")

	p := Page{Title: "Mobile Quartz: Front Page", Body: string(b.Bytes())}
	page.Execute(w, p)
}
