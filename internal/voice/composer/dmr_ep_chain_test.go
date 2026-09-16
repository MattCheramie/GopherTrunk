package composer

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/MattCheramie/GopherTrunk/internal/voice/cryptocap"
)

// buildPIHeaderBurst frames a Privacy Indicator header as a data burst
// (BPTC(196,96), DTPIHeader slot type, BS-Data sync).
func buildPIHeaderBurst(t *testing.T, h dmr.PIHeader, colorCode uint8) []uint8 {
	t.Helper()
	info := dmr.AssemblePIHeader(h)
	bits := make([]byte, 96)
	for i := 0; i < 96; i++ {
		bits[i] = (info[i>>3] >> uint(7-(i&7))) & 1
	}
	payloadDibits := framing.BitsToDibits(framing.EncodeBPTC196_96(bits))
	slotDibits := framing.BitsToDibits(dmr.AssembleSlotType(dmr.SlotType{ColorCode: colorCode, DataType: dmr.DTPIHeader}))
	burst := make([]uint8, 0, dmr.BurstDibits)
	burst = append(burst, payloadDibits[:dmr.HalfPayloadDibits]...)
	burst = append(burst, slotDibits[:dmr.SlotTypeDibits]...)
	burst = append(burst, dmr.BSData.Dibits[:]...)
	burst = append(burst, slotDibits[dmr.SlotTypeDibits:]...)
	burst = append(burst, payloadDibits[dmr.HalfPayloadDibits:]...)
	return burst
}

// buildEnhancedPrivacyStream models an Enhanced Privacy transmission the
// way a radio sends it: a clock-settling lead, three PI header bursts
// (algorithm RC4, key id keyID, Message Indicator mi0), then n voice
// superframes whose AMBE+2 payloads are RC4-scrambled with key‖MI (the MI
// advancing per superframe) and whose on-air frames carry the embedded IV
// nibbles. Returns the dibits and the CLEAR 49-bit payloads in order.
func buildEnhancedPrivacyStream(t *testing.T, key []byte, keyID uint8, mi0 uint32, n int) (dibits []uint8, clear [][]byte) {
	t.Helper()
	dibits = make([]uint8, 240)
	for i := range dibits {
		dibits[i] = uint8(i % 4)
	}
	h := dmr.PIHeader{AlgID: dmr.PIAlgRC4, FID: dmr.PIFIDDMRA, KeyID: keyID, DstAddr: 7}
	h.MI = [4]byte{byte(mi0 >> 24), byte(mi0 >> 16), byte(mi0 >> 8), byte(mi0)}
	for i := 0; i < 3; i++ {
		dibits = append(dibits, buildPIHeaderBurst(t, h, 1)...)
	}
	mi := mi0
	for s := 0; s < n; s++ {
		scrambled := make([][]byte, dmrvoice.FramesPerSuperframe)
		for k := range scrambled {
			info := mkInfo(s*dmrvoice.FramesPerSuperframe + k)
			clear = append(clear, info)
			scrambled[k] = append([]byte(nil), info...)
		}
		// XOR is its own inverse: the descrambler scrambles the fixture.
		if _, _, err := dmrvoice.DescrambleSuperframe(key, mi, scrambled); err != nil {
			t.Fatal(err)
		}
		var onair [dmrvoice.FramesPerSuperframe][]byte
		for k := range scrambled {
			f, err := dmrvoice.EncodeAMBEFrame(scrambled[k])
			if err != nil {
				t.Fatal(err)
			}
			onair[k] = f
		}
		// After encryption, as a radio does — and the IV a superframe carries
		// is the NEXT superframe's MI (capture-pinned, dmrvoice ep_test.go).
		dmrvoice.EmbedIV(onair, dmrvoice.AdvanceMI(mi))
		for b := 0; b < dmrvoice.BurstsPerSuperframe; b++ {
			sync := dmr.BSData.Dibits
			if b == 0 {
				sync = dmr.BSVoice.Dibits
			}
			base := b * dmrvoice.FramesPerBurst
			dibits = append(dibits, voiceBurstDibits(onair[base:base+dmrvoice.FramesPerBurst], sync)...)
		}
		mi = dmrvoice.AdvanceMI(mi)
	}
	return dibits, clear
}

