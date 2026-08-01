package broadcast

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// grantSink is an httptest server that records every POSTed grant payload.
type grantSink struct {
	*httptest.Server
	mu   sync.Mutex
	got  []map[string]any
	auth string
	ct   string
}

func newGrantSink(t *testing.T) *grantSink {
	t.Helper()
	gs := &grantSink{}
	gs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		var m map[string]any
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			t.Errorf("decode body: %v", err)
		}
		gs.mu.Lock()
		gs.auth = r.Header.Get("Authorization")
		gs.ct = r.Header.Get("Content-Type")
		gs.got = append(gs.got, m)
		gs.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(gs.Server.Close)
	return gs
}

func (gs *grantSink) count() int {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return len(gs.got)
}

func (gs *grantSink) waitFor(t *testing.T, want int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if gs.count() >= want {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d POSTed grants (have %d)", want, gs.count())
}

func startGrantWebhook(t *testing.T, opts GrantWebhookOptions) *GrantWebhook {
	t.Helper()
	w, err := NewGrantWebhook(opts)
	if err != nil {
		t.Fatalf("NewGrantWebhook: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go w.Run(ctx)
	t.Cleanup(func() { cancel(); w.Close() })
	return w
}

// TestGrantWebhookPostsDecodedGrant is the issue #915 / #268 guard for the push
// grant sink: a grant decoded off the control channel is POSTed as one JSON
// object carrying the source RID (and talkgroup, frequency, site, encryption)
// on the GrantDTO schema, so a consumer reads it the way SDRtrunk's grant log
// exposes it — without holding an SSE connection or polling /api/v1/grants.
func TestGrantWebhookPostsDecodedGrant(t *testing.T) {
	sink := newGrantSink(t)
	bus := events.NewBus(16)
	defer bus.Close()

	fixed := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	startGrantWebhook(t, GrantWebhookOptions{
		Bus:  bus,
		HTTP: sink.Client(),
		Now:  func() time.Time { return fixed },
		Config: GrantWebhookConfig{
			URL:        sink.URL,
			AuthHeader: "Bearer t0ken",
			Loc:        time.UTC,
		},
	})

	bus.Publish(events.Event{Kind: events.KindGrant, Payload: trunking.Grant{
		System: "MMR", Protocol: "p25", GroupID: 20001, SourceID: 202419,
		FrequencyHz: 422612500, RFSSID: 1, SiteID: 9, NAC: 359,
		Encrypted: true, AlgorithmID: 0x84, KeyID: 3,
	}})
	sink.waitFor(t, 1)

	sink.mu.Lock()
	defer sink.mu.Unlock()
	if sink.ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", sink.ct)
	}
	if sink.auth != "Bearer t0ken" {
		t.Errorf("Authorization = %q, want Bearer t0ken", sink.auth)
	}
	g := sink.got[0]
	if g["event"] != "grant" {
		t.Errorf("event = %v, want grant", g["event"])
	}
	// JSON numbers decode as float64.
	if g["source_id"].(float64) != 202419 {
		t.Errorf("source_id = %v, want 202419", g["source_id"])
	}
	if g["group_id"].(float64) != 20001 {
		t.Errorf("group_id = %v, want 20001", g["group_id"])
	}
	if g["frequency_hz"].(float64) != 422612500 {
		t.Errorf("frequency_hz = %v, want 422612500", g["frequency_hz"])
	}
	if g["rfss_id"].(float64) != 1 || g["site_id"].(float64) != 9 {
		t.Errorf("site identity = rfss %v site %v, want 1/9", g["rfss_id"], g["site_id"])
	}
	if g["encrypted"] != true {
		t.Errorf("encrypted = %v, want true", g["encrypted"])
	}
	if g["algorithm_id"].(float64) != 0x84 {
		t.Errorf("algorithm_id = %v, want 132", g["algorithm_id"])
	}
	if g["at"] != "2026-08-01T12:00:00Z" {
		t.Errorf("at = %v, want the stamped decode time", g["at"])
	}
}

// TestGrantWebhookSourceZeroTruthful pins the coverage-truthful contract: a
// grant whose TSBK carried src=0 is POSTed with source_id=0 present (not
// omitted, not backfilled), so the feed reports exactly what the control
// channel saw — the property #915 relies on to distinguish a real grant log
// from the voice-side completed-call source.
func TestGrantWebhookSourceZeroTruthful(t *testing.T) {
	sink := newGrantSink(t)
	bus := events.NewBus(16)
	defer bus.Close()

	startGrantWebhook(t, GrantWebhookOptions{
		Bus:    bus,
		HTTP:   sink.Client(),
		Config: GrantWebhookConfig{URL: sink.URL},
	})

	bus.Publish(events.Event{Kind: events.KindGrant, Payload: trunking.Grant{
		System: "MMR", Protocol: "p25", GroupID: 20001, SourceID: 0,
		FrequencyHz: 422612500,
	}})
	sink.waitFor(t, 1)

	sink.mu.Lock()
	defer sink.mu.Unlock()
	g := sink.got[0]
	v, ok := g["source_id"]
	if !ok {
		t.Fatal("source_id absent; must be present and zero (coverage-truthful)")
	}
	if v.(float64) != 0 {
		t.Errorf("source_id = %v, want 0", v)
	}
}

// TestGrantWebhookSystemFilter confirms the Systems filter drops grants from
// other systems and delivers only the configured one.
func TestGrantWebhookSystemFilter(t *testing.T) {
	sink := newGrantSink(t)
	bus := events.NewBus(16)
	defer bus.Close()

	startGrantWebhook(t, GrantWebhookOptions{
		Bus:    bus,
		HTTP:   sink.Client(),
		Config: GrantWebhookConfig{URL: sink.URL, Systems: []string{"Metro"}},
	})

	bus.Publish(events.Event{Kind: events.KindGrant, Payload: trunking.Grant{
		System: "Regional", Protocol: "p25", GroupID: 1, FrequencyHz: 1,
	}})
	bus.Publish(events.Event{Kind: events.KindGrant, Payload: trunking.Grant{
		System: "Metro", Protocol: "p25", GroupID: 2, FrequencyHz: 2,
	}})
	sink.waitFor(t, 1)
	// Give any erroneously-delivered second grant a chance to land.
	time.Sleep(50 * time.Millisecond)

	sink.mu.Lock()
	defer sink.mu.Unlock()
	if len(sink.got) != 1 {
		t.Fatalf("delivered %d grants, want 1 (filtered)", len(sink.got))
	}
	if sink.got[0]["system"] != "Metro" {
		t.Errorf("delivered system = %v, want Metro", sink.got[0]["system"])
	}
}

// TestGrantWebhookIgnoresNonGrant confirms non-grant events never reach the
// sink (the sink is a grant feed, not the call feed).
func TestGrantWebhookIgnoresNonGrant(t *testing.T) {
	sink := newGrantSink(t)
	bus := events.NewBus(16)
	defer bus.Close()

	startGrantWebhook(t, GrantWebhookOptions{
		Bus:    bus,
		HTTP:   sink.Client(),
		Config: GrantWebhookConfig{URL: sink.URL},
	})

	bus.Publish(events.Event{Kind: events.KindCallComplete, Payload: trunking.CallComplete{}})
	bus.Publish(events.Event{Kind: events.KindGrant, Payload: trunking.Grant{
		System: "MMR", Protocol: "p25", GroupID: 1, FrequencyHz: 1,
	}})
	sink.waitFor(t, 1)

	if n := sink.count(); n != 1 {
		t.Fatalf("delivered %d payloads, want only the grant (1)", n)
	}
}

// TestNewGrantWebhookRequiresURL guards the constructor validation.
func TestNewGrantWebhookRequiresURL(t *testing.T) {
	bus := events.NewBus(1)
	defer bus.Close()
	if _, err := NewGrantWebhook(GrantWebhookOptions{Bus: bus}); err == nil {
		t.Fatal("NewGrantWebhook with no URL: want error, got nil")
	}
	if _, err := NewGrantWebhook(GrantWebhookOptions{Config: GrantWebhookConfig{URL: "http://x"}}); err == nil {
		t.Fatal("NewGrantWebhook with no Bus: want error, got nil")
	}
}
