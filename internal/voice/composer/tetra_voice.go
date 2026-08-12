package composer

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	tetrarx "github.com/MattCheramie/GopherTrunk/internal/radio/tetra/receiver"
)

// tetraVoiceIntermediateHz is the rate the wideband voice-tap IQ is
// decimated to before the TETRA receiver runs. It matches the per-protocol
// channel rate the control-channel DDC normalises to (144 kHz), giving the
// 18000-baud π/4-DQPSK stream 8 samples/symbol — the rate the receiver's
// Gardner loop, AFC and channel filter are tuned for. A tap already at or
// below this rate is used as-is (the receiver needs only ≥ 2×18 kHz).
const tetraVoiceIntermediateHz = 144_000

// tetraVoiceChannelSelectHz caps the voice front-end channel filter at half
// the 25 kHz TETRA channel spacing when the tap already streams at the
// intermediate rate (so the front end still band-limits to one carrier
// before the receiver). Mirrors dmrVoiceChannelSelectHz.
const tetraVoiceChannelSelectHz = 12_500.0

// newTETRAVoiceFrontEnd builds the channel-select + decimation front end for
// the TETRA voice chain, mirroring newDMRVoiceFrontEnd: a wideband DDC tap
// already at the intermediate rate (decim==1) is band-limited to a single
// 25 kHz channel; a higher-rate tap is decimated with the anti-alias FIR.
func newTETRAVoiceFrontEnd(iqHz float64, bw uint32) *decimatingFIR {
	return newVoiceFrontEnd(iqHz, bw, tetraVoiceIntermediateHz, tetraVoiceChannelSelectHz)
}

// runTETRAVoiceChain consumes IQ for one TETRA traffic-channel call on a SOLO tap
// — a dedicated retuned voice SDR carrying a single call (one call per traffic
// carrier), NOT a same-carrier SCBS. (Concurrent same-carrier calls go through the
// shared per-carrier tetraSlotDemux instead — see followTETRASameCarrier.) It
// decimates the tap IQ to the TETRA symbol rate, recovers the π/4-DQPSK dibit
// stream with the shared TETRA receiver, extracts each Normal Continuous Downlink
// Burst's two data blocks (tetra.TrafficExtractor), TCH/S-decodes each burst, and
// emits the recovered 137-bit speech frames to the recorder — which renders them
// to PCM with the clean-room ACELP vocoder ("tetra-acelp") and writes both the
// decoded WAV and a `.raw` sidecar. This mirrors the DMR/P25 shape.
//
// Each burst is tagged with the AACH downlink usage marker of the slot it came
// from; the chain keeps only bursts whose marker matches the granted call's
// (onTETRATrafficBurst). When the grant carries no usage marker, or a burst's AACH
// does not decode, the chain falls back to TCH/S-CRC single-call isolation so a
// call's own speech is never dropped on a guess — which is the common case on a
// solo tap (one call present). Encrypted calls (TEA1-4) fail the CRC and produce
// no decoded audio (their raw bursts still exist upstream).
func (c *Composer) runTETRAVoiceChain(ctx context.Context, serial string, iqCh <-chan []complex64, iqHz float64, groupID uint32, timeslot uint8, colourExt uint32, usageMarker uint8, done chan<- struct{}) {
	defer close(done)
	defer gtlog.Recover(c.log, "voice-chain-tetra:"+serial, nil)

	// Shared boundary controller: hangtime end-of-call + Touch heartbeat.
	// Talkgroup gating is disabled (grantTG 0) until the chain surfaces a
	// per-burst in-band identity.
	bt := c.newBoundaryTracker(serial, 0, nil)
	go bt.run(ctx)

	fe := newTETRAVoiceFrontEnd(iqHz, c.bw)
	symbolHz := fe.OutRateHz()

	rs, _ := c.sink.(rawFrameSink)
	var bursts, speech, offSlot, bfi atomic.Uint64
	extractor := tetra.NewTrafficExtractor(colourExt, func(frame []byte, softType5 []float32, slot, usage uint8) {
		c.onTETRATrafficBurst(bt, rs, serial, frame, softType5, usage, usageMarker, &bursts, &speech, &offSlot, &bfi)
	})

	rx := tetrarx.New(tetrarx.Options{
		SampleRateHz:        symbolHz,
		DibitSink:           func(d []uint8, base int) { extractor.Process(d, base) },
		SoftSink:            func(diffs []complex64, base int) { extractor.StashSoft(diffs, base) },
		ClockMode:           tetrarx.ClockGardner,
		GardnerGain:         0.005,
		EnableAFC:           true,
		EnableChannelFilter: true,
		EnableEqualizer:     true, // invert linear channel/ISI that garbles TCH/S (#764/#771 follow-up)
		EnableDCBlock:       true, // strip the front-end DC spur that leaks into same-carrier voice under heavy multislot
	})

	c.log.Info("composer: tetra voice follow started — TCH/S decode + ACELP vocoder",
		"serial", serial, "group", groupID, "timeslot", timeslot,
		"usage_marker", usageMarker, "colour_code", colourExt&0x3F, "rate_hz", symbolHz)

	defer func() {
		c.log.Info("composer: tetra voice follow ended",
			"serial", serial, "bursts", bursts.Load(),
			"speech_frames", speech.Load(), "tch_frames", speech.Load(),
			"stch_bursts", extractor.StolenBursts(), "bfi_count", bfi.Load(),
			"other_call_bursts", offSlot.Load())
	}()

	process := func(iq []complex64) {
		bt.observe(iq)
		rx.Process(fe.Process(nil, iq))
	}
	touchTicker := time.NewTicker(c.touchEvery)
	defer touchTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			// The call is being torn down (usually a control-channel D-RELEASE,
			// which cancels immediately and bypasses the voice hangtime). Drain
			// whatever IQ is already buffered in iqCh before returning, so the tail
			// of the transmission — the last speech bursts that arrived just before
			// the release — is still demodulated and recorded instead of dropped
			// mid-flight. The recorder is finalized only after this goroutine
			// returns (handleEnd's <-ch.done), so these late frames still land in
			// the .raw. Mirrors the same-carrier owner worker's cancel drain
			// (tetraOwnerWorker).
			drainTETRAIQ(iqCh, process)
			return
		case <-touchTicker.C:
			// Touch + hangtime end-of-call are driven by the boundary tracker.
		case iq, ok := <-iqCh:
			if !ok {
				return
			}
			process(iq)
		}
	}
}

