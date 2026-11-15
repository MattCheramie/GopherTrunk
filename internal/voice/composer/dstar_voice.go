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
// The chain runs two receivers half a symbol apart (dstarVoiceLanes), so
// a call joined at any symbol phase decodes from its first frames.
//
// ⚠️ UNVERIFIED ON AIR. The DV cadence, syncs and AMBE FEC follow the
// MMDVM firmware and are pinned by its constants (#1261), but no real
// D-STAR capture has been decoded yet. Treat live D-STAR audio as
// experimental until then.
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
	// Each lane's VoiceChannel anchors the DV cadence on the Slow Data sync and
	// FEC-decodes each frame's 49-bit AMBE payload. The sink packs each
	// payload MSB-first to 7 bytes and writes it to the recorder, whose
	// base "ambe2" vocoder renders PCM (WriteRawFrameWithErrors when the
	// sink is error-aware, so the FEC corrected-bit count drives the
	// vocoder's adaptive smoothing — mirrors nxdn_voice.go).
	emit := func(f dstar.DVFrame) {
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
	}

	rx := newDStarVoiceLanes(symbolHz, emit)

	touchTicker := time.NewTicker(c.touchEvery)
	defer touchTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.log.Info("composer: dstar voice chain ended", "serial", serial, "voice_frames", voiceFrames.Load(), "lane_switches", rx.switches)
			return
		case <-touchTicker.C:
			// Touch + hangtime end-of-call handled by the boundary tracker.
		case iq, ok := <-iqCh:
			if !ok {
				c.log.Info("composer: dstar voice chain ended (iq closed)", "serial", serial, "voice_frames", voiceFrames.Load(), "lane_switches", rx.switches)
				return
			}
			bt.observe(iq)
			rx.Process(fe.Process(nil, iq))
		}
	}
}

// newDStarVoiceReceiver is the voice chain's D-STAR receiver, with carrier
// correction and a channel filter on (issue #1261). Without the correction
// the GMSK slicer decodes nothing once the carrier is ~150 Hz off, and a real
// radio is routinely a few hundred hertz off; without the filter the
// discriminator sees the front end's whole ±24 kHz stream, neighbours
// included. Measured on synthetic streams; no D-STAR capture exists yet.
func newDStarVoiceReceiver(sampleHz float64, sink dstar.BitSink) *dstarrx.Receiver {
	return dstarrx.New(dstarrx.Options{
		SampleRateHz:        sampleHz,
		EnableAFC:           true,
		EnableChannelFilter: true,
		BitSink:             sink,
	})
}

// dstarVoiceLanes is the voice chain's receiver pair (issue #1261). The
// chain starts on a carrier at an arbitrary symbol phase, and from the
// phase half a symbol off the Mueller-Müller loop's stable point (one
// start in ten at 10 samples per symbol) a lone receiver hung for one or
// two slow-data sync cycles before it pulled in: 1.0-1.4 s of a joined
// call gone, against 0.02-0.04 s from the other phases (measured,
// dstar_voice_join_test.go). The scan-list gate met the same hang and runs
// two receivers for it (conventional.DStarCallDetector); this is the same
// answer for the audio. The second lane is fed half a symbol later, so one
// of the two always starts near its best phase.
//
// Both lanes decode the same frames, so only one may reach the recorder.
// Frames are collected per block and handed over at its end: the lane that
// decodes first owns the output (of two that start in the same block, the
// one that recovered more of the speech before the sync), and the other
// takes over only after the owner has gone dstarLaneSwitchFrames frames
// without one while the other kept decoding.
type dstarVoiceLanes struct {
	lanes    [2]*dstarrx.Receiver
	pending  [2][]dstar.DVFrame
	block    int // samples fed to both lanes at a time
	owner    int // lane whose frames are emitted; -1 until the first frame
	passed   int // frames the other lane decoded since the owner's last
	switches int
	emit     func(dstar.DVFrame)
}

// dstarLaneSwitchFrames is how many frames (20 ms each) the lane that does
// not own the output must decode, with none from the owner, to take over.
const dstarLaneSwitchFrames = 5

func newDStarVoiceLanes(sampleHz float64, emit func(dstar.DVFrame)) *dstarVoiceLanes {
	// Blocks of half a frame: the two lanes are half a symbol apart, so
	// they deliver the same frame in the same block (or the next), and the
	// hand-over rule does not depend on the IQ chunk size.
	l := &dstarVoiceLanes{owner: -1, emit: emit, block: max(1, int(sampleHz/100))}
	halfSymbol := int(sampleHz / dstarrx.SymbolRate / 2)
	for i := range l.lanes {
		vc := dstar.NewVoiceChannel(func(f dstar.DVFrame) { l.pending[i] = append(l.pending[i], f) })
		l.lanes[i] = newDStarVoiceReceiver(sampleHz, func(bits []byte, baseIdx int) {
			vc.Process(bits, baseIdx)
		})
		if d := i * halfSymbol; d > 0 {
			l.lanes[i].Process(make([]complex64, d))
		}
	}
	return l
}

// settle hands one block's frames to the recorder.
func (l *dstarVoiceLanes) settle() {
	a, b := l.pending[0], l.pending[1]
	if len(a) == 0 && len(b) == 0 {
		return
	}
	if l.owner < 0 {
		l.owner = 0
		if len(b) > len(a) {
			l.owner = 1
		}
	}
	own, other := l.pending[l.owner], l.pending[1-l.owner]
	if len(own) > 0 {
		l.passed = 0
	} else {
		// Late-entry frames are a burst of old speech, not evidence that
		// the owner has fallen silent.
		for _, f := range other {
			if !f.LateEntry {
				l.passed++
			}
		}
		if l.passed >= dstarLaneSwitchFrames {
			l.owner = 1 - l.owner
			l.switches++
			l.passed = 0
			for _, f := range other {
				if !f.LateEntry {
					l.emit(f)
				}
			}
		}
	}
	for _, f := range own {
		l.emit(f)
	}
	l.pending[0], l.pending[1] = a[:0], b[:0]
}

// Process feeds channel-rate IQ to both lanes.
func (l *dstarVoiceLanes) Process(iq []complex64) {
	for len(iq) > 0 {
		n := min(l.block, len(iq))
		for _, rx := range l.lanes {
			rx.Process(iq[:n])
		}
		l.settle()
		iq = iq[n:]
	}
}
