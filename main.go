package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/jasontconnell/server"
	"net/http"
	"os"
	"time"
)

type urlResult struct {
	url    string
	status int
	retry  time.Time
	last   time.Time
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

		for i := 0; i < len(list); i++ {
			u := list[i]
			now := time.Now()

			if u.retry.After(now) {
				continue
			}

			u.last = time.Now()
			s, err := get(c, u.url)
			if err != nil {
				fmt.Println(err)
			}

			u.status = s
			u.retry = time.Now().Add(1 * time.Minute)
		}

		time.Sleep(time.Second * 5)
	}
}

func statusHandler(site server.Site, w http.ResponseWriter, r *http.Request) {
	list := site.GetState("urls").([]*urlResult)
	content := "<html><head><style>.success { background-color: #3f3; } .redir { background-color: #999; } .error { background-color: #f33; }</style></head><body><table><thead><tr><th>Url</th><th>Status</th><th>Next Retry</th><th>Last Retry</th></tr></thead><tbody>"
	for _, u := range list {
		var r, l time.Duration
		r = time.Until(u.retry).Truncate(time.Second)
		l = time.Since(u.last).Truncate(time.Second)
		var css string
		switch u.status {
		case 200:
			css = "success"
		case 304:
			css = "redir"
		default:
			css = "error"
		}
		content += fmt.Sprintf("<tr><td>%s</td><td class=\"%s\">%d</td><td>in %s</td><td>%s ago</td></tr>", u.url, css, u.status, r, l)
	}
	content += "</tbody></table></body></html>"
	w.Write([]byte(content))
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
	flag.Parse()

	site := server.NewSite(server.Configuration{HostName: "localhost", Port: 4444})
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
	go poll(&site, list)

	server.Start(site)
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