// drainTETRAIQ consumes whatever IQ is already buffered in ch (non-blocking) and
// feeds each chunk to process. It returns as soon as the buffer is empty or the
// channel is closed — it never waits for more IQ. Used on call teardown to
// recover the tail of a transmission still queued when the chain is cancelled,
// rather than dropping it (the "recordings end a beat early" report).
func drainTETRAIQ(ch <-chan []complex64, process func([]complex64)) {
	for {
		select {
		case iq, ok := <-ch:
			if !ok {
				return
			}
			process(iq)
		default:
			return
		}
	}
}

// onTETRATrafficBurst handles one Normal Continuous Downlink Burst recovered by
// the TrafficExtractor. Call liveness (the hangtime end-of-call) is driven ONLY
// by CRC-valid TCH/S speech for the granted call — not by every raw burst.
//
// This matters because the extractor emits a burst for all four TDMA timeslots
// on the carrier, continuously, whether or not it is the granted call's speech.
// Refreshing the boundary tracker on every raw burst (the previous behaviour)
// kept lastVoiceNano perpetually fresh while the carrier was up, so the hangtime
// never elapsed and the (single, same-carrier) voice device was held forever —
// every later grant was then dropped with "no voice device available for grant".
//
// Usage-marker demultiplexing: the extractor tags each burst with the AACH
// downlink usage marker of the slot it came from (burstUsage, >= DLUsageTraffic
// for a traffic slot, 0 when the AACH did not decode or the slot is not traffic).
// The grant carries the call's usage marker (grantUsage). When both are present
// and differ, the burst belongs to another call sharing the carrier and is
// dropped — this is what lets up to four concurrent same-carrier calls decode
// into independent recordings. The channel-allocation timeslot field is NOT used:
// on real air it does not map to the physical slot (distinct calls collide on one
// value), which silently starved every mis-mapped call; the AACH usage marker is
// the reliable per-slot call identifier.
//
// Fallbacks (never discard the granted call's own speech on a guess):
//   - grantUsage == 0 (the grant was addressed by plain SSI, no usage marker):
//     accept every CRC-valid burst — the pre-demux single-call behaviour. Audio
//     is preserved; true concurrency without markers may mix (rare).
//   - burstUsage == 0 (this burst's AACH did not decode): let the CRC gate decide
//     rather than dropping, so an occasional AACH miss does not drop own speech.
//
// The class-2 CRC gate (tetra.TCHSpeechFrames) then isolates the granted call's
// speech: a non-TCH/S burst (signalling, an encrypted call, or a badly corrupted
// slot) returns no frames. Gating onVoice on that result makes TETRA teardown
// behave like every other protocol — the call ends hangtime after its last
// decoded speech frame (or via the no-voice startup timeout when a grant never
// decodes any speech).
func (c *Composer) onTETRATrafficBurst(bt *boundaryTracker, rs rawFrameSink, serial string, frame []byte, softType5 []float32, burstUsage, grantUsage uint8, bursts, speech, offSlot, bfi *atomic.Uint64) {
	bursts.Add(1)
	// Drop bursts that carry a different call's AACH usage marker. Only when both
	// markers are known (>= DLUsageTraffic) — an unknown marker on either side
	// falls through to the CRC gate so we never drop the granted call's own speech.
	if grantUsage >= tetra.DLUsageTraffic && burstUsage >= tetra.DLUsageTraffic && burstUsage != grantUsage {
		offSlot.Add(1)
		return
	}
	// A burst accepted for this call that yields no speech is a Bad Frame
	// Indication (corrupt/encrypted → concealment); count it.
	if c.decodeTETRASpeech(bt, rs, serial, frame, softType5, speech) == 0 {
		bfi.Add(1)
	}
}

