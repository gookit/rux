package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"
)

var client *http.Client
// var debug bool

func main() {
	client = createHttpClient()

	fmt.Println("Listen on http://127.0.0.1:3000")

	// create server
	if err := http.ListenAndServe(":3000", http.HandlerFunc(doHandle)); err != nil {
		log.Fatal(err)
	}
}

// link https://studygolang.com/articles/21423?fr=sidebar
func doHandle(w http.ResponseWriter, r *http.Request) {
	// get target url. eg: https://baidu.com/ss/yy
	apiUrl := r.Header.Get("Target-Url")
	if apiUrl == "" {
		responseJSON(w, 200, map[string]interface{}{
			"code": 400,
			"msg": "remote target url cannot be empty",
			"data": map[string]string{},
		})
		return
	}

	log.Println("request target url", apiUrl)

	// create request
	req, err := http.NewRequest(r.Method, apiUrl, r.Body)
	if err != nil {
		responseJSON(w, 200, map[string]interface{}{
			"code": 400,
			"msg": "create request fail, error: " + err.Error(),
			"data": map[string]string{},
		})
		return
	}

	// 注： 设置Request头部信息
	for k, v := range r.Header {
		for _, vv := range v {
			req.Header.Add(k, vv)
		}
	}

	// do request
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// 注： 设置Response头部信息
	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}

	// 1. use io.copy
	// l, _ := io.Copy(w, r.Body) // fail
	// fmt.Println("written", l)
	// 2. use io.copy
	// w.(io.ReaderFrom).ReadFrom(r.Body)  // fail
	// 3. direct read
	data, _ := ioutil.ReadAll(resp.Body)  // ok
	_,_ = w.Write(data)
}

func createHttpClient() *http.Client {
	proxy := func(r *http.Request) (*url.URL, error) {
		apiUrl := r.Header.Get("Target-Url")
		log.Println("proxy url", apiUrl)
		return url.Parse(apiUrl) // 127.0.0.1:8099
	}

	dialCtx := (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		// DualStack: true,
		// FallbackDelay: 5 * time.Second,
	}).DialContext
	transport := &http.Transport{
		Proxy: proxy,

		DialContext:  dialCtx,
		MaxIdleConns: 100,

		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   20,
	}

	// create client
	return &http.Client{Transport: transport}
}

func responseJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)

	bs, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	_,_ =w.Write(bs)
}
