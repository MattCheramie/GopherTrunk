package tier2

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier3"
	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// --- synthetic outbound-carrier builders (mirror voice/superframe_test.go) ---

// lcFragmentsFor encodes a Full LC into the four 32-bit embedded fragments
// bursts B–E carry.
func lcFragmentsFor(t *testing.T, f dmr.FLC) [4][]byte {
	t.Helper()
	info := dmr.AssembleFLC(f)
	lc := make([]byte, framing.EmbLCBits)
	for i := 0; i < framing.EmbLCBits; i++ {
		lc[i] = (info[i/8] >> uint(7-(i%8))) & 1
	}
	ch := framing.EncodeEmbeddedLC(lc)
	var frags [4][]byte
	for i := range frags {
		frags[i] = ch[i*framing.EmbeddedFragmentBits : (i+1)*framing.EmbeddedFragmentBits]
	}
	return frags
}

func embeddedSyncField(emb dmr.EMB, frag []byte) [24]uint8 {
	field := dmr.AssembleEmbeddedField(emb, frag)
	var d [24]uint8
	for i := 0; i < 24; i++ {
		d[i] = field[2*i]<<1 | field[2*i+1]
	}
	return d
}

func lcssFor(b int) dmr.LCSS {
	switch b {
	case 1:
		return dmr.LCSSFirst
	case 4:
		return dmr.LCSSLast
	default:
		return dmr.LCSSCont
	}
}

// voiceBurstDibits assembles a 132-dibit voice burst: 108 bits + 24-dibit
// sync/EMB field + 108 bits, with deterministic (non-codeword) AMBE bits.
func voiceBurstDibits(seed int, sync [24]uint8) []uint8 {
	bits := make([]byte, 216)
	for i := range bits {
		bits[i] = byte((seed*3 + i*5) & 1)
	}
	toDibits := func(b []byte) []uint8 {
		d := make([]uint8, len(b)/2)
		for i := range d {
			d[i] = b[2*i]<<1 | b[2*i+1]
		}
		return d
	}
	out := make([]uint8, 0, dmr.BurstDibits)
	out = append(out, toDibits(bits[:108])...)
	out = append(out, sync[:]...)
	out = append(out, toDibits(bits[108:])...)
	return out
}

// idleBurstDibits is a BS-Data-synced burst whose slot type says Idle at
// colour code cc — what the unused timeslot of a keyed repeater carries.
func idleBurstDibits(cc uint8) []uint8 {
	b := burstWithInfo96(make([]byte, 12))
	stampSlotType(b, cc, dmr.DTIdle)
	return b.Dibits[:]
}

// stampSlotType writes the BS-Data sync + a (cc, dt) slot type into b.
func stampSlotType(b *dmr.Burst, cc uint8, dt dmr.DataType) {
	copy(b.Dibits[dmr.HalfPayloadDibits+dmr.SlotTypeDibits:], dmr.BSData.Dibits[:])
	st := dmr.AssembleSlotType(dmr.SlotType{ColorCode: cc, DataType: dt})
	// 20 bits: 10 before the sync, 10 after.
	for i := 0; i < 5; i++ {
		b.Dibits[dmr.HalfPayloadDibits+i] = (st[2*i] << 1) | st[2*i+1]
		b.Dibits[dmr.HalfPayloadDibits+dmr.SlotTypeDibits+dmr.SyncDibits+i] = (st[10+2*i] << 1) | st[10+2*i+1]
	}
}

// outboundTransmission weaves one slot's transmission onto a 2-slot outbound
// carrier with a 12-dibit CACH before every burst (the 288-dibit same-slot
// cadence of real air): headers Voice-LC-Header bursts, then superframes
// voice superframes carrying lc in their embedded signalling, then
// terminators Terminator-with-LC bursts. The other slot idles throughout.
func outboundTransmission(t *testing.T, lc dmr.FLC, cc uint8, headers, superframes, terminators int) []uint8 {
	t.Helper()
	frags := lcFragmentsFor(t, lc)
	cach := make([]uint8, 12)
	var s []uint8
	weave := func(slotA []uint8) {
		s = append(s, cach...)
		s = append(s, slotA...)
		s = append(s, cach...)
		s = append(s, idleBurstDibits(cc)...)
	}
	for i := 0; i < headers; i++ {
		b := burstWithFLC(lc)
		stampSlotType(b, cc, dmr.DTVoiceLCHeader)
		weave(b.Dibits[:])
	}
	seed := 0
	for sf := 0; sf < superframes; sf++ {
		for b := 0; b < dmrvoice.BurstsPerSuperframe; b++ {
			sync := dmr.BSVoice.Dibits
			if b >= 1 && b <= 4 {
				sync = embeddedSyncField(dmr.EMB{ColorCode: cc, LCSS: lcssFor(b)}, frags[b-1])
			} else if b == 5 {
				sync = embeddedSyncField(dmr.EMB{ColorCode: cc, LCSS: dmr.LCSSSingle}, make([]byte, 32))
			}
			weave(voiceBurstDibits(seed, sync))
			seed++
		}
	}
	for i := 0; i < terminators; i++ {
		b := burstWithTerminatorFLC(lc)
		stampSlotType(b, cc, dmr.DTTerminatorWithLC)
		weave(b.Dibits[:])
	}
	// Trailing idle so the interleaved decoder has a full span buffered
	// behind the last superframe anchor (a live carrier is continuous).
	for i := 0; i < 8; i++ {
		weave(idleBurstDibits(cc))
	}
	return s
}

