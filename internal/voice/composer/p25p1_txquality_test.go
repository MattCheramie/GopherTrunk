package composer

import (
	"context"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/MattCheramie/GopherTrunk/internal/voice/imbe"
)

// p25p1OverDibits builds one transmission: ldus LDUs, the LDU1s' link
// control naming src, with flips on-air bits inverted in every voice subframe (each
// lands in one FEC codeword, so the decoder corrects it and counts it).
func p25p1OverDibits(t *testing.T, ldus int, tg uint16, src uint32, flips int, frame *int) []uint8 {
	t.Helper()
	lces := phase1.AssembleLinkControl(phase1.LinkControl{
		LCFormat: phase1.LCOGroupVoiceChannelUser, TalkgroupID: tg, SourceID: src,
	})
	ess := phase1.AssembleEncryptionSync(phase1.EncryptionSync{AlgorithmID: 0x80})
	var dibits []uint8
	for l := 0; l < ldus; l++ {
		var voice [phase1.LDUVoiceSubframeCount][]byte
		for s := range voice {
			onAir, err := imbe.EncodeFrameToChannel(p25p1VoiceInfo(*frame))
			*frame++
			if err != nil {
				t.Fatal(err)
			}
			for f := 0; f < flips; f++ {
				onAir[f*37] ^= 1
			}
			voice[s] = onAir
		}
		var lsd [phase1.LDULSDBlockCount][]byte
		// Alternate LDU1 / LDU2 as a real over does; LDU2 carries a clear
		// Encryption Sync. A stream of identical LDU1s repeats one fixed
		// pattern every 180 ms, which no real transmission does.
		duid, blocks := phase1.DUIDLogicalLink1, lces
		if l%2 == 1 {
			duid, blocks = phase1.DUIDLogicalLink2, ess
		}
		ldu, err := phase1.AssembleLDU(0x123, duid, voice, blocks, lsd)
		if err != nil {
			t.Fatal(err)
		}
		dibits = append(dibits, framing.BitsToDibits(ldu)...)
	}
	return dibits
}

// p25p1TDUDibits is a 144-bit Terminator Data Unit: frame sync + NID, zero
// fill, with a status symbol after every 70 bits.
func p25p1TDUDibits() []uint8 {
	const total = 144
	payload := append(phase1.FrameSyncBits(), phase1.EncodeNIDBits(0x123, phase1.DUIDTerminator)...)
	stride := phase1.LDUStatusInterval + 2
	for len(payload) < total/stride*phase1.LDUStatusInterval {
		payload = append(payload, 0)
	}
	var bits []byte
	for i := 0; i < total/stride; i++ {
		bits = append(bits, payload[i*phase1.LDUStatusInterval:(i+1)*phase1.LDUStatusInterval]...)
		bits = append(bits, 1, 0)
	}
	return framing.BitsToDibits(bits)
}

// TestComposerP25Phase1LogsQualityPerTransmission is the #1242 follow-up:
// with voice_call_grouping: conversation one voice chain carries every over
// on a talkgroup, and its decode-quality line is cumulative, so a 34 s call
// whose corrected bit errors climbed from ~3 to ~100 per LDU could not say
// whether one radio was bad or the receiver drifted. Two overs here — a
// clean radio, a terminator, then a radio whose every voice subframe carries
// a bit error — must give one transmission line each, naming the radio, with
// the errors on the second only. A third over whose terminator was lost
// (straight from radio B to radio C) must still be split, on the source
// change.
func TestComposerP25Phase1LogsQualityPerTransmission(t *testing.T) {
	const (
		sampleRate = 48_000.0
		deviation  = 1800.0
		tg         = 301
		srcA       = uint32(0x0A0001)
		srcB       = uint32(0x0B0002)
		srcC       = uint32(0x0C0003)
		perOver    = 8
	)
	lead := make([]uint8, 400)
	for i := range lead {
		lead[i] = uint8(i % 4)
	}
	frame := 0
	dibits := append([]uint8(nil), lead...)
	dibits = append(dibits, p25p1OverDibits(t, perOver, tg, srcA, 0, &frame)...)
	dibits = append(dibits, p25p1TDUDibits()...)
	dibits = append(dibits, p25p1OverDibits(t, perOver, tg, srcB, 1, &frame)...)
	// No terminator: radio C keys straight after B.
	dibits = append(dibits, p25p1OverDibits(t, perOver, tg, srcC, 0, &frame)...)
	iq := demod.ModulateP25C4FM(dibits, sampleRate, deviation)
	// Filler so the last LDU's window completes; C's transmission is then
	// closed by the end of the call (the chain-end path), not a terminator.
	iq = append(iq, demod.ModulateP25C4FM(append(append([]uint8(nil), lead...), make([]uint8, 3000)...), sampleRate, deviation)...)

	buf := &syncBuffer{}
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	src := newFakeSource()
	bus := events.NewBus(64)
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
			Grant:        trunking.Grant{System: "P25P1Site", Protocol: "p25", GroupID: tg, FrequencyHz: 450_200_000},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})
	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	lineRe := regexp.MustCompile(`msg="composer: p25p1 transmission quality" serial=VOICE-1 tx=(\d+) src=(\d+) ldus=(\d+) corrected_bit_errs=(\d+) errs_per_ldu=([\d.]+)`)
	waitFor(t, 6*time.Second, func() bool { return len(lineRe.FindAllString(buf.String(), -1)) >= 2 })
	// Let the chain finish radio C's IQ, then end the call.
	time.Sleep(1500 * time.Millisecond)
	publishEnd(bus, "VOICE-1")
	waitFor(t, 6*time.Second, func() bool { return len(lineRe.FindAllString(buf.String(), -1)) >= 3 })
	out := buf.String()
	m := lineRe.FindAllStringSubmatch(out, -1)
	if len(m) != 3 {
		t.Fatalf("got %d transmission lines, want 3:\n%s", len(m), out)
	}
	t.Logf("transmission lines:\n%s", strings.Join(lineRe.FindAllString(out, -1), "\n"))
	perLDU := make([]float64, 3)
	for i, wantSrc := range []uint32{srcA, srcB, srcC} {
		got := m[i]
		if got[1] != strconv.Itoa(i+1) || got[2] != strconv.FormatUint(uint64(wantSrc), 10) {
			t.Errorf("line %d = tx %s src %s, want tx %d src %d", i, got[1], got[2], i+1, wantSrc)
		}
		// The receiver loses an LDU or two of a clean synthetic stream (its
		// acquisition, and pattern-dependent slips on this fixture); the
		// split, not the yield, is under test.
		if n, _ := strconv.Atoi(got[3]); n < perOver-4 || n > perOver {
			t.Errorf("tx %d: ldus = %d, want ~%d", i+1, n, perOver)
		}
		perLDU[i], _ = strconv.ParseFloat(got[5], 64)
	}
	// Radio B's one corrected bit per subframe is 9 per LDU on top of
	// whatever the clean radios show (the receiver's acquisition costs the
	// first over a few). It must stand out from both of its neighbours.
	if perLDU[1] < perLDU[0]+5 || perLDU[1] < perLDU[2]+5 {
		t.Errorf("errs_per_ldu A/B/C = %v; radio B's errors are not attributed to B", perLDU)
	}
}
