package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

type fakeGrantsProvider struct {
	grants []trunking.Grant
}

// RecentGrants returns newest-first up to limit (0 = all), mirroring the real
// tracker's contract so the handler's fetch/limit logic is exercised faithfully.
func (f *fakeGrantsProvider) RecentGrants(limit int) []trunking.Grant {
	out := append([]trunking.Grant(nil), f.grants...)
	if limit > 0 && limit < len(out) {
		out = out[:limit]
	}
	return out
}

func getGrants(t *testing.T, base, query string) []GrantLogEntry {
	t.Helper()
	resp, err := http.Get(base + "/api/v1/grants" + query)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var body struct {
		Grants []GrantLogEntry `json:"grants"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return body.Grants
}

// TestGrantsEndpointCarriesSourceRID is the API-side #915 guard: the grant log
// endpoint surfaces the source RID (and talkgroup, frequency, encryption) off
// the control-channel grant, newest first.
func TestGrantsEndpointCarriesSourceRID(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()

	now := time.Unix(1700, 0).UTC()
	live := &fakeGrantsProvider{grants: []trunking.Grant{
		// newest first, as the tracker returns
		{System: "MMR", Protocol: "p25", GroupID: 20001, SourceID: 202419,
			FrequencyHz: 422612500, Encrypted: true, AlgorithmID: 0x84, KeyID: 0x1234, At: now.Add(2 * time.Second)},
		{System: "MMR", Protocol: "p25", GroupID: 20002, SourceID: 0, FrequencyHz: 422637500, At: now.Add(time.Second)},
	}}

	base, teardown := mkServer(t, ServerOptions{Bus: bus, Grants: live})
	defer teardown()

	grants := getGrants(t, base, "")
	if len(grants) != 2 {
		t.Fatalf("got %d grants, want 2", len(grants))
	}
	g0 := grants[0]
	if g0.SourceID != 202419 || g0.GroupID != 20001 || g0.FrequencyHz != 422612500 {
		t.Fatalf("first grant fields wrong: %+v", g0)
	}
	if !g0.Encrypted || g0.AlgorithmID != 0x84 || g0.KeyID != 0x1234 {
		t.Fatalf("encryption params lost: %+v", g0)
	}
	if g0.At.IsZero() {
		t.Error("grant row missing timestamp")
	}
	// A source-less grant is still surfaced (coverage-truthful).
	if grants[1].SourceID != 0 || grants[1].GroupID != 20002 {
		t.Fatalf("second grant wrong: %+v", grants[1])
	}
}

// TestGrantsEndpointEmptyWhenUnwired: no tracker ⇒ stable empty shape, 200.
func TestGrantsEndpointEmptyWhenUnwired(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{Bus: bus})
	defer teardown()

	if grants := getGrants(t, base, ""); len(grants) != 0 {
		t.Fatalf("want empty list when tracker unwired, got %d", len(grants))
	}
}

// TestGrantsEndpointLimitAndSystemFilter exercises the query params.
func TestGrantsEndpointLimitAndSystemFilter(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()

	live := &fakeGrantsProvider{grants: []trunking.Grant{
		{System: "A", Protocol: "p25", GroupID: 1, SourceID: 10, FrequencyHz: 1},
		{System: "B", Protocol: "p25", GroupID: 2, SourceID: 20, FrequencyHz: 2},
		{System: "A", Protocol: "p25", GroupID: 3, SourceID: 30, FrequencyHz: 3},
	}}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Grants: live})
	defer teardown()

	if g := getGrants(t, base, "?limit=1"); len(g) != 1 || g[0].SourceID != 10 {
		t.Fatalf("?limit=1 = %+v, want single newest", g)
	}
	// System filter returns only matching rows, across the full window.
	sys := getGrants(t, base, "?system=A")
	if len(sys) != 2 {
		t.Fatalf("?system=A returned %d rows, want 2", len(sys))
	}
	for _, g := range sys {
		if g.System != "A" {
			t.Fatalf("system filter leaked a %q row", g.System)
		}
	}
}
