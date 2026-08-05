package broadcast

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// Default tuning for a GrantWebhook.
const (
	// grantWebhookQueueDepth bounds the in-flight grant backlog. A busy
	// control channel emits thousands of grants an hour; when the endpoint
	// wedges we drop the oldest overflow rather than block the event loop
	// (and the decoder behind it), the same discipline the call Manager uses.
	grantWebhookQueueDepth = 256
	grantWebhookMaxRetries = 3
	grantWebhookRetryBase  = 2 * time.Second
)

// GrantWebhookConfig configures one push grant-webhook sink.
type GrantWebhookConfig struct {
	// Name is an optional log label; defaults to "grant-webhook".
	Name string
	// URL is the endpoint each grant is POSTed to (required).
	URL string
	// AuthHeader, when set, is sent verbatim as the Authorization header
	// (e.g. "Bearer <token>"). Empty omits the header.
	AuthHeader string
	// Systems restricts the feed to these trunking-system names. Empty
	// streams every system.
	Systems []string
	// Loc is the timezone the `at` RFC3339 timestamp renders in. Nil means
	// time.Local. The format stays RFC3339, so the offset keeps the instant
	// unambiguous whichever zone is chosen.
	Loc *time.Location
}

// grantWebhookPayload is the JSON object POSTed for each decoded
// control-channel grant. It mirrors the api.GrantDTO schema that
// GET /api/v1/grants and the KindGrant SSE stream already publish — same
// field names, same omitempty rules — plus an "event" discriminator and the
// decode timestamp, so a consumer reads one schema across the REST snapshot,
// the SSE live stream, and this push feed (issue #915 / #268).
//
// Grants are surfaced exactly as decoded off the control channel: the source
// RID rides the grant at call-setup time for every protocol, and a grant whose
// TSBK carried src=0 is POSTed with source_id=0 (never backfilled), so the feed
// reports what the control channel actually saw — the way SDRtrunk's grant log
// does. That makes it distinct from the completed-call webhook, whose source
// RID depends on a separate voice-side decode landing.
type grantWebhookPayload struct {
	Event         string `json:"event"` // always "grant"
	System        string `json:"system"`
	Protocol      string `json:"protocol"`
	GroupID       uint32 `json:"group_id"`
	SourceID      uint32 `json:"source_id"`
	FrequencyHz   uint32 `json:"frequency_hz"`
	ChannelID     uint8  `json:"channel_id,omitempty"`
	ChannelNumber uint16 `json:"channel_number,omitempty"`
	RFSSID        uint8  `json:"rfss_id,omitempty"`
	SiteID        uint8  `json:"site_id,omitempty"`
	NAC           uint16 `json:"nac,omitempty"`
	Timeslot      uint8  `json:"timeslot,omitempty"`
	Encrypted     bool   `json:"encrypted,omitempty"`
	Emergency     bool   `json:"emergency,omitempty"`
	Priority      uint8  `json:"priority,omitempty"`
	DataCall      bool   `json:"data_call,omitempty"`
	Individual    bool   `json:"individual,omitempty"`
	AlgorithmID   uint8  `json:"algorithm_id,omitempty"`
	KeyID         uint16 `json:"key_id,omitempty"`
	At            string `json:"at"` // RFC3339 decode time
}

// grantWebhookPayloadFrom projects a trunking.Grant onto the wire schema.
func grantWebhookPayloadFrom(g trunking.Grant, at time.Time, loc *time.Location) grantWebhookPayload {
	return grantWebhookPayload{
		Event:         "grant",
		System:        g.System,
		Protocol:      g.Protocol,
		GroupID:       g.GroupID,
		SourceID:      g.SourceID,
		FrequencyHz:   g.FrequencyHz,
		ChannelID:     g.ChannelID,
		ChannelNumber: g.ChannelNum,
		RFSSID:        g.RFSSID,
		SiteID:        g.SiteID,
		NAC:           g.NAC,
		Timeslot:      g.Timeslot,
		Encrypted:     g.Encrypted,
		Emergency:     g.Emergency,
		Priority:      g.Priority,
		DataCall:      g.DataCall,
		Individual:    g.Individual,
		AlgorithmID:   g.AlgorithmID,
		KeyID:         g.KeyID,
		At:            rfc3339In(at, loc),
	}
}

// GrantWebhookOptions configure a GrantWebhook.
type GrantWebhookOptions struct {
	// Bus is the event bus to subscribe to. Required.
	Bus *events.Bus
	// Log defaults to slog.Default().
	Log *slog.Logger
	// Config is the sink's destination and filter. URL is required.
	Config GrantWebhookConfig
	// MaxRetries is the per-grant retry budget on a failed POST. Default 3.
	MaxRetries int
	// RetryBase is the first backoff delay; it doubles per attempt. Default 2s.
	RetryBase time.Duration
	// Now is injectable for tests; defaults to time.Now. It stamps a grant
	// that reached the bus with a zero At.
	Now func() time.Time
	// HTTP is the client used for delivery; defaults to http.DefaultClient.
	HTTP *http.Client
}

// GrantWebhook POSTs each decoded control-channel grant to a JSON endpoint as
// it lands — the push form of GET /api/v1/grants (issue #915). It subscribes to
// the bus at construction, so grants decoded before Run starts are not lost,
// and delivers on a single worker with retry/backoff. A wedged endpoint fills
// the bounded queue and overflow grants are dropped rather than stalling the
// decoder.
type GrantWebhook struct {
	name       string
	endpoint   string
	authHeader string
	filter     systemFilter
	loc        *time.Location
	http       *http.Client
	log        *slog.Logger
	now        func() time.Time
	maxRetries int
	retryBase  time.Duration

	sub       *events.Subscription
	jobs      chan grantWebhookPayload
	wg        sync.WaitGroup
	runDone   chan struct{}
	closeOnce sync.Once

	mu      sync.Mutex
	queued  int
	dropped int
	sent    int
	failed  int
}

