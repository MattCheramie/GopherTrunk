package tier2

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
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
		ServiceOptions: 0xC0, // emergency + encrypted
		DstAddr:        0x000064,
		SrcAddr:        0x100200,
	}
	cc.IngestBurst(burstWithFLC(flc), dmr.SlotType{ColorCode: 5, DataType: dmr.DTVoiceLCHeader})

	select {
	case ev := <-sub.C:
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
	case <-time.After(time.Second):
		t.Fatal("no grant event")
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

	count := 0
	timeout := time.After(150 * time.Millisecond)
	for {
		select {
		case <-sub.C:
			count++
		case <-timeout:
			if count != 1 {
				t.Errorf("got %d grant events, want 1 (dedup expected)", count)
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

	cc.IngestBurst(burstWithFLC(flc), hdr) // grant 1
	cc.IngestBurst(burstWithFLC(flc), term) // call ended
	cc.IngestBurst(burstWithFLC(flc), hdr) // grant 2 (state cleared)

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

	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event for CSBK burst: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestConventionalIgnoresNonGroupFLCO(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "S", FrequencyHz: 1})
	// Unit-to-unit calls are intentionally not republished as grants
	// in this PR; the engine's grant model is talkgroup-keyed.
	flc := dmr.FLC{FLCO: dmr.FLCOUnitToUnitVoice, DstAddr: 0x100, SrcAddr: 0x200}
	cc.IngestBurst(burstWithFLC(flc), dmr.SlotType{ColorCode: 1, DataType: dmr.DTVoiceLCHeader})

	select {
	case ev := <-sub.C:
		t.Errorf("unit-to-unit FLCO produced an event: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
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