// payloadMatchesClear compares a recorded 7-byte raw frame with a clear
// 49-bit payload on bits 0..44 only: bits 45..48 (C3[0..3]) are overwritten
// on air by the embedded-IV nibble, exactly as on a real radio.
func payloadMatchesClear(raw []byte, clear []byte) bool {
	p := packBits(clear)
	return len(raw) == 7 && bytes.Equal(raw[:5], p[:5]) && raw[5]&0xF8 == p[5]&0xF8
}

func anySuperframeMatchesClear(got [][]byte, clear [][]byte) bool {
	const sf = dmrvoice.FramesPerSuperframe
	for in := 0; in+sf <= len(clear); in += sf {
		for g := 0; g+sf <= len(got); g++ {
			ok := true
			for k := 0; k < sf; k++ {
				if !payloadMatchesClear(got[g+k], clear[in+k]) {
					ok = false
					break
				}
			}
			if ok {
				return true
			}
		}
	}
	return false
}

type captureSink struct {
	mu     sync.Mutex
	frames []cryptocap.Frame
}

func (s *captureSink) WriteCryptoFrame(f cryptocap.Frame) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.frames = append(s.frames, f)
}

func (s *captureSink) all() []cryptocap.Frame {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]cryptocap.Frame(nil), s.frames...)
}

// runEPChain drives the composer DMR chain over an Enhanced Privacy stream
// and returns the recorder's raw frames, the crypto-capture records and the
// call.encryption events it published.
func runEPChain(t *testing.T, dibits []uint8, resolver KeyResolver) (raw [][]byte, captured []cryptocap.Frame, encEvents []trunking.CallEncryption) {
	t.Helper()
	const (
		sampleRate = 48_000.0
		sps        = 10
		span       = 8
		alpha      = 0.20
		deviation  = 1944.0
	)
	iq := demod.ModulateC4FM(dibits, sps, span, alpha, sampleRate, deviation)

	src := newFakeSource()
	bus := events.NewBus(64)
	sub := bus.Subscribe()
	defer sub.Close()
	sink := &recordingSink{}
	cap := &captureSink{}
	eng := &fakeEngine{}
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-1": src}},
		Sink:          sink,
		Engine:        eng,
		IQSampleRate:  uint32(sampleRate),
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
		CryptoSink:    cap,
		KeyResolver:   resolver,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	defer c.Close()
	defer bus.Close()

	var evMu sync.Mutex
	go func() {
		for ev := range sub.C {
			if ev.Kind == events.KindCallEncryption {
				if ce, ok := ev.Payload.(trunking.CallEncryption); ok {
					evMu.Lock()
					encEvents = append(encEvents, ce)
					evMu.Unlock()
				}
			}
		}
	}()

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant: trunking.Grant{
				System: "DMRSite", Protocol: "dmr-tier3",
				GroupID: 7, FrequencyHz: 460_000_000, Encrypted: true,
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})
	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)
	waitFor(t, 10*time.Second, func() bool {
		return len(sink.rawFrames("VOICE-1")) >= 6*dmrvoice.FramesPerSuperframe
	})
	evMu.Lock()
	defer evMu.Unlock()
	return sink.rawFrames("VOICE-1"), cap.all(), append([]trunking.CallEncryption(nil), encEvents...)
}

