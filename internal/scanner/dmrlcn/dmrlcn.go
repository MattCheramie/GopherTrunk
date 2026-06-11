package dmrlcn

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier3"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/iqtap"
)

const (
	// maxConcurrentProbes bounds simultaneous decode-confirmation taps so
	// the learner never spins up an unbounded number of DDC receivers.
	maxConcurrentProbes = 2
	// candidateBuffer is the depth of the onset→probe handoff queue.
	candidateBuffer = 4
	// expireInterval is how often stale pending grants are swept.
	expireInterval = time.Second
)

// Options configures a Learner.
type Options struct {
	Log    *slog.Logger
	Bus    *events.Bus
	Broker *iqtap.Broker

	// CenterHz / SampleRateHz describe the wideband dongle the broker
	// fans out. Required.
	CenterHz     uint32
	SampleRateHz uint32
	GuardFrac    float64

	// System is the trunking system name this learner serves; only
	// KindDMRGrantObserved events whose System matches are consumed.
	System string

	// SetResolver is bound to the running tier3.ControlChannel's
	// SetResolver so a learned plan applies live. Required for live
	// application; nil is allowed (learn + publish only).
	SetResolver func(tier3.Resolver)

	// Persist, if non-nil, is invoked once with the locked fit so the
	// caller can write the plan back to config.
	Persist func(FitResult)

	// Confirmer overrides the decode-confirmation strategy. When nil a
	// real DDC-tap confirmer is built from Broker/CenterHz/SampleRateHz.
	Confirmer Confirmer

	// FitOptions overrides the band-plan fitter tuning (zero = defaults).
	FitOptions FitOptions

	// ExcludeHz lists frequencies the onset detector must never report —
	// typically the always-on control-channel carriers on this dongle, so
	// they don't generate phantom onset edges.
	ExcludeHz []uint32

	Now func() time.Time
}

// Learner discovers a DMR Tier III system's LCN→frequency band plan from
// observed voice grants and the RF carriers that key up in response, then
// hot-swaps the learned plan into the running control channel. One
// Learner serves one wideband dongle / system. Run owns its goroutines
// and returns when its context is cancelled.
type Learner struct {
	log         *slog.Logger
	bus         *events.Bus
	broker      *iqtap.Broker
	system      string
	setResolver func(tier3.Resolver)
	persist     func(FitResult)
	confirmer   Confirmer
	fitOpts     FitOptions
	onsetCfg    onsetConfig
	now         func() time.Time

	corr *correlator

	mu     sync.Mutex
	pairs  []Pair
	locked bool
}

// New constructs a Learner. Broker, CenterHz and SampleRateHz are
// required.
func New(opts Options) *Learner {
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	guard := opts.GuardFrac
	if guard <= 0 {
		guard = 0.05
	}

	confirmer := opts.Confirmer
	if confirmer == nil && opts.Broker != nil {
		confirmer = newDDCConfirmer(opts.Broker, opts.CenterHz, opts.SampleRateHz, guard)
	}

	oc := onsetConfig{
		CenterHz:     opts.CenterHz,
		SampleRateHz: opts.SampleRateHz,
		GuardFrac:    guard,
		Exclude:      opts.ExcludeHz,
		Now:          now,
	}

	return &Learner{
		log:         log.With("subsystem", "dmrlcn", "system", opts.System),
		bus:         opts.Bus,
		broker:      opts.Broker,
		system:      opts.System,
		setResolver: opts.SetResolver,
		persist:     opts.Persist,
		confirmer:   confirmer,
		fitOpts:     opts.FitOptions,
		onsetCfg:    oc,
		now:         now,
		corr:        newCorrelator(0, 0, 0),
	}
}

