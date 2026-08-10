package tier2

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier3"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// burstWithFLC builds a Voice-LC-Header-shaped burst whose 96-bit
// info block is the supplied FLC followed by a properly computed
// RS(12,9) parity trailer XOR'd with the Voice LC Header seed —
// matching what a DMR transmitter would put on air.
func burstWithFLC(f dmr.FLC) *dmr.Burst {
	flcBytes := dmr.AssembleFLC(f)
	var data [9]byte
	copy(data[:], flcBytes)
	cw := framing.EncodeRS12_9(data)
	for i := 0; i < 3; i++ {
		cw[9+i] ^= framing.RS129SeedVoiceLCHeader[i]
	}
	info := cw[:]

	bits := make([]byte, 96)
	for i := 0; i < 96; i++ {
		bits[i] = (info[i>>3] >> uint(7-(i&7))) & 1
	}
	channel := framing.EncodeBPTC196_96(bits)

	var b dmr.Burst
	for i := 0; i < dmr.HalfPayloadDibits; i++ {
		b.Dibits[i] = (channel[2*i] << 1) | channel[2*i+1]
	}
	for i := 0; i < dmr.HalfPayloadDibits; i++ {
		b.Dibits[dmr.BurstDibits-dmr.HalfPayloadDibits+i] =
			(channel[2*(dmr.HalfPayloadDibits+i)] << 1) | channel[2*(dmr.HalfPayloadDibits+i)+1]
	}
	return &b
}

// burstWithInfo96 builds a DMR burst whose 196-bit BPTC channel payload
// carries the supplied 12-byte (96-bit) information block — the generic
// form of burstWithFLC, used to inject a CSBK beacon block.
func burstWithInfo96(info []byte) *dmr.Burst {
	if len(info) != 12 {
		panic("burstWithInfo96 requires a 12-byte info block")
	}
	bits := make([]byte, 96)
	for i := 0; i < 96; i++ {
		bits[i] = (info[i>>3] >> uint(7-(i&7))) & 1
	}
	channel := framing.EncodeBPTC196_96(bits)

	var b dmr.Burst
	for i := 0; i < dmr.HalfPayloadDibits; i++ {
		b.Dibits[i] = (channel[2*i] << 1) | channel[2*i+1]
	}
	for i := 0; i < dmr.HalfPayloadDibits; i++ {
		b.Dibits[dmr.BurstDibits-dmr.HalfPayloadDibits+i] =
			(channel[2*(dmr.HalfPayloadDibits+i)] << 1) | channel[2*(dmr.HalfPayloadDibits+i)+1]
	}
	return &b
}

// TestConventionalCSBKBeaconMarksSiteAlive pins issue #1036: a CRC-valid
// CSBK burst on a parked conventional/IPSC channel is recognised as a
// "site alive" beacon — the Beacons counter increments — instead of being
// ignored or surfaced as a decode error. Before the DTCSBK handler, CSBK
// bursts fell straight through IngestBurst's switch and Beacons stayed 0.
func TestConventionalCSBKBeaconMarksSiteAlive(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		SystemName:  "IPSCConv",
		FrequencyHz: 451_012_500,
		Now:         func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})

	// A Preamble CSBK is a canonical conventional/IPSC idle beacon.
	info := tier3.AssembleCSBK(tier3.CSBK{Opcode: tier3.OpPreamble, FID: 0x00})
	cc.IngestBurst(burstWithInfo96(info), dmr.SlotType{ColorCode: 3, DataType: dmr.DTCSBK})

	if got := cc.Counters().Beacons; got != 1 {
		t.Fatalf("Beacons = %d, want 1 (CRC-valid CSBK beacon should mark the site alive)", got)
	}
	if got := cc.Counters().FECFail; got != 0 {
		t.Errorf("FECFail = %d, want 0 (a valid beacon must not count as a FEC failure)", got)
	}
	// A beacon must not forge a grant or a decode error.
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				t.Fatalf("beacon wrongly published a grant: %+v", ev.Payload)
			}
			if ev.Kind == events.KindDecodeError {
				t.Fatalf("beacon wrongly published a decode error: %+v", ev.Payload)
			}
		default:
			return
		}
	}
}

