package composer

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// The identity p25p2TestSeed is built from.
const (
	p25p2TestWACN  = 0xBEE00
	p25p2TestSysID = 0x164
	p25p2TestNAC   = 0x161
)

// p25p2KeyupStream is a Phase 2 call whose key-up MAC_PTT (source radio +
// encryption sync) rides only the first superframe, as on air; the rest is
// voice. A chain that cannot descramble that first superframe never learns the
// call's source.
func p25p2KeyupStream(t *testing.T, superframes int) (iq []complex64, wantSrc uint32) {
	t.Helper()
	wantSrc = 200062
	ptt := p25p2.AirMACPTTBytes(p25p2.MACPTT{
		Encryption: p25p2.EncryptionSync{AlgorithmID: 0x80},
		Source:     wantSrc, Group: 0x4EEA,
	})
	calls := 0
	dibits := p25p2MACStream(t, superframes, func(slot int) *p25p2.AirSlotContent {
		calls++
		if slot == 0 && calls == 1 {
			return &p25p2.AirSlotContent{DUID: p25p2.DUIDFACCHScrambled, MAC: ptt}
		}
		return nil
	})
	return demod.ModulateHDQPSKSpec(dibits, 8, 8, 0.20), wantSrc
}

// runP25P2ChainSplit runs the voice chain over iq in two halves. Between
// them (after the first half has been fully processed) it calls between.
func runP25P2ChainSplit(t *testing.T, c *Composer, seed uint64, grantNAC uint16, iq []complex64, between func()) {
	t.Helper()
	iqCh := make(chan []complex64)
	done := make(chan struct{})
	go c.runP25Phase2VoiceChain(context.Background(), "VOICE-1", "MMR",
		p25p2.MACDecodeConfig{Seed: seed}, 1, grantNAC, iqCh, 48_000, done)
	half := len(iq) / 2
	iqCh <- iq[:half]
	// The chain reads its channel from one goroutine, so this send returns
	// only once the first half has been processed.
	iqCh <- nil
	if between != nil {
		between()
	}
	iqCh <- iq[half:]
	close(iqCh)
	<-done
}

func newP25P2SeedTestComposer(t *testing.T, bus *events.Bus, sink *recordingSink, log *bytes.Buffer) *Composer {
	t.Helper()
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{}},
		Sink:          sink,
		Engine:        &fakeEngine{},
		IQSampleRate:  48_000,
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
		Log:           slog.New(slog.NewTextHandler(log, &slog.HandlerOptions{Level: slog.LevelInfo})),
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// A Phase 2 call granted before its control channel had decoded the site's
// Network Status Broadcast carries no descramble seed (issue #1364). The
// chain used to drop every voice and scrambled signalling burst for the whole
// call. It now holds them and, once the site's identity is published,
// decodes them: the key-up MAC_PTT that names the source radio, and every
// voice frame, exactly as a chain that had the seed from the start.
func TestComposerP25Phase2LearnsSeedFromSiteIdentityAfterGrant(t *testing.T) {
	iq, wantSrc := p25p2KeyupStream(t, 6)

	// Reference: the seed known from the start.
	refSink := &recordingSink{}
	var refLog bytes.Buffer
	ref := newP25P2SeedTestComposer(t, events.NewBus(256), refSink, &refLog)
	runP25P2ChainSplit(t, ref, p25p2TestSeed, p25p2TestNAC, iq, nil)
	wantFrames := len(refSink.raw["VOICE-1"])
	if wantFrames == 0 {
		t.Fatalf("reference chain wrote no voice frames:\n%s", refLog.String())
	}

	bus := events.NewBus(256)
	sub := bus.Subscribe()
	defer sub.Close()
	sink := &recordingSink{}
	var log bytes.Buffer
	c := newP25P2SeedTestComposer(t, bus, sink, &log)
	// The grant names the NAC it was heard on; the site update that
	// follows names the network. Neither alone is a seed.
	runP25P2ChainSplit(t, c, 0, p25p2TestNAC, iq, func() {
		c.p25IDs.Note(trunking.SiteUpdate{System: "MMR", WACN: p25p2TestWACN, SystemID: p25p2TestSysID})
	})

	if got := len(sink.raw["VOICE-1"]); got != wantFrames {
		t.Errorf("voice frames written: %d, want %d (the chain with the seed from the start)", got, wantFrames)
	}
	for i, f := range sink.raw["VOICE-1"] {
		if i < wantFrames && !bytes.Equal(f, refSink.raw["VOICE-1"][i]) {
			t.Fatalf("voice frame %d differs from the reference chain's", i)
		}
	}
	var gotSrc bool
	for drained := false; !drained; {
		select {
		case ev := <-sub.C:
			if s, ok := ev.Payload.(trunking.CallSourceUpdate); ok && ev.Kind == events.KindCallSourceUpdate && s.SourceID == wantSrc {
				gotSrc = true
			}
		default:
			drained = true
		}
	}
	if !gotSrc {
		t.Errorf("no call source update naming %d: the key-up MAC_PTT held before the seed was lost", wantSrc)
	}
	out := log.String()
	for _, want := range []string{"decoding the held bursts", "descramble_seed_late=true", "seed_wait_dropped_bursts=0"} {
		if !strings.Contains(out, want) {
			t.Errorf("log is missing %q:\n%s", want, out)
		}
	}
}

// A chain whose seed never arrives still ends with a census that says so,
// and counts the bursts it could not decode.
func TestComposerP25Phase2CensusCountsBurstsHeldWithoutSeed(t *testing.T) {
	iq, _ := p25p2KeyupStream(t, 3)
	sink := &recordingSink{}
	var log bytes.Buffer
	c := newP25P2SeedTestComposer(t, events.NewBus(256), sink, &log)
	runP25P2ChainSplit(t, c, 0, p25p2TestNAC, iq, nil)
	if n := len(sink.raw["VOICE-1"]); n != 0 {
		t.Fatalf("%d voice frames written with no seed", n)
	}
	out := log.String()
	if !strings.Contains(out, "descramble_seed_known=false") || strings.Contains(out, "seed_wait_dropped_bursts=0 ") {
		t.Fatalf("census should report the unknown seed and the undecoded held bursts:\n%s", out)
	}
}

// The composer learns site identities from the bus: a chain started after a
// site update takes its seed at once.
func TestComposerNotesSiteIdentityFromBus(t *testing.T) {
	bus := events.NewBus(16)
	var log bytes.Buffer
	c := newP25P2SeedTestComposer(t, bus, &recordingSink{}, &log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	defer c.Close()
	bus.Publish(events.Event{Kind: events.KindSiteUpdate, Payload: trunking.SiteUpdate{
		System: "MMR", WACN: p25p2TestWACN, SystemID: p25p2TestSysID, NAC: p25p2TestNAC,
	}})
	waitFor(t, 2*time.Second, func() bool { return c.p25IDs.Seed("MMR", 0) == p25p2TestSeed })
}
