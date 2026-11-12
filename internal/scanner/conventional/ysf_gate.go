package conventional

import (
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/radio/ysf"
	ysfrx "github.com/MattCheramie/GopherTrunk/internal/radio/ysf/receiver"
)

// ysfCallHold is how long one frame sync keeps the YSF gate open. Every
// System Fusion frame opens with the FSW (one per 100 ms), so 500 ms rides
// out four lost frames without letting a finished transmission hold the
// channel much past its terminator.
const ysfCallHold = 500 * time.Millisecond

// ysfGateMinDwell is how long a YSF channel whose power squelch has broken
// may stay in its scan window waiting for a frame sync: the receiver's
// carrier correction and timing loop settle (~100 ms) and the next FSW
// arrives (100 ms) with margin. A carrier that is not System Fusion never
// produces one, so the scan moves on after this.
const ysfGateMinDwell = 500 * time.Millisecond

// ysfGateSyncTolerance is the 20-dibit FSW mismatch the gate accepts, the
// voice chain's AirFramer value. A chance match at that tolerance is about
// one in 10^9 dibit positions.
const ysfGateSyncTolerance = 2

// ysfGateDeviationHz is the C4FM outer-symbol deviation the receiver's
// slicer is calibrated to, the voice chain's value.
const ysfGateDeviationHz = 1800.0

// YSFCallDetector is the System Fusion analogue of the DMR call gate (issue
// #1261): it runs the production YSF receiver on the channel's own
// narrowband IQ and reports present only while a YSF frame sync (the FSW
// that opens every 100 ms frame) was heard within ysfCallHold. On power
// alone a `ysf` channel opened a call on ANY carrier: the reporter's first
// capture was an analog FM carrier with a 1 kHz tone on the channel, and the
// chain recorded nothing from it. With the gate such a carrier is skipped.
type YSFCallDetector struct {
	fe  *toneFrontEnd
	rx  *ysfrx.Receiver
	det *ysf.SyncDetector

	hits     []int
	pos      int   // absolute dibit index of the next dibit from the receiver
	samples  int64 // input samples processed since Reset
	lastSync int64 // input-sample index of the last FSW; -1 = none
	syncs    int   // FSWs heard since Reset
	hold     int64 // ysfCallHold in input samples
}

// NewYSFCallDetector builds a YSF call gate for IQ at sampleHz. Returns nil
// when sampleHz is not set.
func NewYSFCallDetector(sampleHz float64) *YSFCallDetector {
	if sampleHz <= 0 {
		return nil
	}
	d := &YSFCallDetector{
		fe:       newToneFrontEnd(sampleHz),
		det:      ysf.NewSyncDetector(ysfGateSyncTolerance),
		lastSync: -1,
		hold:     int64(ysfCallHold.Seconds() * sampleHz),
	}
	d.rx = ysfrx.New(ysfrx.Options{
		SampleRateHz:        d.fe.rate,
		DeviationHz:         ysfGateDeviationHz,
		EnableAFC:           true,
		EnableChannelFilter: true,
		DibitSink:           d.onDibits,
	})
	// The scanner tunes onto a carrier that is usually already up, at an
	// arbitrary symbol phase: acquire it feed-forward instead of letting
	// the loop pull in from wherever it starts (measured: 0.6–1.2 s from
	// three of ten phases, past the scan window).
	d.rx.ArmTimingAcquisition()
	return d
}

// Process feeds one IQ chunk and reports whether a System Fusion
// transmission is on the air.
func (d *YSFCallDetector) Process(iq []complex64) bool {
	d.rx.Process(d.fe.channel(iq))
	d.samples += int64(len(iq))
	return d.Present()
}

func (d *YSFCallDetector) onDibits(dibits []uint8, _ int) {
	d.hits, d.pos = d.det.Process(d.hits[:0], dibits, d.pos)
	if len(d.hits) > 0 {
		d.syncs += len(d.hits)
		d.lastSync = d.samples
	}
}

// Present reports whether a frame sync was seen within ysfCallHold.
func (d *YSFCallDetector) Present() bool {
	return d.lastSync >= 0 && d.samples-d.lastSync <= d.hold
}

// SyncHits is the number of frame syncs heard since the last Reset.
func (d *YSFCallDetector) SyncHits() int { return d.syncs }

// Reset clears all state; the scanner calls it on every retune.
func (d *YSFCallDetector) Reset() {
	d.fe.reset()
	d.rx.Reset()
	d.rx.ArmTimingAcquisition()
	d.det = ysf.NewSyncDetector(ysfGateSyncTolerance)
	d.hits = d.hits[:0]
	d.pos = 0
	d.samples = 0
	d.lastSync = -1
	d.syncs = 0
}

func (d *YSFCallDetector) gateName() string        { return "YSF" }
func (d *YSFCallDetector) minDwell() time.Duration { return ysfGateMinDwell }
