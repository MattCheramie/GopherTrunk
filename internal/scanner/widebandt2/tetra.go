package widebandt2

import (
	"log/slog"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	tetra "github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	tetrarx "github.com/MattCheramie/GopherTrunk/internal/radio/tetra/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// tetraChannelRateHz is the per-tap sample rate a TETRA channel needs: its
// 18000-baud π/4-DQPSK wants a wider channel than the 4800-baud C4FM family, so
// it channelizes to 144 kHz where DMR/P25 use 48 kHz. Mirrors
// ccdecoder.DDCTargetForProtocol / tetra/receiver so a wideband-channelized
// TETRA stream behaves identically to the dedicated-dongle path.
const tetraChannelRateHz = 144_000.0

// TETRA control-channel resync/stale timers, mirrored from the single-frequency
// ccdecoder tetraPipeline so a wideband TETRA tap reacquires and surfaces lock
// loss the same way. TETRA's maybeLock is edge-triggered and never re-affirms a
// steady lock, so without these a decode drought (the multislot case) would
// never reacquire and a dead carrier would never surface cc.lost.
const (
	tetraResyncTimeout      = 1500 * time.Millisecond
	tetraLockStaleTimeout   = 5 * time.Second
	tetraStaleCheckInterval = 1 * time.Second
	// Payload-drought escape, mirrored from ccdecoder (see that package's
	// tetraPayloadResyncTimeout for the full rationale): a locked channel
	// decoding sync bursts but zero CRC-clean SCH payload for this much signal
	// is "locked but deaf" (e.g. a wrong AFC alias latch) and gets the same
	// destructive reset; after tetraPayloadResyncMaxAttempts fruitless resets
	// the lock is declared lost so the supervisor re-hunts.
	tetraPayloadResyncTimeout     = 12 * time.Second
	tetraPayloadResyncMaxAttempts = 3
)

// channelOutRateHz is the per-tap output rate a channel's protocol needs. TETRA
// gets tetraChannelRateHz; every other protocol uses the C4FM-family
// narrowbandRateHz. Used to give each DDC tap its own rate (AddTapAtRate) so a
// TETRA channel can share one wideband SDR with 48 kHz DMR/P25 channels.
func channelOutRateHz(proto trunking.Protocol) float64 {
	if proto == trunking.ProtocolTETRA {
		return tetraChannelRateHz
	}
	return narrowbandRateHz
}

// anyTETRA reports whether any configured channel belongs to a TETRA system.
// The polyphase channelizer can only emit one bank-global rate, so a TETRA tap
// (144 kHz) forces the per-tap DDC strategy.
func anyTETRA(channels []ChannelConfig, byName map[string]trunking.System) bool {
	for _, ch := range channels {
		if sys, ok := byName[ch.SystemName]; ok && sys.Protocol == trunking.ProtocolTETRA {
			return true
		}
	}
	return false
}

// tetraChannelReceiver adapts a TETRA receiver + control channel to the engine's
// narrowbandReceiver interface, driving the same resync + lock-stale watchdogs
// the ccdecoder tetraPipeline runs after every Process. The tap sink calls
// Process(iq); this feeds the receiver, credits any decode, and reacquires symbol
// timing after a full tetraResyncTimeout-worth of signal with no CRC-clean CC
// decode (the multislot drought). Runs on the single wideband pump goroutine.
type tetraChannelReceiver struct {
	rx     *tetrarx.Receiver
	cc     *tetra.ControlChannel
	log    *slog.Logger
	system string
	rateHz float64
	now    func() time.Time

	samplesSinceDecode int64
	lastSeenActivity   int64
	lastStale          time.Time

	// Payload-drought trigger, mirrored from ccdecoder.tetraPipeline (the
	// "locked but deaf" escape — sync bursts decoding, zero SCH payload).
	samplesSincePayload int64
	lastSeenPayload     int64
	payloadResyncStreak int
}

func (t *tetraChannelReceiver) Process(iq []complex64) {
	t.rx.Process(iq)
	t.checkResync(len(iq))
	t.checkPayloadResync(len(iq))
	t.checkLockStale()
}

// checkResync reacquires symbol timing after a full window of processed signal
// with no decode. Signal-time (not wall-clock) budget, mirroring
// ccdecoder.tetraPipeline.checkResync so a wideband tap recovers from the same
// multislot decode drought.
func (t *tetraChannelReceiver) checkResync(n int) {
	act := t.cc.LastActivityNano()
	if act == 0 {
		return // no decode has ever landed — nothing to reacquire toward
	}
	if act != t.lastSeenActivity {
		t.lastSeenActivity = act
		t.samplesSinceDecode = 0
		return
	}
	if t.rateHz <= 0 {
		return
	}
	t.samplesSinceDecode += int64(n)
	if t.samplesSinceDecode < int64(tetraResyncTimeout.Seconds()*t.rateHz) {
		return
	}
	t.samplesSinceDecode = 0
	// The DSP was just reset, so the payload drought (if any) restarts too.
	t.samplesSincePayload = 0
	t.rx.Reset()
	t.cc.ResyncReset()
	if t.log != nil {
		t.log.Debug("widebandt2: tetra dsp resync (signal-time decode drought; reacquiring symbol timing from centre)",
			"system", t.system)
	}
}

// checkPayloadResync mirrors ccdecoder.tetraPipeline.checkPayloadResync: the
// escape from the "locked but deaf" state where sync bursts keep the ordinary
// heartbeat fed while every SCH payload block fails CRC (a wrong AFC alias
// latch). Signal-time budget; escalates to MarkLost after
// tetraPayloadResyncMaxAttempts fruitless resets so the supervisor re-hunts.
func (t *tetraChannelReceiver) checkPayloadResync(n int) {
	pay := t.cc.LastPayloadNano()
	if pay != t.lastSeenPayload {
		t.lastSeenPayload = pay
		t.samplesSincePayload = 0
		t.payloadResyncStreak = 0
		return
	}
	if !t.cc.Locked() {
		t.samplesSincePayload = 0
		return
	}
	if t.rateHz <= 0 {
		return
	}
	t.samplesSincePayload += int64(n)
	if t.samplesSincePayload < int64(tetraPayloadResyncTimeout.Seconds()*t.rateHz) {
		return
	}
	t.samplesSincePayload = 0
	t.payloadResyncStreak++
	if t.payloadResyncStreak >= tetraPayloadResyncMaxAttempts {
		t.payloadResyncStreak = 0
		if t.log != nil {
			t.log.Warn("widebandt2: tetra payload drought persists across resyncs (sync bursts decoding, no SCH payload) — declaring lock lost to force a re-hunt",
				"system", t.system, "attempts", tetraPayloadResyncMaxAttempts)
		}
		t.cc.MarkLost()
		return
	}
	t.rx.Reset()
	t.cc.ResyncReset()
	if t.log != nil {
		t.log.Debug("widebandt2: tetra dsp resync (payload drought; sync bursts still decoding but no SCH payload)",
			"system", t.system, "attempt", t.payloadResyncStreak)
	}
}

// checkLockStale runs the control-channel lock watchdog on a light throttle so a
// silent carrier surfaces cc.lost and the supervisor can re-hunt.
func (t *tetraChannelReceiver) checkLockStale() {
	now := t.now()
	if now.Sub(t.lastStale) < tetraStaleCheckInterval {
		return
	}
	t.lastStale = now
	t.cc.CheckStale(now, tetraLockStaleTimeout)
}

// buildTETRAChannel wires a TETRA control-channel tap for the wideband engine,
// mirroring ccdecoder.newTETRAPipeline: a tetra.ControlChannel fed by a
// tetrarx.Receiver (AFC + channel filter + equalizer + soft-decision + DC block
// on, as the dedicated-dongle voice/CC paths use). It publishes cc.locked /
// grant / site.update on the bus exactly like the single-frequency path, so the
// trunking engine's voice-follow allocator handles TETRA grants unchanged.
func buildTETRAChannel(sys trunking.System, ch ChannelConfig, outRateHz float64, bus *events.Bus, log *slog.Logger, now func() time.Time) (*engineChannel, error) {
	freqHz := ch.FrequencyHz
	if err := requireControlChannel(sys, freqHz, "tetra"); err != nil {
		return nil, err
	}
	if now == nil {
		now = time.Now
	}
	clog := log.With("system", sys.Name, "freq_hz", freqHz, "proto", "tetra")
	cc := tetra.New(tetra.Options{
		Bus:         bus,
		Log:         clog,
		SystemName:  sys.Name,
		FrequencyHz: freqHz,
	})
	codingMode, ok := tetra.ParseChannelCoding(sys.TETRAChannelCoding)
	if !ok {
		clog.Warn("widebandt2: unrecognised tetra_channel_coding; falling back to on", "value", sys.TETRAChannelCoding)
	}
	cc.SetChannelCoding(codingMode)
	if codingMode == tetra.ChannelCodingOn {
		chType, chOK := tetra.ParseChannelType(sys.TETRAChannel)
		if !chOK {
			clog.Warn("widebandt2: unrecognised tetra_channel; falling back to SCH/HD", "value", sys.TETRAChannel)
		}
		cc.SetExpectedChannel(chType)
		cc.SetColourCode(sys.TETRAColourCode)
	}
	clockMode, clockOK := tetrarx.ParseClockMode(sys.TETRAClockMode)
	if !clockOK {
		clog.Warn("widebandt2: unrecognised tetra_clock_mode; falling back to gardner", "value", sys.TETRAClockMode)
	}
	rx := tetrarx.New(tetrarx.Options{
		SampleRateHz: outRateHz,
		DibitSink:    func(dibits []uint8, baseIdx int) { cc.Process(dibits, baseIdx) },
		SoftSink:     func(diffs []complex64, baseIdx int) { cc.StashSoft(diffs, baseIdx) },
		ClockMode:    clockMode,
		GardnerGain:  0.005,
		// The wideband DDC has no AFC and channelizes far wider than a 25 kHz
		// TETRA channel; these mirror the ccdecoder pipeline's on-air settings.
		EnableAFC:           true,
		EnableChannelFilter: true,
		EnableEqualizer:     true,
		EnableDCBlock:       true,
	})
	rcv := &tetraChannelReceiver{
		rx: rx, cc: cc, log: clog, system: sys.Name, rateHz: outRateHz, now: now,
	}
	return &engineChannel{
		freqHz:    freqHz,
		sysName:   sys.Name,
		protoTag:  "tetra",
		processor: cc,
		receiver:  rcv,
		decoded:   func() uint64 { ok, _ := cc.BSCHCounts(); return uint64(ok) },
	}, nil
}
