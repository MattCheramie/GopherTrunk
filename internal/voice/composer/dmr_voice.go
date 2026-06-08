package composer

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
	dmrrx "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/receiver"
	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
)

// dmrVoiceIntermediateHz is the rate the wideband IQ is decimated to
// before the DMR receiver runs. 48 kHz gives the 4800-baud DMR symbol
// stream 10 samples per symbol — ample for the receiver's RRC matched
// filter and Mueller-Müller clock recovery, without the cost of
// running them at the SDR's native multi-MS/s rate.
const dmrVoiceIntermediateHz = 48_000

// rawFrameSink is the subset of voice.Recorder the DMR voice chain
// needs. The composer holds its sink as a PCMSink; runDMRVoiceChain
// type-asserts to this, so a sink that only implements WritePCM
// (analog-only callers, test stubs) still works for the FM path.
type rawFrameSink interface {
	WriteRawFrame(deviceSerial string, frame []byte) error
}

// runDMRVoiceChain consumes IQ for one DMR voice call. It decimates
// the wideband IQ to a DMR-symbol-friendly rate, recovers the dibit
// stream with the shared DMR receiver, assembles A–F voice
// superframes, FEC-decodes each superframe's 18 AMBE+2 frames to
// their 49-bit vocoder payload, and appends them (packed into 7
// bytes) to the recorder's .raw sidecar.
//
// AMBE forward-error-correction is applied per frame
// (dmrvoice.DecodeAMBEFrame): the 72-bit on-air frame is FEC-decoded
// to its 49-bit vocoder payload before being written. Vocoder decode
// to PCM is still out of scope — the .raw sidecar carries the
// post-FEC frames for out-of-band decode (issue #276).
//
// When interleaved is true (System.DMRInterleavedVoice), the carrier is
// decoded as 2-slot TDMA: the interleaved decoder emits a superframe per
// timeslot and a slotRouter keeps only the ones belonging to this call's
// groupID (by embedded-LC talkgroup, then by bound phase). Otherwise the
// single-slot decoder runs and every superframe is written.
//
// slotRouter decides which superframes of a 2-slot interleaved DMR
// carrier belong to this call. A carrier runs two calls (one per
// timeslot) whose burst-A voice sync is identical, so the only reliable
// discriminator is the embedded Link Control: once a superframe's LC
// names this call's talkgroup, its Phase is bound and subsequent
// superframes are routed by Phase even when their LC is absent or fails
// CRC. Superframes whose LC names a different talkgroup are dropped.
type slotRouter struct {
	groupID uint32
	bound   int // -1 until a matching LC binds this call's phase
}

func newSlotRouter(groupID uint32) *slotRouter {
	return &slotRouter{groupID: groupID, bound: -1}
}

// accept reports whether sf belongs to this call's timeslot.
func (r *slotRouter) accept(sf dmrvoice.VoiceSuperframe) bool {
	if sf.HasLC {
		if gv, ok := sf.LC.AsGroupVoiceUser(); ok {
			if gv.GroupAddress == r.groupID {
				r.bound = int(sf.Phase)
				return true
			}
			return false // LC names a different talkgroup — the other slot
		}
	}
	// No usable group LC this superframe: accept only once our phase is
	// known, and only for that phase.
	return r.bound >= 0 && int(sf.Phase) == r.bound
}