// TestConventionalCSBKNoiseIsSilent pins the anti-spam half of #1036: a
// CSBK-shaped burst whose CRC does not check (between-beacon noise on a
// parked channel) must NOT increment Beacons and must NOT publish a
// decode error — surfacing those gaps as corrupt data is the reported
// symptom.
func TestConventionalCSBKNoiseIsSilent(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		SystemName:  "IPSCConv",
		FrequencyHz: 451_012_500,
		Now:         func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})

	// Valid CSBK bytes with the CRC trailer corrupted so ParseCSBK's
	// CRC check fails after a clean BPTC decode.
	info := tier3.AssembleCSBK(tier3.CSBK{Opcode: tier3.OpPreamble, FID: 0x00})
	info[11] ^= 0xFF // break the stored CRC
	cc.IngestBurst(burstWithInfo96(info), dmr.SlotType{ColorCode: 3, DataType: dmr.DTCSBK})

	if got := cc.Counters().Beacons; got != 0 {
		t.Fatalf("Beacons = %d, want 0 (a CRC-failing CSBK is noise, not a beacon)", got)
	}
	select {
	case ev := <-sub.C:
		if ev.Kind == events.KindDecodeError {
			t.Fatalf("between-beacon noise wrongly published a decode error: %+v", ev.Payload)
		}
	default:
	}
}

// TestConventionalColorCodeFilterDropsOtherCC pins the DMR IPSC /
// linked-repeater colour-code hard filter: a channel pinned to one colour
// code drops a FEC-valid Voice LC Header on any other colour code before it
// can grant, lock, or count as a FEC pass — the DroppedOffCC counter records
// it — while a header on the matching colour code decodes normally. Without
// the filter (the historical default) every colour code grants; this test
// fails against that behaviour and passes with it.
func TestConventionalColorCodeFilterDropsOtherCC(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	want := uint8(12)
	cc := New(Options{
		Bus:             bus,
		SystemName:      "IPSC",
		FrequencyHz:     443_237_500,
		Now:             func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
		ColorCodeFilter: &want,
	})

	flc := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 0x42, SrcAddr: 0x100}

	// A perfectly valid header on colour code 7 (a co-channel system bleeding
	// into the shared passband) must be dropped by the filter.
	cc.IngestBurst(burstWithFLC(flc), dmr.SlotType{ColorCode: 7, DataType: dmr.DTVoiceLCHeader})
	if got := cc.Counters().DroppedOffCC; got != 1 {
		t.Fatalf("DroppedOffCC = %d, want 1 (off-colour-code burst should be dropped)", got)
	}
	if got := cc.Counters().FECPass; got != 0 {
		t.Fatalf("FECPass = %d, want 0 (a dropped burst must not reach FEC)", got)
	}
	select {
	case ev := <-sub.C:
		t.Fatalf("off-colour-code burst wrongly published %s: %+v", ev.Kind, ev.Payload)
	default:
	}

	// The same header on the configured colour code 12 must decode and grant.
	cc.IngestBurst(burstWithFLC(flc), dmr.SlotType{ColorCode: 12, DataType: dmr.DTVoiceLCHeader})
	if got := cc.Counters().FECPass; got != 1 {
		t.Fatalf("FECPass = %d, want 1 (matching-colour-code header should decode)", got)
	}
	sawGrant := false
	for done := false; !done; {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				g := ev.Payload.(trunking.Grant)
				if g.ChannelID != 12 {
					t.Errorf("grant colour code = %d, want 12", g.ChannelID)
				}
				sawGrant = true
			}
		default:
			done = true
		}
	}
	if !sawGrant {
		t.Fatal("matching-colour-code header produced no grant")
	}
}

// TestConventionalProtocolTagDirectMode pins the Tier I direct-mode
// parameterization: a ConventionalChannel constructed with
// ProtocolTag="dmr-tier1" must publish grants tagged "dmr-tier1" (the wire
// decode is identical to Tier II — only the tag + sync set differ).
func TestConventionalProtocolTagDirectMode(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		SystemName:  "DirectMode",
		FrequencyHz: 446_006_250,
		ProtocolTag: "dmr-tier1",
		Now:         func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})
	flc := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 0x000064, SrcAddr: 0x100200}
	cc.IngestBurst(burstWithFLC(flc), dmr.SlotType{ColorCode: 7, DataType: dmr.DTVoiceLCHeader})

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindGrant {
				continue
			}
			g := ev.Payload.(trunking.Grant)
			if g.Protocol != "dmr-tier1" {
				t.Errorf("grant Protocol = %q, want dmr-tier1", g.Protocol)
			}
			return
		case <-deadline:
			t.Fatal("no grant event")
		}
	}
}