// NewGrantWebhook validates opts and returns a sink that has already
// subscribed to the bus. The caller must invoke Run and, on shutdown, Close.
func NewGrantWebhook(opts GrantWebhookOptions) (*GrantWebhook, error) {
	if opts.Bus == nil {
		return nil, errors.New("broadcast/grantwebhook: events.Bus is required")
	}
	if opts.Config.URL == "" {
		return nil, errors.New("broadcast/grantwebhook: url is required")
	}
	if opts.Log == nil {
		opts.Log = slog.Default()
	}
	if opts.MaxRetries <= 0 {
		opts.MaxRetries = grantWebhookMaxRetries
	}
	if opts.RetryBase <= 0 {
		opts.RetryBase = grantWebhookRetryBase
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.HTTP == nil {
		opts.HTTP = http.DefaultClient
	}
	name := opts.Config.Name
	if name == "" {
		name = "grant-webhook"
	}
	w := &GrantWebhook{
		name:       name,
		endpoint:   opts.Config.URL,
		authHeader: opts.Config.AuthHeader,
		filter:     newSystemFilter(opts.Config.Systems),
		loc:        opts.Config.Loc,
		http:       opts.HTTP,
		log:        opts.Log,
		now:        opts.Now,
		maxRetries: opts.MaxRetries,
		retryBase:  opts.RetryBase,
		jobs:       make(chan grantWebhookPayload, grantWebhookQueueDepth),
		runDone:    make(chan struct{}),
	}
	w.sub = opts.Bus.Subscribe()
	w.wg.Add(1)
	go w.worker()
	return w, nil
}

// Name reports the sink's log label.
func (w *GrantWebhook) Name() string { return w.name }

// Run drains KindGrant events until ctx is cancelled or the bus subscription
// closes.
func (w *GrantWebhook) Run(ctx context.Context) error {
	defer close(w.runDone)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-w.sub.C:
			if !ok {
				return nil
			}
			if ev.Kind != events.KindGrant {
				continue
			}
			g, ok := ev.Payload.(trunking.Grant)
			if !ok {
				continue
			}
			w.enqueue(g)
		}
	}
}

// enqueue applies the system filter and hands the grant to the worker,
// dropping it when the queue is full rather than blocking the event loop.
func (w *GrantWebhook) enqueue(g trunking.Grant) {
	if !w.filter.Accepts(g.System) {
		return
	}
	at := g.At
	if at.IsZero() {
		at = w.now()
	}
	p := grantWebhookPayloadFrom(g, at, w.loc)
	select {
	case w.jobs <- p:
		w.mu.Lock()
		w.queued++
		w.mu.Unlock()
	default:
		w.mu.Lock()
		w.dropped++
		w.mu.Unlock()
		w.log.Warn("broadcast/grantwebhook: queue full, dropping grant",
			"name", w.name, "system", g.System, "tg", g.GroupID)
	}
}

func (w *GrantWebhook) worker() {
	defer w.wg.Done()
	for p := range w.jobs {
		w.sendWithRetry(p)
	}
}

// sendWithRetry attempts delivery up to maxRetries+1 times with exponential
// backoff between attempts.
func (w *GrantWebhook) sendWithRetry(p grantWebhookPayload) {
	backoff := w.retryBase
	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := w.post(ctx, p)
		cancel()
		if err == nil {
			w.mu.Lock()
			w.sent++
			w.mu.Unlock()
			return
		}
		w.log.Warn("broadcast/grantwebhook: post failed",
			"name", w.name, "system", p.System, "tg", p.GroupID,
			"attempt", attempt+1, "of", w.maxRetries+1, "err", err)
		if attempt < w.maxRetries {
			time.Sleep(backoff)
			backoff *= 2
		}
	}
	w.mu.Lock()
	w.failed++
	w.mu.Unlock()
	w.log.Error("broadcast/grantwebhook: giving up on grant",
		"name", w.name, "system", p.System, "tg", p.GroupID)
}

// post marshals and POSTs one grant as application/json.
func (w *GrantWebhook) post(ctx context.Context, p grantWebhookPayload) error {
	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("%s: marshal: %w", w.name, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if w.authHeader != "" {
		req.Header.Set("Authorization", w.authHeader)
	}
	resp, err := w.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s: post: %w", w.name, err)
	}
	defer resp.Body.Close()
	msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("%s: HTTP %d: %s", w.name, resp.StatusCode,
			strings.TrimSpace(string(msg)))
	}
	return nil
}

// GrantWebhookStats is a point-in-time snapshot of a GrantWebhook's counters.
type GrantWebhookStats struct {
	Name    string `json:"name"`
	Queued  int    `json:"queued"`
	Dropped int    `json:"dropped"`
	Sent    int    `json:"sent"`
	Failed  int    `json:"failed"`
}

// Stats returns a snapshot of the sink's counters.
func (w *GrantWebhook) Stats() GrantWebhookStats {
	w.mu.Lock()
	defer w.mu.Unlock()
	return GrantWebhookStats{
		Name:    w.name,
		Queued:  w.queued,
		Dropped: w.dropped,
		Sent:    w.sent,
		Failed:  w.failed,
	}
}

// Close releases the bus subscription, waits for Run to drain, then stops the
// delivery worker. Safe to call multiple times.
func (w *GrantWebhook) Close() error {
	w.closeOnce.Do(func() {
		w.sub.Close()
		select {
		case <-w.runDone:
		case <-time.After(2 * time.Second):
		}
		close(w.jobs)
		w.wg.Wait()
	})
	return nil
}