// decodeTETRASpeech TCH/S-decodes one traffic frame for an owning call: it
// recovers the CRC-valid 137-bit speech frames, refreshes call liveness (only on
// real speech, never on raw bursts) and appends each frame to the recorder's
// `.raw` sidecar (the ACELP vocoder renders it to PCM downstream). A non-TCH/S
// burst (signalling, encrypted, or corrupt) yields no frames and does not touch
// liveness. Shared by the solo traffic tap and the same-carrier slot demux.
// Returns the number of CRC-valid 137-bit speech frames recovered: 0 for a burst
// that yielded no speech (a Bad Frame Indication the callers count toward
// bfi_count), a positive count for real speech (feeds tch_frames).
func (c *Composer) decodeTETRASpeech(bt *boundaryTracker, rs rawFrameSink, serial string, frame []byte, softType5 []float32, speech *atomic.Uint64) int {
	// Prefer soft-decision TCH/S when the extractor supplied the burst's LLRs:
	// the soft Viterbi's ~2 dB coding gain recovers real speech bursts the
	// hard-decision gate drops on a marginal same-carrier signal (the fix for
	// the short/garbled recordings). Fall back to the hard gate when no soft
	// info is available (never worse than before).
	var frames [][]byte
	if softType5 != nil {
		frames = tetra.TCHSpeechFramesSoft(softType5)
	} else {
		frames = tetra.TCHSpeechFrames(frame)
	}
	if len(frames) == 0 {
		// Not the granted call's speech — do NOT touch call liveness. A burst
		// routed to an owner that yields nothing here is a Bad Frame Indication
		// (the caller counts it).
		return 0
	}
	// CRC-valid speech: keep the call alive and reset the hangtime timer.
	bt.onVoice(0)
	if rs == nil {
		return len(frames)
	}
	// Emit each recovered 137-bit speech frame to the recorder, which renders it
	// with the ACELP vocoder and appends it to the `.raw` sidecar.
	for _, sf := range frames {
		if speech != nil {
			speech.Add(1)
		}
		if err := rs.WriteRawFrame(serial, sf); err != nil {
			c.log.Warn("composer: TETRA speech-frame write failed", "serial", serial, "err", err)
		}
	}
	return len(frames)
}

