package conventional

import (
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dpmr"
	nxdnrx "github.com/MattCheramie/GopherTrunk/internal/radio/nxdn/receiver"
)

// dPMR call gate (issue #1374). A `dpmr` scan-list channel is a licence-free
// dPMR446 handheld or any other dPMR Mode 1 / conventional transmission:
// nothing on the air grants it, so the scanner is the only way a dPMR call
// reaches the composer's voice chain. Like the nxdn / ysf / dstar channels it
// must open on dPMR, not on any carrier above its power squelch.
//
// The gate runs the voice chain's own receiver and framer
// (composer.newDPMRVoiceReceiver, dpmr.TrafficChannel) on the channel's
// narrowband IQ, with the receiver's feed-forward timing acquisition armed:
// without it a call started at about one sub-symbol phase in four never
// framed at all. The framer already demands corroboration before it locks: an
// FS2 sync word (12 symbols, which noise matches at the hunt tolerance about
// once a minute and a half) only locks when a control channel behind it
// passes its CRC7 with the frame number its position must carry and at most
// two Hamming corrections, or a clean channel code is followed by the next
// sync word one frame pair later. The gate is open while that framer accepts
// payload frames, and closes at once on an end frame (FS3): the radio
// released the channel, so the scanner's hangtime starts there.

// dpmrCallHold is how long one accepted payload frame keeps the gate open:
// about six 80 ms frames.
const dpmrCallHold = 500 * time.Millisecond

// dpmrGateMinDwell is how long a dpmr channel whose power squelch has broken
// may wait in its scan window for the framer to lock. The framer needs an FS2
// (frames 1 and 3 of each 320 ms superframe, so one every 160 ms), the frame
// pair behind it and the next sync word (165 ms), after the receiver's carrier
// tracker has pulled in. The same physical layer as NXDN48, whose measured
// worst case on real air was 770 ms.
const dpmrGateMinDwell = 1000 * time.Millisecond

// dpmrGateDeviationHz is the deviation of dPMR's outer symbols (ETSI TS 102
// 658 clause 14.2.2.1: +1050 Hz for +3), the voice chain's slicer level.
const dpmrGateDeviationHz = 1050.0

// DPMRCallDetector is the dPMR call gate.
type DPMRCallDetector struct {
	fe *toneFrontEnd
	rx *nxdnrx.Receiver
	tc *dpmr.TrafficChannel

	samples    int64 // input samples processed since Reset
	lastFrame  int64 // input-sample index of the last accepted frame; -1 = none
	seenFrames uint64
	frames     int // accepted payload frames since Reset
	ends       int // end frames since Reset
	hold       int64
}

// NewDPMRCallDetector builds a dPMR call gate for IQ at sampleHz. Returns nil
// when sampleHz is not set.
func NewDPMRCallDetector(sampleHz float64) *DPMRCallDetector {
	if sampleHz <= 0 {
		return nil
	}
	d := &DPMRCallDetector{
		fe:        newToneFrontEnd(sampleHz),
		lastFrame: -1,
		hold:      int64(dpmrCallHold.Seconds() * sampleHz),
	}
	d.build()
	return d
}

// build makes a fresh receiver and framer: the voice chain's configuration
// (carrier tracking, AFC and channel filter at 2400 symbols a second).
func (d *DPMRCallDetector) build() {
	d.tc = dpmr.NewTrafficChannel(nil)
	d.tc.OnEnd = d.onEnd
	d.seenFrames = 0
	d.rx = nxdnrx.New(nxdnrx.Options{
		SampleRateHz:          d.fe.rate,
		SymbolRateHz:          nxdnrx.SymbolRateNXDN48,
		DeviationHz:           dpmrGateDeviationHz,
		EnableAFC:             true,
		EnableCarrierTracking: true,
		EnableChannelFilter:   true,
		DibitSink:             d.onDibits,
	})
	d.rx.ArmTimingAcquisition()
}

func (d *DPMRCallDetector) onDibits(dibits []uint8, base int) {
	d.tc.Process(dibits, base)
	if n := d.tc.Stats().Frames; n != d.seenFrames {
		d.frames += int(n - d.seenFrames)
		d.seenFrames = n
		d.lastFrame = d.samples
	}
}

// onEnd closes the gate on an end frame.
func (d *DPMRCallDetector) onEnd() {
	d.ends++
	d.lastFrame = -1
}

// Process feeds one IQ chunk and reports whether a dPMR call is on the air.
func (d *DPMRCallDetector) Process(iq []complex64) bool {
	d.rx.Process(d.fe.channel(iq))
	d.samples += int64(len(iq))
	// Keep the timing acquisition armed while nothing is locked, so a
	// transmission that keys up mid-dwell gets a seeded loop too
	// (composer.rearmDPMRTiming has the measurement).
	if !d.tc.Locked() && !d.rx.TimingAcquisitionPending() {
		d.rx.ArmTimingAcquisition()
	}
	return d.Present()
}

// Present reports whether the framer accepted a payload frame within
// dpmrCallHold and no end frame has followed it.
func (d *DPMRCallDetector) Present() bool {
	return d.lastFrame >= 0 && d.samples-d.lastFrame <= d.hold
}

// Frames is the number of payload frames accepted since the last Reset.
func (d *DPMRCallDetector) Frames() int { return d.frames }

// EndFrames is the number of end frames received since the last Reset.
func (d *DPMRCallDetector) EndFrames() int { return d.ends }

// Reset clears all state; the scanner calls it on every retune. The receiver
// is rebuilt, not reset, so the next channel does not start from this one's
// symbol phase and carrier estimate (the NXDN gate's lesson).
func (d *DPMRCallDetector) Reset() {
	d.fe.reset()
	d.build()
	d.samples = 0
	d.lastFrame = -1
	d.frames = 0
	d.ends = 0
}

func (d *DPMRCallDetector) gateName() string        { return "dPMR" }
func (d *DPMRCallDetector) minDwell() time.Duration { return dpmrGateMinDwell }
