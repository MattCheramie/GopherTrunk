package composer

import (
	"context"
	"math"
	"sync/atomic"
	"time"

	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
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
	chanBW := float64(bw)
	decim := int(math.Round(iqHz)) / nxdnVoiceIntermediateHz
	filterAtUnity := false
	if decim <= 1 {
		filterAtUnity = true
		if chanBW > nxdnChannelSelectHz {
			chanBW = nxdnChannelSelectHz
		}
	}
	return newDecimatingFIR(iqHz, nxdnVoiceIntermediateHz, chanBW, filterAtUnity)
}

// runNXDNVoiceChain consumes IQ for one NXDN voice call: it decimates
// the wideband IQ to a 4-FSK-friendly rate and recovers the dibit
// stream with the NXDN receiver (the same receiver the control channel
// uses, with the symbol-AGC that lets it decode real captures).
//
// STAGE 1 (this commit) wires the follow → tune → chain lifecycle: the
// grant is followed, a Voice device is tuned, and this chain runs the
// receiver over the traffic channel. The VCH voice-channel decoder that
// turns traffic frames into 7-byte AMBE+2 payloads is Stage 2; until it
// lands, extractVoiceFrames returns nothing and the chain produces no
// audio (the recorder still maps "nxdn" → the "ambe2" vocoder, so once
// Stage 2 feeds 7-byte frames through rs.WriteRawFrame they render to
// PCM with no further wiring). Kept structurally identical to
// runP25Phase2VoiceChain so Stage 2 is a drop-in of the extractor.
func (c *Composer) runNXDNVoiceChain(ctx context.Context, serial, system string, iqCh <-chan []complex64, iqHz float64, groupID uint32, done chan<- struct{}) {
	defer close(done)
	defer gtlog.Recover(c.log, "voice-chain-nxdn:"+serial, nil)

	// Shared boundary controller: hangtime end-of-call + Touch heartbeat.
	// Talkgroup gating disabled (grantTG 0) until the VCH decoder surfaces
	// a per-voice-frame talkgroup.
	bt := c.newBoundaryTracker(serial, 0, nil)
	go bt.run(ctx)

	c.log.Info("composer: nxdn voice chain started (stage 1: follow+tune; VCH decode is a follow-up)",
		"serial", serial, "system", system, "group", groupID)

	fe := newNXDNVoiceFrontEnd(iqHz, c.bw)
	symbolHz := fe.OutRateHz()

	rs, _ := c.sink.(rawFrameSink)

	// extractVoiceFrames is the Stage 2 hook: the NXDN VCH voice
	// channel-coding decoder (deinterleave + AMBE FEC → 7-byte AMBE+2
	// payloads). Stage 1 ships an empty extractor so the chain exercises
	// the receiver + lifecycle without yet producing audio.
	extractVoiceFrames := func(dibits []uint8, baseIdx int) [][]byte { return nil }

	var voiceFrames atomic.Uint64
	rx := nxdnrx.New(nxdnrx.Options{
		SampleRateHz: symbolHz,
		DeviationHz:  nxdnVoiceDeviationHz,
		DibitSink: func(dibits []uint8, baseIdx int) {
			for _, f := range extractVoiceFrames(dibits, baseIdx) {
				if f == nil {
					continue
				}
				voiceFrames.Add(1)
				bt.onVoice(0)
				if rs != nil {
					if err := rs.WriteRawFrame(serial, f); err != nil {
						c.log.Warn("composer: nxdn raw-frame write failed", "serial", serial, "err", err)
					}
				}
			}
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
