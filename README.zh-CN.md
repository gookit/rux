# Rux

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/gookit/rux?style=flat-square)
[![Actions Status](https://github.com/gookit/rux/workflows/unit-tests/badge.svg)](https://github.com/gookit/rux/actions)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/tag/gookit/rux)](https://github.com/gookit/rux)
[![GoDoc](https://godoc.org/github.com/gookit/rux?status.svg)](https://pkg.go.dev/github.com/gookit/rux?tab=doc)
[![Coverage Status](https://coveralls.io/repos/github/gookit/rux/badge.svg?branch=master)](https://coveralls.io/github/gookit/rux?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/gookit/rux)](https://goreportcard.com/report/github.com/gookit/rux)

`rux` 简单且快速的 Go web 框架，支持中间件，兼容 http.Handler 接口。

- 支持路由参数，支持路由组，支持给路由命名
- 支持方便的静态文件/目录处理
- 支持缓存最近访问的动态路由以获得更高性能
- 支持中间件: 路由中间件，组中间件，全局中间件
- 支持快速添加 `RESETFul` 或 `Controller` 风格的结构体
- 兼容支持 `http.Handler` 接口，可以直接使用其他的常用中间件
- 支持添加 `NotFound` 和 `NotAllowed` 处理
- 支持添加 `Error` 和 `Panic` 处理错误或异常

## [English](README.md)

English introduction, please see **[EN README](README.md)**

## GoDoc

- [godoc for github](https://pkg.go.dev/github.com/gookit/rux?tab=doc)

## 安装

```bash
go get github.com/gookit/rux
```

## 快速开始

> NOTICE: `v1.3.x` 不完全兼容 `v1.2.x` 版本

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

	// 快速添加多个METHOD支持
	r.Add("/post[/{id}]", func(c *rux.Context) {
		if c.Param("id") == "" {
			// do create post
			c.Text(200, "created")
			return
		}
		
		id := c.Params.Int("id")
		// do update post
		c.Text(200, "updated " + fmt.Sprint(id))
	}, rux.POST, rux.PUT)

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

使用示例:

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
	
	// 通过参数添加中间件
	route := r.GET("/middle", func(c *rux.Context) { // main handler
		c.WriteString("-O-")
	}, func(c *rux.Context) { // middle 1
        c.WriteString("a")
        c.Next() // Notice: call Next()
        c.WriteString("A")
		// if call Abort(), will abort at the end of this middleware run
		// c.Abort() 
    })

	// 通过 Use() 添加中间件
	route.Use(func(c *rux.Context) { // middle 2
		c.WriteString("b")
		c.Next()
		c.WriteString("B")
	})

	// 启动server访问: /middle
	// 将会看到输出: ab-O-BA
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

## 更多功能

### 静态资源

```go
package main

import (
	"embed"	
	"net/http"

	"github.com/gookit/rux"
)

//go:embed static
var embAssets embed.FS

func main() {
	r := rux.New()

	// one file
	r.StaticFile("/site.js", "testdata/site.js")

	// allow any files in the directory.
	r.StaticDir("/static", "testdata")

	// file type limit in the directory
	r.StaticFiles("/assets", "testdata", "css|js")

	// go 1.16+: use embed assets. access: /embed/static/some.html
	r.StaticFS("/embed", http.FS(embAssets))
}
```

### 路由命名

rux 中你可以添加命名路由，根据名称可以从路由器里拿到对应的路由实例 `rux.Route`。

有几种方式添加命名路由：

```go
	r := rux.New()
	
	// Method 1
	myRoute := rux.NewNamedRoute("name1", "/path4/some/{id}", emptyHandler, "GET")
	r.AddRoute(myRoute)

	// Method 2
	rux.AddNamed("name2", "/", func(c *rux.Context) {
		c.Text(200, "hello")
	})

	// Method 3
	r.GET("/hi", func(c *rux.Context) {
		c.Text(200, "hello")
	}).NamedTo("name3", r)
	
	// get route by name
	myRoute = r.GetRoute("name1")
```

### 重定向跳转

```go
r.GET("/", func(c *rux.Context) {
    c.AbortThen().Redirect("/login", 302)
})

// Or
r.GET("/", func(c *rux.Context) {
    c.Redirect("/login", 302)
    c.Abort()
})

r.GET("/", func(c *rux.Context) {
    c.Back()
    c.Abort()
})
```

### 操作Cookies

您可以通过以下方式快速操作Cookies `FastSetCookie()` `DelCookie()`

> 注意：您必须先设置或删除Cookies，然后再调用写入BODY内容的相关方法

```go
r.GET("/setcookie", func(c *rux.Context) {
    c.FastSetCookie("rux_cookie2", "test-value2", 3600)
    c.SetCookie("rux_cookie", "test-value1", 3600, "/", c.Req.URL.Host, false, true)
	c.WriteString("hello, in " + c.URL().Path)
})

r.GET("/delcookie", func(c *rux.Context) {
	val := ctx.Cookie("rux_cookie") // "test-value1"
	c.DelCookie("rux_cookie", "rux_cookie2")
})
```

### 多个域名

> code is ref from `julienschmidt/httprouter`

```go
package main

import (
	"log"
	"net/http"

	"github.com/gookit/rux"
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

### RESETful 风格

```go
package main

import (
	"log"
	"net/http"

	"github.com/gookit/rux"
)

type Product struct {
}

// middlewares [optional]
func (Product) Uses() map[string][]rux.HandlerFunc {
	return map[string][]rux.HandlerFunc{
		// function name: handlers
		"Delete": []rux.HandlerFunc{
			handlers.HTTPBasicAuth(map[string]string{"test": "123"}),
			handlers.GenRequestID(),
		},
	}
}

// all products [optional]
func (p *Product) Index(c *rux.Context) {
	// do something ...
}

// create product [optional]
func (p *Product) Create(c *rux.Context) {
	// do something ...
}

// save new product [optional]
func (p *Product) Store(c *rux.Context) {
	// do something ...
}

// show product with {id} [optional]
func (p *Product) Show(c *rux.Context) {
	// do something ...
}

// edit product [optional]
func (p *Product) Edit(c *rux.Context) {
	// do something ...
}

// save edited product [optional]
func (p *Product) Update(c *rux.Context) {
	// do something ...
}

// delete product [optional]
func (p *Product) Delete(c *rux.Context) {
	// do something ...
}

func main() {
	router := rux.New()

	// methods	Path	Action	Route Name
    // GET	/product	index	product_index
    // GET	/product/create	create	product_create
    // POST	/product	store	product_store
    // GET	/product/{id}	show	product_show
    // GET	/product/{id}/edit	edit	product_edit
    // PUT/PATCH	/product/{id}	update	product_update
    // DELETE	/product/{id}	delete	product_delete
    // resetful style
	router.Resource("/", new(Product))

	log.Fatal(http.ListenAndServe(":12345", router))
}
```

### Controller 风格

```go
package main

import (
	"log"
	"net/http"

	"github.com/gookit/rux"
)

// News controller
type News struct {
}

func (n *News) AddRoutes(g *rux.Router) {
	g.GET("/", n.Index)
	g.POST("/", n.Create)
	g.PUT("/", n.Edit)
}

func (n *News) Index(c *rux.Context) {
	// Do something
}

func (n *News) Create(c *rux.Context) {
	// Do something
}

func (n *News) Edit(c *rux.Context) {
	// Do something
}


func main() {
	router := rux.New()

	// controller style
	router.Controller("/news", new(News))

	log.Fatal(http.ListenAndServe(":12345", router))
}
```

### 获取路由与生成请求URL

```go
package main

import (
	"log"
	"net/http"
	"net/url"

	"github.com/gookit/rux"
)

func main() {
	// Initialize a router as usual
	router := rux.New()
	router.GET(`/news/{category_id}/{new_id:\d+}/detail`, func(c *rux.Context) {
		var u = make(url.Values)
	    u.Add("username", "admin")
	    u.Add("password", "12345")
		
		b := rux.NewBuildRequestURL()
	    // b.Scheme("https")
	    // b.Host("www.mytest.com")
	    b.Queries(u)
	    b.Params(rux.M{"{category_id}": "100", "{new_id}": "20"})
		// b.Path("/dev")
	    // println(b.Build().String())
	    
	    println(c.Router().BuildRequestURL("new_detail", b).String())
		// result:  /news/100/20/detail?username=admin&password=12345
		// get current route name
		if c.MustGet(rux.CTXCurrentRouteName) == "new_detail" {
	        // post data etc....
	    }
	}).NamedTo("new_detail", router)

	// Use the HostSwitch to listen and serve on port 12345
	log.Fatal(http.ListenAndServe(":12345", router))
}
```

## 帮助

- lint

```bash
golint ./...
```

- 格式检查

```bash
# list error files
gofmt -s -l ./
# fix format and write to file
gofmt -s -w some.go
```

- 单元测试

```bash
go test -cover ./...
```

## Gookit 工具包

- [gookit/ini](https://github.com/gookit/ini) INI配置读取管理，支持多文件加载，数据覆盖合并, 解析ENV变量, 解析变量引用
- [gookit/rux](https://github.com/gookit/rux) Simple and fast request router for golang HTTP 
- [gookit/gcli](https://github.com/gookit/gcli) Go的命令行应用，工具库，运行CLI命令，支持命令行色彩，用户交互，进度显示，数据格式化显示
- [gookit/slog](https://github.com/gookit/slog) 简洁易扩展的go日志库
- [gookit/event](https://github.com/gookit/event) Go实现的轻量级的事件管理、调度程序库, 支持设置监听器的优先级, 支持对一组事件进行监听
- [gookit/cache](https://github.com/gookit/cache) 通用的缓存使用包装库，通过包装各种常用的驱动，来提供统一的使用API
- [gookit/config](https://github.com/gookit/config) Go应用配置管理，支持多种格式（JSON, YAML, TOML, INI, HCL, ENV, Flags），多文件加载，远程文件加载，数据合并
- [gookit/color](https://github.com/gookit/color) CLI 控制台颜色渲染工具库, 拥有简洁的使用API，支持16色，256色，RGB色彩渲染输出
- [gookit/filter](https://github.com/gookit/filter) 提供对Golang数据的过滤，净化，转换
- [gookit/validate](https://github.com/gookit/validate) Go通用的数据验证与过滤库，使用简单，内置大部分常用验证、过滤器
- [gookit/goutil](https://github.com/gookit/goutil) Go 的一些工具函数，格式化，特殊处理，常用信息获取等
- 更多请查看 https://github.com/gookit

## 相关项目

- https://github.com/gin-gonic/gin
- https://github.com/gorilla/mux
- https://github.com/julienschmidt/httprouter
- https://github.com/xialeistudio/go-dispatcher

## License

**[MIT](LICENSE)**
