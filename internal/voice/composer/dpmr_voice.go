package composer

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dpmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	nxdnrx "github.com/MattCheramie/GopherTrunk/internal/radio/nxdn/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// dpmrVoiceIntermediateHz is the rate the wideband IQ is decimated to
// before the dPMR receiver runs. 48 kHz gives the 2400-baud 4-FSK
// symbol stream 20 samples per symbol — the same intermediate rate the
// C4FM-family voice chains use (dPMR runs half their symbol rate).
const dpmrVoiceIntermediateHz = 48_000

// dpmrChannelSelectHz caps the voice front-end channel filter when the
// IQ already arrives at the intermediate rate (a wideband DDC tap). It is
// the 6.25 kHz channel spacing, not half of it: a transmitter a kilohertz
// or two off frequency has to reach the receiver whole, which recentres it
// and then selects the channel itself (+-3.5 kHz). This was 3125 Hz, which
// cut an off-frequency signal before anything could track it.
const dpmrChannelSelectHz = 6250.0

// dpmrVoiceDeviationHz is the deviation of dPMR's outer symbols, the level
// the receiver's slicer is calibrated against: ETSI TS 102 658 clause
// 14.2.2.1 puts symbol +3 at +1050 Hz and +1 at +350 Hz. (This was 900 Hz
// until the air-format rebuild; the control-channel pipeline still uses
// that.)
const dpmrVoiceDeviationHz = 1050.0

// dpmrSymbolRate is dPMR's 2400 symbols a second.
const dpmrSymbolRate = nxdnrx.SymbolRateNXDN48

// newDPMRVoiceFrontEnd builds the channel-select + decimation front end
// for the dPMR voice chain, mirroring newNXDNVoiceFrontEnd.
func newDPMRVoiceFrontEnd(iqHz float64, bw uint32) *decimatingFIR {
	return newVoiceFrontEnd(iqHz, bw, dpmrVoiceIntermediateHz, dpmrChannelSelectHz)
}

// newDPMRVoiceReceiver builds the receiver of the dPMR voice chain: the
// NXDN receiver at NXDN48's rate. The two are one physical layer, 4-level
// FSK at 2400 symbols a second with +-350 / +-1050 Hz deviation and a root
// raised cosine of 0.2, down to the dibit mapping (TS 102 658 table 14.1).
//
// The dPMR receiver in internal/radio/dpmr/receiver has no carrier
// correction, and at +-350 Hz inner deviation a dongle a few hundred hertz
// off frequency slices nothing right: the NXDN48 lesson of #1261, where
// both of the reporter's captures sat about 400 Hz off. The NXDN receiver's
// carrier tracking, AFC and channel filter were set on those captures. Its
// tracker assumes a balanced symbol stream; dPMR does not scramble its
// voice payload, but vocoder frames are close to balanced, and the worst
// steady case, a run of silence frames, biases it by under 80 Hz.
func newDPMRVoiceReceiver(sampleHz float64, sink nxdn.DibitSink) *nxdnrx.Receiver {
	return nxdnrx.New(nxdnrx.Options{
		SampleRateHz:          sampleHz,
		SymbolRateHz:          dpmrSymbolRate,
		DeviationHz:           dpmrVoiceDeviationHz,
		EnableAFC:             true,
		EnableCarrierTracking: true,
		EnableChannelFilter:   true,
		DibitSink:             sink,
	})
}

