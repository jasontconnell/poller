package main

import (
	"bufio"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/jasontconnell/server"
)

type urlResult struct {
	url    string
	status int
	retry  time.Time
	last   time.Time
}

type UrlResultModel struct {
	Url        string
	StatusText string
	StatusCode int
	Success    bool
	Retry      string
	Last       string
}

func get(c *http.Client, url string) (int, error) {
	resp, err := c.Get(url)
	if err != nil {
		return -1, err
	}

	return resp.StatusCode, nil
}

func poll(site *server.Site, list []*urlResult) {
	for {
		c := &http.Client{
			Timeout: time.Second * 5,
		}

		var wg sync.WaitGroup
		wg.Add(len(list))

		for i := 0; i < len(list); i++ {
			u := list[i]
			now := time.Now()

			if u.retry.After(now) {
				wg.Done()
				continue
			}

			u.last = time.Now()
			go func(u *urlResult) {
				s, err := get(c, u.url)
				if err != nil {
					fmt.Println(err)
				}
				u.status = s
				wg.Done()
			}(u)

			u.retry = time.Now().Add(1 * time.Minute)
		}

		wg.Wait()

		time.Sleep(time.Second * 5)
	}
}

func statusHandler(site server.Site, w http.ResponseWriter, r *http.Request) {
	list := site.GetState("urls").([]*urlResult)
	t := site.GetState("template").(*template.Template)
	model := []UrlResultModel{}

	for _, u := range list {

		success := false
		var statusText string
		switch u.status {
		case 200:
			success = true
			statusText = "ok"
		case 304:
			statusText = "redirected"
			success = true
		case 404:
			statusText = "not found"
			success = false
		case 500, 501, 502, 503:
			statusText = "server error"
			success = false
		default:
			statusText = "no response"
			success = false
		}

		var r, l time.Duration
		r = time.Until(u.retry).Truncate(time.Second)
		l = time.Since(u.last).Truncate(time.Second)

		res := UrlResultModel{
			Url:        u.url,
			StatusCode: u.status,
			StatusText: statusText,
			Success:    success,
			Last:       l.String(),
			Retry:      r.String(),
		}

		model = append(model, res)
	}

	t.Execute(w, struct{ Items []UrlResultModel }{model})

}

func reloadHandler(site server.Site, w http.ResponseWriter, r *http.Request) {
	path := site.GetState("path").(string)
	list, err := getUrls(path)

	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	site.AddState("urls", list)

	w.Write([]byte("Done"))
}

func main() {
	urlsfile := flag.String("u", "", "url filename")
	host := flag.String("h", "", "hostname")
	port := flag.Int("p", 4444, "port")
	flag.Parse()

	site := server.NewSite(server.Configuration{HostName: *host, Port: *port})
	t := template.Must(template.New("Results").Parse(templateString))

	site.AddHandler("/status", statusHandler)
	site.AddHandler("/reload", reloadHandler)

	if *urlsfile == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	list, err := getUrls(*urlsfile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	site.AddState("urls", list)
	site.AddState("path", *urlsfile)
	site.AddState("template", t)
	go poll(site, list)

	server.Start(*site)
}

func getUrls(path string) ([]*urlResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	list := []*urlResult{}
	sc := bufio.NewScanner(f)
	now := time.Now()
	ret := now.Add(5 * time.Second)
	for sc.Scan() {
		res := urlResult{url: sc.Text(), retry: ret.Add(1 * time.Second)}
		list = append(list, &res)
	}

	return list, nil
}
