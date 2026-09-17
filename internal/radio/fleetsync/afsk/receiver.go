// Package afsk is the DSP front end for Kenwood FleetSync: an IQ stream
// (or already-discriminated FM audio) becomes 1200-baud FFSK audio, then
// a sliced bit stream, then framed FleetSync ANI bursts via
// internal/radio/fleetsync.Framer. The pipeline is:
//
//	IQ chunks (Fs Hz, complex64)
//	  → FM demod (internal/dsp/demod.FM)              [skipped for audio input]
//	  → real resampler (internal/dsp.RealResampler) to AudioRateHz
//	  → FFSK tone discriminator (internal/dsp/demod.FFSK,
//	    markHz=1200, spaceHz=1800 — CCIR FFSK)
//	  → Mueller-Müller symbol-timing recovery
//	    (internal/dsp/sync.MuellerMuller, Oversample sps → 1 sample/symbol)
//	  → zero-threshold slicer (NRZ bit on the wire)
//	  → fleetsync.Framer.Push(bit)
//	  → OnMessage(fleetsync.Message)
//
// Layout mirrors internal/radio/mdc1200/afsk — same modulation class,
// same building blocks, one Receiver per (SDR, frequency) pair. Two
// deliberate differences: the protocol framer stays callback-based (so
// it unit-tests in isolation) and this front end is what publishes onto
// the events bus — with Options.Bus set every framed burst becomes an
// events.KindFleetSyncMessage carrying a storage.FleetSyncMessage, which
// storage.FleetSyncLog persists, GET /api/v1/fleetsync/messages serves
// and the /fleetsync web panel renders; and the baud rate is an option
// (FleetSync signalling is 1200 baud; the reference decoder also
// accepts 2400, so the replay harness can sweep both against a capture
// instead of anyone guessing).
//
// Verification status: decodes the #1184 reporter's two SDR# captures
// (Fleet 107 / Unit 1772, FleetSync-I 5/6 bursts and FleetSync-II 8/8)
// through this exact chain — realair_test.go pins channelized slices of
// them. The daemon wiring itself awaits the reporter's live run.
package afsk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	dspsync "github.com/MattCheramie/GopherTrunk/internal/dsp/sync"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/fleetsync"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// CCIR FFSK tone frequencies — the FleetSync signalling convention
// (1200/1800 Hz, confirmed from the reporter's captures in #437).
const (
	MarkHz  = 1200.0 // binary 1
	SpaceHz = 1800.0 // binary 0
)

// Oversample is the number of FFSK-discriminator samples per bit fed
// into the Mueller-Müller timing-recovery loop. 8 matches the MDC1200 /
// APRS / POCSAG front ends.
const Oversample = 8

// mmGain is the Mueller-Müller loop gain — the conventional AFSK-paired
// starting point the other 1200-baud front ends use.
const mmGain = 0.05

// DefaultBaudHz is the FleetSync signalling rate.
const DefaultBaudHz = 1200

// Options configures a Receiver.
type Options struct {
	// InputRateHz is the sample rate of the IQ (ProcessIQ / Process) or
	// discriminator-audio (ProcessAudio) stream. Required.
	InputRateHz uint32

	// BaudHz is the signalling rate; 0 selects DefaultBaudHz (1200).
	BaudHz int

	// OnMessage receives every framed burst, including CRC-failed ones
	// (Message.CRCOK == false) so a caller can surface marginal signals.
	// Invoked synchronously on the processing goroutine. At least one of
	// OnMessage and Bus must be set; both may be.
	OnMessage func(fleetsync.Message)

	// Bus, when non-nil, receives every framed burst as an
	// events.KindFleetSyncMessage whose payload is a
	// storage.FleetSyncMessage — the daemon's path to the SQLite log,
	// the REST endpoint and the web panel.
	Bus *events.Bus

	// DropBadCRC, when true, keeps block-check-failed bursts off the Bus
	// (OnMessage still sees them). Defaults to false so marginal bursts
	// publish with CRCOK=false and the panel can flag them.
	DropBadCRC bool

	// SourceName is stamped on log lines.
	SourceName string

	// Serial and FrequencyHz identify this receiver's SDR and channel;
	// both are stamped on every published storage.FleetSyncMessage so the
	// log, REST endpoint and panel can say which channel produced an ID
	// (#1184). Optional — a harness or offline decode may leave them zero.
	Serial      string
	FrequencyHz uint32

	// Log is optional; defaults to slog.Default.
	Log *slog.Logger
}