// TestComposerDMREnhancedPrivacyDecryptsWithConfiguredKey is the issue #1187
// chain test: an RC4 "Enhanced Privacy" transmission (PI header + scrambled
// voice with embedded IVs) recorded through the full composer path with the
// matching key configured must reach the recorder as CLEAR AMBE+2 payloads.
// Before the descramble landed the .raw sidecar held ciphertext (the
// no-key test below is that behaviour, still correct when no key matches).
func TestComposerDMREnhancedPrivacyDecryptsWithConfiguredKey(t *testing.T) {
	key := []byte{0x0F, 0x1E, 0x2D, 0x3C, 0x4B}
	dibits, clear := buildEnhancedPrivacyStream(t, key, 7, 0xDEADBEEF, 12)
	resolver := func(system, algorithm string, keyID uint16) ([]byte, bool) {
		if system == "DMRSite" && algorithm == "rc4" && keyID == 7 {
			return key, true
		}
		return nil, false
	}
	raw, captured, evs := runEPChain(t, dibits, resolver)
	if !anySuperframeMatchesClear(raw, clear) {
		t.Fatalf("no recorded superframe matched the clear payloads — descramble not applied (%d raw frames)", len(raw))
	}
	if len(evs) == 0 {
		t.Fatalf("no call.encryption event published for the PI header")
	}
	if ev := evs[0]; ev.Protocol != "dmr" || ev.AlgorithmID != dmr.PIAlgRC4 || ev.KeyID != 7 ||
		ev.MessageIndicator != [9]byte{0xDE, 0xAD, 0xBE, 0xEF} {
		t.Fatalf("call.encryption = %+v", ev)
	}
	// The crypto-frame bridge sees the CIPHERTEXT with each superframe's MI,
	// so offline analysis works whether or not a key was configured.
	if len(captured) == 0 {
		t.Fatalf("no crypto-capture records")
	}
	var sawFirst bool
	for _, f := range captured {
		if f.Protocol != "dmr" || f.AlgID != dmr.PIAlgRC4 || f.KeyID != 7 || len(f.CT) != dmrvoice.EPSuperframeBytes || len(f.MI) != 4 {
			t.Fatalf("capture record = %+v", f)
		}
		if bytes.Equal(f.MI, []byte{0xDE, 0xAD, 0xBE, 0xEF}) {
			sawFirst = true
		}
	}
	if !sawFirst {
		t.Fatalf("no capture carried the PI header's MI; MIs: %x", func() [][]byte {
			var m [][]byte
			for _, f := range captured {
				m = append(m, f.MI)
			}
			return m
		}())
	}
}

// Without a configured key the frames must reach the recorder untouched
// (ciphertext) — never a guessed key — while the PI header still surfaces
// the algorithm and key id.
func TestComposerDMREnhancedPrivacyRecordsCiphertextWithoutKey(t *testing.T) {
	key := []byte{0x0F, 0x1E, 0x2D, 0x3C, 0x4B}
	dibits, clear := buildEnhancedPrivacyStream(t, key, 7, 0x0BADF00D, 12)
	raw, _, evs := runEPChain(t, dibits, nil)
	if anySuperframeMatchesClear(raw, clear) {
		t.Fatalf("recorded clear payloads with no key configured")
	}
	if len(evs) == 0 || evs[0].KeyID != 7 || evs[0].AlgorithmID != dmr.PIAlgRC4 {
		t.Fatalf("call.encryption events = %+v", evs)
	}
}

// A key configured under a DIFFERENT key id must not be applied.
func TestComposerDMREnhancedPrivacyIgnoresWrongKeyID(t *testing.T) {
	key := []byte{0x0F, 0x1E, 0x2D, 0x3C, 0x4B}
	dibits, clear := buildEnhancedPrivacyStream(t, key, 7, 0x12345678, 12)
	resolver := func(system, algorithm string, keyID uint16) ([]byte, bool) {
		if keyID == 8 {
			return key, true
		}
		return nil, false
	}
	raw, _, _ := runEPChain(t, dibits, resolver)
	if anySuperframeMatchesClear(raw, clear) {
		t.Fatalf("key for id 8 was applied to a call announcing key id 7")
	}
}
