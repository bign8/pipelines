package main

import (
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"sync"

	"github.com/bign8/pipelines/new"
	"github.com/bign8/pipelines/new/web"
)

var (
	tpl   = template.Must(template.New("").Parse(page))
	port  = flag.Int("port", 9999, "port to start server on")
	title = regexp.MustCompile(`<title>(.*)</title>`)
)

type ui struct {
	Added  []string
	Stored []string
	Error  error
	mutex  sync.RWMutex
}

func (u *ui) Gen(pipelines.Stream, pipelines.Key) pipelines.Worker {
	return u
}

func (u *ui) Work(unit pipelines.Unit) error {
	u.mutex.Lock()
	u.Stored = append(u.Stored, string(title.FindSubmatch(unit.Load())[1]))
	u.mutex.Unlock()
	return nil
}

func (u *ui) handleAdd(w http.ResponseWriter, r *http.Request) {
	// Validate
	uri := r.FormValue("url")
	_, err := url.Parse(uri)
	if err != nil {
		u.mutex.Lock()
		u.Error = err
		tpl.Execute(w, u)
		u.Error = nil
		u.mutex.Unlock()
		return
	}

	// Add to UI
	u.mutex.Lock()
	u.Added = append(u.Added, uri)
	u.mutex.Unlock()

	// Emit type
	pipelines.EmitType(web.StreamINDEX, web.TypeADDR, []byte(uri))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (u *ui) handleIndex(w http.ResponseWriter, r *http.Request) {
	u.mutex.RLock()
	tpl.Execute(w, u)
	u.mutex.RUnlock()
}

func main() {
	flag.Parse()
	server := &ui{
		Added:  make([]string, 0),
		Stored: make([]string, 0),
	}
	http.HandleFunc("/add", server.handleAdd)
	http.HandleFunc("/", server.handleIndex)

	pipelines.Register(pipelines.Config{
		Name: "ui",
		Inputs: map[pipelines.Stream]pipelines.Mine{
			web.StreamSTORE: pipelines.MineConstant,
		},
		Output: map[pipelines.Stream]pipelines.Type{
			web.StreamINDEX: web.TypeADDR,
		},
		Create: server.Gen,
	})

	fmt.Printf("Serving on :%d\n", *port)
	http.ListenAndServe(":"+strconv.Itoa(*port), nil)
}

var page = `
<html>
  <head>
  </head>
  <body>
    <h1>Pipelines Admin</h1>
    <p>Yippie!!!</p>
    {{if .Error}}
      <p><strong>Invalid URL</strong> - <small>{{.Error}}</small> - Try again</p>
    {{end}}
    <hr/>
    <form action="/add" method="POST">
      <input type="text" name="url" />
      <input type="submit" value="Crawl" />
    </form>
    <hr/>
    <h1>Added</h1>
    <ul>
      {{range .Added}}
      <li>{{.}}</li>
      {{end}}
    </ul>
    <hr/>
    <h1>Stored</h1>
    <ul>
      {{range .Stored}}
      <li>{{.}}</li>
      {{end}}
    </ul>
  </body>
</html>
`