// Receiver runs the FleetSync FFSK decode pipeline against a stream of
// IQ (or audio) chunks. Not safe for concurrent use.
type Receiver struct {
	inputRate uint32
	baud      int
	audioRate uint32
	source    string
	log       *slog.Logger

	fm     *demod.FM
	rsmp   *dsp.RealResampler
	ffsk   *demod.FFSK
	mm     *dspsync.MuellerMuller
	framer *fleetsync.Framer

	onMessage  func(fleetsync.Message)
	bus        *events.Bus
	dropBadCRC bool

	// Scratch buffers reused across chunks so the hot path never
	// allocates.
	demodBuf []float32
	rsmpBuf  []float32
	ffskBuf  []float32
	symBuf   []float32

	samplesSeen   atomic.Uint64
	bitsEmitted   atomic.Uint64
	burstsPublish atomic.Uint64 // events published to the bus
	burstsDropped atomic.Uint64 // block-check failures kept off the bus

	serial string // Options.Serial, stamped on every published message
	freqHz uint32 // Options.FrequencyHz, likewise
}

// New constructs a Receiver. Returns an error if neither OnMessage nor
// Bus is set, InputRateHz is unset, or the input rate cannot be
// resampled to the FFSK audio rate.
func New(opts Options) (*Receiver, error) {
	if opts.OnMessage == nil && opts.Bus == nil {
		return nil, errors.New("fleetsync/afsk: OnMessage or Bus is required")
	}
	if opts.InputRateHz == 0 {
		return nil, errors.New("fleetsync/afsk: InputRateHz is required")
	}
	baud := opts.BaudHz
	if baud == 0 {
		baud = DefaultBaudHz
	}
	if baud < 0 {
		return nil, fmt.Errorf("fleetsync/afsk: bad BaudHz %d", baud)
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}

	audioRate := uint32(baud * Oversample)
	g := gcd(audioRate, opts.InputRateHz)
	L := int(audioRate / g)
	M := int(opts.InputRateHz / g)
	if L == 0 || M == 0 {
		return nil, fmt.Errorf("fleetsync/afsk: bad resample ratio L=%d M=%d", L, M)
	}
	// Size the anti-alias prototype like the wideband DDC does (~12 taps
	// per unit of cutoff) so a large decimation (a 48 kHz or 250 kHz
	// capture straight in) still rejects the out-of-band voice/noise
	// energy that would otherwise fold onto the tones; 16 is the floor
	// the other 1200-baud front ends use and 512 a sanity cap.
	tapsPerBranch := (12*M + L - 1) / L
	if tapsPerBranch < 16 {
		tapsPerBranch = 16
	}
	if tapsPerBranch > 512 {
		tapsPerBranch = 512
	}

	r := &Receiver{
		inputRate:  opts.InputRateHz,
		baud:       baud,
		audioRate:  audioRate,
		source:     opts.SourceName,
		log:        log,
		fm:         demod.NewFM(),
		rsmp:       dsp.NewRealResampler(L, M, tapsPerBranch, 7.0),
		ffsk:       demod.NewFFSK(float64(audioRate), MarkHz, SpaceHz),
		mm:         dspsync.NewMuellerMuller(float64(Oversample), mmGain),
		onMessage:  opts.OnMessage,
		bus:        opts.Bus,
		dropBadCRC: opts.DropBadCRC,
		serial:     opts.Serial,
		freqHz:     opts.FrequencyHz,
	}
	r.framer = fleetsync.NewFramer(r.onFrame)
	return r, nil
}

// onFrame is the framer callback: it hands the burst to OnMessage (when
// set) and publishes it on the bus (when set), honouring DropBadCRC for
// the bus only — a caller that asked for every burst still gets them.
func (r *Receiver) onFrame(m fleetsync.Message) {
	if r.onMessage != nil {
		r.onMessage(m)
	}
	if r.bus == nil {
		return
	}
	if !m.CRCOK && r.dropBadCRC {
		r.burstsDropped.Add(1)
		return
	}
	msg := MessageToStorage(m, time.Now())
	msg.Serial, msg.FrequencyHz = r.serial, r.freqHz
	r.bus.Publish(events.Event{
		Kind:      events.KindFleetSyncMessage,
		Timestamp: time.Now(),
		Payload:   msg,
	})
	r.burstsPublish.Add(1)
}

