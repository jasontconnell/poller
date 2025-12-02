package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jasontconnell/poller/conf"
	"github.com/jasontconnell/poller/templ"
)

type site struct {
	http.Handler
	domains   map[string]conf.Domain
	requests  []request
	responses sync.Map
	template  *template.Template
	interval  int
}

func newSite(domains map[string]conf.Domain, reqs []request, tpl *template.Template, interval int) *site {
	s := new(site)
	s.domains = domains
	s.requests = reqs
	s.template = tpl
	s.responses = sync.Map{}
	s.interval = interval
	if s.interval <= 60 {
		s.interval = 60
	}
	return s
}

type request struct {
	index       int
	method      string
	domainKey   string
	path        string
	contentType string
	body        string
}

func (req request) String() string {
	return fmt.Sprintf("%s %s %s %s %d", req.method, req.domainKey, req.path, req.contentType, len(req.body))
}

type response struct {
	request request
	status  int
	last    time.Time
}

type UrlResultModel struct {
	Index      int
	Url        string
	StatusText string
	StatusCode int
	Success    bool
	Last       string
}

func process(c *http.Client, url, method, contentType, body string, headers map[string]string) (int, error) {
	bd := bytes.NewBufferString(body)
	req, err := http.NewRequest(method, url, bd)
	if err != nil {
		return 0, fmt.Errorf("can't create request %s %s %s %s %w", url, method, contentType, body, err)
	}

	if headers != nil {
		for k, v := range headers {
			req.Header.Add(k, v)
		}
	}

	if contentType != "" {
		req.Header.Add("Content-Type", contentType)
	}

	resp, err := c.Do(req)
	if err != nil {
		return -1, err
	}

	return resp.StatusCode, nil
}

func poll(s *site) {
	for {
		c := &http.Client{
			Timeout: time.Second * 5,
		}

		var wg sync.WaitGroup
		wg.Add(len(s.requests))

		for i := 0; i < len(s.requests); i++ {
			req := s.requests[i]

			domain, ok := s.domains[req.domainKey]
			if !ok {
				log.Println("can't locate domain", req.domainKey)
				wg.Done()
				continue
			}

			url := domain.Scheme + "://" + domain.Domain + req.path

			go func(u request) {
				now := time.Now()
				status, err := process(c, url, u.method, u.contentType, u.body, domain.Headers)
				if err != nil {
					log.Println(err)
				}
				resp := response{request: u, status: status, last: now}
				s.responses.Store(u.String(), resp)
				wg.Done()
			}(req)
		}

		wg.Wait()

		time.Sleep(time.Second * time.Duration(s.interval))
	}
}

func (s *site) statusHandler(w http.ResponseWriter, r *http.Request) {
	t := s.template
	model := []UrlResultModel{}

	s.responses.Range(func(k any, v any) bool {
		resp := v.(response)
		success := false
		var statusText string
		switch resp.status {
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

		var l time.Duration
		l = time.Since(resp.last).Truncate(time.Second)

		res := UrlResultModel{
			Index:      resp.request.index,
			Url:        resp.request.domainKey + " " + resp.request.path,
			StatusCode: resp.status,
			StatusText: statusText,
			Success:    success,
			Last:       l.String(),
		}

		model = append(model, res)

		return true
	})

	sort.Slice(model, func(i, j int) bool {
		return model[i].Index < model[j].Index
	})
	t.Execute(w, struct{ Items []UrlResultModel }{model})
}

func main() {
	cfn := flag.String("c", "config.json", "config filename")
	reqFile := flag.String("r", "", "requests filename")
	flag.Parse()

	cfg, err := conf.LoadConfig(*cfn)
	if err != nil {
		log.Fatal(err)
	}

	domains := make(map[string]conf.Domain)
	for _, d := range cfg.Domains {
		domains[d.Key] = d
	}

	t := template.Must(template.New("Results").Parse(templ.TemplateString))

	if *reqFile == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	list, err := getUrls(*reqFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	s := newSite(domains, list, t, cfg.Interval)
	mux := http.NewServeMux()
	mux.HandleFunc("/status", s.statusHandler)

	go poll(s)

	s.Handler = mux

	log.Println("starting listening...")
	http.ListenAndServe(fmt.Sprintf("%s:%d", cfg.Site.Host, cfg.Site.Port), s)
}

func getUrls(path string) ([]request, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	list := []request{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		var body string
		if fields[0] == "POST" && len(fields) >= 5 {
			body = strings.Join(fields[4:], " ")
		}

		var contentType string
		if len(fields) > 3 {
			contentType = fields[3]
		}

		res := request{index: len(list), method: fields[0], domainKey: fields[1], path: fields[2], contentType: contentType, body: body}
		list = append(list, res)
	}

	return list, nil
}
