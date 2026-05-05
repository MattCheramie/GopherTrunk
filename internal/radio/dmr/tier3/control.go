package tier3

import (
	"log/slog"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// LockState is the payload of cc.locked / cc.lost events emitted by the
// DMR Tier III control-channel state machine.
type LockState struct {
	FrequencyHz uint32
	ColorCode   uint8
	SystemID    uint16
}

// ControlChannel ingests detected DMR bursts whose Slot Type identifies a
// CSBK, runs BPTC(196,96) decode + CRC, and emits cc.locked the first time
// it sees an Aloha or System-Info CSBK with self-consistent fields. This
// mirrors the P25 phase1 control channel; the fuller Tier III state
// machine (Aloha tracking, neighbor list, channel-grant follow) lives in
// the trunking engine, which subscribes to the events this package emits.
type ControlChannel struct {
	bus    *events.Bus
	log    *slog.Logger
	freqHz uint32
	locked bool
	last   LockState
}

func NewControlChannel(bus *events.Bus, log *slog.Logger, freqHz uint32) *ControlChannel {
	if log == nil {
		log = slog.Default()
	}
	return &ControlChannel{bus: bus, log: log, freqHz: freqHz}
}

// IngestBurst hands one DMR burst to the state machine. The burst's slot
// type must already be parsed by the caller; the 20-bit Hamming(20,8)
// over the slot type is not yet wired (see dmr/slottype.go).
func (c *ControlChannel) IngestBurst(b *dmr.Burst, slot dmr.SlotType) {
	if slot.DataType != dmr.DTCSBK {
		return
	}
	bits, errs := framing.DecodeBPTC196_96(b.PayloadBits())
	if errs < 0 {
		c.log.Debug("dmr: BPTC uncorrectable")
		return
	}
	csbk, err := ParseCSBK(InfoBitsToBytes(bits))
	if err != nil {
		c.log.Debug("dmr: CSBK CRC failed")
		return
	}
	c.handleCSBK(slot.ColorCode, csbk)
}

func (c *ControlChannel) handleCSBK(cc uint8, csbk CSBK) {
	switch csbk.Opcode {
	case OpAloha:
		c.maybeLock(LockState{FrequencyHz: c.freqHz, ColorCode: cc, SystemID: ParseAloha(csbk.Payload).SystemID})
	case OpSysInfo:
		si := ParseSystemInfoBroadcast(csbk.Payload)
		c.maybeLock(LockState{FrequencyHz: c.freqHz, ColorCode: cc, SystemID: si.SystemID})
	}
}

func (c *ControlChannel) maybeLock(s LockState) {
	if !c.locked || c.last != s {
		c.locked = true
		c.last = s
		c.bus.Publish(events.Event{Kind: events.KindCCLocked, Payload: s})
		c.log.Info("dmr cc locked", "freq", s.FrequencyHz, "cc", s.ColorCode, "sysid", s.SystemID)
	}
}

// MarkLost publishes cc.lost and resets the locked flag. Wired up by the
// engine's watchdog.
func (c *ControlChannel) MarkLost() {
	if !c.locked {
		return
	}
	c.locked = false
	c.bus.Publish(events.Event{Kind: events.KindCCLost, Payload: c.last})
}
