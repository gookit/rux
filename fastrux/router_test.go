package fastrux_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gookit/rux/fastrux"
)

func TestStaticRoutes(t *testing.T) {
	r := fastrux.New()

	r.GET("/", func(c *fastrux.Context) {
		c.Text(200, "home")
	})

	r.GET("/users", func(c *fastrux.Context) {
		c.Text(200, "users list")
	})

	r.POST("/users", func(c *fastrux.Context) {
		c.Text(201, "user created")
	})

	// Test GET /
	result := r.Match("GET", "/")
	if result.Route == nil {
		t.Error("expected route for GET /")
	}

	// Test GET /users
	result = r.Match("GET", "/users")
	if result.Route == nil {
		t.Error("expected route for GET /users")
	}

	// Test POST /users
	result = r.Match("POST", "/users")
	if result.Route == nil {
		t.Error("expected route for POST /users")
	}

	// Test not found
	result = r.Match("GET", "/notfound")
	if result.Route != nil {
		t.Error("expected no route for GET /notfound")
	}
}

func TestDynamicRoutes(t *testing.T) {
	r := fastrux.New()

	r.GET("/users/{id}", func(c *fastrux.Context) {
		c.Text(200, "user: "+c.Param("id"))
	})

	r.GET("/users/{id}/posts/{postId}", func(c *fastrux.Context) {
		c.Text(200, "post")
	})

	// Test dynamic param
	result := r.Match("GET", "/users/123")
	if result.Route == nil {
		t.Error("expected route for GET /users/123")
	}
	if result.Params["id"] != "123" {
		t.Errorf("expected param id=123, got %v", result.Params["id"])
	}

	// Test nested params
	result = r.Match("GET", "/users/456/posts/789")
	if result.Route == nil {
		t.Error("expected route for GET /users/456/posts/789")
	}
	if result.Params["id"] != "456" {
		t.Errorf("expected param id=456, got %v", result.Params["id"])
	}
	if result.Params["postId"] != "789" {
		t.Errorf("expected param postId=789, got %v", result.Params["postId"])
	}
}

func TestWildcardRoutes(t *testing.T) {
	r := fastrux.New()

	r.GET("/assets/*file", func(c *fastrux.Context) {
		c.Text(200, "file: "+c.Param("file"))
	})

	result := r.Match("GET", "/assets/css/style.css")
	if result.Route == nil {
		t.Error("expected route for GET /assets/css/style.css")
	}
	if result.Params["file"] != "css/style.css" {
		t.Errorf("expected file param css/style.css, got %v", result.Params["file"])
	}
}

func TestRegexParamStripping(t *testing.T) {
	r := fastrux.New()

	// regex constraint should be stripped: {id:\d+} -> :id
	r.GET("/users/{id:\\d+}", func(c *fastrux.Context) {
		c.Text(200, "user: "+c.Param("id"))
	})

	result := r.Match("GET", "/users/123")
	if result.Route == nil {
		t.Error("expected route for GET /users/123 (regex stripped)")
	}
	if result.Params["id"] != "123" {
		t.Errorf("expected id=123, got %v", result.Params["id"])
	}

	// non-numeric value should also match (regex is stripped)
	result = r.Match("GET", "/users/abc")
	if result.Route == nil {
		t.Error("expected route for GET /users/abc (regex constraint stripped)")
	}
}

func TestOptionalSegments(t *testing.T) {
	r := fastrux.New()

	r.GET("/posts[/{id}]", func(c *fastrux.Context) {
		c.Text(200, "posts")
	})

	// Without optional segment
	result := r.Match("GET", "/posts")
	if result.Route == nil {
		t.Error("expected route for GET /posts")
	}

	// With optional segment
	result = r.Match("GET", "/posts/42")
	if result.Route == nil {
		t.Error("expected route for GET /posts/42")
	}
	if result.Params["id"] != "42" {
		t.Errorf("expected id=42, got %v", result.Params["id"])
	}
}

func TestMatchResult(t *testing.T) {
	r := fastrux.New()

	r.GET("/api/v1/users/{id}", func(c *fastrux.Context) {
		c.Text(200, "user")
	})

	result := r.Match("GET", "/api/v1/users/99")
	if result == nil {
		t.Fatal("expected non-nil MatchResult")
	}
	if result.Route == nil {
		t.Error("expected non-nil Route in MatchResult")
	}
	if result.Method != "GET" {
		t.Errorf("expected method GET, got %v", result.Method)
	}
	if result.Params["id"] != "99" {
		t.Errorf("expected id=99, got %v", result.Params["id"])
	}
}

func TestMethodNotAllowed(t *testing.T) {
	r := fastrux.New(fastrux.HandleMethodNotAllowed)

	r.GET("/resource", func(c *fastrux.Context) {
		c.Text(200, "ok")
	})

	result := r.Match("POST", "/resource")
	if result.Route != nil {
		t.Error("expected no route for POST /resource")
	}
	if len(result.Allowed) == 0 {
		t.Error("expected allowed methods for POST /resource")
	}
	found := false
	for _, m := range result.Allowed {
		if m == "GET" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected GET in allowed methods, got %v", result.Allowed)
	}
}

func TestGroupRoutes(t *testing.T) {
	r := fastrux.New()

	r.Group("/api/v1", func() {
		r.GET("/users", func(c *fastrux.Context) {
			c.Text(200, "users")
		})
		r.POST("/users", func(c *fastrux.Context) {
			c.Text(201, "created")
		})
	})

	result := r.Match("GET", "/api/v1/users")
	if result.Route == nil {
		t.Error("expected route for GET /api/v1/users")
	}

	result = r.Match("POST", "/api/v1/users")
	if result.Route == nil {
		t.Error("expected route for POST /api/v1/users")
	}
}

func TestServeHTTP(t *testing.T) {
	r := fastrux.New()

	r.GET("/hello", func(c *fastrux.Context) {
		c.Text(200, "hello world")
	})

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "hello world" {
		t.Errorf("expected body 'hello world', got %q", w.Body.String())
	}
}

func TestNotFound(t *testing.T) {
	r := fastrux.New()

	r.NotFound(func(c *fastrux.Context) {
		c.Text(404, "not found")
	})

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestHeadFallbackToGet(t *testing.T) {
	r := fastrux.New()

	r.GET("/page", func(c *fastrux.Context) {
		c.Text(200, "page content")
	})

	result := r.Match("HEAD", "/page")
	if result.Route == nil {
		t.Error("expected HEAD to fall back to GET route")
	}
}
