package rux

import (
	"testing"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "/"},
		{"/", "/"},
		{"/home", "/home"},
		{"home", "/home"},
		{"/home/", "/home"},
		{"/home/user", "/home/user"},
		{"/home/user/", "/home/user"},
		{"//home", "/home"},
		{"  /home  ", "/home"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizePath(tt.input)
			if result != tt.expected {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLongestCommonPrefix(t *testing.T) {
	tests := []struct {
		a        string
		b        string
		expected int
	}{
		{"/home", "/home/user", 5},
		{"/home/user", "/home", 5},
		{"/home", "/user", 1},
		{"", "/home", 0},
		{"/home", "", 0},
		{"/home", "/home", 5},
		{"/api/v1/users", "/api/v2/users", 6},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			result := longestCommonPrefix(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("longestCommonPrefix(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestRadixTree_AddStaticRoute(t *testing.T) {
	tree := newRadixTree()

	handler := func(c *Context) {}
	handlers := HandlersChain{handler}

	// 添加静态路由
	tree.AddRoute("/home", handlers, []string{"GET"})
	tree.AddRoute("/about", handlers, []string{"GET"})

	// 测试查找
	h, params, found := tree.FindRoute("GET", "/home")
	if !found {
		t.Errorf("Expected to find route /home")
	}
	if h == nil {
		t.Errorf("Expected handler to be set")
	}
	if len(params) != 0 {
		t.Errorf("Expected empty params, got %v", params)
	}

	h, params, found = tree.FindRoute("GET", "/about")
	if !found {
		t.Errorf("Expected to find route /about")
	}
	if h == nil {
		t.Errorf("Expected handler to be set")
	}
}

func TestRadixTree_AddParamRoute(t *testing.T) {
	tree := newRadixTree()

	handler := func(c *Context) {}
	handlers := HandlersChain{handler}

	// 添加参数路由
	tree.AddRoute("/users/:id", handlers, []string{"GET"})

	// 测试查找
	h, params, found := tree.FindRoute("GET", "/users/123")
	if !found {
		t.Errorf("Expected to find route /users/:id")
	}
	if h == nil {
		t.Errorf("Expected handler to be set")
	}
	if params["id"] != "123" {
		t.Errorf("Expected param id=123, got %q", params["id"])
	}

	h, params, found = tree.FindRoute("GET", "/users/abc")
	if !found {
		t.Errorf("Expected to find route /users/:id")
	}
	if params["id"] != "abc" {
		t.Errorf("Expected param id=abc, got %q", params["id"])
	}
}

func TestRadixTree_AddWildcardRoute(t *testing.T) {
	tree := newRadixTree()

	handler := func(c *Context) {}
	handlers := HandlersChain{handler}

	// 添加通配符路由
	tree.AddRoute("/static/*filepath", handlers, []string{"GET"})

	// 测试查找
	h, params, found := tree.FindRoute("GET", "/static/css/style.css")
	if !found {
		t.Errorf("Expected to find route /static/*filepath")
	}
	if h == nil {
		t.Errorf("Expected handler to be set")
	}
	if params["filepath"] != "css/style.css" {
		t.Errorf("Expected param filepath=css/style.css, got %q", params["filepath"])
	}

	h, params, found = tree.FindRoute("GET", "/static/js/app.js")
	if !found {
		t.Errorf("Expected to find route /static/*filepath")
	}
	if params["filepath"] != "js/app.js" {
		t.Errorf("Expected param filepath=js/app.js, got %q", params["filepath"])
	}
}

func TestRadixTree_MultipleParams(t *testing.T) {
	tree := newRadixTree()

	handler := func(c *Context) {}
	handlers := HandlersChain{handler}

	// 添加多参数路由
	tree.AddRoute("/articles/:year/:month/:day", handlers, []string{"GET"})

	// 测试查找
	h, params, found := tree.FindRoute("GET", "/articles/2025/02/07")
	if !found {
		t.Errorf("Expected to find route /articles/:year/:month/:day")
	}
	if h == nil {
		t.Errorf("Expected handler to be set")
	}
	if params["year"] != "2025" {
		t.Errorf("Expected param year=2025, got %q", params["year"])
	}
	if params["month"] != "02" {
		t.Errorf("Expected param month=02, got %q", params["month"])
	}
	if params["day"] != "07" {
		t.Errorf("Expected param day=07, got %q", params["day"])
	}
}

func TestRadixTree_PathCompression(t *testing.T) {
	tree := newRadixTree()

	handler := func(c *Context) {}
	handlers := HandlersChain{handler}

	// 添加有公共前缀的路由
	tree.AddRoute("/api/v1/users", handlers, []string{"GET"})
	tree.AddRoute("/api/v1/posts", handlers, []string{"GET"})

	// 验证路径压缩 - v1 节点应该被压缩
	// 测试查找
	h, params, found := tree.FindRoute("GET", "/api/v1/users")
	if !found {
		t.Errorf("Expected to find route /api/v1/users")
	}
	if h == nil {
		t.Errorf("Expected handler to be set")
	}
	if len(params) != 0 {
		t.Errorf("Expected empty params, got %v", params)
	}

	h, params, found = tree.FindRoute("GET", "/api/v1/posts")
	if !found {
		t.Errorf("Expected to find route /api/v1/posts")
	}
	if h == nil {
		t.Errorf("Expected handler to be set")
	}
}

func TestRadixTree_NodeSplit(t *testing.T) {
	tree := newRadixTree()

	handler := func(c *Context) {}
	handlers := HandlersChain{handler}

	// 先添加一个路由
	tree.AddRoute("/api/users", handlers, []string{"GET"})

	// 再添加一个有部分公共前缀的路由，触发节点分裂
	tree.AddRoute("/api/posts", handlers, []string{"GET"})

	// 测试查找
	h, _, found := tree.FindRoute("GET", "/api/users")
	if !found {
		t.Errorf("Expected to find route /api/users")
	}
	if h == nil {
		t.Errorf("Expected handler to be set")
	}

	h, _, found = tree.FindRoute("GET", "/api/posts")
	if !found {
		t.Errorf("Expected to find route /api/posts")
	}
	if h == nil {
		t.Errorf("Expected handler to be set")
	}
}

func TestRadixTree_NotFound(t *testing.T) {
	tree := newRadixTree()

	handler := func(c *Context) {}
	handlers := HandlersChain{handler}

	tree.AddRoute("/home", handlers, []string{"GET"})

	// 测试不存在的路由
	h, params, found := tree.FindRoute("GET", "/about")
	if found {
		t.Errorf("Expected not to find route /about")
	}
	if h != nil {
		t.Errorf("Expected handler to be nil")
	}
	if len(params) != 0 {
		t.Errorf("Expected empty params, got %v", params)
	}
}

func TestRadixTree_DifferentMethods(t *testing.T) {
	tree := newRadixTree()

	getHandler := func(c *Context) {}
	postHandler := func(c *Context) {}

	// 添加不同方法的路由
	tree.AddRoute("/users", HandlersChain{getHandler}, []string{"GET"})
	tree.AddRoute("/users", HandlersChain{postHandler}, []string{"POST"})

	// 测试 GET
	h, params, found := tree.FindRoute("GET", "/users")
	if !found {
		t.Errorf("Expected to find GET /users")
	}
	if h == nil {
		t.Errorf("Expected handler to be set")
	}
	if len(params) != 0 {
		t.Errorf("Expected empty params, got %v", params)
	}

	// 测试 POST
	_, params, found = tree.FindRoute("POST", "/users")
	if !found {
		t.Errorf("Expected to find POST /users")
	}
}

func TestRadixTree_MixedRoutes(t *testing.T) {
	tree := newRadixTree()

	handler := func(c *Context) {}
	handlers := HandlersChain{handler}

	// 添加混合路由
	tree.AddRoute("/home", handlers, []string{"GET"})
	tree.AddRoute("/users/:id", handlers, []string{"GET"})
	tree.AddRoute("/posts/:slug", handlers, []string{"GET"})
	tree.AddRoute("/static/*filepath", handlers, []string{"GET"})

	// 测试静态路由
	_, params, found := tree.FindRoute("GET", "/home")
	if !found {
		t.Errorf("Expected to find route /home")
	}
	if len(params) != 0 {
		t.Errorf("Expected empty params, got %v", params)
	}

	// 测试参数路由
	_, params, found = tree.FindRoute("GET", "/users/123")
	if !found {
		t.Errorf("Expected to find route /users/:id")
	}
	if params["id"] != "123" {
		t.Errorf("Expected param id=123, got %q", params["id"])
	}

	// 测试通配符路由
	_, params, found = tree.FindRoute("GET", "/static/css/style.css")
	if !found {
		t.Errorf("Expected to find route /static/*filepath")
	}
	if params["filepath"] != "css/style.css" {
		t.Errorf("Expected param filepath=css/style.css, got %q", params["filepath"])
	}
}

func TestRadixTree_OptionalSegments(t *testing.T) {
	tree := newRadixTree()

	handler := func(c *Context) {}
	handlers := HandlersChain{handler}

	// 添加带可选参数的路由 /posts[/{id}]
	tree.AddRoute("/posts[/{id}]", handlers, []string{"GET"})

	// 测试不带可选参数的路径 /posts
	h, params, found := tree.FindRoute("GET", "/posts")
	if !found {
		t.Errorf("Expected to find route /posts")
	}
	if h == nil {
		t.Errorf("Expected handler to be set for /posts")
	}
	if len(params) != 0 {
		t.Errorf("Expected empty params for /posts, got %v", params)
	}

	// 测试带可选参数的路径 /posts/123
	h, params, found = tree.FindRoute("GET", "/posts/123")
	if !found {
		t.Errorf("Expected to find route /posts/123")
	}
	if h == nil {
		t.Errorf("Expected handler to be set for /posts/123")
	}
	if params["id"] != "123" {
		t.Errorf("Expected param id=123, got %q", params["id"])
	}
}
