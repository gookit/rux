// Minimal SSE server example using rux + pkg/sse.
//
// Run:
//
//	go run ./_examples/sse-server
//
// Open the page (built-in HTML demo client):
//
//	http://127.0.0.1:18081/
//
// Or hit the stream directly with curl:
//
//	curl -N http://127.0.0.1:18081/events
//
// Pass ?reject=1 to see the OnConnect hook return 401:
//
//	curl -i http://127.0.0.1:18081/events?reject=1
package main

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gookit/rux/v2"
	"github.com/gookit/rux/v2/pkg/sse"
	"github.com/gookit/rux/v2/server"
)

const indexPage = `<!doctype html>
<title>rux sse demo</title>
<pre id="log"></pre>
<script>
const log = document.getElementById('log');
const es = new EventSource('/events');
es.onmessage = (e) => log.textContent += e.data + '\n';
es.addEventListener('tick', (e) => log.textContent += '[tick] ' + e.data + '\n');
es.onerror = () => log.textContent += '[stream closed]\n';
</script>`

func main() {
	s := server.New(true)
	s.Addr = "127.0.0.1:18081"
	// SSE is a long-lived response — relax the default 30s WriteTimeout
	// so producers can run for hours without being killed mid-stream.
	s.WriteTimeout = 0

	s.GET("/", func(c *rux.Context) {
		c.HTMLString(http.StatusOK, indexPage)
	})

	s.GET("/events", eventsHandler)

	log.Printf("listening on http://%s", s.Addr)
	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}

// eventsHandler demonstrates a 1-Hz ticker stream with all four hooks
// wired up. In a real app these would live in middleware or a service
// layer — kept inline here for the example.
func eventsHandler(c *rux.Context) {
	hooks := &sse.Hooks{
		// Pretend we have auth — reject if ?reject=1 is set.
		OnConnect: func(c *rux.Context) error {
			if c.Query("reject") == "1" {
				http.Error(c.Resp, "no token", http.StatusUnauthorized)
				return errors.New("rejected by demo")
			}
			log.Printf("sse open  remote=%s", c.Req.RemoteAddr)
			return nil
		},
		OnDisconnect: func(c *rux.Context, reason error) {
			log.Printf("sse close remote=%s reason=%v", c.Req.RemoteAddr, reason)
		},
		OnSend: func(c *rux.Context, e *sse.Event) error {
			// Tag every event with an auto-incrementing id so the browser
			// can resume via Last-Event-ID after a reconnect.
			return nil
		},
		OnError: func(c *rux.Context, err error) {
			log.Printf("sse send err: %v", err)
		},
	}

	_ = sse.Stream(c, hooks, func(send sse.SendFunc, done <-chan struct{}) error {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		// Suggest a 2s reconnect delay on the very first frame.
		if err := send(sse.Event{Retry: 2000, Data: "stream started"}); err != nil {
			return err
		}

		var n int
		for {
			select {
			case <-done:
				return nil
			case t := <-ticker.C:
				n++
				err := send(sse.Event{
					ID:   strconv.Itoa(n),
					Name: "tick",
					Data: t.Format(time.RFC3339),
				})
				if err != nil {
					return err
				}
			}
		}
	})
}