func feedChunks(c *ConventionalChannel, stream []uint8, chunk int) {
	idx := 0
	for idx < len(stream) {
		e := idx + chunk
		if e > len(stream) {
			e = len(stream)
		}
		c.Process(stream[idx:e], idx)
		idx = e
	}
}

func groupLC(tg, src uint32) dmr.FLC {
	return dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: tg, SrcAddr: src}
}

// TestConventionalLateEntryGrantsHeaderlessTransmission pins DMR late entry
// (field report: "GT said the call ended but the conversation continued for
// 20 s" — the next transmissions' Voice LC Headers were lost to a fade on a
// −60 dBFS tap, and the conventional path had no other way to grant). A
// transmission with ZERO decodable headers must still be granted from its
// embedded Link Control, and the terminator must still release it. Before
// late entry this produced no grant at all.
func TestConventionalLateEntryGrantsHeaderlessTransmission(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	c := New(Options{Bus: bus, SystemName: "ipsc", FrequencyHz: 442387500, InterleavedVoice: true})

	stream := outboundTransmission(t, groupLC(11, 199), 12, 0, 4, 3)
	feedChunks(c, stream, 1000)

	evs := drainEvents(sub, 100*time.Millisecond)
	grants := grantsOf(evs)
	if len(grants) != 1 {
		t.Fatalf("late entry: got %d grants, want exactly 1 (%+v)", len(grants), grants)
	}
	g := grants[0]
	if g.GroupID != 11 || g.SourceID != 199 || g.Protocol != "dmr-tier2" || !g.DMRInterleavedVoice {
		t.Errorf("late-entry grant = %+v, want tg=11 src=199 dmr-tier2 interleaved", g)
	}
	if g.ChannelID != 12 {
		t.Errorf("late-entry grant colour code = %d, want 12 (majority EMB CC)", g.ChannelID)
	}
	if c.Counters().LateEntries != 1 {
		t.Errorf("LateEntries = %d, want 1", c.Counters().LateEntries)
	}
	var released, locked bool
	for _, ev := range evs {
		switch ev.Kind {
		case events.KindCallRelease:
			released = true
		case events.KindCCLocked:
			locked = true
		}
	}
	if !released {
		t.Error("terminator after a late-entered call did not publish call.release")
	}
	if !locked {
		t.Error("late entry did not declare the channel locked")
	}
}

// TestConventionalLateEntryNeedsTwoAgreeingLCs pins the confirm-twice gate: a
// single embedded LC (one superframe) is not enough to grant, so a lone
// miscorrected LC cannot forge a phantom call.
func TestConventionalLateEntryNeedsTwoAgreeingLCs(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	c := New(Options{Bus: bus, SystemName: "ipsc", FrequencyHz: 1, InterleavedVoice: true})

	feedChunks(c, outboundTransmission(t, groupLC(11, 199), 12, 0, 1, 0), 1000)
	if n := len(grantsOf(drainEvents(sub, 50*time.Millisecond))); n != 0 {
		t.Fatalf("one embedded LC granted %d call(s); want 0 until a second agreeing LC", n)
	}
}

