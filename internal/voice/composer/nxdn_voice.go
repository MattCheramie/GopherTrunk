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

// nxdnSymbolRate maps a grant protocol to the NXDN symbol rate: "nxdn48"
// is the 6.25 kHz channel, anything else NXDN96.
func nxdnSymbolRate(proto string) float64 {
	if proto == "nxdn48" {
		return nxdnrx.SymbolRateNXDN48
	}
	return nxdnrx.SymbolRate
}

// runNXDNVoiceChain consumes IQ for one NXDN voice call: it decimates
// the wideband IQ to a 4-FSK-friendly rate, recovers the dibit stream
// with the NXDN receiver, and routes the dibits through an
// nxdn.TrafficChannel that frames them in the real air format
// (internal/radio/nxdn/airframe.go), decodes the AMBE+2 voice frames and
// hands 7-byte frames to the recorder — which maps "nxdn"/"nxdn48" to the
// "ambe2-dmr" vocoder and renders them to PCM.
//
// symbolRateHz is 4800 for NXDN96 (12.5 kHz) and 2400 for NXDN48
// (6.25 kHz); the frame is the same. Checked against the issue #1261
// reporter's captures of both (testdata in internal/radio/nxdn), not yet
// confirmed on air through the daemon.
func (c *Composer) runNXDNVoiceChain(ctx context.Context, serial, system string, iqCh <-chan []complex64, iqHz, symbolRateHz float64, groupID uint32, done chan<- struct{}) {
	defer close(done)
	defer gtlog.Recover(c.log, "voice-chain-nxdn:"+serial, nil)

	// Shared boundary controller: hangtime end-of-call + Touch heartbeat.
	// Talkgroup gating disabled (grantTG 0) until the VCH decoder surfaces
	// a per-voice-frame talkgroup.
	bt := c.newBoundaryTracker(serial, 0, nil)
	go bt.run(ctx)

	c.log.Info("composer: nxdn voice chain started",
		"serial", serial, "system", system, "group", groupID, "symbol_rate", symbolRateHz)

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
		SymbolRateHz: symbolRateHz,
		DeviationHz:  nxdnVoiceDeviationHz,
		// Carrier-offset correction ON for voice. The CC receiver keeps
		// it off (see Options.EnableAFC), but the voice frame is
		// scrambled, so there are no long constant-symbol runs for the
		// tracker to walk onto. Both #1261 captures sat ~400 Hz off: at
		// NXDN48's ±350 Hz inner deviation that decodes nothing without
		// AFC, and at NXDN96 it roughly halves the AMBE Golay errors.
		EnableAFC: true,
		DibitSink: func(dibits []uint8, baseIdx int) {
			tc.Process(dibits, baseIdx)
		},
	})

	touchTicker := time.NewTicker(c.touchEvery)
	defer touchTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.log.Info("composer: nxdn voice chain ended", "serial", serial, "voice_frames", voiceFrames.Load(),
				"frames", tc.Frames, "lich_ok", tc.LICHOK)
			return
		case <-touchTicker.C:
			// Touch + hangtime end-of-call handled by the boundary tracker.
		case iq, ok := <-iqCh:
			if !ok {
				c.log.Info("composer: nxdn voice chain ended (iq closed)", "serial", serial, "voice_frames", voiceFrames.Load(),
					"frames", tc.Frames, "lich_ok", tc.LICHOK)
				return
			}
			bt.observe(iq)
			rx.Process(fe.Process(nil, iq))
		}
	}
}