// --- Shared per-carrier TETRA voice demultiplexer --------------------------
//
// On a TETRA single-carrier base station every concurrent call rides a different
// TDMA timeslot of the SAME control carrier, and all same-carrier voice taps see
// the SAME post-DDC IQ (the control decoder's voice fan-out, ccdecoder/voicetap.go).
// Running a fresh receiver + TrafficExtractor per call caused cross-slot audio
// leaks:
//
//   - pre-anchor accept-all: a fresh per-call extractor had no reference frame yet,
//     so it could not tell one call's speech from another's for the first second.
//   - hangtime marker reuse: a call lingers in hangtime after its speech stops; if
//     the network reuses its usage marker for a new call, a per-call chain (which
//     cannot evict a peer) kept decoding it into the old call's recording.
//
// tetraSlotDemux replaces the per-call extractors with ONE receiver + extractor
// for the whole carrier whose AACH/SB state stays warm across calls. It routes
// each decoded burst to the single call that owns that burst's AACH downlink
// usage marker (the reliable per-call identifier — the channel-allocation timeslot
// does not map to the physical slot on real air, and the SB slot index jitters
// across adjacent slots). owners is keyed by usage marker with most-recent-grant-
// wins, so a new call for a reused marker immediately displaces the lingering one.
// One demux exists per carrier for the composer's lifetime; per-call chains are
// thin owners that register/unregister a marker.
type tetraSlotDemux struct {
	c      *Composer
	key    string
	colour uint32
	cancel context.CancelFunc
	done   chan struct{}

	mu     sync.Mutex
	owners map[uint8]*tetraSlotOwner // usage marker (>=DLUsageTraffic) -> current owner
	// wildcards are owners whose grant carried no usage marker (addressed by plain
	// SSI). Each claims the first unclaimed traffic marker the demux observes, in
	// registration order, so concurrent marker-less calls still separate instead of
	// falling back to accept-all/mix. Best-effort; verified against real DDC IQ.
	wildcards []*tetraSlotOwner

	// Drop/fallback counters — the demux's routing was previously a telemetry
	// blind spot (a slot-loss showed up only as an empty recording). Logged at
	// demux teardown so a "gappy recordings" report is self-triaging: undecoded =
	// bursts with no routable AACH marker dropped (no single owner to fall back
	// to), ownerless = decoded traffic markers with no registered call, crcFallback
	// = undecoded bursts routed to the sole active call via the CRC gate,
	// collisions = two calls contending for one marker.
	undecodedDrops   atomic.Uint64
	ownerlessDrops   atomic.Uint64
	crcFallbacks     atomic.Uint64
	markerCollisions atomic.Uint64
	// concurrencySuppressed counts bursts the single-owner CRC fallback WOULD have
	// routed but dropped instead because ≥2 physical TDMA slots were carrying
	// traffic — i.e. the carrier had concurrent calls even though GT had only one
	// owner registered. Those fallbacks were the cross-slot audio-leak vector:
	// a valid TCH/S CRC proves a burst is speech, not WHOSE speech, so funnelling
	// another slot's speech into the sole recording bled calls together.
	concurrencySuppressed atomic.Uint64

	// Voice-path decode telemetry, aggregated across all owners on this carrier
	// and logged at demux teardown (the voice analogue of the CC decode-status
	// line). tchFrames = CRC-valid speech frames routed to the ACELP vocoder;
	// bfiFrames = bursts routed to an owner that yielded no speech (Bad Frame
	// Indication → concealment). Stolen-slot (STCH) bursts are counted by the
	// TrafficExtractor and read from it directly at teardown.
	tchFrames atomic.Uint64
	bfiFrames atomic.Uint64

	// slotRing is a sliding window of the physical TDMA slot (1..4) of recent
	// traffic-marked bursts, used only to decide whether the carrier is currently
	// running concurrent calls. Unlike the AACH usage marker (which decodes on a
	// minority of bursts) the SB-anchored physical slot is available on ~100% of
	// bursts, so counting distinct busy slots is a robust concurrency signal. The
	// ring is touched only on the single demux goroutine (onBurst), so it needs no
	// lock. 0 entries are empty slots.
	slotRing    [concurrencyWindow]uint8
	slotRingPos int
}

const (
	// concurrencyWindow is how many recent traffic-marked bursts the slot ring
	// remembers — ~one TETRA multiframe of traffic across four slots.
	concurrencyWindow = 48
	// minSlotBurstsForActive is how many of a slot's traffic-marked bursts must
	// appear in the window before it counts as an active call, high enough that
	// the ~5% SB-anchor slot jitter (a burst tagged one slot off) cannot make a
	// single active call look like two.
	minSlotBurstsForActive = 3
)

