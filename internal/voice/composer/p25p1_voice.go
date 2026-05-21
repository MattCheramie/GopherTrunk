package composer

import (
	"context"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	p25p1rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/receiver"
)

// p25p1VoiceIntermediateHz is the rate the wideband IQ is decimated to
// before the P25 Phase 1 receiver runs. 48 kHz gives the 4800-baud
// C4FM symbol stream 10 samples per symbol — ample for the receiver's
// RRC matched filter and Mueller-Müller clock recovery.
const p25p1VoiceIntermediateHz = 48_000

// p25p1DeviationHz is the C4FM peak frequency deviation at symbol ±3
// per TIA-102.BAAA. It calibrates the receiver's 4-level slicer.
const p25p1DeviationHz = 1800.0

// runP25Phase1VoiceChain consumes IQ for one P25 Phase 1 voice call. It
// decimates the wideband IQ to a C4FM-friendly rate, recovers the dibit
// stream with the Phase 1 receiver, assembles complete 1728-bit LDUs,
// and for each LDU extracts its 9 IMBE voice frames and appends them to
// the recorder's .raw sidecar.
//
// The recorder maps protocol "p25" to the pure-Go IMBE vocoder
// (voice.DefaultVocoderForProtocol), so WriteRawFrame here decodes each
// 11-byte frame to PCM and into the call's WAV.
func (c *Composer) runP25Phase1VoiceChain(ctx context.Context, serial string, iqCh <-chan []complex64, done chan<- struct{}) {
	defer close(done)

	decim := int(c.iqHz) / p25p1VoiceIntermediateHz
	if decim < 1 {
		decim = 1
	}
	symbolHz := float64(c.iqHz) / float64(decim)

	// Front-end LPF: doubles as the anti-aliasing filter for the
	// decimation, so it is only needed when the IQ is actually
	// decimated (decim == 1 only in tests that feed IQ already at the
	// intermediate rate).
	cutoff := float64(c.bw) / float64(c.iqHz)
	if cutoff > 0.45 {
		cutoff = 0.45
	}
	lpf := filter.NewFIR(filter.LowpassKaiser(81, cutoff, 8.6))

	rs, _ := c.sink.(rawFrameSink)
	rx := p25p1rx.New(p25p1rx.Options{
		SampleRateHz: symbolHz,
		DeviationHz:  p25p1DeviationHz,
		Sink: func(ldu []byte) {
			if rs == nil {
				return
			}
			frames, _, err := phase1.ExtractVoiceFrames(ldu)
			if err != nil {
				c.log.Warn("composer: p25p1 voice extract failed",
					"serial", serial, "err", err)
			}
			for _, f := range frames {
				if f == nil {
					continue
				}
				if werr := rs.WriteRawFrame(serial, f); werr != nil {
					c.log.Warn("composer: p25p1 raw-frame write failed",
						"serial", serial, "err", werr)
				}
			}
			// An LDU1 carries a Link Control word — surface the call's
			// talkgroup + source unit it identifies.
			if duid, derr := phase1.LDUDuid(ldu); derr == nil && duid == phase1.DUIDLogicalLink1 {
				if blocks, berr := phase1.ExtractLCESBlocks(ldu); berr == nil {
					if lc, _, lerr := phase1.ParseLinkControl(blocks); lerr == nil {
						c.log.Debug("composer: p25p1 link control",
							"serial", serial, "lcf", lc.LCFormat,
							"tg", lc.TalkgroupID, "src", lc.SourceID)
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
			return
		case <-touchTicker.C:
			if c.engine != nil {
				c.engine.Touch(serial)
			}
		case iq, ok := <-iqCh:
			if !ok {
				return
			}
			samples := iq
			if decim > 1 {
				samples = decimateComplex(lpf.Process(nil, iq), decim)
			}
			rx.Process(samples)
		}
	}
}