// TestConventionalHeaderAndEmbeddedLCGrantOnce pins no-harm on the normal
// path: a transmission whose headers decode is granted exactly once — the
// following superframes' embedded LCs refresh the tracked call instead of
// re-granting — and LateEntries stays 0.
func TestConventionalHeaderAndEmbeddedLCGrantOnce(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	c := New(Options{Bus: bus, SystemName: "ipsc", FrequencyHz: 1, InterleavedVoice: true})

	feedChunks(c, outboundTransmission(t, groupLC(11, 199), 12, 4, 4, 3), 1000)
	grants := grantsOf(drainEvents(sub, 100*time.Millisecond))
	if len(grants) != 1 {
		t.Fatalf("header-granted transmission produced %d grants, want 1", len(grants))
	}
	if c.Counters().LateEntries != 0 {
		t.Errorf("LateEntries = %d on a header-granted call, want 0", c.Counters().LateEntries)
	}
	if c.Counters().FECPass != 4 {
		t.Errorf("FECPass = %d, want 4 (one per header burst)", c.Counters().FECPass)
	}
}

// TestConventionalLateEntryHonoursColorCodeFilter pins that the IPSC
// colour-code filter still applies to a header-less transmission: the
// majority EMB colour code of the LC-bearing bursts is what gets filtered.
func TestConventionalLateEntryHonoursColorCodeFilter(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	want := uint8(12)
	c := New(Options{Bus: bus, SystemName: "ipsc", FrequencyHz: 1, InterleavedVoice: true, ColorCodeFilter: &want})

	feedChunks(c, outboundTransmission(t, groupLC(11, 199), 7, 0, 4, 0), 1000)
	if n := len(grantsOf(drainEvents(sub, 50*time.Millisecond))); n != 0 {
		t.Fatalf("colour-code 7 transmission late-entered %d call(s) through a cc=12 filter", n)
	}
	if c.Counters().DroppedOffCC == 0 {
		t.Error("off-colour embedded LCs were not counted as DroppedOffCC")
	}
}

// TestConventionalCSBKFailureLogIsParked pins the log parking for the
// field-reported "CSBK CRC mismatch" storm: a keyed idle IPSC repeater put
// one identical Debug line on the log every 30 ms. Now the first failure
// logs once (with the decoded header + info hex for diagnosis), unchanged
// repeats within csbkFailLogInterval are suppressed and summarised, and the
// counter keeps the exact count.
func TestConventionalCSBKFailureLogIsParked(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	bus := events.NewBus(8)
	defer bus.Close()
	clock := time.Unix(1_700_000_000, 0)
	c := New(Options{Bus: bus, Log: log, SystemName: "ipsc", FrequencyHz: 1,
		Now: func() time.Time { return clock }})

	// A CSBK-shaped block whose CRC does not check (bytes from a Motorola-FID
	// looking header, trailer deliberately wrong).
	info := tier3.AssembleCSBK(tier3.CSBK{LB: true, Opcode: 0x3E, FID: 0x10})
	info[11] ^= 0xFF
	b := burstWithInfo96(info)
	slot := dmr.SlotType{ColorCode: 7, DataType: dmr.DTCSBK}
	for i := 0; i < 100; i++ {
		c.IngestBurst(b, slot)
		clock = clock.Add(30 * time.Millisecond)
	}
	lines := strings.Count(buf.String(), "CSBK CRC mismatch")
	if lines != 1 {
		t.Fatalf("100 identical CSBK CRC failures in 3 s logged %d lines, want 1 (parked)\n%s", lines, buf.String())
	}
	if !strings.Contains(buf.String(), "csbko=0x3e") || !strings.Contains(buf.String(), "fid=0x10") || !strings.Contains(buf.String(), "info_hex=") {
		t.Errorf("parked CSBK failure line lacks the diagnostic header/info fields:\n%s", buf.String())
	}
	if got := c.Counters().CSBKCRCFail; got != 100 {
		t.Errorf("CSBKCRCFail = %d, want 100", got)
	}
	// Past the interval the summary carries the suppressed count.
	clock = clock.Add(csbkFailLogInterval)
	c.IngestBurst(b, slot)
	if !strings.Contains(buf.String(), "suppressed_repeats=99") {
		t.Errorf("summary line after the interval lacks suppressed_repeats=99:\n%s", buf.String())
	}
}

