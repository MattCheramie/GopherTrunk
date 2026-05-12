package client

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Stream opens GET /api/v1/events as Server-Sent Events and returns
// a channel of decoded Events plus an error channel that receives
// at most one terminal error before being closed.
//
// The stream lives until ctx is cancelled or the server closes the
// connection. Reconnection is the caller's responsibility — the
// bubbletea Update loop owns the retry policy.
//
// SSE framing reference: dispatch on blank line, accumulate `event:`
// and `data:` fields. Comments (lines starting with `:`) are
// ignored. Multi-line data fields are joined with `\n` per the
// W3C spec; the daemon currently emits single-line `data:` so we
// honour the spec without exercising it heavily.
func (c *Client) Stream(ctx context.Context) (<-chan Event, <-chan error) {
	out := make(chan Event, 64)
	errCh := make(chan error, 1)

	go func() {
		defer close(out)
		defer close(errCh)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/api/v1/events", nil)
		if err != nil {
			errCh <- err
			return
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")
		c.authorize(req)

		// SSE is long-lived; use a transport without per-request
		// timeout. We rely on ctx for cancellation.
		hc := &http.Client{Transport: c.hc.Transport}
		resp, err := hc.Do(req)
		if err != nil {
			errCh <- err
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			errCh <- c.httpErr(http.MethodGet, req.URL.String(), resp)
			return
		}

		if err := parseSSE(resp.Body, out); err != nil && !errors.Is(err, io.EOF) && ctx.Err() == nil {
			errCh <- err
		}
	}()

	return out, errCh
}

// parseSSE drains one SSE response body, emitting one Event per
// dispatched event. It returns nil on clean EOF or context-induced
// closes; otherwise the underlying read error.
func parseSSE(r io.Reader, out chan<- Event) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)

	var (
		eventName string
		data      strings.Builder
	)
	dispatch := func() {
		if data.Len() == 0 {
			eventName = ""
			return
		}
		raw := data.String()
		data.Reset()
		// The daemon's data line is itself a JSON envelope:
		// {"kind","timestamp","payload"}. Decode the envelope
		// and forward the payload as Event.Raw so callers can
		// type-decode it.
		var env struct {
			Kind      string          `json:"kind"`
			Timestamp time.Time       `json:"timestamp"`
			Payload   json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal([]byte(raw), &env); err == nil {
			ev := Event{Kind: env.Kind, Time: env.Timestamp, Raw: env.Payload}
			if ev.Kind == "" {
				ev.Kind = eventName
			}
			out <- ev
		} else {
			// Fall back to using the SSE event: name and the raw
			// data as-is, so callers still see something.
			out <- Event{Kind: eventName, Time: time.Now(), Raw: json.RawMessage(raw)}
		}
		eventName = ""
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			dispatch()
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue // comment
		}
		colon := strings.IndexByte(line, ':')
		var field, value string
		if colon < 0 {
			field, value = line, ""
		} else {
			field = line[:colon]
			value = line[colon+1:]
			if strings.HasPrefix(value, " ") {
				value = value[1:]
			}
		}
		switch field {
		case "event":
			eventName = value
		case "data":
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(value)
		case "id", "retry":
			// Not used by the daemon; safely ignored.
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("sse read: %w", err)
	}
	// Flush trailing event if the stream ended without a blank line.
	dispatch()
	return nil
}
