package composer

import (
	"context"
	"math"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	dmrrx "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/receiver"
	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// dmrVoiceIntermediateHz is the rate the wideband IQ is decimated to
// before the DMR receiver runs. 48 kHz gives the 4800-baud DMR symbol
// stream 10 samples per symbol — ample for the receiver's RRC matched
// filter and Mueller-Müller clock recovery, without the cost of
// running them at the SDR's native multi-MS/s rate.
const dmrVoiceIntermediateHz = 48_000

// dmrVoiceChannelSelectHz caps the voice front-end channel filter at half the
// 12.5 kHz DMR channel spacing. A wideband DDC voice tap hands the chain a
// stream already at the intermediate rate but band-limited only to the tap's
// output Nyquist (~±24 kHz), NOT to a single 12.5 kHz channel — so the old
// pass-through front end fed that whole ±24 kHz span into the receiver's FM
// discriminator, where the capture effect locks onto a stronger ±12.5 kHz
// neighbour during voice gaps. DMR gating is disabled (grantTG 0), so a leaked
// neighbour is recorded AS the call — wrong audio, not just dropped. Filtering
// to ±6.25 kHz drops an adjacent channel centred at ±12.5 kHz deep into the
// 81-tap front end's stopband while the wanted 4FSK (deviation ±1944 Hz,
// Carson ≈ ±4.8 kHz) passes flat. Both DMR timeslots share the one 12.5 kHz
// carrier, so the filter isolates the carrier without splitting the slots.
const dmrVoiceChannelSelectHz = 6250.0

// newDMRVoiceFrontEnd builds the channel-select + decimation front end for the
// DMR voice chain. Mirrors newP25P1VoiceFrontEnd: the dedicated-tuner path
// (iqHz well above the intermediate rate) already band-limits as it decimates;
// the pass-through path (a wideband DDC tap already at the intermediate rate,
// decim==1) applied NO filter, so this channel-selects to
// ±min(bw, dmrVoiceChannelSelectHz) before the FM discriminator. Factored out
// so the selectivity is unit-testable.
func newDMRVoiceFrontEnd(iqHz float64, bw uint32) *decimatingFIR {
	chanBW := float64(bw)
	decim := int(math.Round(iqHz)) / dmrVoiceIntermediateHz
	filterAtUnity := false
	if decim <= 1 {
		filterAtUnity = true
		if chanBW > dmrVoiceChannelSelectHz {
			chanBW = dmrVoiceChannelSelectHz
		}
	}
	return newDecimatingFIR(iqHz, dmrVoiceIntermediateHz, chanBW, filterAtUnity)
}

// aliasPrintable reports whether a reassembled talker alias looks like a
// clean decode: it holds at least one printable character and no C0/C1
// control characters or Unicode replacement runes (the hallmark of a
// bit-error-corrupted or wrong-format decode). Unlike the P25 ASCII-only
// cleaner it tolerates accented / non-Latin names, since DMR aliases carry
// a UTF-8 / UTF-16 format bit. Mirrors the trunking.TalkerAlias.Unreliable
// contract (#711).
func aliasPrintable(s string) bool {
	if s == "" {
		return false
	}
	printable := false
	for _, r := range s {
		if r == utf8.RuneError || r < 0x20 || (r >= 0x7F && r < 0xA0) {
			return false
		}
		if !unicode.IsSpace(r) {
			printable = true
		}
	}
	return printable
}

// rawFrameSink is the subset of voice.Recorder the DMR voice chain
// needs. The composer holds its sink as a PCMSink; runDMRVoiceChain
// type-asserts to this, so a sink that only implements WritePCM
// (analog-only callers, test stubs) still works for the FM path.
type rawFrameSink interface {
	WriteRawFrame(deviceSerial string, frame []byte) error
}

// errAwareRawSink is the optional richer sink the P25 Phase 1 voice chain
// uses to pass the per-frame FEC corrected-bit count to the IMBE decoder's
// adaptive smoothing. The recorder implements it; sinks that don't are
// handled via the plain rawFrameSink path.
type errAwareRawSink interface {
	WriteRawFrameWithErrors(deviceSerial string, frame []byte, correctedBits int) error
}

// callAwareRawSink is the richest sink: it carries the per-frame FEC
// corrected-bit count AND the call identity (Grant.CallID) the frame was
// decoded for, so the recorder can reject a stale frame from the call that
// previously held this tap serial (the voice-tap mix-up fence). The recorder
// implements it; a chain that finds the sink supports it routes frames here
// in preference to errAwareRawSink, falling back when it doesn't.
type callAwareRawSink interface {
	WriteRawFrameForCall(deviceSerial string, callID uint64, frame []byte, correctedBits int) error
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
// timeslot) whose burst-A voice sync is identical, so the preferred
// discriminator is the embedded Link Control: once a superframe's LC
// names this call's talkgroup, its Phase is bound and subsequent
// superframes are routed by Phase even when their LC is absent or fails
// CRC. Superframes whose LC names a different talkgroup are dropped.
//
// Real outbound DMR embedded signalling is still capture-pending (the
// EMB FEC / 5-bit CRC constants are unvalidated against live air, see
// docs/status.md / issue #644), so on a carrier whose embedded LC never
// decodes the router would otherwise bind nothing and drop the whole
// call — recording empty files. To degrade gracefully it falls back to
// the active slot's Phase after a short grace window with no LC bind, so
// audio still records; a later valid LC re-binds and corrects the routing.
type slotRouter struct {
	groupID          uint32
	bound            int   // -1 until a phase is bound (by LC or by fallback)
	unboundSeen      int   // LC-less superframes seen while still unbound
	foreignPhaseMask uint8 // bit p set when phase p was positively another talkgroup
	byFallback       bool  // bound was set by the phase fallback, not an embedded LC
}

// unboundPhaseFallbackGrace is how many consecutive LC-less voice superframes
// the router tolerates before binding the active slot's phase. Two lets a
// clean embedded LC on superframe 1 or 2 bind the correct phase first while
// losing at most one leading superframe to the fallback.
const unboundPhaseFallbackGrace = 2

func newSlotRouter(groupID uint32) *slotRouter {
	return &slotRouter{groupID: groupID, bound: -1}
}

// accept reports whether sf belongs to this call's timeslot. An embedded LC
// naming this call's talkgroup is authoritative and binds (or re-binds) the
// phase. Absent a usable LC, once a phase is bound we route by it; while still
// unbound we wait a couple of superframes for an LC and otherwise fall back to
// the active slot's phase so the call still records rather than dropping it.
func (r *slotRouter) accept(sf dmrvoice.VoiceSuperframe) bool {
	if sf.HasLC {
		if gv, ok := sf.LC.AsGroupVoiceUser(); ok {
			if gv.GroupAddress == r.groupID {
				r.bound = int(sf.Phase)
				r.byFallback = false
				return true
			}
			// LC names a different talkgroup — this phase is positively the
			// other slot, so never fall back onto it later.
			r.foreignPhaseMask |= 1 << (sf.Phase & 1)
			return false
		}
	}
	if r.bound >= 0 {
		// Phase already known (by LC or fallback): route by it.
		return int(sf.Phase) == r.bound
	}
	// Unbound and no usable group LC. Never fall back onto a phase we've
	// positively identified as another talkgroup (if the embedded LC decodes
	// well enough to read the other TG on a slot, that slot isn't ours).
	if r.foreignPhaseMask&(1<<(sf.Phase&1)) != 0 {
		return false
	}
	// Otherwise give a valid LC a couple of superframes to bind the correct
	// phase; if none decodes, fall back to this slot's phase so audio records
	// instead of the whole call being dropped.
	r.unboundSeen++
	if r.unboundSeen >= unboundPhaseFallbackGrace {
		r.bound = int(sf.Phase)
		r.byFallback = true
		return true
	}
	return false
}

func (c *Composer) runDMRVoiceChain(ctx context.Context, serial, system string, iqCh <-chan []complex64, iqHz float64, groupID uint32, interleaved bool, done chan<- struct{}) {
	defer close(done)
	defer gtlog.Recover(c.log, "voice-chain-dmr:"+serial, nil)

	// Shared boundary controller: universal hangtime end-of-call + Touch
	// heartbeat. Talkgroup gating is left disabled (grantTG 0) until the
	// DMR chain surfaces a per-superframe embedded-LC talkgroup.
	bt := c.newBoundaryTracker(serial, 0, nil)
	go bt.run(ctx)

	// Front-end decimator: an 81-tap anti-alias FIR that convolves ONLY at
	// the output positions, replacing the old full-rate FIR + every-Nth-
	// sample decimation that filtered every input sample and discarded ~98%
	// of the result. At 2.4 MS/s that wasted ~194M MACs/sec per voice call
	// and starved the live IQ consumer until the SDR dropped chunks. Same
	// coefficients and same kept samples as before, so the decode is
	// byte-for-byte unchanged — only the wasted work is removed. decim==1 (a
	// source already at the intermediate rate — a wideband DDC voice tap) is NOT
	// a pass-through: it still channel-selects to a single DMR channel before the
	// FM discriminator (see newDMRVoiceFrontEnd / dmrVoiceChannelSelectHz) so an
	// adjacent ±12.5 kHz channel can't capture the demod and be recorded as this
	// call.
	fe := newDMRVoiceFrontEnd(iqHz, c.bw)
	symbolHz := fe.OutRateHz()

	rs, _ := c.sink.(rawFrameSink)
	// ers, when the sink supports it, carries the per-frame FEC corrected-bit
	// count into the AMBE+2 decoder's adaptive smoothing (the recorder
	// implements it). Falls back to the plain rawFrameSink path otherwise.
	ers, _ := c.sink.(errAwareRawSink)
	voiceDec := dmrvoice.NewDecoder()
	var router *slotRouter
	if interleaved {
		voiceDec = dmrvoice.NewInterleavedDecoder()
		router = newSlotRouter(groupID)
	}
	// Talker-alias reassembly and GPS Info both ride the embedded LC, which
	// carries no source address, so both are attributed to the most recent
	// SourceID seen from a group-voice embedded LC on this call.
	aliasAsm := dmr.NewTalkerAliasAssembler(nil)
	var callSrc uint32
	publishAlias := func(src uint32, alias string) {
		if src == 0 || alias == "" || c.bus == nil {
			return
		}
		unreliable := !aliasPrintable(alias)
		c.log.Info("composer: dmr talker alias",
			"system", system, "serial", serial, "src", src, "alias", alias,
			"unreliable", unreliable)
		c.bus.Publish(events.Event{
			Kind: events.KindTalkerAlias,
			Payload: trunking.TalkerAlias{
				System:     system,
				Protocol:   "dmr",
				SourceID:   src,
				Alias:      alias,
				Unreliable: unreliable,
				At:         time.Now(),
			},
		})
	}
	// lastGPS dedupes repeated identical fixes — a stationary radio rebroadcasts
	// the same GPS Info LC, and one row per superframe would flood the log.
	var lastGPS dmr.GPSInfo
	var haveGPS bool
	publishGPS := func(src uint32, g dmr.GPSInfo) {
		if src == 0 || c.bus == nil || !g.Valid() {
			return
		}
		if haveGPS && g == lastGPS {
			return
		}
		lastGPS, haveGPS = g, true
		c.log.Info("composer: dmr gps",
			"system", system, "serial", serial, "src", src,
			"lat", g.Latitude, "lon", g.Longitude)
		c.bus.Publish(events.Event{
			Kind: events.KindLocation,
			Payload: trunking.Location{
				System:    system,
				Protocol:  "dmr",
				RadioID:   src,
				Talkgroup: groupID,
				Latitude:  g.Latitude,
				Longitude: g.Longitude,
				At:        time.Now(),
			},
		})
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
		// lcSuperframes counts superframes that carried a CRC-valid embedded
		// LC, so the decode-quality log shows whether slot routing is driven
		// by embedded signalling or by the phase fallback (issue #644).
		lcSuperframes  atomic.Uint64
		loggedFallback bool
	)
	rx := dmrrx.New(dmrrx.Options{
		SampleRateHz: symbolHz,
		// DMR spec peak deviation per ETSI TS 102 361-1 §6.3 — matches
		// the control-channel receiver in internal/scanner/ccdecoder.
		DeviationHz: 1944.0,
		ClockGain:   0.025,
		DibitSink: func(dibits []uint8, baseIdx int) {
			for _, sf := range voiceDec.Process(dibits, baseIdx) {
				if sf.HasLC {
					lcSuperframes.Add(1)
					if gv, ok := sf.LC.AsGroupVoiceUser(); ok && gv.SourceID != 0 {
						callSrc = gv.SourceID
					}
				}
				// Talker-alias header/block and GPS Info embedded LCs are
				// call-metadata, not this slot's audio, so they run on every
				// superframe (both slots) before the slot-router gate.
				if sf.HasTalkerAlias {
					if alias, done := aliasAsm.Add(callSrc, sf.TalkerAlias); done {
						publishAlias(callSrc, alias)
					}
				}
				if sf.HasGPS {
					publishGPS(callSrc, sf.GPS)
				}
				if router != nil && !router.accept(sf) {
					// A superframe from the other timeslot (or an
					// as-yet-unbound phase) — not this call's audio.
					continue
				}
				// One-shot: surface when the call records via the active-slot
				// phase fallback rather than an embedded-LC bind, so empty-vs-
				// fallback recordings are diagnosable from the logs (#644).
				if router != nil && router.byFallback && !loggedFallback {
					loggedFallback = true
					c.log.Info("composer: DMR slot router bound by phase fallback — embedded LC did not decode; embedded signalling is capture-pending (see #644)",
						"serial", serial, "phase", router.bound)
				}
				superframes.Add(1)
				bt.onVoice(0)
				if sf.HasRC {
					// In-call Reverse Channel signalling — control-plane
					// visibility only (see dmr/rc.go); it does not affect the
					// audio path.
					c.log.Debug("composer: DMR reverse channel",
						"serial", serial, "rc_info", sf.RC.Info, "phase", sf.Phase)
				}
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
					packed := packBits(info)
					var werr error
					if ers != nil {
						werr = ers.WriteRawFrameWithErrors(serial, packed, errBits)
					} else {
						werr = rs.WriteRawFrame(serial, packed)
					}
					if werr != nil {
						c.log.Warn("composer: DMR raw-frame write failed",
							"serial", serial, "err", werr)
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
			"corrected_bit_errs", corrErrBits.Load(),
			"lc_superframes", lcSuperframes.Load())
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
			bt.observe(iq)
			rx.Process(fe.Process(nil, iq))
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
