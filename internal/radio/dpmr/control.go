package dpmr

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// LockState is the payload of the cc.locked / cc.lost events a dPMR Mode 3
// beacon channel emits.
type LockState struct {
	FrequencyHz uint32
	// SystemID is the beacon's 14-bit System Identity Code (SYScast1 and
	// Aloha), 0 until one has been confirmed.
	SystemID uint32
}

// LockedFrequencyHz / LockedNAC make LockState satisfy
// trunking.LockedPayload so the cchunt supervisor's state machine
// recognises dPMR lock events alongside the other protocols'. dPMR has no
// NAC; the System Identity Code is the closest per-site identifier and
// fills the slot.
func (s LockState) LockedFrequencyHz() uint32 { return s.FrequencyHz }
func (s LockState) LockedNAC() uint16         { return uint16(s.SystemID) }

// ControlChannel follows a dPMR Mode 3 beacon channel (clause 4.2.3.7):
// it frames the beacon (BeaconFramer), declares the channel locked once the
// beacon has identified itself, and turns each Goto Channel broadcast into
// an events.KindGrant with `trunking.Grant.Protocol = "dpmr"`, so the
// composer's dPMR voice chain follows the call to its traffic channel.
//
// What counts as the beacon identifying itself: an Aloha (a broadcast only
// a Mode 3 beacon sends, carrying the System Identity Code), or the same
// System Identity Code in the SYScast1 of two framesets. A single SYScast1
// is not enough: the framer can briefly frame a traffic channel's header,
// which also opens with FS1, and the slot after it then holds payload whose
// 17-bit field passes CRC7 now and then.
//
// A Goto Channel names the called party and the traffic channel; the
// calling party rides the SYScast2 and SYScast3 of the same frameset. The
// grant's frequency is the MS receive frequency, the traffic channel's
// downlink. A beacon repeats an unacknowledged Goto Channel, so a grant that
// repeats one already published within grantRepeatWindow is not published
// again.
type ControlChannel struct {
	bus        *events.Bus
	log        *slog.Logger
	systemName string
	freqHz     uint32
	now        func() time.Time

	framer *BeaconFramer

	mu         sync.Mutex
	locked     bool
	last       LockState
	sysCode    uint16 // last SYScast1 system code, for the two-in-a-row check
	sysCodeN   int
	lastGrant  grantKey
	lastGrantT time.Time
	customWarn time.Time

	lastActivityNano atomic.Int64
	decoded          atomic.Uint64
}

// grantKey identifies a grant for repeat suppression.
type grantKey struct {
	called, calling, freq uint32
}

// grantRepeatWindow is how long a repeated Goto Channel for the same call
// is taken as the beacon's repetition (clause 4.2.3.11 e), not a new call.
const grantRepeatWindow = 2 * time.Second

// Options configure a ControlChannel.
type Options struct {
	Bus         *events.Bus
	Log         *slog.Logger
	SystemName  string
	FrequencyHz uint32
	Now         func() time.Time
}

// New constructs a ControlChannel.
func New(opts Options) *ControlChannel {
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	c := &ControlChannel{
		bus:        opts.Bus,
		log:        log,
		systemName: opts.SystemName,
		freqHz:     opts.FrequencyHz,
		now:        now,
	}
	c.framer = NewBeaconFramer(c.ingest)
	return c
}

// Process consumes dibits from the receiver. It returns
// baseIdx+len(dibits), the ccdecoder pipeline contract.
func (c *ControlChannel) Process(dibits []uint8, baseIdx int) int {
	return c.framer.Process(dibits, baseIdx)
}

// ResyncReset drops the framer's lock and buffered dibits after the
// receiver was reset, so no frameset is spliced across the restart.
func (c *ControlChannel) ResyncReset() { c.framer.Reset() }

// Framer exposes the beacon framer, for its counters.
func (c *ControlChannel) Framer() *BeaconFramer { return c.framer }

// Locked reports whether the beacon has identified itself.
func (c *ControlChannel) Locked() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.locked
}

// LastActivityNano is the Unix-nano time of the most recent CRC-valid
// message or SYScast, 0 before the first: the pipeline's signal-time
// decode-drought watchdog compares successive values.
func (c *ControlChannel) LastActivityNano() int64 { return c.lastActivityNano.Load() }

// DecodedFrames counts framesets that carried a CRC-valid message or
// SYScast field.
func (c *ControlChannel) DecodedFrames() uint64 { return c.decoded.Load() }

