package composer

import (
	"context"
	"math"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	p25p2rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/sigfollow"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// p25p2VoiceIntermediateHz is the rate the wideband IQ is decimated to
// before the P25 Phase 2 receiver runs. 48 kHz gives the 6000-baud
// H-DQPSK symbol stream 8 samples per symbol — ample for the matched
// filter and Gardner clock recovery without running them at the SDR's
// native multi-MS/s rate.
const p25p2VoiceIntermediateHz = 48_000

// p25p2ChannelSelectHz caps the voice front-end channel filter at half the
// 12.5 kHz P25 channel spacing. A wideband DDC voice tap hands the chain a
// stream already at the intermediate rate but band-limited only to the tap's
// output Nyquist (~±24 kHz), NOT to a single 12.5 kHz channel — so the old
// pass-through front end fed that whole ±24 kHz span into the receiver. Unlike
// Phase 1 (C4FM/FM discriminator) there is no FM capture effect here: Phase 2
// is linear H-DQPSK, and adjacent-channel energy instead pumps the receiver's
// AGC down (shrinking the wanted constellation's decision margin) and raises the
// noise floor the coarse-carrier / Costas loops estimate against — degrading
// carrier recovery and raising EVM. Filtering to ±6.25 kHz drops an adjacent
// channel centred at ±12.5 kHz deep into the 81-tap front end's stopband while
// the wanted H-DQPSK (occupied ≈ ±3.6 kHz at 6000 baud, α=0.2) passes flat.
const p25p2ChannelSelectHz = 6250.0

// newP25P2VoiceFrontEnd builds the channel-select + decimation front end for
// the P25 Phase 2 voice chain. Mirrors newP25P1VoiceFrontEnd: on the
// dedicated-tuner path (iqHz well above the intermediate rate) the decimating
// FIR already band-limits as it decimates; on the pass-through path (a wideband
// DDC tap already at the intermediate rate, decim==1) the old front end applied
// NO filter at all, so this channel-selects to ±min(bw, p25p2ChannelSelectHz)
// before the receiver. Factored out so the selectivity is unit-testable.
func newP25P2VoiceFrontEnd(iqHz float64, bw uint32) *decimatingFIR {
	return newVoiceFrontEnd(iqHz, bw, p25p2VoiceIntermediateHz, p25p2ChannelSelectHz)
}

// p25p2VoiceGardnerGain matches the value newP25Phase2Pipeline settled
// on for the H-DQPSK symbol clock (smaller than the receiver default
// because H-DQPSK slips differently than C4FM).
const p25p2VoiceGardnerGain = 0.005

// runP25Phase2VoiceChain consumes IQ for one P25 Phase 2 voice call. It
// decimates the wideband IQ to an H-DQPSK-friendly rate, recovers the
// dibit stream with the Phase 2 receiver, assembles 360 ms superframes,
// and for every voice-bearing sub-frame FEC-decodes its AMBE+2 frames
// and appends them to the recorder's .raw sidecar.
//
// In parallel it dispatches the SACCH/FACCH bursts that ride the same
// carrier (Phase 2 voice traffic channels interleave MAC PDUs — talker
// alias, encryption sync, in-call signalling — with voice); macCfg.Seed
// descrambles them. On a completed talker-alias the chain publishes a
// trunking.TalkerAlias event itself, mirroring the CC's
// publishTalkerAlias path. This is the only way display names surface
// on Phase 2 systems whose CCs never emit alias fragments (e.g. MMR).
//
// The recorder maps protocol "p25-phase2" to the pure-Go AMBE+2
// 3600x2450 (half-rate) decoder, registered as "ambe2-dmr"
// (voice.DefaultVocoderForProtocol), so WriteRawFrame here decodes each
// FEC-decoded 7-byte frame to PCM and into the call's WAV. It is the
// decoder DMR and NXDN use: the 72-bit codeword and the 49-bit payload
// are the same on all three.
//
// Voice is framed in the real TDMA air format (p25p2.AirFramer: S-/I-ISCH
// timeslot grid, DUID, PN44 descramble, OP25/SDRTrunk frame positions)
// and only the granted logical channel is recorded: timeslot is the
// grant's 1-based Timeslot (channel number n mod 2, +1), 0 when unknown,
// in which case the chain binds to the first channel that carries voice.
// Two calls share a Phase 2 carrier, one per logical channel, and the
// chain used to record both into one WAV (#1273). The MAC signalling
// (SACCH/FACCH) rides the same bursts and decodes on the air format too.
//
// grantNAC is the NAC the grant was heard on. A grant made before the site's
// identity was known carries no seed (issue #1364); the chain then holds its
// bursts (p25p2SeedWaitMaxBursts) and decodes them once the control channel
// publishes the identity (KindSiteUpdate), so the call loses nothing.
func (c *Composer) runP25Phase2VoiceChain(ctx context.Context, serial string, system string, macCfg p25p2.MACDecodeConfig, timeslot uint8, grantNAC uint16, iqCh <-chan []complex64, iqHz float64, done chan<- struct{}) {
	defer close(done)
	defer gtlog.Recover(c.log, "voice-chain-p25p2:"+serial, nil)

	// Shared boundary controller: universal hangtime end-of-call + Touch
	// heartbeat. Talkgroup gating is left disabled (grantTG 0) until the
	// Phase 2 chain surfaces a per-voice-frame MAC talkgroup.
	bt := c.newBoundaryTracker(serial, 0, nil)
	go bt.run(ctx)

	// Issue #376 field-diagnostic: the existing "composer: p25p2 mac
	// pdu" log fires only after a successful FEC decode, so every
	// failure mode (zero macCfg, ScramblerOff against scrambled
	// on-air traffic, ISCH FEC corruption) is silent. A single
	// once-per-call line at chain entry confirms the chain actually
	// ran and surfaces the exact FEC config the grant carried.
	c.log.Info("composer: p25p2 voice chain started",
		"serial", serial, "system", system,
		"timeslot", timeslot, "seed", macCfg.Seed)
	// The air format fixes the channel coding, so the trellis / rs /
	// interleave / scrambler modes no longer change anything; the seed is
	// the one input that matters (issue #451): without it neither the voice
	// frames nor the scrambled SACCH/FACCH can be descrambled.
	seedLate := false
	if macCfg.Seed == 0 {
		if s := c.p25IDs.seed(system, grantNAC); s != 0 {
			macCfg.Seed, seedLate = s, true
			c.log.Info("composer: p25p2 grant carried no scrambler seed; using the site identity the control channel has published since",
				"serial", serial, "system", system, "seed", s)
		} else {
			c.log.Warn("composer: p25p2 has no scrambler seed yet; holding the call's bursts until the control channel decodes the site's Network Status Broadcast",
				"serial", serial, "system", system, "max_wait_bursts", p25p2SeedWaitMaxBursts)
		}
	}

	// Front-end decimator: an 81-tap anti-alias FIR that convolves ONLY at
	// the output positions, replacing the old full-rate FIR + every-Nth-
	// sample decimation that filtered every input sample and discarded ~98%
	// of the result. At 2.4 MS/s that wasted ~194M MACs/sec per voice call
	// and starved the live IQ consumer until the SDR dropped chunks. Same
	// coefficients and same kept samples as before, so the decode is
	// byte-for-byte unchanged — only the wasted work is removed. decim==1 (a
	// source already at the intermediate rate — a wideband DDC voice tap) is NOT
	// a pass-through: it still channel-selects to a single P25 channel before the
	// receiver (see newP25P2VoiceFrontEnd / p25p2ChannelSelectHz) so an adjacent
	// ±12.5 kHz channel can't pump the AGC / degrade carrier recovery.
	fe := newP25P2VoiceFrontEnd(iqHz, c.bw)
	symbolHz := fe.OutRateHz()

	rs, _ := c.sink.(rawFrameSink)
	airFramer := p25p2.NewAirFramer()
	var airMasks *p25p2.AirMasks
	if macCfg.Seed != 0 {
		airMasks = p25p2.NewAirMasks(macCfg.Seed)
	}
	// wantLCH is the logical channel this call's voice rides; -1 binds to
	// the first one that carries voice.
	wantLCH := -1
	if timeslot > 0 {
		wantLCH = int(timeslot-1) & 1
	}
	// dispatcher decodes the MAC PDUs interleaved with voice on the
	// traffic channel (talker alias, in-call source, encryption sync).
	// It is the same shared dispatch the signalling follower runs off the
	// traffic-channel signalling stream (internal/sigfollow), so the
	// voice and follower paths never diverge on what they decode (#376).
	// The voice chain wires the call-bound source / encryption PDUs to
	// its engine-backfill publishers; the alias the dispatcher publishes
	// itself (identical in both paths).
	// srcFilter keeps the call's source updates to the ones that say
	// something new; see callSourceFilter. encFilter does the same for the
	// encryption sync, which now also arrives every 360 ms in the voice
	// bursts' ESS (below).
	var srcFilter callSourceFilter
	var encFilter callEncryptionFilter
	dispatcher := sigfollow.NewMACDispatcher(sigfollow.MACDispatcherOptions{
		Bus:              c.bus,
		Log:              c.log,
		LogPrefix:        "composer",
		System:           system,
		Serial:           serial,
		OnCallSource:     func(u p25p2.GroupVoiceChannelUser) { c.publishP25Phase2CallSource(serial, u, &srcFilter) },
		OnCallEncryption: func(es p25p2.EncryptionSync) { c.publishP25Phase2CallEncryption(serial, es, &encFilter) },
		// MAC_END_PTT is the radio releasing its PTT: the Phase 2 form of
		// the Phase 1 terminator. In per-transmission recording mode it
		// rolls the file, so each over is a file of its own carrying its
		// own talker (issue #1379 did this for Phase 1 and TETRA).
		OnTransmissionEnd: bt.onTransmissionEnd,
	})
	// ess reads the encryption sync carried in the voice bursts themselves
	// (four parts in the VOICE-4 bursts, the Reed-Solomon parity in the
	// VOICE-2 burst). MAC_PTT gives the same sync, but only at key-up: a
	// call this chain joined late, or whose key-up burst was lost, had no
	// algorithm or key at all (issue #1364).
	var ess p25p2.ESSTracker
	// voiceSubframes counts P25 Phase 2 voice-bearing subframes the
	// receiver delivered — i.e. real voice activity. The touch ticker
	// (below) only refreshes the engine's LastHeardAt when this counter
	// has advanced since the previous tick. Without this gate a stalled
	// decoder still kept the call alive forever via an unconditional
	// 1 s heartbeat (issue #356).
	var voiceSubframes atomic.Uint64
	// Decode-quality telemetry — see runP25Phase1VoiceChain for the
	// rationale. A high uncorrectable rate is the measurable signature of
	// weak signal / wrong gain behind garbled audio (issue #356 follow-up).
	var corrErrBits atomic.Uint64
	// Call census (issues #813, #915). A field test on real Phase 2
	// traffic saw the per-PDU "composer: p25p2 mac pdu" line — which fires
	// only on a *successful* MAC decode — never appear at all, which is
	// ambiguous: it can't tell framing failure from a missing seed from
	// FEC failure. These counters feed the unconditional end-of-call census
	// (logCallCensus) that disambiguates them. They are read from the
	// for-select goroutine below, so they are atomic like the decode-quality
	// counters above.
	var (
		// Air-format census: bursts framed, voice bursts dropped because
		// they belong to the carrier's other logical channel (another
		// call), and this call's signalling bursts / CRC-valid MAC PDUs.
		airBursts      atomic.Uint64
		otherSlotVoice atomic.Uint64
		macBursts      atomic.Uint64
		macPDUsDecoded atomic.Uint64
		duidHist       [16]atomic.Uint64
		// essSyncs counts encryption syncs decoded from the voice bursts.
		essSyncs atomic.Uint64
		// Where the voice FEC corrections fell: by the frame's position in
		// its burst (four) and by the burst's voice timeslot (five per
		// logical channel). The first on-air Phase 2 log had one call with
		// about two corrections in every voice frame beside signalling that
		// decoded 70 of 72 bursts. Corrections that sit in one position or
		// one timeslot are a layout or mask fault; spread evenly they are
		// the signal. The next log can tell which without a capture.
		errsByFrame [4]atomic.Uint64
		errsBySlot  [5]atomic.Uint64
		// What reached the chain at all, for a call whose framer never
		// locked (air_bursts=0): IQ samples in, dibits out of the receiver,
		// and the stream position of the first burst framed (-1: none).
		iqSamples  atomic.Uint64
		dibitsSeen atomic.Uint64
		firstBurst atomic.Int64
	)
	firstBurst.Store(-1)
	// oneBurst hands the dispatcher one signalling burst at a time, in
	// stream order with the voice bursts around it: a MAC_END_PTT must roll
	// the recording before the next over's first voice burst is written.
	var oneBurst [1]p25p2.AirBurst
	dispatchSig := func(b p25p2.AirBurst) {
		oneBurst[0] = b
		nSig, nDec := dispatcher.DispatchAir(oneBurst[:], airMasks)
		macBursts.Add(uint64(nSig))
		macPDUsDecoded.Add(uint64(nDec))
	}
	decodeVoice := func(b p25p2.AirBurst) {
		if es, _, ok := ess.Add(b, airMasks); ok {
			essSyncs.Add(1)
			c.publishP25Phase2CallEncryption(serial, es, &encFilter)
		}
		if rs == nil {
			return
		}
		frames, errs := p25p2.ExtractAirVoiceFramesDetail(b, airMasks)
		for i, e := range errs {
			if e <= 0 {
				continue
			}
			corrErrBits.Add(uint64(e))
			errsByFrame[i].Add(uint64(e))
			if vs := b.Slot >> 1; vs >= 0 && vs < len(errsBySlot) {
				errsBySlot[vs].Add(uint64(e))
			}
		}
		for _, f := range frames {
			if werr := rs.WriteRawFrame(serial, f); werr != nil {
				c.log.Warn("composer: p25p2 raw-frame write failed",
					"serial", serial, "err", werr)
			}
		}
	}
	// held is the call's own bursts (its voice and its scrambled
	// signalling) kept in stream order while the chain has no seed;
	// heldDropped counts the oldest ones a long wait pushed out.
	var held []p25p2.AirBurst
	var heldDropped, heldReplayed atomic.Uint64
	hold := func(b p25p2.AirBurst) {
		if len(held) >= p25p2SeedWaitMaxBursts {
			copy(held, held[1:])
			held = held[:len(held)-1]
			heldDropped.Add(1)
		}
		held = append(held, b)
	}
	// adoptSeed installs a seed the site's identity named after the chain
	// started and decodes what was held, in order.
	adoptSeed := func() {
		s := c.p25IDs.seed(system, grantNAC)
		if s == 0 {
			return
		}
		macCfg.Seed, seedLate = s, true
		airMasks = p25p2.NewAirMasks(s)
		c.log.Info("composer: p25p2 scrambler seed learned from the site's identity; decoding the held bursts",
			"serial", serial, "system", system, "seed", s,
			"held_bursts", len(held), "dropped_bursts", heldDropped.Load())
		for _, b := range held {
			heldReplayed.Add(1)
			if b.DUID.IsSignalling() {
				dispatchSig(b)
			} else {
				decodeVoice(b)
			}
		}
		held = nil
	}
	// drainAirBursts records the granted logical channel's voice frames and
	// dispatches its MAC signalling.
	drainAirBursts := func(bursts []p25p2.AirBurst) {
		if airMasks == nil && len(bursts) > 0 {
			adoptSeed()
		}
		for _, b := range bursts {
			if airBursts.Add(1) == 1 {
				firstBurst.Store(int64(b.StartDibit))
			}
			if b.DUID >= 0 && int(b.DUID) < len(duidHist) {
				duidHist[b.DUID].Add(1)
			}
			if b.DUID.IsSignalling() {
				// Like OP25, signalling on the other logical channel
				// belongs to the other call; a clear LCCH is the
				// carrier's own and is always taken.
				if b.DUID == p25p2.DUIDLCCHClear || wantLCH < 0 || b.LogicalChannel == wantLCH {
					if airMasks == nil && b.DUID != p25p2.DUIDLCCHClear {
						hold(b)
					} else {
						dispatchSig(b)
					}
				}
				continue
			}
			if !b.DUID.IsVoice() {
				continue
			}
			if wantLCH < 0 {
				wantLCH = b.LogicalChannel
				c.log.Info("composer: p25p2 grant names no timeslot; following the first logical channel with voice",
					"serial", serial, "system", system, "logical_channel", wantLCH)
			}
			if b.LogicalChannel != wantLCH {
				otherSlotVoice.Add(1)
				continue
			}
			voiceSubframes.Add(1)
			bt.onVoice(0)
			if airMasks == nil {
				hold(b)
				continue
			}
			decodeVoice(b)
		}
	}
	// Build the receiver on the hard slicer by default; when the grant
	// carries the soft-decision flag (issue #915) wire the soft sink. The
	// air framer slices hard dibits either way; the soft path keeps the
	// receiver's soft-decision demod.
	rxOpts := p25p2rx.Options{
		SampleRateHz:  symbolHz,
		ClockMode:     p25p2rx.ClockGardner,
		GardnerGain:   p25p2VoiceGardnerGain,
		Equalizer:     macCfg.Equalizer,
		EnableDCBlock: macCfg.DCBlock,
	}
	if macCfg.SoftDecision {
		rxOpts.SoftDecision = true
		rxOpts.SoftSink = func(dibits []uint8, soft []complex64, baseIdx int) {
			dibitsSeen.Add(uint64(len(dibits)))
			drainAirBursts(airFramer.Process(dibits, baseIdx))
		}
	} else {
		rxOpts.DibitSink = func(dibits []uint8, baseIdx int) {
			dibitsSeen.Add(uint64(len(dibits)))
			drainAirBursts(airFramer.Process(dibits, baseIdx))
		}
	}
	rx := p25p2rx.New(rxOpts)
	// The chain starts at the grant, at an arbitrary sub-symbol phase of a
	// carrier that is already up, and MAC_PTT (the call's source radio) rides
	// the key-up bursts: seed the timing loop feed-forward instead of waiting
	// up to a second for it to pull in (issue #1364).
	rx.ArmTimingAcquisition()

	touchTicker := time.NewTicker(c.touchEvery)
	defer touchTicker.Stop()
	// logDecodeQuality emits a rolling decode-quality summary, gated to a
	// burst of voice subframes so it does not spam the log every touch
	// tick (issue #356 follow-up). See runP25Phase1VoiceChain.
	var lastQualityLogSubframes uint64
	const qualityLogEverySubframes = 50
	logDecodeQuality := func(final bool) {
		n := voiceSubframes.Load()
		if n == 0 || (!final && n-lastQualityLogSubframes < qualityLogEverySubframes) {
			return
		}
		lastQualityLogSubframes = n
		c.log.Info("composer: p25p2 decode quality",
			"serial", serial, "system", system,
			"voice_bursts", n, "corrected_bit_errs", corrErrBits.Load())
	}
	// logCallCensus emits exactly one line at chain end, unconditionally —
	// even when nothing ever framed. It is the field diagnostic that
	// disambiguates the failure modes when no "composer: p25p2 mac pdu"
	// line ever appears:
	//   - air_bursts=0                     → the TDMA framer never locked;
	//                                         iq_seconds / dibits say whether
	//                                         the chain was fed at all
	//   - descramble_seed_known=false      → voice and scrambled MAC cannot
	//                                         be descrambled (no identity)
	//   - mac_bursts>0, mac_pdus=0         → every signalling burst failed
	//                                         RS(63,35) or its CRC: weak
	//                                         signal or a wrong seed
	// The duid_<type> buckets reuse DUID.String() so the line is
	// self-describing.
	logCallCensus := func() {
		attrs := []any{
			"serial", serial, "system", system,
			"air_bursts", airBursts.Load(),
			"voice_bursts", voiceSubframes.Load(),
			"other_slot_voice_bursts", otherSlotVoice.Load(),
			"logical_channel", wantLCH,
			"descramble_seed_known", airMasks != nil,
			"descramble_seed_late", seedLate,
			"seed_wait_replayed_bursts", heldReplayed.Load(),
			"seed_wait_dropped_bursts", heldDropped.Load()+uint64(len(held)),
			"mac_bursts", macBursts.Load(),
			"mac_pdus", macPDUsDecoded.Load(),
			"ess_syncs", essSyncs.Load(),
			"voice_errs_by_frame", p25p2CensusCounts(errsByFrame[:]),
			"voice_errs_by_timeslot", p25p2CensusCounts(errsBySlot[:]),
			"iq_seconds", math.Round(float64(iqSamples.Load())/iqHz*100) / 100,
			"dibits", dibitsSeen.Load(),
			"first_burst_ms", p25p2FirstBurstMs(firstBurst.Load()),
		}
		for d := range duidHist {
			if cnt := duidHist[d].Load(); cnt > 0 {
				attrs = append(attrs, p25p2CensusDUIDKey(p25p2.DUID(d)), cnt)
			}
		}
		c.log.Info("composer: p25p2 call census", attrs...)
	}

	for {
		select {
		case <-ctx.Done():
			logDecodeQuality(true)
			logCallCensus()
			return
		case <-touchTicker.C:
			// Touch + hangtime end-of-call handled by the shared boundary
			// tracker; this ticker only drives the decode-quality summary.
			logDecodeQuality(false)
		case iq, ok := <-iqCh:
			if !ok {
				logDecodeQuality(true)
				logCallCensus()
				return
			}
			bt.observe(iq)
			iqSamples.Add(uint64(len(iq)))
			rx.Process(fe.Process(nil, iq))
		}
	}
}

// p25p2CensusDUIDKey names a DUID's bucket in the call census. The defined
// DUIDs use DUID.String(); the seven reserved values all print "reserved", so
// they carry their number (duid_reserved_5): with one shared name the census
// line repeated the key six times on real air (issue #1364), and a reader
// could not tell which bucket was which. A reserved DUID on a locked carrier
// is a misread burst, so the count is the useful part.
func p25p2CensusDUIDKey(d p25p2.DUID) string {
	name := d.String()
	if name == "reserved" {
		return "duid_reserved_" + strconv.Itoa(int(d))
	}
	return "duid_" + name
}

// p25p2CensusCounts renders a row of counters as "a/b/c".
func p25p2CensusCounts(c []atomic.Uint64) string {
	var sb strings.Builder
	for i := range c {
		if i > 0 {
			sb.WriteByte('/')
		}
		sb.WriteString(strconv.FormatUint(c[i].Load(), 10))
	}
	return sb.String()
}

// p25p2FirstBurstMs is the stream time of the first burst the framer cut,
// from its dibit index at 6000 symbols a second; -1 when it cut none.
func p25p2FirstBurstMs(startDibit int64) int64 {
	if startDibit < 0 {
		return -1
	}
	return startDibit * 1000 / p25p2rx.SymbolRate
}

// callEncryptionFilter passes a call's encryption sync only when its
// algorithm or key differs from the last one passed. The sync arrives at
// every key-up (MAC_PTT) and, from the voice bursts' ESS, every 360 ms; its
// message indicator changes each time, but the engine keeps only the
// algorithm and key. Not safe for concurrent use; one per voice chain.
type callEncryptionFilter struct {
	alg  uint8
	key  uint16
	have bool
}

func (f *callEncryptionFilter) admit(es p25p2.EncryptionSync) bool {
	if f.have && f.alg == es.AlgorithmID && f.key == es.KeyID {
		return false
	}
	f.alg, f.key, f.have = es.AlgorithmID, es.KeyID, true
	return true
}

// callSourceFilter passes a call's in-call source update only when it tells
// the engine something the last one it passed did not. A Phase 2 traffic
// channel repeats GROUP_VOICE_CHANNEL_USER on every FACCH / SACCH burst of a
// talk spurt, roughly every 60 ms, and each one used to go to the bus: the
// first on-air Phase 2 log had 79 "call source update" lines in four seconds
// across three calls, and the same flood reached every event subscriber (the
// web activity feed, alerts, the affiliation tracker). The engine's backfill
// is additive, so a repeat carries nothing. Not safe for concurrent use; one
// per voice chain, called from the chain's own goroutine.
type callSourceFilter struct {
	last trunking.CallSourceUpdate
	have bool
}

// admit reports whether upd (with At unset) is news. A hang-time update names
// no radio (source 0), and the engine keeps the source it has, so such an
// update is compared as if it named the last one: it passes only when its
// encryption / emergency / priority differ.
func (f *callSourceFilter) admit(upd trunking.CallSourceUpdate) bool {
	if upd.SourceID == 0 && f.have {
		upd.SourceID = f.last.SourceID
	}
	if f.have && upd == f.last {
		return false
	}
	f.last, f.have = upd, true
	return true
}

// publishP25Phase2CallSource publishes a KindCallSourceUpdate event so
// the trunking engine can backfill the bound ActiveCall's SourceID +
// Encrypted from an in-call GROUP_VOICE_CHANNEL_USER PDU. The engine
// fills in System / Protocol / GroupID from the bound Grant on
// republish — leave them blank here. Repeats are dropped by filter.
func (c *Composer) publishP25Phase2CallSource(serial string, u p25p2.GroupVoiceChannelUser, filter *callSourceFilter) {
	so := p25p2.ServiceOptions(u.ServiceOptions)
	upd := trunking.CallSourceUpdate{
		DeviceSerial: serial,
		SourceID:     u.SourceID,
		Encrypted:    so.Encrypted(),
		Emergency:    so.Emergency(),
		Priority:     so.Priority(),
	}
	if filter != nil && !filter.admit(upd) {
		return
	}
	upd.At = time.Now().UTC()
	c.publishCallSource(upd)
}

// publishP25Phase2CallEncryption mirrors the Phase 1 LDU2 in-call
// encryption-sync path for the Phase 2 traffic channel's two copies of the
// sync: MAC_PTT at key-up and the voice bursts' ESS. The engine backfills
// ALGID/KID onto the bound ActiveCall's Grant and republishes with the
// call's identity. Repeats of the same algorithm and key are dropped by
// filter (nil publishes every one).
func (c *Composer) publishP25Phase2CallEncryption(serial string, es p25p2.EncryptionSync, filter *callEncryptionFilter) {
	if c.bus == nil {
		return
	}
	// Validity gate: a mis-decoded MAC Encryption Sync yields an Algorithm
	// ID outside the TIA-102 set (the field-observed smear across 0x00-0xFF,
	// one Key ID per call). Drop it rather than surface a plausible-but-wrong
	// algid+key that downstream can't tell from a real key. Issue #924.
	if !p25.AlgorithmKnown(es.AlgorithmID) {
		c.log.Debug("composer: p25p2 dropping out-of-set encryption sync",
			"serial", serial, "alg", p25.FormatAlgorithm(es.AlgorithmID), "key", es.KeyID)
		return
	}
	if filter != nil && !filter.admit(es) {
		return
	}
	c.bus.Publish(events.Event{
		Kind: events.KindCallEncryption,
		Payload: trunking.CallEncryption{
			DeviceSerial:     serial,
			AlgorithmID:      es.AlgorithmID,
			KeyID:            es.KeyID,
			MessageIndicator: es.MessageIndicator,
			At:               time.Now().UTC(),
		},
	})
}
