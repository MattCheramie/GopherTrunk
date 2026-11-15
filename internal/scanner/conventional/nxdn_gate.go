package conventional

import (
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	nxdnrx "github.com/MattCheramie/GopherTrunk/internal/radio/nxdn/receiver"
)

// NXDN call gate (issue #1261). An `nxdn` / `nxdn48` scan-list channel used
// to squelch on in-channel power alone, so ANY carrier on the channel opened
// an "nxdn" call that recorded nothing: an analog FM radio, a carrier left
// keyed, or an NXDN control channel, which is not a call. YSF (#1355) and DMR
// had the same hole and got the same fix. This gate runs the production
// NXDN receiver on the channel's own narrowband IQ and is present only while
// NXDN TRAFFIC frames are being heard.
//
// The evidence is deliberately two-part, because the frame sync alone is
// weak: at the framer's search tolerance (1 of 10 dibits) a random dibit
// stream matches about 3e-5 per position, roughly one false sync every 7 s
// at NXDN96 — often enough to open on an analog carrier. So a frame counts
// only when
//   - a second frame sync follows it exactly one frame (192 dibits) later,
//     which a chance match almost never does, and
//   - its descrambled LICH has good parity and names a traffic channel
//     (RTCH, RDCH or composite), not the control channel (RCCH).

// nxdnCallHold is how long one confirmed traffic frame keeps the gate open:
// about six NXDN48 frames (80 ms each) or twelve NXDN96 ones (40 ms).
const nxdnCallHold = 500 * time.Millisecond

// nxdnGateMinDwell / nxdn48GateMinDwell are how long an NXDN channel whose
// power squelch has broken may wait in its scan window for a confirmed
// frame. Measured on the reporter's real-air slices at 48 kHz and 2.4 MS/s
// from ten start offsets: NXDN96 confirmed its first frame after 60-100 ms,
// NXDN48 after 440-770 ms (80 ms frames, and the carrier tracker pulling in
// that capture's ~400 Hz offset against NXDN48's ±350 Hz inner deviation).
// A carrier that is not NXDN traffic never confirms, so the scan moves on
// after this.
const (
	nxdnGateMinDwell   = 500 * time.Millisecond
	nxdn48GateMinDwell = 1000 * time.Millisecond
)

// nxdnGateColdTolerance / nxdnGatePairTolerance are the airFramer's own
// frame-sync tolerances: a cold search, and the expected position one frame
// after a sync.
const (
	nxdnGateColdTolerance = 1
	nxdnGatePairTolerance = 3
)

// NXDNCallDetector is the NXDN call gate.
type NXDNCallDetector struct {
	fe *toneFrontEnd
	rx *nxdnrx.Receiver

	// win holds the most recent dibits: a frame sync, its LICH, and room
	// to reach the next frame's sync.
	win  []uint8
	n    int64 // dibits seen since Reset
	cand []nxdnCand

	samples   int64 // input samples processed since Reset
	lastFrame int64 // input-sample index of the last confirmed frame; -1 = none
	frames    int   // confirmed traffic frames since Reset
	hold      int64 // nxdnCallHold in input samples
	symRate   float64
}

// nxdnCand is a frame sync waiting for its LICH and for the next frame's
// sync. end is the dibit index just after the sync; traffic is set once the
// LICH (8 dibits after end) has been read.
type nxdnCand struct {
	end      int64
	lichRead bool
	traffic  bool
}

// NewNXDNCallDetector builds an NXDN call gate for IQ at sampleHz, at
// symbolRateHz (nxdnrx.SymbolRate for NXDN96, SymbolRateNXDN48 for NXDN48).
// Returns nil when sampleHz is not set.
func NewNXDNCallDetector(sampleHz, symbolRateHz float64) *NXDNCallDetector {
	if sampleHz <= 0 {
		return nil
	}
	d := &NXDNCallDetector{
		fe:        newToneFrontEnd(sampleHz),
		lastFrame: -1,
		hold:      int64(nxdnCallHold.Seconds() * sampleHz),
		symRate:   symbolRateHz,
	}
	d.rx = d.newReceiver()
	return d
}

