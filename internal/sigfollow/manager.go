package sigfollow

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	p25p2rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/wbvoice"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// gardnerGain matches composer.p25p2VoiceGardnerGain — the H-DQPSK
// symbol clock slips differently than C4FM, so the loop runs slower than
// the receiver default.
const gardnerGain = 0.005

// Follow-lifecycle bounds.
const (
	// defaultIdleTimeout ends a follow after this long with no decoded
	// superframe — the call has dropped (or never decoded, e.g. a wrong
	// scrambler config), so the tap is freed. Aliases ride FACCH-S during
	// hangtime, which immediately follows the voice payload, so a few
	// seconds covers the gap between voice end and the alias burst.
	defaultIdleTimeout = 4 * time.Second
	// defaultMaxDuration caps a single follow so a stuck decode can't pin
	// a signalling tap forever; generous enough for a long transmission
	// to reach its hangtime alias.
	defaultMaxDuration = 30 * time.Second
)

// Manager subscribes to the event bus and, for each in-window P25 Phase 2
// grant, runs a signalling-only decode on a free wideband DDC tap to
// harvest talker aliases — independent of the voice pool and of the
// encrypted-call teardown. This is the architectural fix for issue #376:
// alias capture is decoupled from voice-following, matching how SDRTrunk
// harvests aliases off the traffic channel's signalling stream.
type Manager struct {
	bus *events.Bus
	log *slog.Logger

	// tuners is the signalling-tap pool: standalone wideband DDC taps not
	// registered in the voice pool. A follow borrows a free tuner whose
	// IQ window covers the grant frequency.
	tuners []*wbvoice.VirtualTuner

	idleTimeout time.Duration
	maxDuration time.Duration

	mu         sync.Mutex
	busy       map[string]bool               // tuner serial -> in use
	active     map[uint32]context.CancelFunc // grant frequency -> follow cancel
	voiceFreqs map[uint32]int                // frequency -> active voice-call count

	wg      sync.WaitGroup
	runDone chan struct{}
}

// Options configures a Manager.
type Options struct {
	// Bus carries the grants the follower acts on and the talker-alias
	// events it publishes. Required.
	Bus *events.Bus
	// Log labels diagnostics. Optional; defaults to slog.Default().
	Log *slog.Logger
	// Tuners is the signalling-tap pool. Empty disables the follower
	// (every grant is ignored) — the daemon builds these per `role:
	// wideband` device from its `signalling_taps` count.
	Tuners []*wbvoice.VirtualTuner
	// IdleTimeout / MaxDuration override the follow-lifecycle bounds; zero
	// uses the defaults.
	IdleTimeout time.Duration
	MaxDuration time.Duration
}

