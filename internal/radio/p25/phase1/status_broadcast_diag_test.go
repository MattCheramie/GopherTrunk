package phase1

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestControlChannelLogsStatusBroadcastPayload confirms that the HANDLED
// status broadcasts (adjacent 0x3C, network 0x3B, RFSS 0x3A) now dump their
// raw payload bytes plus the decoded fields, sampled per opcode and capped at
// maxStatusBroadcastSamples. This is pure instrumentation — added so a field
// tester (and the JSONL event sink) captures the bytes behind a reported
// neighbour/identity mismatch when there is no raw IQ to replay. It must not
// change parsing: the decoded rfss/site attrs are asserted against the real
// Mt Anakie ADJ frame to prove the spec-correct layout is what we log.
func TestControlChannelLogsStatusBroadcastPayload(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	bus := events.NewBus(16)
	defer bus.Close()
	cc := New(Options{Bus: bus, Log: log, SystemName: "S"})

	// Real on-air Mt Anakie ADJ_STS_BCST argument bytes: RFSS 4, Site 0x2E,
	// channel F.065 — the same vector pinned by TestParseAdjacentSiteStatusBroadcast.
	adj := TSBK{Opcode: OpAdjacentSiteStatusBroadcast,
		Payload: [8]byte{0x00, 0x31, 0x64, 0x04, 0x2E, 0xF0, 0x65, 0x70}}
	const fires = maxStatusBroadcastSamples + 4
	for i := 0; i < fires; i++ {
		lead := 0
		if i == 0 {
			lead = 10 // establish lock on the first frame
		}
		cc.Process(buildLockedStreamWithTSBK(lead, 0x293, DUIDTrunkingSignaling, adj), i<<20)
	}

	out := buf.String()
	if n := strings.Count(out, `msg="p25: status broadcast payload"`); n != maxStatusBroadcastSamples {
		t.Errorf("status-broadcast payload lines = %d, want %d (capped)", n, maxStatusBroadcastSamples)
	}
	if !strings.Contains(out, "payload=003164042ef06570") {
		t.Error("status-broadcast line missing the raw payload hex")
	}
	// Decoded attrs must reflect the spec-correct layout (RFSS@byte3=4,
	// Site@byte4=0x2E=46), proving instrumentation didn't alter the parse.
	if !strings.Contains(out, "rfss=4") || !strings.Contains(out, "site=46") {
		t.Errorf("status-broadcast line missing/incorrect decoded rfss/site:\n%s", firstLine(out))
	}
	if !strings.Contains(out, "opcode=0x3C") {
		t.Error("status-broadcast line missing numeric opcode")
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