// runDPMRVoiceChain consumes IQ for one dPMR voice call: it decimates the
// wideband IQ to a 4-FSK-friendly rate, recovers the dibit stream, and
// routes the dibits through a dpmr.TrafficChannel, which frames them in
// the air format (internal/radio/dpmr/airframe.go), follows the
// transmission from an FS2 to its end frame, and hands out the AMBE+2
// vocoder frames of its TCH blocks. Each goes to the recorder as 7 bytes;
// the recorder maps "dpmr" to the "ambe2-dmr" vocoder.
//
// The calling ID the control channels carry is published as the call's
// source, and an end frame rolls the recording in per-transmission mode.
//
// The frame format is pinned to ETSI TS 102 658 and to two decoders that
// receive dPMR on the air; this chain has not yet received a dPMR
// transmission itself (#764/#771).
func (c *Composer) runDPMRVoiceChain(ctx context.Context, serial, system string, iqCh <-chan []complex64, iqHz float64, groupID uint32, done chan<- struct{}) {
	defer close(done)
	defer gtlog.Recover(c.log, "voice-chain-dpmr:"+serial, nil)

	// Shared boundary controller: hangtime end-of-call + Touch heartbeat.
	// Talkgroup gating disabled (grantTG 0): the called ID arrives in two
	// halves over two frames, too late to gate the first of them.
	bt := c.newBoundaryTracker(serial, 0, nil)
	go bt.run(ctx)

	c.log.Info("composer: dpmr voice chain started (air format not yet confirmed on a real signal)",
		"serial", serial, "system", system, "group", groupID)

	fe := newDPMRVoiceFrontEnd(iqHz, c.bw)
	symbolHz := fe.OutRateHz()

	rs, _ := c.sink.(rawFrameSink)
	ers, _ := c.sink.(errAwareRawSink)

	var voiceFrames atomic.Uint64
	// rx is built after tc (tc is its dibit sink) but read from tc's frame
	// callback, which only runs inside rx.Process.
	var rx *nxdnrx.Receiver
	var tc *dpmr.TrafficChannel
	var voiceOffHz float64
	var iqSamples int
	var source uint32
	// The sink packs each 49-bit payload MSB-first to 7 bytes and writes it
	// to the recorder (WriteRawFrameWithErrors when the sink is
	// error-aware, so the FEC's corrected-bit count drives the vocoder's
	// smoothing, as in nxdn_voice.go).
	tc = dpmr.NewTrafficChannel(func(frames []dpmr.TCHFrame) {
		if st := tc.Stats(); st.HaveCalling && st.CallingID != source {
			source = st.CallingID
			c.publishCallSource(trunking.CallSourceUpdate{
				DeviceSerial: serial,
				SourceID:     source,
				At:           time.Now().UTC(),
			})
		}
		for _, vf := range frames {
			packed := framing.PackBitsMSB(vf.Payload)
			voiceFrames.Add(1)
			voiceOffHz = rx.CarrierOffsetHz()
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
	// An end frame closes the transmission: roll the recording there.
	tc.OnEnd = bt.onTransmissionEnd

	rx = newDPMRVoiceReceiver(symbolHz, func(dibits []uint8, baseIdx int) { tc.Process(dibits, baseIdx) })

	touchTicker := time.NewTicker(c.touchEvery)
	defer touchTicker.Stop()

	// ended logs the chain's summary. locks is how often the framer found
	// a superframe grid, drops how often it lost one; frames and cch_ok
	// count payload frames and their control channels by checksum.
	ended := func(msg string) {
		st := tc.Stats()
		c.log.Info(msg, "serial", serial, "voice_frames", voiceFrames.Load(),
			"locks", st.Locks, "drops", st.Drops, "end_frames", st.EndFrames,
			"frames", st.Frames, "cch_ok", st.CCHValid, "cch_bad", st.CCHInvalid,
			"data_frames", st.DataFrames, "inverted", st.Inverted,
			"channel_code", dpmrHex(st.ChannelCode, st.HaveChannelCode),
			"called_id", dpmrHex(st.CalledID, st.HaveCalled), "calling_id", dpmrHex(st.CallingID, st.HaveCalling),
			"carrier_offset_hz", int(rx.CarrierOffsetHz()), "carrier_offset_at_voice_hz", int(voiceOffHz),
			"iq_samples", iqSamples)
		if st.Locks == 0 && iqSamples > 0 {
			c.log.Warn("composer: dpmr call carried no dPMR superframe — no frame sync (FS2) was followed by a valid control channel, so the carrier was not dPMR voice (analog FM, another digital mode, noise) or was too far off frequency or too weak to slice",
				"serial", serial, "carrier_offset_hz", int(rx.CarrierOffsetHz()), "iq_samples", iqSamples)
		}
	}

	for {
		select {
		case <-ctx.Done():
			ended("composer: dpmr voice chain ended")
			return
		case <-touchTicker.C:
			// Touch + hangtime end-of-call handled by the boundary tracker.
		case iq, ok := <-iqCh:
			if !ok {
				ended("composer: dpmr voice chain ended (iq closed)")
				return
			}
			iqSamples += len(iq)
			bt.observe(iq)
			rx.Process(fe.Process(nil, iq))
		}
	}
}

// dpmrHex renders a 24-bit field for the log, or "" when it was not
// received.
func dpmrHex(v uint32, have bool) string {
	if !have {
		return ""
	}
	return fmt.Sprintf("%06X", v)
}