// New returns a Manager. Returns an error when Bus is missing.
func New(opts Options) (*Manager, error) {
	if opts.Bus == nil {
		return nil, errors.New("sigfollow: Bus is required")
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	idle := opts.IdleTimeout
	if idle <= 0 {
		idle = defaultIdleTimeout
	}
	maxDur := opts.MaxDuration
	if maxDur <= 0 {
		maxDur = defaultMaxDuration
	}
	return &Manager{
		bus:         opts.Bus,
		log:         log,
		tuners:      opts.Tuners,
		idleTimeout: idle,
		maxDuration: maxDur,
		busy:        make(map[string]bool),
		active:      make(map[uint32]context.CancelFunc),
		voiceFreqs:  make(map[uint32]int),
		runDone:     make(chan struct{}),
	}, nil
}

// Run drains the bus, starting and stopping signalling follows, until ctx
// cancels or the bus closes. It blocks until every in-flight follow has
// unwound before returning.
func (m *Manager) Run(ctx context.Context) error {
	defer close(m.runDone)
	sub := m.bus.Subscribe()
	defer sub.Close()
	for {
		select {
		case <-ctx.Done():
			m.wg.Wait()
			return ctx.Err()
		case ev, ok := <-sub.C:
			if !ok {
				m.wg.Wait()
				return nil
			}
			switch ev.Kind {
			case events.KindGrant:
				if g, ok := ev.Payload.(trunking.Grant); ok {
					m.onGrant(ctx, g)
				}
			case events.KindCallStart:
				if cs, ok := ev.Payload.(trunking.CallStart); ok {
					m.onVoiceStart(cs.Grant.FrequencyHz)
				}
			case events.KindCallEnd:
				if ce, ok := ev.Payload.(trunking.CallEnd); ok {
					m.onVoiceEnd(ce.Grant.FrequencyHz)
				}
			}
		}
	}
}

// onGrant starts a signalling follow for an in-window Phase 2 grant that
// isn't already being followed and isn't covered by a voice call.
func (m *Manager) onGrant(ctx context.Context, g trunking.Grant) {
	if g.Protocol != "p25-phase2" || g.FrequencyHz == 0 {
		return
	}
	m.mu.Lock()
	if _, following := m.active[g.FrequencyHz]; following {
		m.mu.Unlock()
		return
	}
	if m.voiceFreqs[g.FrequencyHz] > 0 {
		m.mu.Unlock()
		return // a voice chain already harvests aliases on this channel
	}
	vt := m.allocTunerLocked(g.FrequencyHz)
	if vt == nil {
		m.mu.Unlock()
		return // outside every tap's window, or all signalling taps busy
	}
	fctx, cancel := context.WithCancel(ctx)
	m.active[g.FrequencyHz] = cancel
	m.mu.Unlock()

	m.wg.Add(1)
	go m.follow(fctx, vt, g)
}

// allocTunerLocked claims the first free signalling tuner whose window
// covers freq. Caller holds m.mu.
func (m *Manager) allocTunerLocked(freq uint32) *wbvoice.VirtualTuner {
	for _, t := range m.tuners {
		if m.busy[t.Serial()] || !t.CanTune(freq) {
			continue
		}
		m.busy[t.Serial()] = true
		return t
	}
	return nil
}

// follow decodes the Phase 2 traffic channel at the grant frequency for
// talker-alias MAC signalling until it idles out, hits the duration cap,
// or ctx cancels (call ended, voice chain took over, or shutdown).
func (m *Manager) follow(ctx context.Context, vt *wbvoice.VirtualTuner, g trunking.Grant) {
	defer m.wg.Done()
	defer m.releaseFollow(g.FrequencyHz, vt.Serial())
	defer gtlog.Recover(m.log, "sigfollow:"+vt.Serial(), nil)

	if err := vt.SetCenterFreq(g.FrequencyHz); err != nil {
		m.log.Debug("sigfollow: tap cannot tune grant",
			"serial", vt.Serial(), "freq", g.FrequencyHz, "err", err)
		return
	}
	iqCh, err := vt.StreamIQ(ctx)
	if err != nil {
		m.log.Warn("sigfollow: StreamIQ failed", "serial", vt.Serial(), "err", err)
		return
	}
	rateHz := vt.SampleRateExactHz()
	if rateHz <= 0 {
		rateHz = float64(vt.SampleRateHz())
	}

	macCfg := p25p2.MACDecodeConfig{
		Trellis:      p25p2.TrellisMode(g.P25Phase2Decode.Trellis),
		RS:           p25p2.RSMode(g.P25Phase2Decode.RS),
		Interleave:   p25p2.InterleaveMode(g.P25Phase2Decode.Interleave),
		Scrambler:    p25p2.ScramblerMode(g.P25Phase2Decode.Scrambler),
		Seed:         g.P25Phase2Decode.Seed,
		SoftDecision: g.P25Phase2Decode.SoftDecision,
		Equalizer:    g.P25Phase2Decode.Equalizer,
	}

	sfDec := p25p2.NewSuperframeDecoder()
	dispatcher := NewMACDispatcher(MACDispatcherOptions{
		Bus:       m.bus,
		Log:       m.log,
		LogPrefix: "sigfollow",
		System:    g.System,
		Serial:    vt.Serial(),
		// No OnCallSource / OnCallEncryption: a signalling follow has no
		// bound call to backfill, so it harvests talker aliases only (the
		// affiliation tracker binds those by System + SourceID).
	})

	var lastActivity atomic.Int64
	lastActivity.Store(time.Now().UnixNano())

	// drainSuperframes is shared by the hard DibitSink and, when the grant
	// requests soft-decision demod (issue #915), the soft SoftSink whose
	// superframes carry per-symbol soft differentials in Subframe.Soft.
	drainSuperframes := func(sfs []p25p2.Superframe) {
		for _, sf := range sfs {
			lastActivity.Store(time.Now().UnixNano())
			dispatcher.Dispatch(sf, macCfg)
		}
	}
	rxOpts := p25p2rx.Options{
		SampleRateHz: rateHz,
		ClockMode:    p25p2rx.ClockGardner,
		GardnerGain:  gardnerGain,
		Equalizer:    macCfg.Equalizer,
	}
	if macCfg.SoftDecision {
		rxOpts.SoftDecision = true
		rxOpts.SoftSink = func(dibits []uint8, soft []complex64, baseIdx int) {
			drainSuperframes(sfDec.ProcessSoft(dibits, soft, baseIdx))
		}
	} else {
		rxOpts.DibitSink = func(dibits []uint8, baseIdx int) {
			drainSuperframes(sfDec.Process(dibits, baseIdx))
		}
	}
	rx := p25p2rx.New(rxOpts)

	m.log.Info("sigfollow: p25p2 signalling follow started",
		"system", g.System, "serial", vt.Serial(),
		"freq", g.FrequencyHz, "group", g.GroupID,
		"scrambler", macCfg.Scrambler, "seed", macCfg.Seed)

	idle := time.NewTicker(m.idleTimeout / 2)
	defer idle.Stop()
	hardCap := time.NewTimer(m.maxDuration)
	defer hardCap.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-hardCap.C:
			m.log.Debug("sigfollow: follow hit max duration",
				"serial", vt.Serial(), "freq", g.FrequencyHz)
			return
		case <-idle.C:
			if time.Since(time.Unix(0, lastActivity.Load())) > m.idleTimeout {
				m.log.Debug("sigfollow: follow idle, ending",
					"serial", vt.Serial(), "freq", g.FrequencyHz)
				return
			}
		case iq, ok := <-iqCh:
			if !ok {
				return
			}
			rx.Process(iq)
		}
	}
}

// releaseFollow frees the tap and clears the per-frequency follow slot.
// Idempotent: a follow cancelled by onVoiceStart and then unwinding here
// double-clears harmlessly.
func (m *Manager) releaseFollow(freq uint32, serial string) {
	m.mu.Lock()
	if cancel, ok := m.active[freq]; ok {
		cancel()
		delete(m.active, freq)
	}
	delete(m.busy, serial)
	m.mu.Unlock()
}

// onVoiceStart marks a frequency as voice-followed and cancels any
// redundant signalling follow on it — the voice chain harvests aliases on
// that channel itself.
func (m *Manager) onVoiceStart(freq uint32) {
	if freq == 0 {
		return
	}
	m.mu.Lock()
	m.voiceFreqs[freq]++
	cancel := m.active[freq]
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// onVoiceEnd clears a frequency's voice-followed mark.
func (m *Manager) onVoiceEnd(freq uint32) {
	if freq == 0 {
		return
	}
	m.mu.Lock()
	if m.voiceFreqs[freq] > 0 {
		m.voiceFreqs[freq]--
		if m.voiceFreqs[freq] == 0 {
			delete(m.voiceFreqs, freq)
		}
	}
	m.mu.Unlock()
}
