package voice

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestRecorderWriteRawGovernsDecodedProtocols pins #1321: recordings.write_raw
// decides the .raw sidecar for TETRA (TMO and DMO) and DMR like every other
// protocol with an in-process vocoder. Failing-first: the recorder used to
// force a sidecar for those protocols, so `write_raw: false` still wrote a .raw
// for every TETRA call. (EDACS ProVoice, which has no in-process decoder, still
// forces one: TestRecorderProVoiceForcesRawSidecar.)
func TestRecorderWriteRawGovernsDecodedProtocols(t *testing.T) {
	for _, proto := range []string{"tetra", "tetra-dmo", "dmr-tier2"} {
		for _, writeRaw := range []bool{false, true} {
			name := proto + "/write_raw=false"
			if writeRaw {
				name = proto + "/write_raw=true"
			}
			t.Run(name, func(t *testing.T) {
				rawPath, size := recordOneRawFrame(t, proto, writeRaw)
				if !writeRaw {
					if _, err := os.Stat(rawPath); !os.IsNotExist(err) {
						t.Fatalf(".raw sidecar written with write_raw=false: %s (err=%v)", rawPath, err)
					}
					return
				}
				if size != 18 {
					t.Fatalf("raw sidecar size = %d, want 18 (%s)", size, rawPath)
				}
			})
		}
	}
}

// recordOneRawFrame runs one call of the given protocol through the recorder,
// writes a single 18-byte frame (the composer's packed 137-bit TCH/S shape;
// the sidecar write is verbatim) and returns where its .raw sidecar would be
// and its size (-1 if absent).
func recordOneRawFrame(t *testing.T, proto string, writeRaw bool) (string, int) {
	t.Helper()
	r, bus, dir := mkRecorder(t, writeRaw)
	defer r.Close()
	defer bus.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "S", Protocol: proto, GroupID: 7, SourceID: 1},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 1, 2, 3, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	waitSession(t, r, "VOICE-1", true)

	frame := make([]byte, 18)
	for i := range frame {
		frame[i] = byte(i)
	}
	if err := r.WriteRawFrame("VOICE-1", frame); err != nil {
		t.Fatal(err)
	}
	bus.Publish(events.Event{
		Kind: events.KindCallEnd,
		Payload: trunking.CallEnd{
			Grant: cs.Grant, DeviceSerial: "VOICE-1",
			StartedAt: cs.StartedAt, EndedAt: cs.StartedAt.Add(time.Second),
			Reason: trunking.EndReasonNormal,
		},
	})
	waitSession(t, r, "VOICE-1", false)

	rawPath := filepath.Join(dir, "S", "7", "20260505_010203_7.raw")
	data, err := os.ReadFile(rawPath)
	if err != nil {
		return rawPath, -1
	}
	return rawPath, len(data)
}
