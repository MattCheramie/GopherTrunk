package phase1

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestMotorolaProbeLogsEachPayloadOnceAsBroadcast is issue #1273: on a
// Motorola-built P25 site (MOSWIN), the INFO line for opcodes 0x05 / 0x09
// read "MOTOROLA_OSP_TRAFFIC_CHANNEL_ID … named, not field-decoded", 64
// times per opcode, and the reporter took it for the call grants
// GopherTrunk could not decode. The grants were standard TSBKs all along
// (logged at Debug), and the site sent only a handful of constant payloads.
// The line now says it is a site broadcast, not a grant, and logs each
// distinct payload once. Payloads are the reporter's literal on-air values.
func TestMotorolaProbeLogsEachPayloadOnceAsBroadcast(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	bus := events.NewBus(16)
	defer bus.Close()
	cc := New(Options{Bus: bus, Log: log, SystemName: "MOSWIN"})

	frames := []struct {
		op      Opcode
		payload [8]byte
	}{
		{OpMotorolaTrafficChannelID, [8]byte{0x40, 0x00, 0x80, 0x00, 0x00, 0x00, 0x08, 0x00}},
		{OpMotorolaTrafficChannelID, [8]byte{0xC0, 0x00, 0x80, 0x00, 0x00, 0x00, 0x08, 0x00}},
		{OpMotorolaSystemLoading, [8]byte{0x11, 0x80}},
	}
	i := 0
	for range 5 { // the site repeats each one all day
		for _, f := range frames {
			lead := 0
			if i == 0 {
				lead = 10 // establish lock on the first frame
			}
			tsbk := TSBK{Opcode: f.op, MFID: MFIDMotorola, Payload: f.payload}
			cc.Process(buildLockedStreamWithTSBK(lead, 0x1C4, DUIDTrunkingSignaling, tsbk), i<<20)
			i++
		}
	}

	out := buf.String()
	lines := strings.Count(out, "p25: motorola opcode")
	if lines != len(frames) {
		t.Fatalf("%d motorola opcode lines for %d distinct payloads sent 5 times each, want one per payload:\n%s", lines, len(frames), out)
	}
	if n := strings.Count(out, "not a call grant"); n != len(frames) {
		t.Errorf("%d of %d lines say they are not a call grant:\n%s", n, len(frames), out)
	}
	for _, want := range []string{"payload=4000800000000800", "payload=c000800000000800", "payload=1180000000000000"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s:\n%s", want, out)
		}
	}
}
