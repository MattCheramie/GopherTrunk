package composer

import (
	"context"
	"math"
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
	// TCH/S-decode the burst. Only bursts whose class-2 CRC verifies are TCH/S
	// speech for the granted call; everything else yields no frames.
	frames := tetra.TCHSpeechFrames(frame)
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
		speech.Add(1)
		if err := rs.WriteRawFrame(serial, sf); err != nil {
			c.log.Warn("composer: TETRA speech-frame write failed", "serial", serial, "err", err)
		}
	}
}
