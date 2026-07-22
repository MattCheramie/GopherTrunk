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
// decimates the tap IQ to the TETRA symbol rate, recovers the π/4-DQPSK
// dibit stream with the shared TETRA receiver, extracts each Normal
// Continuous Downlink Burst's two data blocks (tetra.TrafficExtractor), and
// appends them to the recorder's `.raw` sidecar as raw full-slot traffic
// frames.
//
// This is the traffic-following + capture half of TETRA voice support. TCH/S
// channel decoding (§8.4) and the TETRA ACELP vocoder that would turn the
// raw frames into PCM are the labelled follow-ups; until they land the `.raw`
// sidecar is the capture, exactly as for DMR/P25 (post-FEC frames) and EDACS
// ProVoice (raw frames). Encrypted calls (TEA1-4) still capture raw — the
// bytes exist even when no in-process decode does.
//
// The extractor emits a frame for every burst on the carrier (all four TDMA
// timeslots), not just the granted one; groupID/timeslot are threaded
// through for a future per-slot filter and logging only.
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
	var bursts atomic.Uint64
	extractor := tetra.NewTrafficExtractor(colourExt, func(frame []byte) {
		bursts.Add(1)
		// Each recovered burst counts as traffic activity: keep the call
		// alive and reset the hangtime timer.
		bt.onVoice(0)
		if rs == nil {
			return
		}
		if err := rs.WriteRawFrame(serial, frame); err != nil {
			c.log.Warn("composer: TETRA raw-frame write failed", "serial", serial, "err", err)
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

	c.log.Info("composer: tetra voice follow started — descrambled traffic sidecar (TCH/S FEC + ACELP vocoder are follow-ups)",
		"serial", serial, "group", groupID, "timeslot", timeslot,
		"colour_code", colourExt&0x3F, "rate_hz", symbolHz)

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