// ingest handles one framed frameset.
func (c *ControlChannel) ingest(fs BeaconFrameset) {
	anySY := false
	for _, f := range fs.SYScast {
		anySY = anySY || f.OK
	}
	if fs.MessageOK || anySY || fs.AppendedOK {
		c.decoded.Add(1)
		c.lastActivityNano.Store(c.now().UnixNano())
	}

	// Identity: SYScast1 twice in a row, or an Aloha.
	if fs.SYScast[0].OK {
		s1 := ParseSYScast1(fs.SYScast[0].Bits)
		c.mu.Lock()
		if s1.SystemCode == c.sysCode {
			c.sysCodeN++
		} else {
			c.sysCode, c.sysCodeN = s1.SystemCode, 1
		}
		confirmed := c.sysCodeN >= 2
		c.mu.Unlock()
		if confirmed {
			c.maybeLock(uint32(s1.SystemCode))
		}
	}
	if !fs.MessageOK {
		return
	}
	if a, ok := fs.Message.Aloha(); ok {
		c.maybeLock(uint32(a.SystemCode))
		return
	}
	if g, ok := fs.Message.GotoChannel(); ok {
		c.gotoChannel(g, fs)
		return
	}
	c.log.Debug("dpmr: beacon message", "system", c.systemName, "msg", fs.Message.Name())
}

func (c *ControlChannel) gotoChannel(g GotoChannel, fs BeaconFrameset) {
	// A Goto Channel is only sent by a beacon: the channel is a beacon.
	c.maybeLock(0)
	calling, haveCalling := uint32(0), false
	if fs.SYScast[1].OK && fs.SYScast[2].OK {
		calling, haveCalling = CallingParty(fs.SYScast[1].Bits, fs.SYScast[2].Bits)
	}
	freq, freqOK := g.RxHz()
	if !g.Voice() {
		c.log.Debug("dpmr: goto channel for a data call (not followed)",
			"system", c.systemName, "called", g.Called, "mode", g.Mode, "freq_hz", freq)
		return
	}
	now := c.now()
	if !freqOK {
		c.mu.Lock()
		warn := now.Sub(c.customWarn) > 10*time.Minute
		if warn {
			c.customWarn = now
		}
		c.mu.Unlock()
		if warn {
			c.log.Warn("dpmr: goto channel names a custom or reserved frequency band, which the standard leaves to the operator; the call cannot be followed",
				"system", c.systemName, "band", g.Band, "rx_field", g.RxField, "sep_5khz", g.Sep5k)
		}
		return
	}
	key := grantKey{called: g.Called, calling: calling, freq: freq}
	c.mu.Lock()
	repeat := key == c.lastGrant && now.Sub(c.lastGrantT) < grantRepeatWindow
	c.lastGrant, c.lastGrantT = key, now
	c.mu.Unlock()
	if repeat {
		return
	}
	group := IsWildcardTalkgroup(g.Called)
	digits, _ := Address(g.Called)
	if c.bus != nil {
		c.bus.Publish(events.Event{
			Kind: events.KindGrant,
			Payload: trunking.Grant{
				System:      c.systemName,
				Protocol:    "dpmr",
				GroupID:     g.Called,
				SourceID:    calling,
				FrequencyHz: freq,
				ChannelNum:  g.RxField,
				Emergency:   g.Emergency,
				At:          now,
			},
		})
	}
	c.log.Debug("dpmr: grant",
		"system", c.systemName, "called", g.Called, "called_digits", digits,
		"wildcard_talkgroup", group, "calling", calling, "have_calling", haveCalling,
		"freq_hz", freq, "emergency", g.Emergency, "broadcast", g.Broadcast)
}

// maybeLock publishes cc.locked the first time the beacon identifies
// itself, and again when the System Identity Code first becomes known or
// changes. sys 0 is "no code learned from this evidence".
func (c *ControlChannel) maybeLock(sys uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := LockState{FrequencyHz: c.freqHz, SystemID: sys}
	if c.locked && sys == 0 {
		return
	}
	if c.locked && c.last == s {
		return
	}
	c.locked = true
	c.last = s
	if c.bus != nil {
		c.bus.Publish(events.Event{Kind: events.KindCCLocked, Payload: s})
	}
	c.log.Info("dpmr cc locked", "freq", s.FrequencyHz, "sys", s.SystemID, "system", c.systemName)
}

// MarkLost publishes cc.lost and resets the locked flag. The hunter calls
// this when the beacon goes silent.
func (c *ControlChannel) MarkLost() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.locked {
		return
	}
	c.locked = false
	c.sysCodeN = 0
	if c.bus != nil {
		c.bus.Publish(events.Event{Kind: events.KindCCLost, Payload: c.last})
	}
}
