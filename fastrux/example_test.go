package fastrux_test

import (
	"fmt"
	"net/http/httptest"

	"github.com/gookit/rux/fastrux"
)

// Example_basic demonstrates basic routing
func Example_basic() {
	r := fastrux.New()

	r.GET("/", func(c *fastrux.Context) {
		c.Text(200, "Welcome to FastRux!")
	})

	r.GET("/hello/{name}", func(c *fastrux.Context) {
		name := c.Param("name")
		c.Text(200, "Hello, "+name+"!")
	})

	// Simulate request
	req := httptest.NewRequest("GET", "/hello/world", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	fmt.Println(w.Body.String())
	// Output: Hello, world!
}

// Example_groups demonstrates route grouping
func Example_groups() {
	r := fastrux.New()

	r.Group("/api", func() {
		r.Group("/v1", func() {
			r.GET("/users", func(c *fastrux.Context) {
				c.JSON(200, []string{"alice", "bob"})
			})
		})
	})

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	fmt.Println(w.Body.String())
	// Output: ["alice","bob"]
}

// Example_middleware demonstrates middleware usage
func Example_middleware() {
	r := fastrux.New()

	// Global middleware
	r.Use(func(c *fastrux.Context) {
		fmt.Print("before-")
		c.Next()
		fmt.Print("-after")
	})

	r.GET("/test", func(c *fastrux.Context) {
		fmt.Print("handler")
		c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Output: before-handler-after
}

// Example_optionalParams demonstrates optional parameters
func Example_optionalParams() {
	r := fastrux.New()

	r.GET("/posts[/{id}]", func(c *fastrux.Context) {
		if id := c.Param("id"); id != "" {
			c.Text(200, "Post #"+id)
		} else {
			c.Text(200, "All posts")
		}
	})

	// Without parameter
	req1 := httptest.NewRequest("GET", "/posts", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	fmt.Println(w1.Body.String())

	// With parameter
	req2 := httptest.NewRequest("GET", "/posts/42", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	fmt.Println(w2.Body.String())

	// Output:
	// All posts
	// Post #42
}

// Example_namedRoutes demonstrates named route URL building
func Example_namedRoutes() {
	r := fastrux.New()

	r.AddNamed("user.show", "/users/{id}", func(c *fastrux.Context) {
		c.Text(200, "User page")
	}, "GET")

	// Get the route
	route := r.GetRoute("user.show")
	if route != nil {
		fmt.Println("Route name:", route.Name())
		fmt.Println("Route path:", route.Path())
	}

	// Output:
	// Route name: user.show
	// Route path: /users/{id}
}

// Example_wildcardRoutes demonstrates wildcard routing
func Example_wildcardRoutes() {
	r := fastrux.New()

	r.GET("/files/*filepath", func(c *fastrux.Context) {
		path := c.Param("filepath")
		c.Text(200, "File: "+path)
	})

	req := httptest.NewRequest("GET", "/files/docs/readme.md", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	fmt.Println(w.Body.String())
	// Output: File: docs/readme.md
}

// Example_customHandlers demonstrates custom 404/405 handlers
func Example_customHandlers() {
	r := fastrux.New(fastrux.HandleMethodNotAllowed)

	r.GET("/exists", func(c *fastrux.Context) {
		c.Text(200, "ok")
	})

	// Custom 404
	r.NotFound(func(c *fastrux.Context) {
		c.Text(404, "Custom 404: Page not found")
	})

	// Custom 405
	r.NotAllowed(func(c *fastrux.Context) {
		c.Text(405, "Custom 405: Method not allowed")
	})

	// Test 404
	req1 := httptest.NewRequest("GET", "/nonexistent", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	fmt.Println(w1.Body.String())

	// Test 405
	req2 := httptest.NewRequest("POST", "/exists", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	fmt.Println(w2.Body.String())

	// Output:
	// Custom 404: Page not found
	// Custom 405: Method not allowed
}

// Example_performance demonstrates performance-conscious usage
func Example_performance() {
	r := fastrux.New()

	// Static routes are fastest (O(1) lookup, 1 allocation)
	r.GET("/ping", func(c *fastrux.Context) {
		c.Text(200, "pong")
	})

	// Dynamic routes are still fast (O(m) lookup, 3 allocations)
	r.GET("/users/{id}", func(c *fastrux.Context) {
		c.JSON(200, fastrux.M{"id": c.Param("id")})
	})

	// For repeated Match() calls, reuse MatchResult
	result := r.Match("GET", "/ping")
	if result.Route != nil {
		fmt.Println("Found:", result.Route.Path())
	}
	r.ReleaseMatchResult(result) // Return to pool

	// Output: Found: /ping
}

// Example_contextUtilities demonstrates Context helper methods
func Example_contextUtilities() {
	r := fastrux.New()

	r.POST("/user", func(c *fastrux.Context) {
		// Query parameters
		page := c.Query("page", "1")

		// Headers
		contentType := c.Header("Content-Type")

		// POST data
		name := c.Post("name", "anonymous")

		// Client IP
		ip := c.ClientIP()

		c.JSON(200, fastrux.M{
			"page":        page,
			"contentType": contentType,
			"name":        name,
			"ip":          ip,
		})
	})

	req := httptest.NewRequest("POST", "/user?page=2", nil)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.1:1234"

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	fmt.Println(w.Body.String())
	// Output: {"contentType":"application/json","ip":"192.168.1.1","name":"anonymous","page":"2"}
}
