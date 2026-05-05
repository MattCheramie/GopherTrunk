package tier3

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// burstWithCSBK packs a CSBK into a DMR burst's payload halves.
func burstWithCSBK(c CSBK) *dmr.Burst {
	info := AssembleCSBK(c)
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
		b.Dibits[dmr.BurstDibits-dmr.HalfPayloadDibits+i] = (channel[2*(dmr.HalfPayloadDibits+i)] << 1) | channel[2*(dmr.HalfPayloadDibits+i)+1]
	}
	return &b
}

func TestControlChannelEmitsLockOnAloha(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, nil, 851_000_000)
	c := CSBK{LB: true, Opcode: OpAloha, FID: 0, Payload: [8]byte{0x12, 0x08, 0x12, 0x34, 0, 0, 0, 0}}
	cc.IngestBurst(burstWithCSBK(c), dmr.SlotType{ColorCode: 1, DataType: dmr.DTCSBK})

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked {
			t.Fatalf("kind = %s", ev.Kind)
		}
		ls, ok := ev.Payload.(LockState)
		if !ok {
			t.Fatalf("payload type %T", ev.Payload)
		}
		if ls.SystemID != 0x1234 || ls.ColorCode != 1 || ls.FrequencyHz != 851_000_000 {
			t.Errorf("payload = %+v", ls)
		}
	case <-time.After(time.Second):
		t.Fatal("no event")
	}
}

func TestControlChannelEmitsLockOnSysInfo(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, nil, 851_000_000)
	c := CSBK{LB: true, Opcode: OpSysInfo, Payload: [8]byte{0xCA, 0xFE, 0x01, 0x05, 0xFF, 0, 0, 0}}
	cc.IngestBurst(burstWithCSBK(c), dmr.SlotType{ColorCode: 0xC, DataType: dmr.DTCSBK})

	select {
	case ev := <-sub.C:
		ls := ev.Payload.(LockState)
		if ls.SystemID != 0xCAFE {
			t.Errorf("sysid = %X, want CAFE", ls.SystemID)
		}
	case <-time.After(time.Second):
		t.Fatal("no event")
	}
}

func TestControlChannelIgnoresNonCSBK(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, nil, 851_000_000)
	c := CSBK{LB: true, Opcode: OpAloha, Payload: [8]byte{0, 0x08, 0xCA, 0xFE, 0, 0, 0, 0}}
	cc.IngestBurst(burstWithCSBK(c), dmr.SlotType{ColorCode: 1, DataType: dmr.DTVoiceLCHeader})

	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestControlChannelMarkLost(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, nil, 851_000_000)
	c := CSBK{LB: true, Opcode: OpSysInfo, Payload: [8]byte{0x42, 0x00, 0x01, 0x01, 0, 0, 0, 0}}
	cc.IngestBurst(burstWithCSBK(c), dmr.SlotType{ColorCode: 7, DataType: dmr.DTCSBK})
	<-sub.C

	cc.MarkLost()
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLost {
			t.Errorf("kind = %s, want cc.lost", ev.Kind)
		}
	case <-time.After(time.Second):
		t.Fatal("no cc.lost")
	}
}
