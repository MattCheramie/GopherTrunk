package motorola

import (
	"log/slog"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// LockState is the payload of cc.locked / cc.lost events emitted by
// the Motorola control-channel state machine.
type LockState struct {
	FrequencyHz uint32
	SystemID    uint16
}

// LockedFrequencyHz / LockedNAC make LockState satisfy
// trunking.LockedPayload so the cchunt supervisor's state machine
// recognises Motorola lock events alongside the protocol-neutral P25 /
// DMR / NXDN / TETRA payloads. Motorola doesn't have a P25-style NAC;
// SystemID is the closest per-site identifier and gets plumbed into
// the NAC slot. Without these methods, the supervisor's type-assertion
// on cc.locked silently drops the event and /api/v1/scanner never
// surfaces state=locked.
func (s LockState) LockedFrequencyHz() uint32 { return s.FrequencyHz }
func (s LockState) LockedNAC() uint16         { return s.SystemID }

// oswQueueDepth is how many decoded OSWs the sequencer buffers before
// interpreting the oldest. SmartNet messages span up to three
// consecutive OSWs (system ID + CC broadcast triplets), so the
// sequencer only steps once it can see a full window.
const oswQueueDepth = 3

// ControlChannel ingests OSWs from a single SmartNet / SmartZone
// control channel, emits cc.locked the first time the OSW stream
// carries the system identity, and republishes voice grants as
// events.KindGrant with a `trunking.Grant` payload.
//
// SmartNet OSWs are not self-describing: a grant is "group address +
// channel number" spread over one or two OSWs, and system
// identification rides an 0x308/0x30B pair. The sequencer here is the
// subset of trunk-recorder's SmartnetParser that GopherTrunk's engine
// needs — lock, group voice grants/updates, idle, and alternate/
// adjacent control-channel broadcasts for the hunt topology.
type ControlChannel struct {
	bus        *events.Bus
	log        *slog.Logger
	systemName string
	freqHz     uint32
	plan       BandPlan
	onOSW      func(OSW)
	now        func() time.Time

	// topo accumulates system identity + advertised alternate /
	// adjacent control channels for the hunt layer.
	topo topologyModel

	// proc is the cross-call bit / sync framing state the Process
	// adapter uses (see process.go). Lazily constructed on the
	// first Process call.
	proc *processState

	mu     sync.Mutex
	oswQ   []OSW
	locked bool
	last   LockState
}

// Options configure a ControlChannel.
type Options struct {
	Bus        *events.Bus
	Log        *slog.Logger
	SystemName string
	// FrequencyHz is the control-channel frequency this state machine
	// is bound to. Carried in cc.locked / cc.lost payloads.
	FrequencyHz uint32
	// Plan maps channel numbers to frequencies. nil selects the
	// 800 MHz standard plan.
	Plan BandPlan
	// OnOSW, when set, observes every CRC-clean OSW before the
	// sequencer interprets it — a decode-yield tap for tests and
	// diagnostics.
	OnOSW func(OSW)
	Now   func() time.Time
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
	plan := opts.Plan
	if plan == nil {
		plan, _ = ParseBandPlan("")
	}
	return &ControlChannel{
		bus:        opts.Bus,
		log:        log,
		systemName: opts.SystemName,
		freqHz:     opts.FrequencyHz,
		plan:       plan,
		onOSW:      opts.OnOSW,
		now:        now,
	}
}

// Ingest hands a single decoded OSW to the sequencer. Real captures
// arrive via the receiver + Process framing chain; tests publish OSWs
// directly. Interpretation lags by up to oswQueueDepth-1 OSWs so
// multi-OSW sequences are seen whole.
func (c *ControlChannel) Ingest(o OSW) {
	if c.onOSW != nil {
		c.onOSW(o)
	}
	c.mu.Lock()
	c.oswQ = append(c.oswQ, o)
	for len(c.oswQ) >= oswQueueDepth {
		n := c.stepLocked()
		c.oswQ = c.oswQ[n:]
	}
	c.mu.Unlock()
}

// stepLocked interprets the oldest buffered OSW (plus its successors
// when it opens a multi-OSW sequence) and returns how many OSWs were
// consumed (≥ 1). Caller holds c.mu. Mirrors trunk-recorder
// SmartnetParser::process_osws, oldest = osw2 in its naming.
func (c *ControlChannel) stepLocked() int {
	o2, o1 := c.oswQ[0], c.oswQ[1]

	isChan2 := c.plan.IsChannel(o2.Command)
	switch {
	case isChan2 && o2.Group:
		// Single-OSW group voice update: keeps an in-flight call
		// alive and lets a scanner join late. Same shape as a grant
		// minus the source radio ID.
		c.publishGrant(o2, 0)
		return 1

	case isChan2 && !o2.Group && o2.Address&0xFF00 == 0x1F00:
		// Control-channel broadcast beacon (no system ID by itself).
		return 1

	case o2.Command == CmdIdle, o2.Command == CmdGroupBusy, o2.Command == CmdEmergencyBusy:
		return 1

	case o2.Command == CmdFirstNormal:
		switch {
		case c.plan.IsChannel(o1.Command) && !o1.Group && o1.Address&0xFF00 == 0x1F00:
			// System ID + control-channel broadcast: the lock signal.
			c.observeSystemID(o2.Address)
			return 2
		case c.plan.IsChannel(o1.Command) && o1.Group && o1.Address != 0 && o2.Address != 0:
			// Analog group voice grant: o2 carries the source radio
			// ID, o1 the talkgroup + channel.
			c.publishGrant(o1, uint32(o2.Address))
			return 2
		case c.plan.IsChannel(o1.Command) && !o1.Group && o1.Address != 0 && o2.Address != 0:
			// Private / interconnect call. Observed, not recorded.
			c.log.Debug("motorola: private/interconnect call",
				"system", c.systemName, "src", o2.Address, "dst", o1.Address)
			return 2
		default:
			// Unknown continuation — consume only the opener so a
			// sequence we misread doesn't eat a valid follower.
			return 1
		}

	case o2.Command == CmdFirstAlternate:
		switch {
		case o1.Group && o1.Address&0xFC00 == 0x2800:
			// System ID + control-channel broadcast (alternate form).
			c.observeSystemID(o2.Address)
			return 2
		case o1.Address&0xFC00 == 0x6000:
			// Alternate (same-site) or adjacent-site control channel:
			// the low 10 address bits carry its channel number. Feed
			// the hunt topology either way.
			c.observeSystemID(o2.Address)
			ch := o1.Address & 0x3FF
			if hz, ok := c.plan.Frequency(ch); ok {
				c.topo.applyNeighbor(NeighborSite{LCN: ch, Adjacent: o1.Group})
				c.log.Debug("motorola: alternate/adjacent cc",
					"system", c.systemName, "channel", ch, "freq_hz", hz,
					"adjacent", o1.Group)
			}
			return 2
		default:
			// Extended functions (radio check, inhibit, status acks…):
			// consume the pair, nothing for the engine yet.
			return 2
		}

	default:
		return 1
	}
}

// observeSystemID folds a decoded system ID into topology + lock
// state. Caller holds c.mu.
func (c *ControlChannel) observeSystemID(id uint16) {
	if id == 0 {
		return
	}
	c.topo.applySystemID(id)
	c.maybeLockLocked(LockState{FrequencyHz: c.freqHz, SystemID: id})
}

// Topology returns a snapshot of the system topology (identity +
// advertised control channels) accumulated from the control channel,
// for the hunt/discovery layer.
func (c *ControlChannel) Topology() TopologyConfig { return c.topo.snapshot() }

// NeighborFrequency resolves an advertised control-channel number to
// its downlink frequency via the band plan, quietly (no log/event on
// a miss — an unresolved neighbour is informational).
func (c *ControlChannel) NeighborFrequency(lcn uint16) (uint32, bool) {
	return c.plan.Frequency(lcn)
}

// publishGrant emits a voice grant/update. grantOSW carries the
// talkgroup in its address and the channel in its command; src is the
// source radio ID when the sequence carried one (0 on updates).
// Caller holds c.mu.
func (c *ControlChannel) publishGrant(grantOSW OSW, src uint32) {
	if c.bus == nil {
		return
	}
	freq, ok := c.plan.Frequency(grantOSW.Command)
	if !ok {
		return
	}
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:      c.systemName,
			Protocol:    "motorola",
			GroupID:     uint32(grantOSW.Talkgroup()),
			SourceID:    src,
			FrequencyHz: freq,
			ChannelNum:  grantOSW.Command,
			Encrypted:   grantOSW.Encrypted(),
			Emergency:   grantOSW.Emergency(),
			At:          c.now(),
		},
	})
	c.log.Debug("motorola: grant",
		"system", c.systemName, "tg", grantOSW.Talkgroup(), "src", src,
		"channel", grantOSW.Command, "freq_hz", freq)
}

// maybeLockLocked publishes cc.locked on the first (or a changed)
// system identity. Caller holds c.mu.
func (c *ControlChannel) maybeLockLocked(s LockState) {
	if c.locked && c.last == s {
		return
	}
	c.locked = true
	c.last = s
	if c.bus != nil {
		c.bus.Publish(events.Event{Kind: events.KindCCLocked, Payload: s})
	}
	c.log.Info("motorola cc locked",
		"freq", s.FrequencyHz, "sys", s.SystemID, "system", c.systemName)
}

// MarkLost publishes cc.lost and resets the locked flag. The trunking
// engine's hunter calls this when the control channel goes silent.
func (c *ControlChannel) MarkLost() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.locked {
		return
	}
	c.locked = false
	if c.bus != nil {
		c.bus.Publish(events.Event{Kind: events.KindCCLost, Payload: c.last})
	}
}
