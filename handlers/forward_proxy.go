package handlers

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

/*************************************************************
 * Forward proxy
 *************************************************************/

// ForwardProxy definition.
// ref links:
// https://blog.csdn.net/mengxinghuiku/article/details/65448600
type ForwardProxy struct {
	// open debug
	Debug bool
	// Proxy proxy map
	// {
	// 	"from": "to",
	// 	"/api": "/api/v2",
	// }
	Proxy map[string]string
}

// NewForwardProxy create a ForwardProxy instance
func NewForwardProxy(proxyMap map[string]string) *ForwardProxy {
	fp := &ForwardProxy{Proxy: proxyMap}

	return fp
}

func (fp *ForwardProxy) match(r *http.Request) bool {
	path := r.URL.Path
	for from, to := range fp.Proxy {
		if from == path {
			// r.URL = to // change URL
			r.URL.Path = to // change URL path
			return true
		}
	}

	return false
}

// ServeHTTP implement the http.Handler interface
func (fp *ForwardProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if fp.Debug {
		fmt.Printf("Received request %s %s %s\n", r.Method, r.Host, r.RemoteAddr)
	}

	// step 1
	nr := new(http.Request)
	*nr = *r // this only does shallow copies of maps

	if clientIP, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if prior, ok := nr.Header["X-Forwarded-For"]; ok {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		nr.Header.Set("X-Forwarded-For", clientIP)
	}

	// step 2
	transport := http.DefaultTransport
	res, err := transport.RoundTrip(nr)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	// step 3
	for key, value := range res.Header {
		for _, v := range value {
			w.Header().Add(key, v)
		}
	}

	w.WriteHeader(res.StatusCode)
	io.Copy(w, res.Body)
	res.Body.Close()
}

// HTTPHandler wrap and returns http.Handler func
func (fp *ForwardProxy) HTTPHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !fp.match(r) {
			next.ServeHTTP(w, r)
		}

		fp.ServeHTTP(w, r)
	})
}
