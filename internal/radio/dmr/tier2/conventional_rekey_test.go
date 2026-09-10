package tier2

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// idleCarrier is n two-slot cadences (CACH + burst per slot) of a keyed but
// idle repeater: both slots carry the ETSI Idle burst at colour code cc. At
// 288 dibits per cadence, 17 cadences ≈ 1 s of air.
func idleCarrier(cc uint8, n int) []uint8 {
	cach := make([]uint8, 12)
	idle := realIdleBurstDibits(cc)
	var s []uint8
	for i := 0; i < n; i++ {
		s = append(s, cach...)
		s = append(s, idle...)
		s = append(s, cach...)
		s = append(s, idle...)
	}
	return s
}

// realIdleBurstDibits is an Idle burst carrying the ETSI idle information
// block (IdleInfoPattern) — what a Motorola/Hytera IPSC repeater actually
// transmits while keyed and idle — as opposed to the zero-block idleBurstDibits
// the older fixtures use as neutral filler.
func realIdleBurstDibits(cc uint8) []uint8 {
	b := burstWithInfo96(IdleInfoPattern[:])
	stampSlotType(b, cc, dmr.DTIdle)
	return b.Dibits[:]
}

func countKinds(evs []events.Event) (grants, releases int) {
	for _, ev := range evs {
		switch ev.Kind {
		case events.KindGrant:
			grants++
		case events.KindCallRelease:
			releases++
		}
	}
	return
}

// TestConventionalRekeyWithinHangtimeRegrants pins the 10 Sep IPSC field
// report ("some calls are completely missed by GT while the radios decode
// them"): the operator's own log shows the composer's voice-path terminator
// detector releasing each call at PTT release while the control path's copy
// of the same Terminator-with-LC decoded 0.8–6.5 s later (or not at all) on
// its weaker channelizer tap. Until the control path saw a terminator it kept
// the call tracked, so a reply's Voice LC Header for the same talkgroup and
// radio was "deduped" against a call the engine had already ended — and the
// reply was never granted. A header that lands well past the tracked call's
// last header/LC is a new transmission: it must release the stale call and
// grant again. Fails against the pre-fix dedupe (1 grant, 0 rekeys).
func TestConventionalRekeyWithinHangtimeRegrants(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	c := New(Options{Bus: bus, SystemName: "ipsc", FrequencyHz: 442387500, InterleavedVoice: true})

	lc := groupLC(11, 199)
	var stream []uint8
	// Over 1: headers + voice, terminator LOST (0 copies decoded here).
	stream = append(stream, outboundTransmission(t, lc, 12, 3, 3, 0)...)
	// Repeater hang time: ~2 s of idle beacon on both slots.
	stream = append(stream, idleCarrier(12, 34)...)
	// Over 2 (the reply, same radio, same talkgroup): headers + voice + terminator.
	stream = append(stream, outboundTransmission(t, lc, 12, 3, 3, 3)...)
	feedChunks(c, stream, 1000)

	evs := drainEvents(sub, 100*time.Millisecond)
	grants, releases := countKinds(evs)
	if grants != 2 {
		t.Fatalf("got %d grants, want 2 (the re-key within hangtime must be granted); rekeys=%d",
			grants, c.Counters().Rekeys)
	}
	if releases != 2 {
		t.Errorf("got %d call.release events, want 2 (stale call released on re-key, then the terminator)", releases)
	}
	if got := c.Counters().Rekeys; got != 1 {
		t.Errorf("Rekeys = %d, want 1", got)
	}
	if got := c.Counters().LateEntries; got != 0 {
		t.Errorf("LateEntries = %d, want 0 (the re-key was header-granted)", got)
	}
}

// TestConventionalRekeyByLateEntryAfterSilence covers the harder case: the
// previous over's terminator AND the reply's headers were both lost. The
// reply's embedded LC names the still-tracked (talkgroup, radio) after a
// silence gap longer than any mid-transmission fade, so it re-grants by late
// entry instead of being absorbed as liveness of the dead call.
func TestConventionalRekeyByLateEntryAfterSilence(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	c := New(Options{Bus: bus, SystemName: "ipsc", FrequencyHz: 442387500, InterleavedVoice: true})

	lc := groupLC(11, 199)
	var stream []uint8
	stream = append(stream, outboundTransmission(t, lc, 12, 3, 3, 0)...)
	stream = append(stream, idleCarrier(12, 34)...)
	stream = append(stream, outboundTransmission(t, lc, 12, 0, 3, 3)...)
	feedChunks(c, stream, 1000)

	grants, _ := countKinds(drainEvents(sub, 100*time.Millisecond))
	if grants != 2 {
		t.Fatalf("got %d grants, want 2 (headerless re-key after silence must late-enter); rekeys=%d",
			grants, c.Counters().Rekeys)
	}
	if got := c.Counters().Rekeys; got != 1 {
		t.Errorf("Rekeys = %d, want 1", got)
	}
	if got := c.Counters().LateEntries; got != 1 {
		t.Errorf("LateEntries = %d, want 1", got)
	}
}

