package composer

import (
	"context"
	"math"
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
	chanBW := float64(bw)
	decim := int(math.Round(iqHz)) / tetraVoiceIntermediateHz
	filterAtUnity := false
	if decim <= 1 {
		filterAtUnity = true
		if chanBW > tetraVoiceChannelSelectHz {
			chanBW = tetraVoiceChannelSelectHz
		}
	}
	return newDecimatingFIR(iqHz, tetraVoiceIntermediateHz, chanBW, filterAtUnity)
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
	var bursts, speech, offSlot atomic.Uint64
	extractor := tetra.NewTrafficExtractor(colourExt, func(frame []byte, softType5 []float32, slot, usage uint8) {
		c.onTETRATrafficBurst(bt, rs, serial, frame, softType5, usage, usageMarker, &bursts, &speech, &offSlot)
	})

	rx := tetrarx.New(tetrarx.Options{
		SampleRateHz:        symbolHz,
		DibitSink:           func(d []uint8, base int) { extractor.Process(d, base) },
		SoftSink:            func(diffs []complex64, base int) { extractor.StashSoft(diffs, base) },
		ClockMode:           tetrarx.ClockGardner,
		GardnerGain:         0.005,
		EnableAFC:           true,
		EnableChannelFilter: true,
	})

	c.log.Info("composer: tetra voice follow started — TCH/S decode + ACELP vocoder",
		"serial", serial, "group", groupID, "timeslot", timeslot,
		"usage_marker", usageMarker, "colour_code", colourExt&0x3F, "rate_hz", symbolHz)

	defer func() {
		c.log.Info("composer: tetra voice follow ended",
			"serial", serial, "bursts", bursts.Load(), "speech_frames", speech.Load(),
			"other_call_bursts", offSlot.Load())
	}()

	touchTicker := time.NewTicker(c.touchEvery)
	defer touchTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-touchTicker.C:
			// Touch + hangtime end-of-call are driven by the boundary tracker.
		case iq, ok := <-iqCh:
			if !ok {
				return
			}
			bt.observe(iq)
			rx.Process(fe.Process(nil, iq))
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
func (c *Composer) onTETRATrafficBurst(bt *boundaryTracker, rs rawFrameSink, serial string, frame []byte, softType5 []float32, burstUsage, grantUsage uint8, bursts, speech, offSlot *atomic.Uint64) {
	bursts.Add(1)
	// Drop bursts that carry a different call's AACH usage marker. Only when both
	// markers are known (>= DLUsageTraffic) — an unknown marker on either side
	// falls through to the CRC gate so we never drop the granted call's own speech.
	if grantUsage >= tetra.DLUsageTraffic && burstUsage >= tetra.DLUsageTraffic && burstUsage != grantUsage {
		offSlot.Add(1)
		return
	}
	c.decodeTETRASpeech(bt, rs, serial, frame, softType5, speech)
}

// decodeTETRASpeech TCH/S-decodes one traffic frame for an owning call: it
// recovers the CRC-valid 137-bit speech frames, refreshes call liveness (only on
// real speech, never on raw bursts) and appends each frame to the recorder's
// `.raw` sidecar (the ACELP vocoder renders it to PCM downstream). A non-TCH/S
// burst (signalling, encrypted, or corrupt) yields no frames and does not touch
// liveness. Shared by the solo traffic tap and the same-carrier slot demux.
func (c *Composer) decodeTETRASpeech(bt *boundaryTracker, rs rawFrameSink, serial string, frame []byte, softType5 []float32, speech *atomic.Uint64) {
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
		// Not the granted call's speech — do NOT touch call liveness.
		return
	}
	// CRC-valid speech: keep the call alive and reset the hangtime timer.
	bt.onVoice(0)
	if rs == nil {
		return
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
}

// tetraSlotOwner is one call's registration with the carrier demux: the usage
// marker it follows (0 until a wildcard claims one) plus the boundary tracker +
// recorder sink its decoded speech drives. bursts/speech feed the ended log line.
type tetraSlotOwner struct {
	serial         string
	marker         uint8 // grant usage marker; 0 = wildcard until it claims one
	bt             *boundaryTracker
	rs             rawFrameSink
	bursts, speech atomic.Uint64
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
	})
	d.c.log.Info("composer: tetra shared voice demux started (usage-marker routing)",
		"key", d.key, "colour_code", d.colour&0x3F, "rate_hz", symbolHz)
	defer func() {
		d.c.log.Info("composer: tetra shared voice demux ended", "key", d.key,
			"undecoded_drops", d.undecodedDrops.Load(),
			"ownerless_drops", d.ownerlessDrops.Load(),
			"crc_fallbacks", d.crcFallbacks.Load(),
			"marker_collisions", d.markerCollisions.Load())
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
// marker. Non-traffic bursts (usage < DLUsageTraffic, including an undecoded AACH
// reported as 0) carry no speech and are dropped; a marker with no live owner is
// dropped too. When a marker has no owner but a wildcard call is waiting, the
// oldest wildcard claims that marker. Runs on the single demux goroutine, so each
// owner's boundary tracker has exactly one writer.
func (d *tetraSlotDemux) onBurst(frame []byte, softType5 []float32, slot, usage uint8) {
	// A burst whose AACH did not decode (usage 0) or is a control slot
	// (< DLUsageTraffic) carries no routable marker. Dropping it outright loses
	// the granted call's own speech whenever its AACH is momentarily
	// un-decodable — the shared-demux gap the solo tap did not have. Fall back to
	// the CRC gate, but ONLY when exactly one call is active (one owner and no
	// waiting wildcard, or one waiting wildcard and no owner): with a single call
	// there is no concurrent call to cross-talk into, so this matches the solo
	// tap's burstUsage==0 behaviour. With ≥2 active calls an unrouteable burst is
	// dropped (and counted) rather than mixed into an arbitrary recording.
	if usage < tetra.DLUsageTraffic {
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
		d.c.decodeTETRASpeech(sole.bt, sole.rs, sole.serial, frame, softType5, &sole.speech)
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
	d.mu.Unlock()
	if o == nil {
		d.ownerlessDrops.Add(1)
		return
	}
	o.bursts.Add(1)
	d.c.decodeTETRASpeech(o.bt, o.rs, o.serial, frame, softType5, &o.speech)
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

	o := &tetraSlotOwner{serial: serial, marker: usageMarker, bt: bt, rs: rs}
	d.addOwner(o)
	c.log.Info("composer: tetra voice follow started (shared demux) — TCH/S decode + ACELP vocoder",
		"serial", serial, "group", groupID, "timeslot", grantSlot, "usage_marker", usageMarker)
	defer func() {
		d.removeOwner(o)
		c.log.Info("composer: tetra voice follow ended",
			"serial", serial, "bursts", o.bursts.Load(), "speech_frames", o.speech.Load())
	}()

	<-ctx.Done()
}
