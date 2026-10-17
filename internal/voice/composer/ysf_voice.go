package composer

import (
	"context"
	"sync/atomic"
	"time"

	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/ysf"
	ysfrx "github.com/MattCheramie/GopherTrunk/internal/radio/ysf/receiver"
)

// ysfVoiceIntermediateHz / ysfChannelSelectHz / ysfVoiceDeviationHz mirror
// the NXDN chain: YSF is 4800-baud C4FM on a 12.5 kHz channel, with the
// same 1800 Hz peak deviation the YSF control-channel pipeline uses.
const (
	ysfVoiceIntermediateHz = 48_000
	ysfChannelSelectHz     = 6250.0
	ysfVoiceDeviationHz    = 1800.0
)

func newYSFVoiceFrontEnd(iqHz float64, bw uint32) *decimatingFIR {
	return newVoiceFrontEnd(iqHz, bw, ysfVoiceIntermediateHz, ysfChannelSelectHz)
}

// runYSFVoiceChain consumes IQ for one System Fusion call (issue #1261): the
// YSF receiver recovers dibits, ysf.AirFramer cuts them into frames at each
// sync word, the on-air FICH says which frames carry voice and in which
// mode, and ysf.DecodeVD2Voice extracts the five AMBE+2 frames of each
// V/D mode 2 frame. Those go to the recorder as 7-byte frames, which maps
// "ysf" to the "ambe2-dmr" vocoder (the same 49-bit AMBE+2 payload DMR
// carries; pinned by ysf.TestDecodeVD2VoiceMatchesDMRSilence).
//
// Only V/D mode 2 (the "DN" mode radios use by default) is decoded. V/D
// mode 1 and Voice Full Rate ("VW") frames are counted and logged, not
// decoded.
//
// ⚠️ UNVERIFIED ON AIR. The FICH and VD2 layouts are pinned against
// MMDVMHost's encoder and YSF2DMR's constants, not against a capture.
func (c *Composer) runYSFVoiceChain(ctx context.Context, serial, system string, iqCh <-chan []complex64, iqHz float64, groupID uint32, done chan<- struct{}) {
	defer close(done)
	defer gtlog.Recover(c.log, "voice-chain-ysf:"+serial, nil)

	// YSF voice carries a callsign, not a talkgroup: no talkgroup gating.
	bt := c.newBoundaryTracker(serial, 0, nil)
	go bt.run(ctx)

	c.log.Info("composer: ysf voice chain started (V/D mode 2 only; unverified on air — experimental)",
		"serial", serial, "system", system, "group", groupID)

	fe := newYSFVoiceFrontEnd(iqHz, c.bw)
	rs, _ := c.sink.(rawFrameSink)
	ers, _ := c.sink.(errAwareRawSink)

	var (
		voiceFrames  atomic.Uint64
		vd2Frames    atomic.Uint64
		otherFrames  atomic.Uint64 // V/D mode 1 / voice FR / data: not decoded
		fichFailures atomic.Uint64
		warnedMode   bool
		// lastVD2 lets a frame whose FICH failed its CRC still decode
		// when the call's last good FICH said V/D mode 2 voice, as
		// MMDVMHost does with its last FICH.
		lastVD2 bool
	)
	framer := ysf.NewAirFramer(2)
	write := func(fr ysf.VD2Frame) {
		packed := framing.PackBitsMSB(fr.Info)
		voiceFrames.Add(1)
		bt.onVoice(0)
		var werr error
		switch {
		case ers != nil:
			werr = ers.WriteRawFrameWithErrors(serial, packed, fr.Corrected)
		case rs != nil:
			werr = rs.WriteRawFrame(serial, packed)
		}
		if werr != nil {
			c.log.Warn("composer: ysf raw-frame write failed", "serial", serial, "err", werr)
		}
	}
	rx := newYSFVoiceReceiver(fe.OutRateHz(), func(dibits []uint8, baseIdx int) {
		for _, af := range framer.Process(dibits, baseIdx) {
			isVoice := false
			switch {
			case af.FICHOK:
				lastVD2 = af.FICH.FI == ysf.AirFIComms && af.FICH.DT == ysf.AirDTVDMode2
				isVoice = lastVD2
				if af.FICH.FI == ysf.AirFIComms && !lastVD2 {
					otherFrames.Add(1)
					if !warnedMode {
						warnedMode = true
						c.log.Warn("composer: ysf call is not V/D mode 2 (DN); only V/D mode 2 voice is decoded",
							"serial", serial, "data_type", af.FICH.DT)
					}
				}
			default:
				fichFailures.Add(1)
				isVoice = lastVD2
			}
			if !isVoice {
				continue
			}
			vd2Frames.Add(1)
			for _, fr := range ysf.DecodeVD2Voice(af.Payload) {
				write(fr)
			}
		}
	})

	touchTicker := time.NewTicker(c.touchEvery)
	defer touchTicker.Stop()
	ended := func(how string) {
		c.log.Info("composer: ysf voice chain ended"+how, "serial", serial,
			"voice_frames", voiceFrames.Load(), "vd2_frames", vd2Frames.Load(),
			"undecoded_mode_frames", otherFrames.Load(), "fich_failures", fichFailures.Load())
	}
	for {
		select {
		case <-ctx.Done():
			ended("")
			return
		case <-touchTicker.C:
		case iq, ok := <-iqCh:
			if !ok {
				ended(" (iq closed)")
				return
			}
			bt.observe(iq)
			rx.Process(fe.Process(nil, iq))
		}
	}
}

// newYSFVoiceReceiver is the voice chain's System Fusion receiver, with
// carrier correction and a ±6.25 kHz channel filter on (issue #1261). Without
// the correction the 4-level slicer fails from ~200 Hz of carrier offset, and a
// real radio is routinely a few hundred hertz off; without the filter the
// discriminator sees the front end's whole ±24 kHz stream, neighbours
// included. Measured on synthetic streams; no YSF capture exists yet.
func newYSFVoiceReceiver(sampleHz float64, sink ysf.DibitSink) *ysfrx.Receiver {
	return ysfrx.New(ysfrx.Options{
		SampleRateHz:        sampleHz,
		DeviationHz:         ysfVoiceDeviationHz,
		EnableAFC:           true,
		EnableChannelFilter: true,
		DibitSink:           sink,
	})
}
