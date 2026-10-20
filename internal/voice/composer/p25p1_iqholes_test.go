package composer

import (
	"context"
	"log/slog"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestComposerP25Phase1VoiceChainReportsIQHoles drives the #1242 signature
// through the whole voice chain: C4FM IQ with 10 ms gaps cut out of three
// LDUs, as the reporter's captures had (2-3.5 % of the samples missing in
// 8-14 ms gaps). The chain must keep the LDU after each hole (the assembler
// used to swallow its frame sync, losing two LDUs per hole) and say in the
// log that the stream has holes. Before this, a call like that only showed
// as noise and tones, and the log could not tell a dropped stream from a weak
// signal.
func TestComposerP25Phase1VoiceChainReportsIQHoles(t *testing.T) {
	const (
		sampleRate = 48_000.0
		deviation  = 1800.0
		ldus       = 20
		gap        = 480 // 10 ms at 48 kHz = 48 symbols
	)
	dibits, _ := buildP25P1VoiceStream(t, ldus)
	iq := demod.ModulateP25C4FM(dibits, sampleRate, deviation)
	// 10 samples per symbol; the stream opens with a 400-symbol lead-in.
	const sps, lead, lduSyms = 10, 400, 864
	var holey []complex64
	prev := 0
	for _, l := range []int{4, 9, 14} {
		cut := (lead + l*lduSyms + 500) * sps
		holey = append(holey, iq[prev:cut]...)
		prev = cut + gap
	}
	holey = append(holey, iq[prev:]...)
	// Trailing silence-free filler so the last LDU's window completes.
	holey = append(holey, demod.ModulateP25C4FM(make([]uint8, 1200), sampleRate, deviation)...)

	buf := &syncBuffer{}
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	src := newFakeSource()
	bus := events.NewBus(8)
	c, err := New(Options{
		Bus:           bus,
		Log:           logger,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-1": src}},
		Sink:          &recordingSink{},
		Engine:        &fakeEngine{},
		IQSampleRate:  uint32(sampleRate),
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

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant:        trunking.Grant{System: "P25P1Site", Protocol: "p25", GroupID: 42, FrequencyHz: 851_000_000},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})
	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(holey)

	ldusRe := regexp.MustCompile(`composer: p25p1 decode quality.* ldus=(\d+)`)
	// Let the chain chew through the IQ, then end the call so the final
	// summary (and the holes WARN) is written.
	time.Sleep(1500 * time.Millisecond)
	publishEnd(bus, "VOICE-1")
	waitFor(t, 6*time.Second, func() bool { return regexp.MustCompile(`voice IQ has holes`).MatchString(buf.String()) })
	out := buf.String()

	warn := regexp.MustCompile(`level=WARN msg="composer: p25p1 voice IQ has holes[^"]*".* short_ldus=(\d+) ldus=(\d+) missing_symbols=(\d+)`).FindStringSubmatch(out)
	if warn == nil {
		t.Fatalf("no IQ-holes WARN in the log:\n%s", out)
	}
	if warn[1] != "3" {
		t.Errorf("short_ldus = %s, want 3 (one per gap)", warn[1])
	}
	if ms, _ := strconv.Atoi(warn[3]); ms < 3*40 || ms > 3*56 {
		t.Errorf("missing_symbols = %d, want ~144 (three 48-symbol gaps)", ms)
	}
	// Every LDU reaches the chain: the three short ones, and the three that
	// follow them, which the old assembler lost.
	m := ldusRe.FindAllStringSubmatch(out, -1)
	if len(m) == 0 {
		t.Fatalf("no decode quality line:\n%s", out)
	}
	if n, _ := strconv.Atoi(m[len(m)-1][1]); n < ldus-1 {
		t.Errorf("chain delivered %d LDUs of %d; the frame after each hole must be kept", n, ldus)
	}
}
