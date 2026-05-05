package toneout

import (
	"errors"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// Options configure a Detector.
type Options struct {
	Bus        *events.Bus
	Profiles   []Profile
	SampleRate uint32 // PCM sample rate (must match composer; 8000 typical)
	BlockSize  int    // samples per Goertzel block; default 800 → 100 ms at 8 kHz
	Log        *slog.Logger
	// Now is injectable for tests; defaults to time.Now.
	Now func() time.Time
}

// Detector consumes 16-bit PCM samples from the voice composer (it
// satisfies the same composer.PCMSink shape: WritePCM(serial,
// samples)) and emits events.KindToneAlert when a profile matches.
//
// The detector keeps per-device state so concurrent calls on
// different SDRs don't cross-contaminate match progress.
type Detector struct {
	bus        *events.Bus
	log        *slog.Logger
	profiles   []Profile
	sampleRate uint32
	blockSize  int
	blockDur   time.Duration
	now        func() time.Time

	// Each unique target frequency across all profiles gets one
	// Goertzel filter slot. Profiles store indexes into this table.
	freqIdx     map[float64]int
	freqValues  []float64
	profileBins [][]int // profileBins[i][toneIdx] -> index into freqValues

	mu     sync.Mutex
	states map[string]*deviceState
}

// deviceState is the Goertzel + match state for one Voice device serial.
type deviceState struct {
	gs        []*Goertzel    // per unique frequency
	progress  []*matchProgress // per profile
	lastBlock time.Time
}

// matchProgress tracks one profile's match state on one device.
type matchProgress struct {
	toneIdx          int
	contiguousBlocks int
	matchedFreqs     []float64
	gapBlocks        int
	lastFiredAt      time.Time
}

// New validates options and returns a ready-to-use Detector.
func New(opts Options) (*Detector, error) {
	if opts.Bus == nil {
		return nil, errors.New("toneout: events.Bus is required")
	}
	if opts.SampleRate == 0 {
		opts.SampleRate = 8000
	}
	if opts.BlockSize <= 0 {
		opts.BlockSize = int(opts.SampleRate) / 10 // 100 ms
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	for i := range opts.Profiles {
		if err := opts.Profiles[i].Validate(); err != nil {
			return nil, err
		}
	}

	d := &Detector{
		bus:        opts.Bus,
		log:        log,
		profiles:   opts.Profiles,
		sampleRate: opts.SampleRate,
		blockSize:  opts.BlockSize,
		blockDur:   time.Duration(float64(time.Second) * float64(opts.BlockSize) / float64(opts.SampleRate)),
		now:        opts.Now,
		freqIdx:    make(map[float64]int),
		states:     make(map[string]*deviceState),
	}
	// Index unique target frequencies (rounded) and remember per
	// (profile, tone) which slot to read.
	d.profileBins = make([][]int, len(opts.Profiles))
	for pi, p := range opts.Profiles {
		bins := make([]int, len(p.Tones))
		for ti, t := range p.Tones {
			key := math.Round(t.FrequencyHz*10) / 10 // 0.1 Hz key — coalesces nearly-identical entries
			idx, ok := d.freqIdx[key]
			if !ok {
				idx = len(d.freqValues)
				d.freqIdx[key] = idx
				d.freqValues = append(d.freqValues, t.FrequencyHz)
			}
			bins[ti] = idx
		}
		d.profileBins[pi] = bins
	}
	return d, nil
}

// Profiles returns the configured list (copy).
func (d *Detector) Profiles() []Profile {
	out := make([]Profile, len(d.profiles))
	copy(out, d.profiles)
	return out
}

// WritePCM is the composer.PCMSink interface. Samples are pushed into
// the device's per-frequency Goertzel filters; when block boundaries
// land, every profile's matcher is advanced. Always returns nil so
// composer chains using a fan-out sink don't abort on detector
// hiccups.
func (d *Detector) WritePCM(deviceSerial string, samples []int16) error {
	if len(d.profiles) == 0 {
		return nil
	}
	st := d.stateFor(deviceSerial)

	for _, s := range samples {
		// Feed every Goertzel; collect the (per-frequency) magnitude
		// when a block completes.
		var mags []float64
		ready := false
		for i, g := range st.gs {
			m, r := g.Process(s)
			if r {
				if mags == nil {
					mags = make([]float64, len(st.gs))
				}
				mags[i] = m
				ready = true
			}
			_ = i
		}
		if !ready {
			continue
		}
		d.advanceProfiles(deviceSerial, st, mags)
	}
	return nil
}

// stateFor returns or lazy-creates the per-device state struct.
func (d *Detector) stateFor(serial string) *deviceState {
	d.mu.Lock()
	defer d.mu.Unlock()
	if st, ok := d.states[serial]; ok {
		return st
	}
	st := &deviceState{
		gs:       make([]*Goertzel, len(d.freqValues)),
		progress: make([]*matchProgress, len(d.profiles)),
	}
	for i, f := range d.freqValues {
		st.gs[i] = NewGoertzel(f, float64(d.sampleRate), d.blockSize)
	}
	for i := range st.progress {
		st.progress[i] = &matchProgress{}
	}
	d.states[serial] = st
	return st
}

// ResetDevice drops all per-profile state for the given serial. Called
// when a call ends so the next call starts fresh.
func (d *Detector) ResetDevice(serial string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if st, ok := d.states[serial]; ok {
		for i := range st.progress {
			st.progress[i] = &matchProgress{lastFiredAt: st.progress[i].lastFiredAt}
		}
	}
}

func (d *Detector) advanceProfiles(serial string, st *deviceState, mags []float64) {
	now := d.now()
	for pi, profile := range d.profiles {
		bins := d.profileBins[pi]
		prog := st.progress[pi]
		if prog.toneIdx >= len(profile.Tones) {
			prog.toneIdx = 0
			prog.contiguousBlocks = 0
			prog.matchedFreqs = nil
			prog.gapBlocks = 0
		}
		expected := profile.Tones[prog.toneIdx]
		mag := mags[bins[prog.toneIdx]]

		// Cooldown: if we're in a refractory window, ignore the profile
		// entirely until it expires.
		if !prog.lastFiredAt.IsZero() && now.Sub(prog.lastFiredAt) < profile.Cooldown {
			continue
		}

		if mag >= profile.MagnitudeThreshold {
			prog.contiguousBlocks++
			prog.gapBlocks = 0
		} else {
			// Tone not present this block.
			if prog.contiguousBlocks > 0 {
				// We were in the middle of a tone. Decide if it counted.
				dur := time.Duration(prog.contiguousBlocks) * d.blockDur
				if dur >= expected.MinDuration &&
					(expected.MaxDuration == 0 || dur <= expected.MaxDuration) {
					// Tone matched. Record the achieved frequency (here
					// the configured target; live-frequency refinement
					// is a follow-up) and advance the sequence.
					prog.matchedFreqs = append(prog.matchedFreqs, expected.FrequencyHz)
					prog.toneIdx++
					prog.contiguousBlocks = 0
					if prog.toneIdx >= len(profile.Tones) {
						d.fire(serial, profile, prog, now)
						prog.toneIdx = 0
						prog.matchedFreqs = nil
						prog.lastFiredAt = now
						continue
					}
				} else {
					// Too short. Reset entire sequence (we may have
					// drifted into noise; restart from tone 0).
					prog.toneIdx = 0
					prog.contiguousBlocks = 0
					prog.matchedFreqs = nil
				}
			} else if prog.toneIdx > 0 {
				// We're between tones in a sequence. Track the gap.
				prog.gapBlocks++
				if time.Duration(prog.gapBlocks)*d.blockDur > profile.MaxGap {
					// Gap too long; reset.
					prog.toneIdx = 0
					prog.matchedFreqs = nil
					prog.gapBlocks = 0
				}
			}
		}
	}
}

func (d *Detector) fire(serial string, p Profile, prog *matchProgress, now time.Time) {
	freqs := append([]float64(nil), prog.matchedFreqs...)
	d.bus.Publish(events.Event{
		Kind: events.KindToneAlert,
		Payload: Alert{
			Profile:       p.Name,
			AlphaTag:      p.AlphaTag,
			System:        p.System,
			DeviceSerial:  serial,
			MatchedAt:     now,
			FrequenciesHz: freqs,
		},
	})
	d.log.Info("toneout: profile matched",
		"profile", p.Name, "device", serial, "tones", freqs)
}