func TestConventionalPublishesGrantOnVoiceLCHeader(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		SystemName:  "TestRepeater",
		FrequencyHz: 451_000_000,
		Now:         func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})
	flc := dmr.FLC{
		FLCO:           dmr.FLCOGroupVoiceUser,
		FID:            0x00,
		ServiceOptions: 0xC5, // emergency + encrypted + priority 5
		DstAddr:        0x000064,
		SrcAddr:        0x100200,
	}
	cc.IngestBurst(burstWithFLC(flc), dmr.SlotType{ColorCode: 5, DataType: dmr.DTVoiceLCHeader})

	// Drain until we hit the grant (cc.locked may fire first).
	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				continue
			}
			if ev.Kind != events.KindGrant {
				t.Fatalf("kind = %s", ev.Kind)
			}
			g, ok := ev.Payload.(trunking.Grant)
			if !ok {
				t.Fatalf("payload type = %T", ev.Payload)
			}
			if g.System != "TestRepeater" || g.Protocol != "dmr-tier2" {
				t.Errorf("identity = %s/%s", g.System, g.Protocol)
			}
			if g.GroupID != 0x64 || g.SourceID != 0x100200 {
				t.Errorf("group/source = %d/%d", g.GroupID, g.SourceID)
			}
			if g.FrequencyHz != 451_000_000 || g.ChannelID != 5 {
				t.Errorf("freq/cc = %d/%d", g.FrequencyHz, g.ChannelID)
			}
			if !g.Encrypted || !g.Emergency {
				t.Errorf("flags = enc=%v emer=%v, want both", g.Encrypted, g.Emergency)
			}
			if g.Priority != 5 {
				t.Errorf("priority = %d, want 5", g.Priority)
			}
			if g.Individual {
				t.Error("group grant wrongly marked Individual")
			}
			return
		case <-deadline:
			t.Fatal("no grant event")
		}
	}
}

// TestConventionalPublishesUnitToUnitGrant confirms a private
// (unit-to-unit) Voice LC Header now yields a grant marked Individual,
// rather than being dropped as a "non-group FLCO" — DMR private calls were
// previously invisible on a Tier II conventional channel.
func TestConventionalPublishesUnitToUnitGrant(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		SystemName:  "TestRepeater",
		FrequencyHz: 451_000_000,
		Now:         func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})
	flc := dmr.FLC{
		FLCO:    dmr.FLCOUnitToUnitVoice,
		DstAddr: 0x00ABCD, // called subscriber, NOT a talkgroup
		SrcAddr: 0x001234,
	}
	cc.IngestBurst(burstWithFLC(flc), dmr.SlotType{ColorCode: 3, DataType: dmr.DTVoiceLCHeader})

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				continue
			}
			if ev.Kind != events.KindGrant {
				t.Fatalf("kind = %s", ev.Kind)
			}
			g := ev.Payload.(trunking.Grant)
			if !g.Individual {
				t.Error("unit-to-unit grant not marked Individual")
			}
			if g.GroupID != 0x00ABCD || g.SourceID != 0x001234 {
				t.Errorf("dst/src = %X/%X", g.GroupID, g.SourceID)
			}
			if g.Protocol != "dmr-tier2" {
				t.Errorf("protocol = %s", g.Protocol)
			}
			return
		case <-deadline:
			t.Fatal("no unit-to-unit grant event (private call dropped?)")
		}
	}
}

