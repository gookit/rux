// SSE server example using rux + pkg/sse with Hub.
//
// Run:
//
//	go run ./_examples/sse-server
//
// Open the demo page (two browser tabs to see fan-out):
//
//	http://127.0.0.1:18081/?uid=alice
//
// Curl side:
//
//	curl -N http://127.0.0.1:18081/events?uid=alice                      # subscribe
//	curl -X POST 'http://127.0.0.1:18081/push?uid=alice&msg=hello'       # → only alice's tabs
//	curl -X POST 'http://127.0.0.1:18081/broadcast?msg=system+update'    # → everyone
//	curl http://127.0.0.1:18081/stats                                    # hub size
//	curl -i http://127.0.0.1:18081/events?uid=alice&reject=1             # OnConnect 401 demo
package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gookit/rux/v2"
	"github.com/gookit/rux/v2/pkg/sse"
	"github.com/gookit/rux/v2/server"
)

// hub is the process-wide registry of active SSE clients. In a real app
// you'd inject this via a service container or wire it through a struct;
// kept as a package var here for example brevity.
var hub = sse.NewHub(64)

const indexPage = `<!doctype html>
<title>rux sse hub demo</title>
<style>body{font-family:monospace;max-width:48em;margin:1em auto}</style>
<h3>SSE Hub Demo</h3>
<p>Connected as <b id="uid"></b> — open this page in two tabs (same uid)
to watch the same user fan-out.</p>
<p>
  <input id="msg" placeholder="message"/>
  <button onclick="push()">push to my uid</button>
  <button onclick="bcast()">broadcast to everyone</button>
</p>
<pre id="log"></pre>
<script>
const qs = new URLSearchParams(location.search);
const uid = qs.get('uid') || 'guest';
document.getElementById('uid').textContent = uid;

const log = document.getElementById('log');
const append = s => log.textContent += s + '\n';

const es = new EventSource('/events?uid=' + encodeURIComponent(uid));
es.onmessage = e => append('msg: ' + e.data);
es.addEventListener('notify', e => append('[notify] ' + e.data));
es.addEventListener('announce', e => append('[announce] ' + e.data));
es.onerror = () => append('[stream closed, browser will retry]');

function push() {
    const m = document.getElementById('msg').value;
    fetch('/push?uid=' + encodeURIComponent(uid) + '&msg=' + encodeURIComponent(m), {method:'POST'});
}
function bcast() {
    const m = document.getElementById('msg').value;
    fetch('/broadcast?msg=' + encodeURIComponent(m), {method:'POST'});
}
</script>`

func main() {
	s := server.New(true)
	s.Addr = "127.0.0.1:18081"
	// SSE is a long-lived response — disable WriteTimeout so producers
	// can run for hours. Heartbeats can't substitute for this; see the
	// pkg/sse godoc.
	s.WriteTimeout = 0

	// Surface drops in the log so a slow-consumer is immediately visible.
	hub.SetOnDrop(func(c *sse.Client, _ sse.Event) {
		log.Printf("sse drop  uid=%s total=%d", c.ID, c.Dropped())
	})

	s.GET("/", func(c *rux.Context) {
		c.HTMLString(http.StatusOK, indexPage)
	})

	s.GET("/events", eventsHandler)
	s.POST("/push", pushHandler)
	s.POST("/broadcast", broadcastHandler)
	s.GET("/stats", statsHandler)

	log.Printf("listening on http://%s", s.Addr)
	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}

// eventsHandler subscribes the caller to the hub under their uid.
// All actual event delivery happens via /push or /broadcast.
func eventsHandler(c *rux.Context) {
	uid := c.Query("uid")
	if uid == "" {
		c.AbortWithStatus(http.StatusBadRequest, "uid required")
		return
	}

	hooks := &sse.Hooks{
		OnConnect: func(c *rux.Context) error {
			if c.Query("reject") == "1" {
				http.Error(c.Resp, "no token", http.StatusUnauthorized)
				return errors.New("rejected by demo")
			}
			log.Printf("sse open  uid=%s remote=%s", uid, c.Req.RemoteAddr)
			return nil
		},
		OnDisconnect: func(c *rux.Context, reason error) {
			log.Printf("sse close uid=%s reason=%v", uid, reason)
		},
	}
	opts := &sse.Options{
		Hooks:             hooks,
		SendConnected:     true,
		KeepaliveInterval: 30 * time.Second, // defeat proxy idle timeout
	}
	_ = sse.StreamWith(c, opts, sse.HubProducer(hub, uid))
}

// pushHandler sends a message to every active connection under uid.
// JSON-style response: {"delivered": N, "dropped": M}.
func pushHandler(c *rux.Context) {
	uid := c.Query("uid")
	msg := c.Query("msg")
	delivered, dropped := hub.Send(uid, sse.Event{
		Name: "notify",
		Data: msg,
	})
	c.JSON(http.StatusOK, rux.M{"delivered": delivered, "dropped": dropped})
}

// broadcastHandler sends a message to every client across all uids.
func broadcastHandler(c *rux.Context) {
	msg := c.Query("msg")
	delivered, dropped := hub.Broadcast(sse.Event{
		Name: "announce",
		Data: msg,
	})
	c.JSON(http.StatusOK, rux.M{"delivered": delivered, "dropped": dropped})
}

// statsHandler exposes hub size — useful for monitoring / admin.
func statsHandler(c *rux.Context) {
	clients, ids := hub.Count()
	c.JSON(http.StatusOK, rux.M{
		"clients": clients,
		"ids":     ids,
		"ids_now": hub.IDs(),
	})
}