// TestConventionalVoiceBurstCannotForgeTerminator pins the root cause of the
// 9 Sep "false call ended" report. A voice burst A carries the BS-Voice sync
// with AMBE bits in the slot-type positions; when those bits happen to form a
// Golay(20,8) codeword reading Terminator-with-LC, the old slicer parsed it and
// the single-call fallback ("LC undecodable but only one call active") ended
// the live call mid-transmission — measured on the operator's capture as a
// terminator six bursts before the next voice superframe of the same over.
// Slot types must only be read from data-sync bursts, and an undecodable
// terminator must at least be a BPTC codeword to end a call.
func TestConventionalVoiceBurstCannotForgeTerminator(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	c := New(Options{Bus: bus, SystemName: "ipsc", FrequencyHz: 1, InterleavedVoice: true})

	lc := groupLC(11, 199)
	frags := lcFragmentsFor(t, lc)
	cach := make([]uint8, 12)
	var s []uint8
	weave := func(slotA []uint8) {
		s = append(s, cach...)
		s = append(s, slotA...)
		s = append(s, cach...)
		s = append(s, idleBurstDibits(12)...)
	}
	hdr := burstWithFLC(lc)
	stampSlotType(hdr, 12, dmr.DTVoiceLCHeader)
	weave(hdr.Dibits[:])
	// Forged terminator slot type: the AMBE bits around burst A's voice sync
	// happen to spell (cc=12, TerminatorWithLC).
	forged := dmr.AssembleSlotType(dmr.SlotType{ColorCode: 12, DataType: dmr.DTTerminatorWithLC})
	seed := 0
	for sf := 0; sf < 3; sf++ {
		for b := 0; b < dmrvoice.BurstsPerSuperframe; b++ {
			sync := dmr.BSVoice.Dibits
			if b >= 1 && b <= 4 {
				sync = embeddedSyncField(dmr.EMB{ColorCode: 12, LCSS: lcssFor(b)}, frags[b-1])
			} else if b == 5 {
				sync = embeddedSyncField(dmr.EMB{ColorCode: 12, LCSS: dmr.LCSSSingle}, make([]byte, 32))
			}
			burst := voiceBurstDibits(seed, sync)
			if b == 0 {
				for i := 0; i < 5; i++ {
					burst[dmr.HalfPayloadDibits+i] = (forged[2*i] << 1) | forged[2*i+1]
					burst[dmr.HalfPayloadDibits+dmr.SlotTypeDibits+dmr.SyncDibits+i] = (forged[10+2*i] << 1) | forged[10+2*i+1]
				}
			}
			weave(burst)
			seed++
		}
	}
	for i := 0; i < 8; i++ {
		weave(idleBurstDibits(12))
	}
	feedChunks(c, s, 1000)

	evs := drainEvents(sub, 100*time.Millisecond)
	if n := len(grantsOf(evs)); n != 1 {
		t.Fatalf("got %d grants, want 1", n)
	}
	for _, ev := range evs {
		if ev.Kind == events.KindCallRelease {
			t.Fatalf("a voice burst whose AMBE bits spell a Terminator slot type released the live call: %+v", ev.Payload)
		}
	}
	if len(c.calls) != 1 {
		t.Fatalf("call was dropped from tracking by the forged terminator")
	}
}

// TestConventionalUndecodableTerminatorNeedsBPTC pins the hardened fallback:
// with one active call, a burst whose slot type says Terminator but whose
// payload is not a BPTC codeword does not end the call; one whose BPTC
// decodes but RS mismatches (a weak real terminator) still does.
func TestConventionalUndecodableTerminatorNeedsBPTC(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	c := New(Options{Bus: bus, SystemName: "ipsc", FrequencyHz: 1})
	lc := groupLC(11, 199)
	c.IngestBurst(burstWithFLC(lc), dmr.SlotType{ColorCode: 12, DataType: dmr.DTVoiceLCHeader})

	var garbage dmr.Burst
	for i := range garbage.Dibits {
		garbage.Dibits[i] = uint8((i*7 + 3) & 3)
	}
	c.IngestBurst(&garbage, dmr.SlotType{ColorCode: 12, DataType: dmr.DTTerminatorWithLC})
	for _, ev := range drainEvents(sub, 50*time.Millisecond) {
		if ev.Kind == events.KindCallRelease {
			t.Fatalf("non-BPTC garbage with a Terminator slot type released the call")
		}
	}

	// BPTC-valid, RS-mismatching terminator (wrong seed): still releases.
	weak := burstWithFLC(lc) // Voice LC Header seed ≠ terminator seed ⇒ RS fails
	c.IngestBurst(weak, dmr.SlotType{ColorCode: 12, DataType: dmr.DTTerminatorWithLC})
	released := false
	for _, ev := range drainEvents(sub, 50*time.Millisecond) {
		if ev.Kind == events.KindCallRelease {
			released = true
		}
	}
	if !released {
		t.Fatal("BPTC-valid terminator with an RS mismatch did not release the single active call")
	}
}