// TestConventionalDoesNotRelockOnColorCodeFlicker pins the fix for the
// flaky TestDaemonCCDecodesDMRTier2: a single Golay(20,8)-miscorrected
// slot type can flip the decoded color code (e.g. 0x7 → 0x5) on
// otherwise-identical traffic. The lock is to the repeater frequency,
// so a transient color-code change must not republish cc.locked — doing
// so churned the event bus / control_channel_transitions metric and made
// the integration test's exact-one-lock assertion timing-dependent.
func TestConventionalDoesNotRelockOnColorCodeFlicker(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		SystemName:  "FlickerRepeater",
		FrequencyHz: 460_500_000,
		Now:         func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})
	flc := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 0x123, SrcAddr: 0x456789}

	// First burst locks at color code 0x7; a later burst on the same
	// frequency decodes 0x5 (the FEC-miss flicker) and must NOT re-lock.
	cc.IngestBurst(burstWithFLC(flc), dmr.SlotType{ColorCode: 0x7, DataType: dmr.DTVoiceLCHeader})
	cc.IngestBurst(burstWithFLC(flc), dmr.SlotType{ColorCode: 0x5, DataType: dmr.DTVoiceLCHeader})

	var locks []LockState
	deadline := time.After(300 * time.Millisecond)
drain:
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindCCLocked {
				continue
			}
			ls, ok := ev.Payload.(LockState)
			if !ok {
				t.Fatalf("CCLocked payload type = %T, want LockState", ev.Payload)
			}
			locks = append(locks, ls)
		case <-deadline:
			break drain
		}
	}

	if len(locks) != 1 {
		t.Fatalf("cc.locked fired %d times, want 1 (color-code flicker must not re-lock): %+v", len(locks), locks)
	}
	if locks[0].ColorCode != 0x7 {
		t.Errorf("locked color code = %#x, want 0x7 (the first decode, not the flicker)", locks[0].ColorCode)
	}
}

func TestConventionalDedupsRepeatedVoiceLCHeader(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "S", FrequencyHz: 1})
	flc := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 0x42, SrcAddr: 0x100}
	slot := dmr.SlotType{ColorCode: 1, DataType: dmr.DTVoiceLCHeader}

	cc.IngestBurst(burstWithFLC(flc), slot) // first header → grant
	cc.IngestBurst(burstWithFLC(flc), slot) // dedup → no event
	cc.IngestBurst(burstWithFLC(flc), slot) // dedup → no event

	grants := 0
	timeout := time.After(150 * time.Millisecond)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				grants++
			}
		case <-timeout:
			if grants != 1 {
				t.Errorf("got %d grant events, want 1 (dedup expected)", grants)
			}
			return
		}
	}
}

func TestConventionalNewCallAfterTerminator(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "S", FrequencyHz: 1})
	flc := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 0x42, SrcAddr: 0x100}
	hdr := dmr.SlotType{ColorCode: 1, DataType: dmr.DTVoiceLCHeader}
	term := dmr.SlotType{ColorCode: 1, DataType: dmr.DTTerminatorWithLC}

	cc.IngestBurst(burstWithFLC(flc), hdr)  // grant 1
	cc.IngestBurst(burstWithFLC(flc), term) // call ended
	cc.IngestBurst(burstWithFLC(flc), hdr)  // grant 2 (state cleared)

	count := 0
	timeout := time.After(150 * time.Millisecond)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				count++
			}
		case <-timeout:
			if count != 2 {
				t.Errorf("got %d grants across two calls, want 2", count)
			}
			return
		}
	}
}

func TestConventionalIgnoresNonHeaderBursts(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "S", FrequencyHz: 1})
	flc := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 0x42}
	cc.IngestBurst(burstWithFLC(flc), dmr.SlotType{ColorCode: 1, DataType: dmr.DTCSBK})

	// A CSBK burst belongs to Tier III: Tier II must ignore it entirely
	// — no grant, and (since the lock is gated on a FEC-validated Voice
	// LC Header) no cc.locked either. Locking on a non-header slot-type
	// alone is the false "instalock" this guards against.
	timeout := time.After(50 * time.Millisecond)
DrainCSBK:
	for {
		select {
		case ev := <-sub.C:
			t.Errorf("unexpected event for CSBK burst: %s", ev.Kind)
		case <-timeout:
			break DrainCSBK
		}
	}
}

