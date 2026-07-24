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

// runTETRAVoiceChain consumes IQ for one TETRA traffic-channel call. It
// decimates the tap IQ to the TETRA symbol rate, recovers the π/4-DQPSK dibit
// stream with the shared TETRA receiver, extracts each Normal Continuous
// Downlink Burst's two data blocks (tetra.TrafficExtractor), TCH/S-decodes each
// burst, and emits the recovered 137-bit speech frames to the recorder — which
// renders them to PCM with the clean-room ACELP vocoder ("tetra-acelp") and
// writes both the decoded WAV and a `.raw` sidecar of the speech frames. This
// mirrors the DMR/P25 shape (post-FEC frames in `.raw` + decoded WAV).
//
// The extractor emits a burst for every timeslot on the carrier (all four TDMA
// slots), each tagged with its timeslot (anchored to the synchronisation burst,
// TN1). This chain keeps only bursts on its granted timeslot and drops the rest
// (onTETRATrafficBurst), so up to four concurrent calls on one carrier — one per
// slot, each on its own same-carrier tap — decode into four independent
// recordings instead of mixing. Until the slot grid anchors (or on a traffic
// carrier with no synchronisation burst) the slot is unknown and the chain falls
// back to the TCH/S-CRC single-call isolation. Encrypted calls (TEA1-4) fail the
// CRC and produce no decoded audio (their raw bursts still exist upstream).
func (c *Composer) runTETRAVoiceChain(ctx context.Context, serial string, iqCh <-chan []complex64, iqHz float64, groupID uint32, timeslot uint8, colourExt uint32, done chan<- struct{}) {
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
	extractor := tetra.NewTrafficExtractor(colourExt, func(frame []byte, slot uint8) {
		c.onTETRATrafficBurst(bt, rs, serial, frame, slot, timeslot, &bursts, &speech, &offSlot)
	})

	rx := tetrarx.New(tetrarx.Options{
		SampleRateHz:        symbolHz,
		DibitSink:           func(d []uint8, base int) { extractor.Process(d, base) },
		ClockMode:           tetrarx.ClockGardner,
		GardnerGain:         0.005,
		EnableAFC:           true,
		EnableChannelFilter: true,
	})

	c.log.Info("composer: tetra voice follow started — TCH/S decode + ACELP vocoder",
		"serial", serial, "group", groupID, "timeslot", timeslot,
		"colour_code", colourExt&0x3F, "rate_hz", symbolHz)

	defer func() {
		c.log.Info("composer: tetra voice follow ended",
			"serial", serial, "bursts", bursts.Load(), "speech_frames", speech.Load(),
			"other_slot_bursts", offSlot.Load())
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
// Slot demultiplexing: the extractor tags each burst with its TDMA timeslot
// (slot, 1..4; 0 = not yet anchored to a synchronisation burst). When the slot
// is known and does NOT match this chain's granted timeslot, the burst belongs
// to another call sharing the carrier and is dropped here — this is what lets up
// to four concurrent calls on one carrier decode into four independent
// recordings. Until the grid anchors (slot 0), the chain falls back to the
// CRC-gated single-call behaviour (a non-same-carrier traffic tap has no SB, so
// slot stays 0 and every chain behaves as before).
//
// The class-2 CRC gate (tetra.TCHSpeechFrames) then isolates the granted call's
// speech: a non-TCH/S burst (signalling on another timeslot, an encrypted call,
// or a badly corrupted slot) returns no frames. Gating onVoice on that result
// makes TETRA teardown behave like every other protocol — the call ends hangtime
// after its last decoded speech frame (or via the no-voice startup timeout when
// a grant never decodes any speech).
func (c *Composer) onTETRATrafficBurst(bt *boundaryTracker, rs rawFrameSink, serial string, frame []byte, slot, grantSlot uint8, bursts, speech, offSlot *atomic.Uint64) {
	bursts.Add(1)
	// Drop bursts from another timeslot once the slot grid is anchored and the
	// grant names a slot. slot==0 (unanchored) or grantSlot==0 (unknown) falls
	// through to the CRC-gated single-call path.
	if slot != 0 && grantSlot != 0 && slot != grantSlot {
		offSlot.Add(1)
		return
	}
	c.decodeTETRASpeech(bt, rs, serial, frame, speech)
}

// decodeTETRASpeech TCH/S-decodes one traffic frame for an owning call: it
// recovers the CRC-valid 137-bit speech frames, refreshes call liveness (only on
// real speech, never on raw bursts) and appends each frame to the recorder's
// `.raw` sidecar (the ACELP vocoder renders it to PCM downstream). A non-TCH/S
// burst (signalling, encrypted, or corrupt) yields no frames and does not touch
// liveness. Shared by the same-carrier slot demux and the solo traffic tap.
func (c *Composer) decodeTETRASpeech(bt *boundaryTracker, rs rawFrameSink, serial string, frame []byte, speech *atomic.Uint64) {
	frames := tetra.TCHSpeechFrames(frame)
	if len(frames) == 0 {
		return
	}
	bt.onVoice(0)
	if rs == nil {
		return
	}
	for _, sf := range frames {
		if speech != nil {
			speech.Add(1)
		}
		if err := rs.WriteRawFrame(serial, sf); err != nil {
			c.log.Warn("composer: TETRA speech-frame write failed", "serial", serial, "err", err)
		}
	}
}

// --- Shared per-carrier TETRA slot demultiplexer ---------------------------
//
// On a TETRA single-carrier base station every concurrent call rides a different
// TDMA timeslot of the SAME control carrier, and all same-carrier voice taps see
// the SAME post-DDC IQ (the control decoder's voice fan-out, ccdecoder/voicetap.go).
// Running a fresh receiver + TrafficExtractor per call was the source of two
// cross-slot audio leaks:
//
//   - L1 pre-anchor accept-all: a fresh extractor has no synchronisation-burst
//     anchor yet, so slotOf returns 0 for up to ~1 s (one multiframe) and the
//     per-call path accepted EVERY slot's speech — the first second of every
//     recording absorbed all four slots.
//   - L2 hangtime slot reuse: a call lingers in hangtime after its speech stops;
//     if a new call reuses that physical slot, the lingering per-call extractor
//     kept decoding it into the old call's recording.
//
// tetraSlotDemux replaces the per-call extractors with ONE receiver + extractor
// for the whole carrier whose SB anchor stays warm across calls. It routes each
// decoded burst to the single call that currently owns that physical slot
// (owners map, most-recent grant wins), and drops unanchored (slot 0) bursts
// instead of broadcasting them. One demux exists per carrier for the composer's
// lifetime; per-call chains are thin owners that register/unregister a slot.
type tetraSlotDemux struct {
	c      *Composer
	key    string
	colour uint32
	cancel context.CancelFunc
	done   chan struct{}

	mu     sync.Mutex
	owners map[uint8]*tetraSlotOwner // slot 1..4 -> current owner (nil = unowned)
}

// tetraSlotOwner is one call's registration with the carrier demux: the slot it
// follows plus the boundary tracker + recorder sink its decoded speech drives.
// bursts/speech are per-call counters surfaced in the follow-ended log line.
type tetraSlotOwner struct {
	serial         string
	slot           uint8
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
	defer gtlog.Recover(d.c.log, "tetra-slot-demux:"+d.key, nil)

	fe := newTETRAVoiceFrontEnd(iqHz, d.c.bw)
	symbolHz := fe.OutRateHz()
	extractor := tetra.NewTrafficExtractor(d.colour, d.onBurst)
	rx := tetrarx.New(tetrarx.Options{
		SampleRateHz:        symbolHz,
		DibitSink:           func(di []uint8, base int) { extractor.Process(di, base) },
		ClockMode:           tetrarx.ClockGardner,
		GardnerGain:         0.005,
		EnableAFC:           true,
		EnableChannelFilter: true,
	})
	d.c.log.Info("composer: tetra shared slot demux started",
		"key", d.key, "colour_code", d.colour&0x3F, "rate_hz", symbolHz)
	defer d.c.log.Info("composer: tetra shared slot demux ended", "key", d.key)

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
	d.mu.Unlock()
}

// onBurst routes one extracted traffic frame to the call that owns its timeslot.
// Unanchored (slot 0) and out-of-range bursts are dropped — never broadcast (L1);
// a slot with no live owner is dropped too. This runs on the single demux
// goroutine, so each owner's boundary tracker has exactly one writer.
func (d *tetraSlotDemux) onBurst(frame []byte, slot uint8) {
	if slot < 1 || slot > 4 {
		return
	}
	d.mu.Lock()
	o := d.owners[slot]
	d.mu.Unlock()
	if o == nil {
		return
	}
	o.bursts.Add(1)
	d.c.decodeTETRASpeech(o.bt, o.rs, o.serial, frame, &o.speech)
}

// addOwner makes o the owner of its slot; the most-recent grant wins so a new
// call reusing a slot immediately displaces a hangtime-lingering one (L2).
func (d *tetraSlotDemux) addOwner(o *tetraSlotOwner) {
	d.mu.Lock()
	d.owners[o.slot] = o
	d.mu.Unlock()
}

// removeOwner releases o's slot, but only if o still owns it — a newer call that
// already claimed the slot keeps it.
func (d *tetraSlotDemux) removeOwner(o *tetraSlotOwner) {
	d.mu.Lock()
	if d.owners[o.slot] == o {
		delete(d.owners, o.slot)
	}
	d.mu.Unlock()
}

// runTETRASameCarrierChain is the thin per-call goroutine for a same-carrier
// TETRA call. It owns the call's boundary tracker (hangtime end-of-call + Touch
// heartbeat) and registers itself as the owner of its granted timeslot with the
// carrier's shared demux; the demux delivers that slot's decoded speech. It does
// no IQ work of its own. On ctx cancel (call end) it releases the slot.
func (c *Composer) runTETRASameCarrierChain(ctx context.Context, d *tetraSlotDemux, serial string, grantSlot uint8, done chan<- struct{}) {
	defer close(done)
	defer gtlog.Recover(c.log, "voice-chain-tetra-sc:"+serial, nil)

	// Talkgroup gating disabled (grantTG 0): the chain surfaces no per-burst
	// in-band identity, so liveness is driven purely by CRC-valid speech.
	bt := c.newBoundaryTracker(serial, 0, nil)
	go bt.run(ctx)
	rs, _ := c.sink.(rawFrameSink)

	o := &tetraSlotOwner{serial: serial, slot: grantSlot, bt: bt, rs: rs}
	d.addOwner(o)
	c.log.Info("composer: tetra voice follow started (shared demux) — TCH/S decode + ACELP vocoder",
		"serial", serial, "timeslot", grantSlot)
	defer func() {
		d.removeOwner(o)
		c.log.Info("composer: tetra voice follow ended",
			"serial", serial, "bursts", o.bursts.Load(), "speech_frames", o.speech.Load())
	}()

	<-ctx.Done()
}
