package server

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gookit/goutil/x/assert"
	"github.com/gookit/rux/v2"
	"github.com/gookit/rux/v2/pkg/sse"
)

// TestSSE_OutlivesWriteTimeout is the end-to-end version of a consumer report:
// server.New's WriteTimeout used to cut a long-lived SSE stream (30s by
// default, 300ms here) and no heartbeat could prevent it. StreamWith now clears
// the write deadline for its own response.
func TestSSE_OutlivesWriteTimeout(t *testing.T) {
	s := addrTestServer("127.0.0.1:0")
	s.WriteTimeout = 300 * time.Millisecond // deliberately shorter than the stream

	s.GET("/events", func(c *rux.Context) {
		_ = sse.StreamWith(c, &sse.Options{SendConnected: true}, func(send sse.SendFunc, done <-chan struct{}) error {
			tick := time.NewTicker(50 * time.Millisecond)
			defer tick.Stop()
			for i := 0; i < 20; i++ {
				select {
				case <-done:
					return nil
				case <-tick.C:
					if err := send(sse.Event{Name: "tick", Data: fmt.Sprintf("%d", i)}); err != nil {
						return err
					}
				}
			}
			return nil
		})
	})

	done := startInBackground(t, s)
	defer stopAndWait(t, s, done)

	start := time.Now()
	resp, err := http.Get(s.LocalURL() + "/events")
	assert.NoErr(t, err)
	defer resp.Body.Close()
	assert.Eq(t, 200, resp.StatusCode)
	assert.StrContains(t, resp.Header.Get("Content-Type"), "text/event-stream")

	reader := bufio.NewReader(resp.Body)
	deadline := start.Add(900 * time.Millisecond)
	var events int
	var lastAt time.Duration
	for time.Now().Before(deadline) {
		line, err := reader.ReadString('\n')
		if err != nil {
			break // the server cut the stream early
		}
		if strings.HasPrefix(line, "event: tick") {
			events++
			lastAt = time.Since(start)
		}
	}

	assert.Gt(t, events, 8)
	assert.True(t, lastAt > 600*time.Millisecond,
		"stream must outlive WriteTimeout(300ms): last event after %v, %d events", lastAt, events)
}
