package composer

import (
	"context"
	"sync/atomic"
	"time"

	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dstar"
	dstarrx "github.com/MattCheramie/GopherTrunk/internal/radio/dstar/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// dstarVoiceIntermediateHz is the rate the wideband IQ is decimated to
// before the D-STAR receiver runs. 48 kHz gives the 4800-baud GMSK bit
// stream 10 samples per symbol — the same intermediate rate the
// C4FM-family voice chains use.
const dstarVoiceIntermediateHz = 48_000

// dstarChannelSelectHz caps the voice front-end channel filter at half
// the 6.25 kHz D-STAR channel spacing. GMSK at BT 0.5 / 4800 baud puts
// its main lobe within ≈ ±2.6 kHz, so the ±3.125 kHz selection passes
// the signal while rejecting adjacent-channel energy from a wideband
// DDC tap.
const dstarChannelSelectHz = 3125.0

// newDStarVoiceFrontEnd builds the channel-select + decimation front
// end for the D-STAR voice chain, mirroring newNXDNVoiceFrontEnd.
func newDStarVoiceFrontEnd(iqHz float64, bw uint32) *decimatingFIR {
	return newVoiceFrontEnd(iqHz, bw, dstarVoiceIntermediateHz, dstarChannelSelectHz)
}

// runDStarVoiceChain consumes IQ for one D-STAR DV call: it decimates
// the wideband IQ to a GMSK-friendly rate, recovers the bit stream with
// the D-STAR receiver (the same receiver the header decoder uses —
// D-STAR is 2-level GMSK, so this is the composer's first bit-based
// chain rather than a dibit one), and routes the bits through a
// dstar.VoiceChannel that anchors the 96-bit DV cadence on the Slow
// Data sync, carves each frame's 72 voice bits, FEC-decodes the AMBE
// payloads, and hands 7-byte frames to the recorder — which maps
// "dstar" → the base "ambe2" vocoder (the AMBE 3600×2400 codebook
// D-STAR uses) and renders them to PCM.
//
// D-STAR is conventional: the grant frequency is the monitored carrier
// itself, so this chain is normally fed by the wideband DDC voice tap
// rather than a retuned voice SDR.
//
// ⚠️ UNVERIFIED ON AIR. The DV cadence + AMBE FEC + codebook are
// transcribed from the JARL spec / mbelib and validated only by
// synthetic round-trip (internal/radio/dstar/voice_ambe.go, voice.go);
// a real D-STAR capture is needed to confirm them. Treat live D-STAR
// audio as experimental until then.
func (c *Composer) runDStarVoiceChain(ctx context.Context, serial, system string, iqCh <-chan []complex64, iqHz float64, groupID uint32, done chan<- struct{}) {
	defer close(done)
	defer gtlog.Recover(c.log, "voice-chain-dstar:"+serial, nil)

	// Shared boundary controller: hangtime end-of-call + Touch heartbeat.
	// Talkgroup gating disabled (grantTG 0) — D-STAR has no talkgroups;
	// the grant's GroupID is a callsign hash used only for filing.
	bt := c.newBoundaryTracker(serial, 0, nil)
	go bt.run(ctx)

	c.log.Info("composer: dstar voice chain started (DV decode is unverified on air — experimental)",
		"serial", serial, "system", system, "group", groupID)

	fe := newDStarVoiceFrontEnd(iqHz, c.bw)
	symbolHz := fe.OutRateHz()

	rs, _ := c.sink.(rawFrameSink)
	ers, _ := c.sink.(errAwareRawSink)

	var voiceFrames atomic.Uint64
	// VoiceChannel anchors the DV cadence on the Slow Data sync and
	// FEC-decodes each frame's 49-bit AMBE payload. The sink packs each
	// payload MSB-first to 7 bytes and writes it to the recorder, whose
	// base "ambe2" vocoder renders PCM (WriteRawFrameWithErrors when the
	// sink is error-aware, so the FEC corrected-bit count drives the
	// vocoder's adaptive smoothing — mirrors nxdn_voice.go).
	vc := dstar.NewVoiceChannel(func(f dstar.DVFrame) {
		packed := framing.PackBitsMSB(f.Payload)
		voiceFrames.Add(1)
		bt.onVoice(0)
		var werr error
		switch {
		case ers != nil:
			werr = ers.WriteRawFrameWithErrors(serial, packed, f.Errors)
		case rs != nil:
			werr = rs.WriteRawFrame(serial, packed)
		}
		if werr != nil {
			c.log.Warn("composer: dstar raw-frame write failed", "serial", serial, "err", werr)
		}
	})

	rx := dstarrx.New(dstarrx.Options{
		SampleRateHz: symbolHz,
		BitSink: func(bits []byte, baseIdx int) {
			vc.Process(bits, baseIdx)
		},
	})

	touchTicker := time.NewTicker(c.touchEvery)
	defer touchTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.log.Info("composer: dstar voice chain ended", "serial", serial, "voice_frames", voiceFrames.Load())
			return
		case <-touchTicker.C:
			// Touch + hangtime end-of-call handled by the boundary tracker.
		case iq, ok := <-iqCh:
			if !ok {
				c.log.Info("composer: dstar voice chain ended (iq closed)", "serial", serial, "voice_frames", voiceFrames.Load())
				return
			}
			bt.observe(iq)
			rx.Process(fe.Process(nil, iq))
		}
	}
}
