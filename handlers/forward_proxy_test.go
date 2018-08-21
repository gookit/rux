package handlers

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMultiHostsReverseProxy(t *testing.T) {
	// art := assert.New(t)
	//
	// rp := NewForwardProxy(&url.URL{
	// 	Scheme: "http",
	// 	Host:   "localhost:9091",
	// }, &url.URL{
	// 	Scheme: "http",
	// 	Host:   "localhost:9092",
	// })
	//
	// art.NotNil(rp)

	// log.Fatal(http.ListenAndServe(":9090", proxy))
}

func TestNewForwardProxy(t *testing.T) {
	art := assert.New(t)
	fp := NewForwardProxy(map[string]string{
		"/api": "127.0.0.1:8345",
	})
	art.NotNil(fp)
}