// tetraVocoderQueueDepth is the per-owner backlog of decoded traffic bursts
// awaiting TCH/S channel decode + ACELP synthesis on the owner's worker
// goroutine. A downlink call occupies one TDMA slot ≈ 18 bursts/s, so a worker
// clears its backlog far faster than it fills; the depth only absorbs scheduling
// jitter (~3 s here). It exists so the demux goroutine's enqueue is non-blocking:
// vocoding must never stall the single goroutine that drains the carrier's IQ (a
// stall there drops post-DDC IQ chunks upstream and garbles every slot — the
// multislot garble). On the rare overflow a burst is dropped and counted rather
// than blocking the demux.
const tetraVocoderQueueDepth = 64

// tetraDecodeJob is one decoded traffic burst handed from the demux goroutine to
// an owner's vocoder worker. The frame / soft-LLR slices are COPIES: the
// TrafficExtractor may reuse its backing arrays for the next burst, so the demux
// goroutine must not hand the worker a slice it will overwrite.
type tetraDecodeJob struct {
	frame     []byte
	softType5 []float32
}

// tetraSlotOwner is one call's registration with the carrier demux: the usage
// marker it follows (0 until a wildcard claims one) plus the boundary tracker +
// recorder sink its decoded speech drives. bursts/speech feed the ended log line.
//
// Concurrency: the demux goroutine routes a burst to this owner and enqueues it on
// jobs; a single per-owner worker goroutine (tetraOwnerWorker) dequeues and runs
// decodeTETRASpeech. One worker per owner keeps decodeTETRASpeech's boundary-tracker
// bookkeeping single-writer (as boundary.go requires), while moving the heavy
// Viterbi + ACELP off the demux goroutine. The demux still folds carrier power into
// bt via observe(), but that touches a disjoint non-atomic field set (sumSq/nSamp)
// from onVoice's (lastMatch/foreignRun/…), so the two goroutines do not race.
type tetraSlotOwner struct {
	serial         string
	marker         uint8           // grant usage marker; 0 = wildcard until it claims one
	d              *tetraSlotDemux // owning carrier demux, for aggregate voice telemetry (nil in the solo tap)
	bt             *boundaryTracker
	rs             rawFrameSink
	bursts, speech atomic.Uint64

	jobs         chan tetraDecodeJob // decoded bursts awaiting the vocoder worker
	vocoderDrops atomic.Uint64       // bursts dropped because the worker queue was full
}

// enqueue hands a decoded burst to the owner's vocoder worker without ever
// blocking the demux goroutine. The slices are copied (the extractor reuses its
// buffers) and, on a full queue, the burst is dropped and counted rather than
// stalling IQ intake.
func (o *tetraSlotOwner) enqueue(frame []byte, softType5 []float32) {
	job := tetraDecodeJob{}
	if frame != nil {
		job.frame = append([]byte(nil), frame...)
	}
	if softType5 != nil {
		job.softType5 = append([]float32(nil), softType5...)
	}
	select {
	case o.jobs <- job:
	default:
		o.vocoderDrops.Add(1)
	}
}

// decode runs one queued burst through decodeTETRASpeech and folds the outcome
// into the carrier demux's aggregate voice telemetry: CRC-valid speech frames
// into tch_frames, a burst that yielded no speech into bfi_count (a Bad Frame
// Indication — this burst was routed to the owner as its slot but could not be
// turned into speech, so it is concealed).
func (o *tetraSlotOwner) decode(c *Composer, job tetraDecodeJob) {
	n := c.decodeTETRASpeech(o.bt, o.rs, o.serial, job.frame, job.softType5, &o.speech)
	if o.d == nil {
		return
	}
	if n > 0 {
		o.d.tchFrames.Add(uint64(n))
	} else {
		o.d.bfiFrames.Add(1)
	}
}

