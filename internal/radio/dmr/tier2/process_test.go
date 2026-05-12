package tier2

import (
	"log/slog"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// TestProcess_DecodesVoiceLCHeaderBurst feeds the Process adapter a
// dibit stream containing a single Voice LC Header burst (sync +
// slot-type around a BPTC-encoded FLC). The state machine should
// publish KindCCLocked + KindGrant on the bus.
func TestProcess_DecodesVoiceLCHeaderBurst(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "TestRepeater",
		FrequencyHz: 460_500_000,
	})

	dibits := buildVoiceLCHeaderBurstDibits(0x7, 0x123, 0x456789)
	// Add some warmup + tail so the sync detector has room.
	stream := make([]uint8, 0, 50+len(dibits)+50)
	for i := 0; i < 50; i++ {
		stream = append(stream, uint8(i&3))
	}
	stream = append(stream, dibits...)
	for i := 0; i < 50; i++ {
		stream = append(stream, uint8(i&3))
	}
	if got := cc.Process(stream, 0); got != len(stream) {
		t.Errorf("Process returned %d, want %d", got, len(stream))
	}

	var sawLock, sawGrant bool
	deadline := time.After(300 * time.Millisecond)
DrainLoop:
	for {
		select {
		case ev := <-sub.C:
			switch ev.Kind {
			case events.KindCCLocked:
				sawLock = true
			case events.KindGrant:
				sawGrant = true
			}
			if sawLock && sawGrant {
				break DrainLoop
			}
		case <-deadline:
			break DrainLoop
		}
	}
	if !sawLock {
		t.Errorf("no KindCCLocked event observed")
	}
	if !sawGrant {
		t.Errorf("no KindGrant event observed")
	}
}

// buildVoiceLCHeaderBurstDibits builds a 132-dibit Voice LC Header
// burst for the Tier II Process adapter.
func buildVoiceLCHeaderBurstDibits(colorCode uint8, groupID, sourceID uint32) []uint8 {
	flc := dmr.FLC{
		FLCO:    dmr.FLCOGroupVoiceUser,
		DstAddr: groupID,
		SrcAddr: sourceID,
	}
	flcBytes := dmr.AssembleFLC(flc)
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
	channelBits := framing.EncodeBPTC196_96(bits)
	payloadDibits := framing.BitsToDibits(channelBits)

	slotBits := dmr.AssembleSlotType(dmr.SlotType{ColorCode: colorCode, DataType: dmr.DTVoiceLCHeader})
	slotDibits := framing.BitsToDibits(slotBits)

	burst := make([]uint8, 0, dmr.BurstDibits)
	burst = append(burst, payloadDibits[:dmr.HalfPayloadDibits]...)
	burst = append(burst, slotDibits[:dmr.SlotTypeDibits]...)
	burst = append(burst, dmr.BSData.Dibits[:]...)
	burst = append(burst, slotDibits[dmr.SlotTypeDibits:]...)
	burst = append(burst, payloadDibits[dmr.HalfPayloadDibits:]...)
	return burst
}
