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
// slots), not just the granted one. Slot isolation is done by the TCH/S CRC:
// only the granted call's TCH/S bursts pass, so the other slots' signalling
// bursts are dropped (tetra.TCHSpeechFrames). This cleanly separates a single
// active call; concurrent TCH/S calls on the same carrier would still mix and
// need TDMA frame-number demux (a follow-up). groupID/timeslot are threaded
// through for logging and that future demux. Encrypted calls (TEA1-4) fail the
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
	var bursts, speech atomic.Uint64
	extractor := tetra.NewTrafficExtractor(colourExt, func(frame []byte) {
		bursts.Add(1)
		// Each recovered burst counts as traffic activity: keep the call
		// alive and reset the hangtime timer.
		bt.onVoice(0)
		if rs == nil {
			return
		}
		// TCH/S-decode the burst. Only bursts whose class-2 CRC verifies are
		// TCH/S speech for the granted call (other TDMA timeslots' bursts fail
		// the CRC); emit each recovered 137-bit speech frame to the recorder,
		// which renders it with the ACELP vocoder and appends it to the `.raw`
		// sidecar.
		for _, sf := range tetra.TCHSpeechFrames(frame) {
			speech.Add(1)
			if err := rs.WriteRawFrame(serial, sf); err != nil {
				c.log.Warn("composer: TETRA speech-frame write failed", "serial", serial, "err", err)
			}
		}
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
			"serial", serial, "bursts", bursts.Load(), "speech_frames", speech.Load())
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
