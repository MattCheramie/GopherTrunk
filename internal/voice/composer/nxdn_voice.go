package composer

import (
	"context"
	"sync/atomic"
	"time"

	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	nxdnrx "github.com/MattCheramie/GopherTrunk/internal/radio/nxdn/receiver"
)

// nxdnVoiceIntermediateHz is the rate the wideband IQ is decimated to
// before the NXDN receiver runs. 48 kHz gives the 4800-baud 4-FSK
// symbol stream 10 samples per symbol — the same intermediate rate the
// DMR / P25 Phase 1 C4FM-family voice chains use.
const nxdnVoiceIntermediateHz = 48_000

// nxdnChannelSelectHz caps the voice front-end channel filter at half
// the 12.5 kHz NXDN (NXDN96) channel spacing, so a wideband DDC voice
// tap (which band-limits only to its tap Nyquist) doesn't feed adjacent
// channel energy into the receiver. Mirrors p25p2ChannelSelectHz.
const nxdnChannelSelectHz = 6250.0

// nxdnVoiceDeviationHz is the NXDN peak deviation the receiver's slicer
// + symbol-AGC calibrate against (same value the CC pipeline uses).
const nxdnVoiceDeviationHz = 1800.0

// newNXDNVoiceFrontEnd builds the channel-select + decimation front end
// for the NXDN voice chain, mirroring newP25P2VoiceFrontEnd: on the
// dedicated-tuner path the decimating FIR band-limits as it decimates;
// on the pass-through path (a wideband DDC tap already at the
// intermediate rate) it channel-selects to ±min(bw, nxdnChannelSelectHz)
// before the receiver.
func newNXDNVoiceFrontEnd(iqHz float64, bw uint32) *decimatingFIR {
	return newVoiceFrontEnd(iqHz, bw, nxdnVoiceIntermediateHz, nxdnChannelSelectHz)
}

// runNXDNVoiceChain consumes IQ for one NXDN voice call: it decimates
// the wideband IQ to a 4-FSK-friendly rate, recovers the dibit stream
// with the NXDN receiver (the same receiver the control channel uses,
// with the symbol-AGC that lets it decode real captures), and routes the
// dibits through an nxdn.TrafficChannel that carves the VCH voice frames
// out of each traffic frame's Information field, FEC-decodes their
// AMBE+2 payloads, and hands 7-byte frames to the recorder — which maps
// "nxdn" → the "ambe2-dmr" vocoder and renders them to PCM.
//
// ⚠️ UNVERIFIED ON AIR. The VCH carve + AMBE FEC + codebook are
// transcribed from spec/reference and validated only by synthetic
// round-trip (internal/radio/nxdn/voice_ambe.go, voice.go, traffic.go);
// a real NXDN VCH capture is needed to confirm them. Treat live NXDN
// audio as experimental until then.
func (c *Composer) runNXDNVoiceChain(ctx context.Context, serial, system string, iqCh <-chan []complex64, iqHz float64, groupID uint32, done chan<- struct{}) {
	defer close(done)
	defer gtlog.Recover(c.log, "voice-chain-nxdn:"+serial, nil)

	// Shared boundary controller: hangtime end-of-call + Touch heartbeat.
	// Talkgroup gating disabled (grantTG 0) until the VCH decoder surfaces
	// a per-voice-frame talkgroup.
	bt := c.newBoundaryTracker(serial, 0, nil)
	go bt.run(ctx)

	c.log.Info("composer: nxdn voice chain started (VCH decode is unverified on air — experimental)",
		"serial", serial, "system", system, "group", groupID)

	fe := newNXDNVoiceFrontEnd(iqHz, c.bw)
	symbolHz := fe.OutRateHz()

	rs, _ := c.sink.(rawFrameSink)
	ers, _ := c.sink.(errAwareRawSink)

	var voiceFrames atomic.Uint64
	// TrafficChannel carves the VCH voice frames out of each RFChTraffic
	// frame and FEC-decodes their 49-bit AMBE+2 payloads. The sink packs
	// each payload MSB-first to 7 bytes and writes it to the recorder,
	// whose "ambe2-dmr" vocoder renders PCM (WriteRawFrameWithErrors when
	// the sink is error-aware, so the FEC corrected-bit count drives the
	// vocoder's adaptive smoothing — mirrors dmr_voice.go).
	tc := nxdn.NewTrafficChannel(func(frames []nxdn.VCHFrame) {
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
				c.log.Warn("composer: nxdn raw-frame write failed", "serial", serial, "err", werr)
			}
		}
	})

	rx := nxdnrx.New(nxdnrx.Options{
		SampleRateHz: symbolHz,
		DeviationHz:  nxdnVoiceDeviationHz,
		DibitSink: func(dibits []uint8, baseIdx int) {
			tc.Process(dibits, baseIdx)
		},
	})

	touchTicker := time.NewTicker(c.touchEvery)
	defer touchTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.log.Info("composer: nxdn voice chain ended", "serial", serial, "voice_frames", voiceFrames.Load())
			return
		case <-touchTicker.C:
			// Touch + hangtime end-of-call handled by the boundary tracker.
		case iq, ok := <-iqCh:
			if !ok {
				c.log.Info("composer: nxdn voice chain ended (iq closed)", "serial", serial, "voice_frames", voiceFrames.Load())
				return
			}
			bt.observe(iq)
			rx.Process(fe.Process(nil, iq))
		}
	}
}
