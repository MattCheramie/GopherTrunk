package phase1

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestUnresolvedChannelIDGrantsWarn is the #1273 observability regression:
// grants that name a channel ID the site never announced used to be dropped
// with only a DEBUG line, so a site that never followed a call looked like a
// site with no traffic. A grant inside pendingGrantTTL of the first miss is
// the normal pre-IDEN_UP race and stays quiet; once the ID stays unresolved
// past it, one WARN names the ID, rate-limited, and an IDEN_UP for the ID
// resets the state.
func TestUnresolvedChannelIDGrantsWarn(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	var buf bytes.Buffer
	now := time.Unix(1_700_000_000, 0).UTC()
	cc := New(Options{
		Bus:        bus,
		Log:        slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})),
		Now:        func() time.Time { return now },
		SystemName: "MOSWIN", FrequencyHz: 161_912_500,
	})
	cc.bandPlan.Apply(IdentifierUpdate{ChannelID: 1, BaseHz: 161_000_000, SpacingHz: 12_500})
	grant := voiceGrant{groupID: 7405, sourceID: 114063, channelID: 2, channelNumber: 9}
	warns := func() int { return strings.Count(buf.String(), "has not announced") }

	cc.publishVoiceGrant(grant, 0x1C4)
	now = now.Add(pendingGrantTTL - time.Second)
	cc.publishVoiceGrant(grant, 0x1C4)
	if n := warns(); n != 0 {
		t.Fatalf("WARN inside pendingGrantTTL (normal pre-IDEN_UP race): %d lines\n%s", n, buf.String())
	}

	now = now.Add(2 * time.Second) // past the TTL since the first miss
	cc.publishVoiceGrant(grant, 0x1C4)
	if n := warns(); n != 1 {
		t.Fatalf("want 1 WARN once the ID stays unresolved past the TTL, got %d\n%s", n, buf.String())
	}
	line := buf.String()
	for _, want := range []string{"id=2", "num=9", "grants=3", "announced_ids=[1]", "p25_band_plan"} {
		if !strings.Contains(line, want) {
			t.Errorf("WARN missing %q: %s", want, line)
		}
	}

	now = now.Add(time.Minute)
	cc.publishVoiceGrant(grant, 0x1C4)
	if n := warns(); n != 1 {
		t.Fatalf("WARN not rate-limited: %d lines", n)
	}

	// The ID gets announced: state resets, so a later miss starts a fresh
	// TTL window rather than warning at once.
	cc.bandPlan.Apply(IdentifierUpdate{ChannelID: 2, BaseHz: 161_000_000, SpacingHz: 12_500})
	cc.drainPendingGrants(2, 0x1C4)
	if got := cc.unresolved[2]; !got.first.IsZero() || got.grants != 0 {
		t.Errorf("unresolved state not reset by IDEN_UP: %+v", got)
	}
}