// tetraOwnerWorker is the single per-owner goroutine that runs the CPU-heavy
// TCH/S channel decode + ACELP synthesis for one call, off the shared demux
// goroutine. On ctx cancel (call end) it drains whatever bursts are already
// queued — so tail speech at the end of a transmission is not lost — then exits.
func (c *Composer) tetraOwnerWorker(ctx context.Context, o *tetraSlotOwner) {
	defer gtlog.Recover(c.log, "tetra-voice-worker:"+o.serial, nil)
	for {
		select {
		case <-ctx.Done():
			for {
				select {
				case job := <-o.jobs:
					o.decode(c, job)
				default:
					return
				}
			}
		case job := <-o.jobs:
			o.decode(c, job)
		}
	}
}

// run streams the carrier's post-DDC IQ through one front end + receiver +
// TrafficExtractor, feeding every burst to onBurst. It self-removes from the
// composer's registry when the IQ stream ends so a later grant rebuilds it.
func (d *tetraSlotDemux) run(ctx context.Context, iqCh <-chan []complex64, iqHz float64) {
	defer close(d.done)
	defer d.c.removeTETRADemux(d.key, d)
	defer gtlog.Recover(d.c.log, "tetra-voice-demux:"+d.key, nil)

	fe := newTETRAVoiceFrontEnd(iqHz, d.c.bw)
	symbolHz := fe.OutRateHz()
	extractor := tetra.NewTrafficExtractor(d.colour, d.onBurst)
	rx := tetrarx.New(tetrarx.Options{
		SampleRateHz:        symbolHz,
		DibitSink:           func(di []uint8, base int) { extractor.Process(di, base) },
		SoftSink:            func(diffs []complex64, base int) { extractor.StashSoft(diffs, base) },
		ClockMode:           tetrarx.ClockGardner,
		GardnerGain:         0.005,
		EnableAFC:           true,
		EnableChannelFilter: true,
		EnableEqualizer:     true, // invert linear channel/ISI that garbles TCH/S (#764/#771 follow-up)
		EnableDCBlock:       true, // strip the front-end DC spur that leaks into same-carrier voice under heavy multislot
	})
	d.c.log.Info("composer: tetra shared voice demux started (usage-marker routing)",
		"key", d.key, "colour_code", d.colour&0x3F, "rate_hz", symbolHz)
	defer func() {
		d.c.log.Info("composer: tetra shared voice demux ended", "key", d.key,
			"tch_frames", d.tchFrames.Load(),
			"stch_bursts", extractor.StolenBursts(),
			"bfi_count", d.bfiFrames.Load(),
			"undecoded_drops", d.undecodedDrops.Load(),
			"ownerless_drops", d.ownerlessDrops.Load(),
			"crc_fallbacks", d.crcFallbacks.Load(),
			"marker_collisions", d.markerCollisions.Load(),
			"concurrency_suppressed", d.concurrencySuppressed.Load())
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case iq, ok := <-iqCh:
			if !ok {
				return
			}
			d.observe(iq)
			rx.Process(fe.Process(nil, iq))
		}
	}
}

// observe folds the carrier's channel power into every current owner's signal
// meter. All calls share one carrier, so its post-DDC power is each call's RSSI.
func (d *tetraSlotDemux) observe(iq []complex64) {
	d.mu.Lock()
	for _, o := range d.owners {
		o.bt.observe(iq)
	}
	for _, o := range d.wildcards {
		o.bt.observe(iq)
	}
	d.mu.Unlock()
}