// TestConventionalPublishesReleaseOnTerminator confirms an explicit
// Terminator-with-LC ends the call promptly: the decoder publishes a
// KindCallRelease keyed to the active call's talkgroup, rather than
// leaving the engine to wait out the hangtime timers.
func TestConventionalPublishesReleaseOnTerminator(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		SystemName:  "TestRepeater",
		FrequencyHz: 451_000_000,
		Now:         func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})
	// A group call opens (Voice LC Header), then a Terminator ends it.
	flc := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 0x000064, SrcAddr: 0x100200}
	cc.IngestBurst(burstWithFLC(flc), dmr.SlotType{ColorCode: 5, DataType: dmr.DTVoiceLCHeader})
	cc.IngestBurst(burstWithFLC(flc), dmr.SlotType{ColorCode: 5, DataType: dmr.DTTerminatorWithLC})

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindCallRelease {
				continue
			}
			r := ev.Payload.(trunking.CallRelease)
			if r.System != "TestRepeater" || r.GroupID != 0x64 {
				t.Fatalf("release identity = %s/%X, want TestRepeater/64", r.System, r.GroupID)
			}
			if r.Reason != trunking.EndReasonReleased {
				t.Errorf("reason = %v, want Released", r.Reason)
			}
			return
		case <-deadline:
			t.Fatal("no KindCallRelease on Terminator")
		}
	}
}

func TestConventionalIgnoresTerminatorFLCO(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "S", FrequencyHz: 1})
	// A non-voice FLCO (e.g. Terminator) carries no call, so it yields no
	// grant — but a valid Voice LC Header burst still locks the channel.
	// (Group- and unit-to-unit voice LCs DO grant; see the dedicated tests.)
	flc := dmr.FLC{FLCO: dmr.FLCOTerminator, DstAddr: 0x100, SrcAddr: 0x200}
	cc.IngestBurst(burstWithFLC(flc), dmr.SlotType{ColorCode: 1, DataType: dmr.DTVoiceLCHeader})

	timeout := time.After(50 * time.Millisecond)
	var sawLock bool
DrainTerm:
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				sawLock = true
				continue
			}
			if ev.Kind == events.KindGrant {
				t.Errorf("Terminator FLCO produced a grant event")
				continue
			}
			t.Errorf("unexpected event for Terminator FLCO: %s", ev.Kind)
		case <-timeout:
			break DrainTerm
		}
	}
	if !sawLock {
		t.Errorf("expected cc.locked from valid Voice LC Header, got none")
	}
}

func TestConventionalEmitsRSDecodeErrorOnBadParity(t *testing.T) {
	// Build a Voice LC Header with valid BPTC framing but corrupted
	// RS(12,9) parity (skip the seed XOR step). The state machine
	// should publish decode.error stage="voiceheader-rs" and not a
	// grant.
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "S", FrequencyHz: 1})

	flc := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 0x42, SrcAddr: 0x100}
	flcBytes := dmr.AssembleFLC(flc)
	var data [9]byte
	copy(data[:], flcBytes)
	cw := framing.EncodeRS12_9(data)
	// Skip the seed XOR — the verifier expects the seed-applied form,
	// so skipping it leaves the parity 'wrong' and verify must fail.
	bits := make([]byte, 96)
	for i := 0; i < 96; i++ {
		bits[i] = (cw[i>>3] >> uint(7-(i&7))) & 1
	}
	channel := framing.EncodeBPTC196_96(bits)
	var b dmr.Burst
	for i := 0; i < dmr.HalfPayloadDibits; i++ {
		b.Dibits[i] = (channel[2*i] << 1) | channel[2*i+1]
	}
	for i := 0; i < dmr.HalfPayloadDibits; i++ {
		b.Dibits[dmr.BurstDibits-dmr.HalfPayloadDibits+i] =
			(channel[2*(dmr.HalfPayloadDibits+i)] << 1) | channel[2*(dmr.HalfPayloadDibits+i)+1]
	}

	cc.IngestBurst(&b, dmr.SlotType{ColorCode: 1, DataType: dmr.DTVoiceLCHeader})

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				t.Fatalf("unexpected grant emitted with bad RS parity: %+v", ev.Payload)
			}
			if ev.Kind != events.KindDecodeError {
				continue
			}
			de := ev.Payload.(events.DecodeError)
			if de.Protocol == "dmr-tier2" && de.Stage == "voiceheader-rs" {
				return
			}
		case <-deadline:
			t.Fatal("no decode-error with stage=voiceheader-rs")
		}
	}
}
