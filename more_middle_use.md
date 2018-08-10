# Third party middleware 

## thoas/stats

`thoas/stats` 一个Go中间件，用于存储有关Web应用程序的各种信息（响应时间，状态代码计数等）

```go
package main

import (
        "encoding/json"
        "github.com/gookit/sux"
        "github.com/thoas/stats"
        "net/http"
)

func main() {
	r := sux.New()
	s := stats.New()

	r.GET("/", func(c *sux.Context) {
		c.Text(200, "hello")
	})
	// add routes ...
	r.GET("/stats", func(c *sux.Context) {
		bs, err := json.Marshal(s.Data())
        if err != nil {
        	c.HTTPError(err.Error(), http.StatusInternalServerError)
        	return 
        }
        
	    c.JSONBytes(200, bs)
	})
	
    // Wrap our server with our gzip handler to gzip compress all responses.
    http.ListenAndServe(":8000", s.Handler(r))
}
```