// MessageToStorage converts a decoded burst into the persisted /
// wire shape the bus carries. Exposed so the replay harness and tests
// build the exact payload the daemon publishes.
func MessageToStorage(m fleetsync.Message, at time.Time) storage.FleetSyncMessage {
	return storage.FleetSyncMessage{
		ReceivedAt: at,
		Fleet:      m.Fleet,
		Unit:       m.Unit,
		IsFS2:      m.IsFS2,
		CRCOK:      m.CRCOK,
		RawHex:     m.RawHex,
		Body:       m.Body,
	}
}

// Process pumps IQ chunks from in through the decode pipeline until ctx
// cancels or in closes.
func (r *Receiver) Process(ctx context.Context, in <-chan []complex64) error {
	if in == nil {
		return errors.New("fleetsync/afsk: input channel is nil")
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case chunk, ok := <-in:
			if !ok {
				return nil
			}
			r.ProcessIQ(chunk)
		}
	}
}

// ProcessIQ runs one IQ chunk through FM demod → resample → FFSK
// discrimination → symbol-timing recovery → slice → framer. Messages are
// delivered through OnMessage before it returns.
func (r *Receiver) ProcessIQ(chunk []complex64) {
	r.samplesSeen.Add(uint64(len(chunk)))
	r.demodBuf = r.fm.Process(r.demodBuf, chunk)
	r.processAudio(r.demodBuf)
}

// ProcessAudio runs one chunk of already-discriminated FM audio (real
// samples at InputRateHz — a receiver's discriminator tap or a mono
// audio capture) through the same chain minus the FM demod.
func (r *Receiver) ProcessAudio(chunk []float32) {
	r.samplesSeen.Add(uint64(len(chunk)))
	r.processAudio(chunk)
}

func (r *Receiver) processAudio(audio []float32) {
	r.rsmpBuf = r.rsmp.Process(r.rsmpBuf, audio)
	r.ffskBuf = r.ffsk.Discriminate(r.ffskBuf, r.rsmpBuf)
	r.symBuf = r.mm.Process(r.symBuf, r.ffskBuf)
	for _, s := range r.symBuf {
		r.feedSymbol(s)
	}
}

// feedSymbol slices one recovered symbol to a wire bit at a fixed zero
// threshold and pushes it into the framer. The discriminator output is
// sign-aligned (mark positive) and DC-free by construction — the FFSK
// stage mixes the audio to the tone midpoint and low-passes it, which
// removes the carrier-offset term — so zero is the right threshold, and
// it is what the reference decoder uses (sign of the accumulated
// deviation). A tracked threshold was measured to be HARMFUL here: an
// FS-II frame whose word1 nibbles are small opens with ~68 symbols of
// continuous space tone, a slow bias tracker (1/512 per symbol) drifted
// ~12% of a symbol toward it, and the ISI-attenuated isolated bits that
// follow — the weakest samples in the frame — flipped (23 errors in one
// synthesised 260-bit frame; zero with the fixed threshold). Polarity
// inversion is handled by the framer's complemented-sync lock, not here.
func (r *Receiver) feedSymbol(s float32) {
	var bit byte
	if s > 0 {
		bit = 1
	}
	r.framer.Push(bit)
	r.bitsEmitted.Add(1)
}

// Reset clears every stage's state so a retune or stream restart does
// not bleed the previous stream's filter history into the next.
func (r *Receiver) Reset() {
	r.fm.Reset()
	r.rsmp.Reset()
	r.ffsk.Reset()
	r.mm = dspsync.NewMuellerMuller(float64(Oversample), mmGain)
}

// Framer returns the protocol framer the front end is driving, for its
// Stats().
func (r *Receiver) Framer() *fleetsync.Framer { return r.framer }

// BaudHz reports the configured signalling rate.
func (r *Receiver) BaudHz() int { return r.baud }

// AudioRateHz reports the internal FFSK discriminator rate (baud ×
// Oversample).
func (r *Receiver) AudioRateHz() uint32 { return r.audioRate }

// Stats reports cumulative DSP-front-end counters.
type Stats struct {
	SamplesSeen     uint64 // input samples (IQ or audio) consumed
	BitsEmitted     uint64 // bits handed to the framer
	BurstsPublished uint64 // events published to the bus
	BurstsDropped   uint64 // block-check failures kept off the bus (DropBadCRC)
}

func (r *Receiver) Stats() Stats {
	return Stats{
		SamplesSeen:     r.samplesSeen.Load(),
		BitsEmitted:     r.bitsEmitted.Load(),
		BurstsPublished: r.burstsPublish.Load(),
		BurstsDropped:   r.burstsDropped.Load(),
	}
}

// gcd computes the greatest common divisor via Euclid's algorithm.
func gcd(a, b uint32) uint32 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
