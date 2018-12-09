# rux 路由器

[![GoDoc](https://godoc.org/github.com/gookit/rux?status.svg)](https://godoc.org/github.com/gookit/rux)
[![Build Status](https://travis-ci.org/gookit/rux.svg?branch=master)](https://travis-ci.org/gookit/rux)
[![Coverage Status](https://coveralls.io/repos/github/gookit/rux/badge.svg?branch=master)](https://coveralls.io/github/gookit/rux?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/gookit/rux)](https://goreportcard.com/report/github.com/gookit/rux)

简单且快速的 Go HTTP 请求路由器，支持中间件，兼容 http.Handler 接口。

> **[EN README](README.md)**

- 支持路由参数，支持路由组，支持给路由命名
- 支持方便的静态文件/目录处理
- 支持缓存最近访问的动态路由以获得更高性能
- 支持中间件: 路由中间件，组中间件，全局中间件。
- 兼容支持 `http.Handler` 接口，可以直接使用其他的常用中间件
- 支持添加 `NotFound` 和 `NotAllowed` 处理

## GoDoc

- [godoc for gopkg](https://godoc.org/gopkg.in/gookit/rux.v1)
- [godoc for github](https://godoc.org/github.com/gookit/rux)

## 快速开始

```go
package main

import (
	"github.com/gookit/rux"
)

func main() {
	r := rux.New()
	
	// ===== 静态资源
	// 单个文件
	r.StaticFile("/site.js", "testdata/site.js")
	// 静态资源目录
	r.StaticDir("/static", "testdata")
	// 静态资源目录，但是有后缀限制
	r.StaticFiles("/assets", "testdata", "css|js")

	// ===== 添加路由
	
	r.GET("/", func(c *rux.Context) {
		c.Text(200, "hello")
	})
	r.GET("/hello/{name}", func(c *rux.Context) {
		c.Text(200, "hello " + c.Param("name"))
	})
	r.POST("/post", func(c *rux.Context) {
		c.Text(200, "hello")
	})
	r.Group("/articles", func() {
		r.GET("", func(c *rux.Context) {
			c.Text(200, "view list")
		})
		r.POST("", func(c *rux.Context) {
			c.Text(200, "create ok")
		})
		r.GET(`/{id:\d+}`, func(c *rux.Context) {
			c.Text(200, "view detail, id: " + c.Param("id"))
		})
	})

	// 启动服务并监听
	r.Listen(":8080")
	// 也可以
	// http.ListenAndServe(":8080", r)
}
```

## 使用中间件

支持使用中间件:

- 全局中间件
- 路由组中间件
- 路由中间件

**调用优先级**: `全局中间件 -> 路由组中间件 -> 路由中间件`

### 使用示例

```go
package main

import (
	"fmt"
	"github.com/gookit/rux"
)

func main() {
	r := rux.New()
	
	// 添加一个全局中间件
	r.Use(func(c *rux.Context) {
	    // do something ...
	})
	
	// 使用中间件作为路由
	route := r.GET("/middle", func(c *rux.Context) { // main handler
		c.WriteString("-O-")
	}, func(c *rux.Context) { // middle 1
        c.WriteString("a")
        c.Next() // Notice: call Next()
        c.WriteString("A")
        // if call Abort(), will abort at the end of this middleware run
        // c.Abort() 
    })
	// add by Use()
	route.Use(func(c *rux.Context) { // middle 2
		c.WriteString("b")
		c.Next()
		c.WriteString("B")
	})

	// now, access the URI /middle
	// will output: ab-O-BA
}
```

- **调用流程图**:

```text
        +-----------------------------+
        | middle 1                    |
        |  +----------------------+   |
        |  | middle 2             |   |
 start  |  |  +----------------+  |   | end
------->|  |  |  main handler  |  |   |--->----
        |  |  |________________|  |   |    
        |  |______________________|   |  
        |_____________________________|
```

> 更多的使用请查看 [middleware_test.go](middleware_test.go) 中间件测试

## 使用`http.Handler`

rux 支持通用的 `http.Handler` 接口中间件

> 你可以使用 `rux.WrapHTTPHandler()` 转换 `http.Handler` 为 `rux.HandlerFunc`

```go
package main

import (
	"net/http"
	
	"github.com/gookit/rux"
	// 这里我们使用 gorilla/handlers，它提供了一些通用的中间件
	"github.com/gorilla/handlers"
)

func main() {
	r := rux.New()
	
	// create a simple generic http.Handler
	h0 := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		w.Header().Set("new-key", "val")
	})
	
	r.Use(rux.WrapHTTPHandler(h0), rux.WrapHTTPHandler(handlers.ProxyHeaders()))
	
	r.GET("/", func(c *rux.Context) {
		c.Text(200, "hello")
	})
	// add routes ...
	
    // Wrap our server with our gzip handler to gzip compress all responses.
    http.ListenAndServe(":8000", handlers.CompressHandler(r))
}
```

## 多个域名

> code is ref from `julienschmidt/httprouter`

```go
package main

import (
	"github.com/gookit/rux"
	"log"
	"net/http"
)

type HostSwitch map[string]http.Handler

// Implement the ServeHTTP method on our new type
func (hs HostSwitch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if a http.Handler is registered for the given host.
	// If yes, use it to handle the request.
	if router := hs[r.Host]; router != nil {
		router.ServeHTTP(w, r)
	} else {
		// Handle host names for which no handler is registered
		http.Error(w, "Forbidden", 403) // Or Redirect?
	}
}

func main() {
	// Initialize a router as usual
	router := rux.New()
	router.GET("/", Index)
	router.GET("/hello/{name}", func(c *rux.Context) {})

	// Make a new HostSwitch and insert the router (our http handler)
	// for example.com and port 12345
	hs := make(HostSwitch)
	hs["example.com:12345"] = router

	// Use the HostSwitch to listen and serve on port 12345
	log.Fatal(http.ListenAndServe(":12345", hs))
}
```

## 相关项目

- https://github.com/gin-gonic/gin
- https://github.com/gorilla/mux
- https://github.com/julienschmidt/httprouter
- https://github.com/xialeistudio/go-dispatcher

## License

**[MIT](LICENSE)**