// newReceiver is the voice chain's NXDN receiver configuration
// (composer.newNXDNVoiceReceiver): carrier tracking, AFC and the per-mode
// channel filter, which the real-air captures need.
func (d *NXDNCallDetector) newReceiver() *nxdnrx.Receiver {
	return nxdnrx.New(nxdnrx.Options{
		SampleRateHz:          d.fe.rate,
		SymbolRateHz:          d.symRate,
		DeviationHz:           nxdnVoiceDeviationHz,
		EnableAFC:             true,
		EnableCarrierTracking: true,
		EnableChannelFilter:   true,
		DibitSink:             d.onDibits,
	})
}

// nxdnVoiceDeviationHz is the NXDN peak deviation the receiver's slicer is
// calibrated to, the voice chain's value.
const nxdnVoiceDeviationHz = 1800.0

// Process feeds one IQ chunk and reports whether an NXDN call is on the air.
func (d *NXDNCallDetector) Process(iq []complex64) bool {
	d.rx.Process(d.fe.channel(iq))
	d.samples += int64(len(iq))
	return d.Present()
}

// nxdnWinCap bounds the dibit window: a sync plus one whole frame.
const nxdnWinCap = nxdn.AirFrameDibits + nxdn.AirFSWDibits

func (d *NXDNCallDetector) onDibits(dibits []uint8, _ int) {
	for _, x := range dibits {
		d.win = append(d.win, x)
		if len(d.win) > nxdnWinCap {
			d.win = d.win[1:]
		}
		d.n++
		d.step()
	}
}

// step runs after every dibit, with d.win ending at dibit d.n-1.
func (d *NXDNCallDetector) step() {
	// Read the LICH of syncs whose 8 LICH dibits have now arrived.
	for i := range d.cand {
		c := &d.cand[i]
		if !c.lichRead && d.n == c.end+nxdn.AirLICHDibits {
			lich := append([]uint8(nil), d.win[len(d.win)-nxdn.AirLICHDibits:]...)
			nxdn.DescrambleAir(lich)
			l := nxdn.ParseAirLICH(lich)
			c.lichRead, c.traffic = true, l.ParityOK && l.IsTraffic()
		}
	}
	if len(d.win) < nxdn.AirFSWDibits {
		return
	}
	errs := d.fswErrors()
	confirmed := false
	if errs <= nxdnGatePairTolerance {
		// A sync exactly one frame after a candidate confirms that
		// candidate's frame.
		for _, c := range d.cand {
			if d.n == c.end+nxdn.AirFrameDibits && c.lichRead && c.traffic {
				confirmed = true
			}
		}
	}
	if confirmed {
		d.frames++
		d.lastFrame = d.samples
	}
	// Drop candidates the next frame's sync has passed.
	keep := d.cand[:0]
	for _, c := range d.cand {
		if d.n < c.end+nxdn.AirFrameDibits {
			keep = append(keep, c)
		}
	}
	d.cand = keep
	// A cold-tolerance sync (or one that just confirmed a frame) starts a
	// candidate of its own, so a transmission keeps confirming frame by
	// frame.
	if errs <= nxdnGateColdTolerance || confirmed {
		d.cand = append(d.cand, nxdnCand{end: d.n})
	}
}

// fswErrors counts the dibits of the window's last AirFSWDibits that differ
// from the frame sync word.
func (d *NXDNCallDetector) fswErrors() int {
	tail := d.win[len(d.win)-nxdn.AirFSWDibits:]
	errs := 0
	for i, x := range tail {
		if x != nxdn.AirFSW[i] {
			errs++
		}
	}
	return errs
}

// Present reports whether a confirmed traffic frame was heard within
// nxdnCallHold.
func (d *NXDNCallDetector) Present() bool {
	return d.lastFrame >= 0 && d.samples-d.lastFrame <= d.hold
}

// Frames is the number of confirmed traffic frames since the last Reset.
func (d *NXDNCallDetector) Frames() int { return d.frames }

// Reset clears all state; the scanner calls it on every retune.
func (d *NXDNCallDetector) Reset() {
	d.fe.reset()
	// A fresh receiver, not rx.Reset: that leaves the timing loop and the
	// filter histories in place, so the next channel would start from the
	// previous one's symbol phase.
	d.rx = d.newReceiver()
	d.win = d.win[:0]
	d.cand = d.cand[:0]
	d.n = 0
	d.samples = 0
	d.lastFrame = -1
	d.frames = 0
}

func (d *NXDNCallDetector) gateName() string { return "NXDN" }
func (d *NXDNCallDetector) minDwell() time.Duration {
	if d.symRate == nxdnrx.SymbolRateNXDN48 {
		return nxdn48GateMinDwell
	}
	return nxdnGateMinDwell
}