// Run drives the learner until ctx is cancelled. It subscribes to the
// wideband IQ broker (onset detection) and the event bus (grant
// observations), schedules bounded decode-confirmation probes, and
// applies the band plan once the fitter locks. Run never returns an
// error; failures are logged and the learner degrades to a no-op.
func (l *Learner) Run(ctx context.Context) error {
	if l.bus == nil || l.broker == nil {
		l.log.Warn("dmrlcn: missing bus or broker; learner disabled")
		<-ctx.Done()
		return nil
	}

	iqSub := l.broker.Subscribe()
	defer iqSub.Close()
	busSub := l.bus.Subscribe()
	defer busSub.Close()

	candidates := make(chan Candidate, candidateBuffer)
	results := make(chan Pair, maxConcurrentProbes)

	detector := newOnsetDetector(l.onsetCfg, func(o CarrierOnset) {
		if l.isLocked() {
			return
		}
		cand, ok := l.corr.matchOnset(o)
		if !ok {
			return
		}
		select {
		case candidates <- cand:
		default:
			// Probe queue full; this LCN will be observed again.
		}
	})

	// Probe workers: confirm candidates off the main loop so a ~700 ms
	// DDC tap never stalls IQ pumping into the onset detector.
	var wg sync.WaitGroup
	for i := 0; i < maxConcurrentProbes; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case cand, ok := <-candidates:
					if !ok {
						return
					}
					if l.isLocked() {
						continue
					}
					pair, ok := l.confirmCandidate(ctx, cand)
					if !ok {
						continue
					}
					select {
					case results <- pair:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	ticker := time.NewTicker(expireInterval)
	defer ticker.Stop()

	l.log.Info("dmrlcn: learning DMR band plan", "center_hz", l.onsetCfg.CenterHz, "sample_rate_hz", l.onsetCfg.SampleRateHz)

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return nil
		case chunk, ok := <-iqSub.C:
			if !ok {
				wg.Wait()
				return nil
			}
			if !l.isLocked() {
				detector.Process(chunk)
			}
		case ev := <-busSub.C:
			if ev.Kind != events.KindDMRGrantObserved {
				continue
			}
			obs, ok := ev.Payload.(events.DMRGrantObserved)
			if !ok || obs.System != l.system {
				continue
			}
			l.corr.addGrant(obs)
		case pair := <-results:
			l.handlePair(pair)
		case <-ticker.C:
			l.corr.expire(l.now())
		}
	}
}

// confirmCandidate runs decode-confirmation on a candidate and returns
// the trusted fit Pair. With no confirmer configured it accepts the
// candidate at a reduced weight (temporal-only evidence).
func (l *Learner) confirmCandidate(ctx context.Context, cand Candidate) (Pair, bool) {
	if l.confirmer == nil {
		return Pair{LCN: cand.LCN, FreqHz: cand.FreqHz, Weight: 0.4}, true
	}
	ok, weight := l.confirmer.Confirm(ctx, cand.FreqHz)
	if !ok {
		return Pair{}, false
	}
	return Pair{LCN: cand.LCN, FreqHz: cand.FreqHz, Weight: weight}, true
}

// handlePair records a confirmed pair and re-runs the fitter, applying
// and publishing the plan the first time it locks.
func (l *Learner) handlePair(p Pair) {
	l.mu.Lock()
	if l.locked {
		l.mu.Unlock()
		return
	}
	l.pairs = append(l.pairs, p)
	res, ok := FitBandPlan(l.pairs, l.fitOpts)
	if !ok {
		l.mu.Unlock()
		return
	}
	l.locked = true
	l.mu.Unlock()

	l.apply(res)
}

// apply hot-swaps the learned resolver, publishes the learned event,
// logs, and triggers optional persistence.
func (l *Learner) apply(res FitResult) {
	plan := res.BandPlan()
	if l.setResolver != nil {
		l.setResolver(tier3.ResolverFromPlan(plan))
	}
	if l.bus != nil {
		l.bus.Publish(events.Event{Kind: events.KindDMRBandPlanLearned, Payload: l.learnedPayload(res)})
	}
	if res.Linear != nil {
		l.log.Info("dmrlcn: band plan learned (linear)",
			"base_hz", res.Linear.BaseHz, "spacing_hz", res.Linear.SpacingHz,
			"offset", res.Linear.Offset, "pairs", res.NumPairs,
			"confidence", res.Confidence, "residual_hz", res.ResidualHz)
	} else {
		l.log.Info("dmrlcn: band plan learned (table)",
			"entries", len(res.Table), "pairs", res.NumPairs, "confidence", res.Confidence)
	}
	if l.persist != nil {
		l.persist(res)
	}
}

func (l *Learner) learnedPayload(res FitResult) events.DMRBandPlanLearned {
	p := events.DMRBandPlanLearned{
		System:     l.system,
		NumPairs:   res.NumPairs,
		Confidence: res.Confidence,
		ResidualHz: res.ResidualHz,
	}
	if res.Linear != nil {
		p.BaseHz = res.Linear.BaseHz
		p.SpacingHz = res.Linear.SpacingHz
		p.Offset = res.Linear.Offset
	} else {
		for _, e := range res.Table {
			p.Table = append(p.Table, events.DMRBandPlanLCN{LCN: e.LCN, FreqHz: e.FreqHz})
		}
	}
	return p
}

func (l *Learner) isLocked() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.locked
}

// Locked reports whether the learner has committed a band plan.
func (l *Learner) Locked() bool { return l.isLocked() }
