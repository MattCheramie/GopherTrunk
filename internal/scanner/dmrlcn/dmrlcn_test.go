package dmrlcn

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier3"
)

// stubConfirmer confirms any carrier (optionally only a configured set).
type stubConfirmer struct {
	weight float64
	only   map[uint32]bool
}

func (s stubConfirmer) Confirm(_ context.Context, freqHz uint32) (bool, float64) {
	if s.only != nil && !s.only[freqHz] {
		return false, 0
	}
	w := s.weight
	if w == 0 {
		w = 1
	}
	return true, w
}

// feed drives one grant+onset coincidence through the learner's core
// (correlator → confirm → fit), as Run would but synchronously.
func (l *Learner) feed(t *testing.T, lcn uint8, freqHz uint32, at time.Time) {
	t.Helper()
	l.corr.addGrant(events.DMRGrantObserved{System: l.system, LCN: lcn, At: at})
	cand, ok := l.corr.matchOnset(CarrierOnset{FreqHz: freqHz, At: at.Add(300 * time.Millisecond)})
	if !ok {
		return
	}
	pair, ok := l.confirmCandidate(context.Background(), cand)
	if !ok {
		return
	}
	l.handlePair(pair)
}

func TestLearnerLocksAndAppliesPlan(t *testing.T) {
	bus := events.NewBus(32)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	var applied atomic.Pointer[tier3.Resolver]
	var persisted atomic.Int32

	l := New(Options{
		Bus:          bus,
		System:       "S",
		CenterHz:     851_500_000,
		SampleRateHz: 2_400_000,
		Confirmer:    stubConfirmer{weight: 1},
		SetResolver:  func(r tier3.Resolver) { applied.Store(&r) },
		Persist:      func(FitResult) { persisted.Add(1) },
	})

	// Grid: base 851_000_000, spacing 12.5 kHz, offset 1.
	base := time.Unix(1_700_000_000, 0)
	for i, lcn := range []uint8{1, 2, 3, 4, 5} {
		freq := uint32(851_000_000 + int(lcn-1)*12_500)
		l.feed(t, lcn, freq, base.Add(time.Duration(i)*time.Second))
	}

	if !l.Locked() {
		t.Fatal("learner did not lock after 5 confirmed pairs")
	}
	rp := applied.Load()
	if rp == nil {
		t.Fatal("SetResolver was not called")
	}
	// The applied resolver must resolve a known LCN to its grid frequency.
	got, err := (*rp).Frequency(3)
	if err != nil || got != 851_025_000 {
		t.Errorf("applied resolver: Frequency(3) = %d, %v; want 851025000", got, err)
	}
	if persisted.Load() != 1 {
		t.Errorf("persist called %d times, want 1", persisted.Load())
	}

	// A KindDMRBandPlanLearned event must have been published.
	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindDMRBandPlanLearned {
				continue
			}
			p := ev.Payload.(events.DMRBandPlanLearned)
			if p.System != "S" || p.SpacingHz != 12_500 || p.BaseHz != 851_000_000 {
				t.Errorf("learned payload = %+v", p)
			}
			return
		case <-deadline:
			t.Fatal("no KindDMRBandPlanLearned event")
		}
	}
}

func TestLearnerStaysOpenBelowMinLCNs(t *testing.T) {
	l := New(Options{
		System:       "S",
		CenterHz:     851_500_000,
		SampleRateHz: 2_400_000,
		Confirmer:    stubConfirmer{weight: 1},
	})
	base := time.Unix(1_700_000_000, 0)
	for i, lcn := range []uint8{1, 2, 3} { // 3 < default MinLCNs (4)
		l.feed(t, lcn, uint32(851_000_000+int(lcn-1)*12_500), base.Add(time.Duration(i)*time.Second))
	}
	if l.Locked() {
		t.Fatal("learner locked below MinLCNs")
	}
}

func TestLearnerRunStopsOnContextCancel(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	// No broker → Run takes the disabled path and blocks until ctx done.
	l := New(Options{Bus: bus, System: "S", CenterHz: 851_500_000, SampleRateHz: 2_400_000})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- l.Run(ctx) }()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after context cancel")
	}
}
