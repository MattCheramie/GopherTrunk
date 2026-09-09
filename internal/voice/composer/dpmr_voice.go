package composer

import (
	"context"
	"sync/atomic"
	"time"

	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dpmr"
	dpmrrx "github.com/MattCheramie/GopherTrunk/internal/radio/dpmr/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// dpmrVoiceIntermediateHz is the rate the wideband IQ is decimated to
// before the dPMR receiver runs. 48 kHz gives the 2400-baud 4-FSK
// symbol stream 20 samples per symbol — the same intermediate rate the
// C4FM-family voice chains use (dPMR runs half their symbol rate).
const dpmrVoiceIntermediateHz = 48_000

// dpmrChannelSelectHz caps the voice front-end channel filter at half
// the 6.25 kHz dPMR channel spacing, so a wideband DDC voice tap
// (which band-limits only to its tap Nyquist) doesn't feed adjacent
// channel energy into the receiver. Mirrors nxdnChannelSelectHz.
const dpmrChannelSelectHz = 3125.0

// dpmrVoiceDeviationHz is the dPMR peak deviation the receiver's
// slicer calibrates against (900 Hz at symbol ±3 — half of P25 / DMR,
// matching the 6.25 kHz spacing; same value the CC pipeline uses).
const dpmrVoiceDeviationHz = 900.0

// newDPMRVoiceFrontEnd builds the channel-select + decimation front end
// for the dPMR voice chain, mirroring newNXDNVoiceFrontEnd.
func newDPMRVoiceFrontEnd(iqHz float64, bw uint32) *decimatingFIR {
	return newVoiceFrontEnd(iqHz, bw, dpmrVoiceIntermediateHz, dpmrChannelSelectHz)
}

// runDPMRVoiceChain consumes IQ for one dPMR Mode 3 voice call: it
// decimates the wideband IQ to a 4-FSK-friendly rate, recovers the
// dibit stream with the dPMR receiver (the same receiver the control
// channel uses), and routes the dibits through a dpmr.TrafficChannel
// that anchors on the FS1/FS2 voice syncs, carves the TCH voice frames
// out of each 80 ms frame, FEC-decodes their AMBE+2 payloads, and hands
// 7-byte frames to the recorder — which maps "dpmr" → the "ambe2-dmr"
// vocoder and renders them to PCM.
//
// ⚠️ UNVERIFIED ON AIR. The frame carve + AMBE FEC + codebook are
// transcribed from spec and validated only by synthetic round-trip
// (internal/radio/dpmr/voice_ambe.go, voice.go, traffic.go); a real
// dPMR voice capture is needed to confirm them. Treat live dPMR audio
// as experimental until then.
func (c *Composer) runDPMRVoiceChain(ctx context.Context, serial, system string, iqCh <-chan []complex64, iqHz float64, groupID uint32, done chan<- struct{}) {
	defer close(done)
	defer gtlog.Recover(c.log, "voice-chain-dpmr:"+serial, nil)

	// Shared boundary controller: hangtime end-of-call + Touch heartbeat.
	// Talkgroup gating disabled (grantTG 0) until the TCH decoder surfaces
	// a per-frame talkgroup from the CCH.
	bt := c.newBoundaryTracker(serial, 0, nil)
	go bt.run(ctx)

	c.log.Info("composer: dpmr voice chain started (TCH decode is unverified on air — experimental)",
		"serial", serial, "system", system, "group", groupID)

	fe := newDPMRVoiceFrontEnd(iqHz, c.bw)
	symbolHz := fe.OutRateHz()

	rs, _ := c.sink.(rawFrameSink)
	ers, _ := c.sink.(errAwareRawSink)

	var voiceFrames atomic.Uint64
	// TrafficChannel carves the TCH voice frames out of each FS1/FS2
	// voice frame and FEC-decodes their 49-bit AMBE+2 payloads. The sink
	// packs each payload MSB-first to 7 bytes and writes it to the
	// recorder, whose "ambe2-dmr" vocoder renders PCM
	// (WriteRawFrameWithErrors when the sink is error-aware, so the FEC
	// corrected-bit count drives the vocoder's adaptive smoothing —
	// mirrors nxdn_voice.go).
	tc := dpmr.NewTrafficChannel(func(frames []dpmr.TCHFrame) {
		for _, vf := range frames {
			packed := framing.PackBitsMSB(vf.Payload)
			voiceFrames.Add(1)
			bt.onVoice(0)
			var werr error
			switch {
			case ers != nil:
				werr = ers.WriteRawFrameWithErrors(serial, packed, vf.Errors)
			case rs != nil:
				werr = rs.WriteRawFrame(serial, packed)
			}
			if werr != nil {
				c.log.Warn("composer: dpmr raw-frame write failed", "serial", serial, "err", werr)
			}
		}
	})

	rx := dpmrrx.New(dpmrrx.Options{
		SampleRateHz: symbolHz,
		DeviationHz:  dpmrVoiceDeviationHz,
		DibitSink: func(dibits []uint8, baseIdx int) {
			tc.Process(dibits, baseIdx)
		},
	})

	touchTicker := time.NewTicker(c.touchEvery)
	defer touchTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.log.Info("composer: dpmr voice chain ended", "serial", serial, "voice_frames", voiceFrames.Load())
			return
		case <-touchTicker.C:
			// Touch + hangtime end-of-call handled by the boundary tracker.
		case iq, ok := <-iqCh:
			if !ok {
				c.log.Info("composer: dpmr voice chain ended (iq closed)", "serial", serial, "voice_frames", voiceFrames.Load())
				return
			}
			bt.observe(iq)
			rx.Process(fe.Process(nil, iq))
		}
	}
}