// onBurst routes one extracted traffic frame to the call that owns its AACH usage
// marker. When a marker has no owner but a wildcard call is waiting, the oldest
// wildcard claims that marker. A burst that cannot be routed by marker — an
// undecoded/control AACH (usage < DLUsageTraffic) or a decoded marker with no
// registered owner — is handed to the CRC gate of the sole active call when
// exactly one call is up (a stray/undecoded AACH on that call's own burst), and
// dropped (counted) only when ≥2 calls are concurrent, where cross-talk would be
// possible. Runs on the single demux goroutine, so each owner's boundary tracker
// has exactly one writer.
func (d *tetraSlotDemux) onBurst(frame []byte, softType5 []float32, slot, usage uint8) {
	// Fold this burst's physical slot into the concurrency window (traffic-marked
	// bursts only — those unambiguously mark a slot as carrying an active call).
	d.noteSlotActivity(slot, usage)

	// A burst whose AACH did not decode (usage 0) or is a control slot
	// (< DLUsageTraffic) carries no routable marker. Dropping it outright loses
	// the granted call's own speech whenever its AACH is momentarily
	// un-decodable — the shared-demux gap the solo tap did not have. Fall back to
	// the CRC gate, but ONLY when the carrier is genuinely running one call: a
	// single registered owner AND the physical slots show no concurrent traffic.
	// The registered-owner count alone is not enough — a call GT never granted (a
	// missed grant, a call in hangtime, a wakeup-page ghost) still keys up a
	// physical slot, and routing that foreign slot's speech to GT's one owner via
	// the CRC gate is exactly the cross-slot leak (a valid TCH/S CRC proves the
	// burst is speech, not whose). When ≥2 slots are active the burst is dropped
	// rather than mixed into an arbitrary recording.
	if usage < tetra.DLUsageTraffic {
		if d.onAirConcurrent() {
			d.concurrencySuppressed.Add(1)
			d.undecodedDrops.Add(1)
			return
		}
		d.mu.Lock()
		var sole *tetraSlotOwner
		switch {
		case len(d.owners) == 1 && len(d.wildcards) == 0:
			for _, o := range d.owners {
				sole = o
			}
		case len(d.owners) == 0 && len(d.wildcards) == 1:
			sole = d.wildcards[0]
		}
		d.mu.Unlock()
		if sole == nil {
			d.undecodedDrops.Add(1)
			return
		}
		d.crcFallbacks.Add(1)
		sole.bursts.Add(1)
		sole.enqueue(frame, softType5)
		return
	}
	d.mu.Lock()
	o := d.owners[usage]
	if o == nil && len(d.wildcards) > 0 {
		o = d.wildcards[0]
		d.wildcards = d.wildcards[1:]
		o.marker = usage
		d.owners[usage] = o
	}
	// No owner for this decoded marker and no wildcard to claim it. When exactly
	// one call is active, this is that call's own burst whose AACH usage marker
	// miscorrected to a stray value (a common RM(30,14) miss on a marginal AACH):
	// capture the sole owner so it goes through the CRC gate below instead of
	// being dropped. Guarded on BOTH a single registered owner AND no concurrent
	// on-air traffic — an ownerless traffic marker while another physical slot is
	// active is far more likely a genuine peer call GT isn't tracking than the one
	// owner's own miscorrection, and routing it in is the cross-slot leak.
	var sole *tetraSlotOwner
	if o == nil && len(d.owners) == 1 && len(d.wildcards) == 0 && !d.onAirConcurrent() {
		for _, only := range d.owners {
			sole = only
		}
	}
	d.mu.Unlock()
	if o == nil {
		if sole != nil {
			d.crcFallbacks.Add(1)
			sole.bursts.Add(1)
			sole.enqueue(frame, softType5)
			return
		}
		if d.onAirConcurrent() {
			d.concurrencySuppressed.Add(1)
		}
		d.ownerlessDrops.Add(1)
		return
	}
	o.bursts.Add(1)
	o.enqueue(frame, softType5)
}

// noteSlotActivity folds a traffic-marked burst's physical slot into the sliding
// concurrency window. Only usage >= DLUsageTraffic bursts are recorded: those
// unambiguously mark a slot as carrying a live call, whereas an undecoded AACH
// (usage 0) tells us nothing about which slot is busy. Runs on the demux
// goroutine, so the ring needs no lock.
func (d *tetraSlotDemux) noteSlotActivity(slot, usage uint8) {
	if usage < tetra.DLUsageTraffic || slot < 1 || slot > 4 {
		return
	}
	d.slotRing[d.slotRingPos] = slot
	d.slotRingPos = (d.slotRingPos + 1) % len(d.slotRing)
}

// onAirConcurrent reports whether the carrier is currently running more than one
// call, judged from how many distinct physical slots carry traffic in the recent
// window. This is decoupled from how many owners GT has registered: a call GT
// never granted still keys up a slot, and it is exactly that untracked concurrent
// traffic that the single-owner CRC fallback used to leak into the one recording.
func (d *tetraSlotDemux) onAirConcurrent() bool {
	var counts [5]int // index 1..4
	for _, s := range d.slotRing {
		if s >= 1 && s <= 4 {
			counts[s]++
		}
	}
	active := 0
	for s := 1; s <= 4; s++ {
		if counts[s] >= minSlotBurstsForActive {
			active++
		}
	}
	return active >= 2
}