func (c *Composer) runDMRVoiceChain(ctx context.Context, serial string, iqCh <-chan []complex64, iqHz uint32, groupID uint32, interleaved bool, done chan<- struct{}) {
	defer close(done)

	// Shared boundary controller: universal hangtime end-of-call + Touch
	// heartbeat. Talkgroup gating is left disabled (grantTG 0) until the
	// DMR chain surfaces a per-superframe embedded-LC talkgroup.
	bt := c.newBoundaryTracker(serial, 0, nil)
	go bt.run(ctx)

	decim := int(iqHz) / dmrVoiceIntermediateHz
	if decim < 1 {
		decim = 1
	}
	symbolHz := float64(iqHz) / float64(decim)

	// Front-end LPF: doubles as the anti-aliasing filter for the
	// decimation, so it is only needed when the IQ is actually
	// decimated (the live multi-MS/s path; decim == 1 only in tests
	// that feed IQ already at the intermediate rate).
	cutoff := float64(c.bw) / float64(iqHz)
	if cutoff > 0.45 {
		cutoff = 0.45
	}
	lpf := filter.NewFIR(filter.LowpassKaiser(81, cutoff, 8.6))

	rs, _ := c.sink.(rawFrameSink)
	voiceDec := dmrvoice.NewDecoder()
	var router *slotRouter
	if interleaved {
		voiceDec = dmrvoice.NewInterleavedDecoder()
		router = newSlotRouter(groupID)
	}
	// superframes counts DMR voice superframes the receiver delivered —
	// i.e. real voice activity. The touch ticker (below) only refreshes
	// the engine's LastHeardAt when this counter has advanced since the
	// previous tick. Without this gate a stalled decoder still kept the
	// call alive forever via an unconditional 1 s heartbeat (issue #356).
	var superframes atomic.Uint64
	// Decode-quality telemetry — see runP25Phase1VoiceChain for the
	// rationale. A high uncorrectable AMBE-frame rate is the measurable
	// signature of weak signal / wrong gain behind garbled audio (issue
	// #356 follow-up).
	var (
		uncorrectableFrames atomic.Uint64
		corrErrBits         atomic.Uint64
	)
	rx := dmrrx.New(dmrrx.Options{
		SampleRateHz: symbolHz,
		// DMR spec peak deviation per ETSI TS 102 361-1 §6.3 — matches
		// the control-channel receiver in internal/scanner/ccdecoder.
		DeviationHz: 1944.0,
		ClockGain:   0.025,
		DibitSink: func(dibits []uint8, baseIdx int) {
			for _, sf := range voiceDec.Process(dibits, baseIdx) {
				if router != nil && !router.accept(sf) {
					// A superframe from the other timeslot (or an
					// as-yet-unbound phase) — not this call's audio.
					continue
				}
				superframes.Add(1)
				bt.onVoice(0)
				if rs == nil {
					continue
				}
				for i := range sf.Frames {
					info, errBits, err := dmrvoice.DecodeAMBEFrame(sf.Frames[i])
					if errBits > 0 {
						corrErrBits.Add(uint64(errBits))
					}
					if err != nil {
						uncorrectableFrames.Add(1)
						c.log.Debug("composer: DMR AMBE FEC decode failed",
							"serial", serial, "err", err)
						continue
					}
					if err := rs.WriteRawFrame(serial, packBits(info)); err != nil {
						c.log.Warn("composer: DMR raw-frame write failed",
							"serial", serial, "err", err)
					}
				}
			}
		},
	})

	touchTicker := time.NewTicker(c.touchEvery)
	defer touchTicker.Stop()
	// logDecodeQuality emits a rolling decode-quality summary, gated to a
	// burst of superframes so it does not spam the log every touch tick
	// (issue #356 follow-up). See runP25Phase1VoiceChain.
	var lastQualityLogSuperframes uint64
	const qualityLogEverySuperframes = 25
	logDecodeQuality := func(final bool) {
		n := superframes.Load()
		if n == 0 || (!final && n-lastQualityLogSuperframes < qualityLogEverySuperframes) {
			return
		}
		lastQualityLogSuperframes = n
		c.log.Info("composer: dmr decode quality",
			"serial", serial,
			"superframes", n, "uncorrectable_frames", uncorrectableFrames.Load(),
			"corrected_bit_errs", corrErrBits.Load())
	}

	for {
		select {
		case <-ctx.Done():
			logDecodeQuality(true)
			return
		case <-touchTicker.C:
			// Touch + hangtime end-of-call handled by the shared boundary
			// tracker; this ticker only drives the decode-quality summary.
			logDecodeQuality(false)
		case iq, ok := <-iqCh:
			if !ok {
				logDecodeQuality(true)
				return
			}
			samples := iq
			if decim > 1 {
				samples = decimateComplex(lpf.Process(nil, iq), decim)
			}
			rx.Process(samples)
		}
	}
}

// packBits packs a bit slice (one bit per byte, MSB-first) into bytes
// — 49 FEC-decoded AMBE payload bits become a 7-byte .raw frame.
func packBits(bits []byte) []byte {
	out := make([]byte, (len(bits)+7)/8)
	for i := range bits {
		if bits[i]&1 != 0 {
			out[i>>3] |= 1 << uint(7-(i&7))
		}
	}
	return out
}
