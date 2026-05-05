package nxdn

import (
	"log/slog"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// LockState is the payload of cc.locked / cc.lost events emitted by the
// NXDN control-channel state machine.
type LockState struct {
	FrequencyHz uint32
	BaudRate    BaudRate
	SiteID      uint16
	SystemID    uint16
}

// ControlChannel ingests parsed NXDN frames whose LICH indicates RCCH
// (control channel), reads the CAC payload, and emits cc.locked when it
// observes a SITE_INFO or CCH announcement message that confirms the
// frequency. Mirrors the P25 phase1 / DMR tier3 control channels.
//
// The full RCCH state machine (Aloha tracking, channel-grant follow,
// neighbor list reaction) lives in the trunking engine, which
// subscribes to the events this package emits.
type ControlChannel struct {
	bus    *events.Bus
	log    *slog.Logger
	freqHz uint32
	rate   BaudRate
	locked bool
	last   LockState
}

func NewControlChannel(bus *events.Bus, log *slog.Logger, freqHz uint32, rate BaudRate) *ControlChannel {
	if log == nil {
		log = slog.Default()
	}
	return &ControlChannel{bus: bus, log: log, freqHz: freqHz, rate: rate}
}

// IngestFrame consumes one decoded NXDN frame. The caller must have
// already extracted and parsed the LICH and CAC fields.
func (c *ControlChannel) IngestFrame(lich LICH, cac *CACMessage) {
	if !lich.ParityOK || lich.RFCh != RFChControl {
		return
	}
	if cac == nil {
		return
	}
	switch cac.Type {
	case RCCHSITEINFO:
		s := ParseSiteInfo(cac.Payload)
		c.maybeLock(LockState{FrequencyHz: c.freqHz, BaudRate: c.rate, SiteID: s.SiteID, SystemID: s.SystemID})
	case RCCHCCH:
		c.maybeLock(LockState{FrequencyHz: c.freqHz, BaudRate: c.rate})
	}
}

func (c *ControlChannel) maybeLock(s LockState) {
	if !c.locked || c.last != s {
		c.locked = true
		c.last = s
		c.bus.Publish(events.Event{Kind: events.KindCCLocked, Payload: s})
		c.log.Info("nxdn cc locked", "freq", s.FrequencyHz, "rate", s.BaudRate.String(), "site", s.SiteID, "sys", s.SystemID)
	}
}

// MarkLost publishes cc.lost and resets the locked flag.
func (c *ControlChannel) MarkLost() {
	if !c.locked {
		return
	}
	c.locked = false
	c.bus.Publish(events.Event{Kind: events.KindCCLost, Payload: c.last})
}
