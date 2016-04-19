package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"text/template"
	"time"
)

var mask = fmt.Sprintf("%%0%dd", 20) // int(math.Log10(n))

type tpl struct {
	Title string
	Items []string
}

var t = template.Must(template.New("page").Parse(`<!DOCTYPE html>
<html>
	<head>
		<title>Page {{.Title}}</title>
	</head>
	<body>
		<h1>Page {{.Title}}</h1>
		<ul>{{range .Items}}
			<li>
				<a href="/page/{{ . }}">Link {{ . }}</a>
			</li>{{end}}
		</ul>
	</body>
</html>`))

func toZero(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/page/"+fmt.Sprintf(mask, 0), http.StatusTemporaryRedirect)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/page/", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()

		// Parse Base String
		u, err := strconv.ParseUint(r.URL.Path[6:], 10, 64)
		if err != nil {
			toZero(w, r)
			return
		}

		// Setup Random Links
		links := make([]string, 30)
		random := rand.New(rand.NewSource(int64(u)))
		links[0], links[1] = fmt.Sprintf(mask, u-1), fmt.Sprintf(mask, u+1)
		for j := 2; j < len(links); j++ {
			big := uint64(random.Uint32()) << 32
			links[j] = fmt.Sprintf(mask, big+uint64(random.Uint32()))
		}

		// Randomized first and last position
		x := random.Intn(len(links))
		links[0], links[x] = links[x], links[0]
		x = random.Intn(len(links))
		links[1], links[x] = links[x], links[1]

		// Render template
		t.Execute(w, tpl{fmt.Sprintf(mask, u), links})
		log.Printf("%s %s", r.URL.Path, time.Since(now))
	})
	mux.HandleFunc("/", toZero)
	http.ListenAndServe(":8080", mux)
}