// addOwner registers o. A marker-bearing grant claims its marker (most-recent
// grant wins, so a new call reusing a marker displaces a hangtime-lingering one);
// a marker-less (wildcard) grant queues to claim the first unowned marker seen.
func (d *tetraSlotDemux) addOwner(o *tetraSlotOwner) {
	d.mu.Lock()
	if o.marker >= tetra.DLUsageTraffic {
		if prev := d.owners[o.marker]; prev != nil && prev != o {
			// Two live calls contending for one on-air marker: most-recent-grant
			// wins (a reused marker must displace a hangtime-lingering call), but
			// if these are genuinely concurrent the prior call now starves. Count
			// + warn so the case is visible rather than a silent empty recording.
			d.markerCollisions.Add(1)
			d.c.log.Warn("composer: tetra usage-marker collision — newest call takes the marker, prior call starves",
				"key", d.key, "marker", o.marker, "prev", prev.serial, "new", o.serial)
		}
		d.owners[o.marker] = o
	} else {
		d.wildcards = append(d.wildcards, o)
	}
	d.mu.Unlock()
}

// removeOwner releases o's marker (only if o still owns it — a newer call that
// already displaced it keeps it) and drops it from the wildcard queue if it never
// claimed a marker.
func (d *tetraSlotDemux) removeOwner(o *tetraSlotOwner) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if o.marker >= tetra.DLUsageTraffic && d.owners[o.marker] == o {
		delete(d.owners, o.marker)
	}
	for i, w := range d.wildcards {
		if w == o {
			d.wildcards = append(d.wildcards[:i], d.wildcards[i+1:]...)
			break
		}
	}
}

// runTETRASameCarrierChain is the thin per-call goroutine for a same-carrier
// TETRA call. It owns the call's boundary tracker (hangtime end-of-call + Touch
// heartbeat) and registers itself as the owner of its granted AACH usage marker
// with the carrier's shared demux; the demux delivers that marker's decoded
// speech. It does no IQ work of its own. On ctx cancel (call end) it releases the
// marker.
func (c *Composer) runTETRASameCarrierChain(ctx context.Context, d *tetraSlotDemux, serial string, groupID uint32, grantSlot, usageMarker uint8, done chan<- struct{}) {
	defer close(done)
	defer gtlog.Recover(c.log, "voice-chain-tetra-sc:"+serial, nil)

	// Talkgroup gating disabled (grantTG 0): the chain surfaces no per-burst
	// in-band identity, so liveness is driven purely by CRC-valid speech.
	bt := c.newBoundaryTracker(serial, 0, nil)
	go bt.run(ctx)
	rs, _ := c.sink.(rawFrameSink)

	o := &tetraSlotOwner{
		serial: serial, marker: usageMarker, d: d, bt: bt, rs: rs,
		jobs: make(chan tetraDecodeJob, tetraVocoderQueueDepth),
	}
	// Start the vocoder worker BEFORE registering the owner, so a burst routed the
	// instant addOwner returns already has a draining consumer. The worker exits
	// (after draining) when ctx is cancelled at call end. workerDone closes when
	// the worker has finished draining its queued jobs and written the last
	// speech frame.
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		c.tetraOwnerWorker(ctx, o)
	}()
	d.addOwner(o)
	c.log.Info("composer: tetra voice follow started (shared demux) — TCH/S decode + ACELP vocoder",
		"serial", serial, "group", groupID, "timeslot", grantSlot, "usage_marker", usageMarker)
	defer func() {
		// Stop new bursts routing to this owner, then WAIT for the worker to
		// drain the bursts already queued (the transmission tail) and write
		// their speech frames. This must complete before runTETRASameCarrierChain
		// returns and closes done: composer.handleEnd signals the recorder to
		// finalize right after <-ch.done, so a worker still draining here would
		// otherwise land the tail frames on an already-finalized (deleted)
		// session and drop them (the "missing trailing voice frames" report).
		d.removeOwner(o)
		<-workerDone
		c.log.Info("composer: tetra voice follow ended",
			"serial", serial, "bursts", o.bursts.Load(), "speech_frames", o.speech.Load(),
			"vocoder_drops", o.vocoderDrops.Load())
	}()

	<-ctx.Done()
}