// TestConventionalFadeWithinTransmissionDoesNotRekey is the no-harm bound:
// a short mid-over dropout (two superframes lost, well inside
// superframeRekeyGapDibits) must not split one transmission into two calls.
func TestConventionalFadeWithinTransmissionDoesNotRekey(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	c := New(Options{Bus: bus, SystemName: "ipsc", FrequencyHz: 442387500, InterleavedVoice: true})

	lc := groupLC(11, 199)
	var stream []uint8
	stream = append(stream, outboundTransmission(t, lc, 12, 3, 2, 0)...)
	// ~0.5 s of nothing decodable (a fade), then the same over continues
	// headerless and ends normally.
	stream = append(stream, idleCarrier(12, 4)...)
	stream = append(stream, outboundTransmission(t, lc, 12, 0, 3, 3)...)
	feedChunks(c, stream, 1000)

	grants, _ := countKinds(drainEvents(sub, 100*time.Millisecond))
	if grants != 1 {
		t.Fatalf("got %d grants, want 1 (a short fade must not re-key)", grants)
	}
	if got := c.Counters().Rekeys; got != 0 {
		t.Errorf("Rekeys = %d, want 0", got)
	}
}

// TestConventionalIdleBeaconMarksSiteAliveAndLocks pins the "camp / idle
// beacon" request (10 Sep): the ETSI Idle burst a keyed-but-idle IPSC
// repeater fills both slots with counts as a site-alive beacon and locks the
// channel (site camped) — before the DTIdle handler Idle fell through
// IngestBurst's switch and the channel stayed idle/unlocked until someone
// spoke. An Idle whose information block is not the ETSI pattern is neither
// (it is logged as an unknown vendor variant), so a Golay-only slot-type
// decode cannot forge a beacon.
func TestConventionalIdleBeaconMarksSiteAliveAndLocks(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	c := New(Options{Bus: bus, SystemName: "ipsc", FrequencyHz: 442387500, InterleavedVoice: true})

	feedChunks(c, idleCarrier(12, 4), 1000)
	if got := c.Counters().Beacons; got == 0 {
		t.Fatalf("Beacons = 0 after an idle-beacon train, want > 0")
	}
	var locked bool
	for _, ev := range drainEvents(sub, 100*time.Millisecond) {
		switch ev.Kind {
		case events.KindCCLocked:
			locked = true
		case events.KindGrant, events.KindDecodeError:
			t.Fatalf("idle beacon wrongly published %v", ev.Kind)
		}
	}
	if !locked {
		t.Error("idle beacon did not declare the channel locked (site alive)")
	}

	// A zero-block Idle (BPTC-clean, wrong content) is not a beacon.
	c2 := New(Options{Bus: bus, SystemName: "ipsc", FrequencyHz: 442387500})
	c2.IngestBurst(burstWithInfo96(make([]byte, 12)), dmr.SlotType{ColorCode: 12, DataType: dmr.DTIdle})
	if got := c2.Counters().Beacons; got != 0 {
		t.Errorf("Beacons = %d for an unknown idle block, want 0", got)
	}
}

// TestIdleInfoPatternMatchesMMDVMHostConstant cross-checks IdleInfoPattern
// against an INDEPENDENT reference: MMDVMHost's DMR_IDLE_DATA (DMRDefines.h)
// is the on-air Idle burst as 33 raw bytes (payload half, slot type, BS-Data
// sync, slot type, payload half; colour code 0). BPTC-decoding its payload with GT's own
// decoder must yield the pattern with zero corrections — a literal-vector
// pin, so the pattern can never drift with the decoder (the #764/#771
// self-consistent-synthetic trap).
func TestIdleInfoPatternMatchesMMDVMHostConstant(t *testing.T) {
	mmdvmIdle := []byte{
		0x53, 0xC2, 0x5E, 0xAB, 0xA8, 0x67, 0x1D, 0xC7, 0x38, 0x3B, 0xD9,
		0x36, 0x00, 0x0D, 0xFF, 0x57, 0xD7, 0x5D, 0xF5, 0xD0, 0x03, 0xF6,
		0xE4, 0x65, 0x17, 0x1B, 0x48, 0xCA, 0x6D, 0x4F, 0xC6, 0x10, 0xB4,
	}
	if len(mmdvmIdle) != 33 {
		t.Fatalf("reference burst is %d bytes, want 33 (264 bits = 132 dibits)", len(mmdvmIdle))
	}
	var b dmr.Burst
	for i := 0; i < dmr.BurstDibits; i++ {
		b.Dibits[i] = (mmdvmIdle[i/4] >> uint(6-2*(i%4))) & 3
	}
	// The constant is a template: MMDVMHost stamps the colour code + Idle
	// slot type into it per repeater at transmit time (CDMRSlotType), so
	// only the BPTC-coded information block is compared here.
	bits, errs := framing.DecodeBPTC196_96(b.PayloadBits())
	if errs != 0 {
		t.Fatalf("reference burst BPTC corrections = %d, want 0", errs)
	}
	got := infoBitsToBytes(bits)
	if string(got) != string(IdleInfoPattern[:]) {
		t.Fatalf("MMDVMHost idle block = %x, IdleInfoPattern = %x", got, IdleInfoPattern[:])
	}
}
