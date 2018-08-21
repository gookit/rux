package reverseproxy

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
)

/*************************************************************
 * Reverse proxy
 *************************************************************/

type Proxy struct {
	opts Options
}

// Options for proxy
type Options struct {
	// open debug
	Debug bool
	// Target host http://www.example.org
	Target string
	// Filter custom filter for check whether need proxy
	Filter func(path string, r *http.Request) bool
	// ChangeOrigin needed for virtual hosted sites
	ChangeOrigin bool
	// WS bool
	// PathRewrite url path rewrite
	// 	{
	//      '^/api/old-path' : '/api/new-path',     // rewrite path
	//      '^/api/remove/path' : '/path'           // remove base path
	//		'^/' : '/basePath/'  					// add base path
	//   },
	PathRewrite map[string]string
	LogLevel    int
	Logger      *log.Logger
	Events      map[string]func()
}

// New a proxy instance
func New(opts *Options) *Proxy {
	rp := &Proxy{}

	return rp
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if p.opts.Debug {
		fmt.Printf("Received request %s %s %s\n", req.Method, req.Host, req.RemoteAddr)
	}
}

// MultiReverseProxy create a global reverse proxy.
// usage:
// 	rp := ReverseProxy(&url.URL{
// 		Scheme: "http",
// 		Host:   "localhost:9091",
// 	}, &url.URL{
// 		Scheme: "http",
// 		Host:   "localhost:9092",
// 	})
// 	log.Fatal(http.ListenAndServe(":9090", rp))
func MultiReverseProxy(targets ...*url.URL) *httputil.ReverseProxy {
	if len(targets) == 0 {
		panic("Please add at least one remote target server")
	}

	var target *url.URL

	// if only one target
	if len(targets) == 1 {
		target = targets[0]
	}

	director := func(req *http.Request) {
		if len(targets) > 1 {
			target = targets[rand.Int()%len(targets)]
		}

		fmt.Printf("Received request %s %s %s\n", req.Method, req.Host, req.RemoteAddr)

		targetQuery := target.RawQuery

		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = target.Path
		// req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)

		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}

	return &httputil.ReverseProxy{Director: director}
}
