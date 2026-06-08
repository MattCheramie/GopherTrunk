//go:build integration

package composer

import (
	"context"
	"encoding/binary"
	"math"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestDMR2SlotRealairRoutesByEmbeddedLC validates the opt-in 2-slot
// interleaved voice path against a real DMR capture — the cross-check
// docs/status.md flags as the gate before the interleaved decoder can
// replace the single-slot decoder as the production default.
//
// It is skip-gated on environment so CI (which has no capture) stays
// green; a contributor with a recording runs it as:
//
//	GOPHERTRUNK_DMR_2SLOT_CFILE=cap.cfile \
//	GOPHERTRUNK_DMR_2SLOT_RATE_HZ=48000 \
//	GOPHERTRUNK_DMR_2SLOT_TG=12345 \
//	go test -tags integration ./internal/voice/composer/ -run DMR2SlotRealair -v
//
// The cfile is GNU Radio interleaved float32 IQ at the given rate; TG is
// the talkgroup of the call to follow on the carrier. A pass means the
// interleaved decoder recovered that talkgroup's embedded LC and routed
// its (and only its) audio to the sidecar.
func TestDMR2SlotRealairRoutesByEmbeddedLC(t *testing.T) {
	path := os.Getenv("GOPHERTRUNK_DMR_2SLOT_CFILE")
	if path == "" {
		t.Skip("set GOPHERTRUNK_DMR_2SLOT_CFILE (+ _RATE_HZ, _TG) to run the DMR 2-slot real-air check")
	}
	rate, err := strconv.ParseUint(os.Getenv("GOPHERTRUNK_DMR_2SLOT_RATE_HZ"), 10, 32)
	if err != nil || rate == 0 {
		t.Fatalf("GOPHERTRUNK_DMR_2SLOT_RATE_HZ must be a positive integer: %v", err)
	}
	tg, err := strconv.ParseUint(os.Getenv("GOPHERTRUNK_DMR_2SLOT_TG"), 10, 32)
	if err != nil {
		t.Fatalf("GOPHERTRUNK_DMR_2SLOT_TG must be an integer talkgroup: %v", err)
	}
	iq := readCfileF32(t, path)

	src := newFakeSource()
	bus := events.NewBus(8)
	sink := &recordingSink{}
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-1": src}},
		Sink:          sink,
		Engine:        &fakeEngine{},
		IQSampleRate:  uint32(rate),
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	defer c.Close()
	defer bus.Close()

	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: trunking.CallStart{
		Grant: trunking.Grant{
			System: "DMRSite", Protocol: "dmr-tier3",
			GroupID: uint32(tg), FrequencyHz: 460_000_000,
			Timeslot: 1, DMRInterleavedVoice: true,
		},
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Now().UTC(),
	}})

	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)
	waitFor(t, 10*time.Second, func() bool { return len(sink.rawFrames("VOICE-1")) > 0 })

	if got := len(sink.rawFrames("VOICE-1")); got == 0 {
		t.Fatalf("no AMBE frames routed for talkgroup %d — interleaved decode/cadence/embedded-LC constants need adjustment against this capture", tg)
	} else {
		t.Logf("routed %d AMBE frames for talkgroup %d from %s", got, tg, path)
	}
}

func readCfileF32(t *testing.T, path string) []complex64 {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read cfile: %v", err)
	}
	if len(raw)%8 != 0 {
		t.Fatalf("cfile length %d is not a multiple of 8 (interleaved float32 IQ)", len(raw))
	}
	out := make([]complex64, len(raw)/8)
	for i := range out {
		re := math.Float32frombits(binary.LittleEndian.Uint32(raw[i*8:]))
		im := math.Float32frombits(binary.LittleEndian.Uint32(raw[i*8+4:]))
		out[i] = complex(re, im)
	}
	return out
}
