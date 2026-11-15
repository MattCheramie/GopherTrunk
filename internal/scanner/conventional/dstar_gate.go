package conventional

import (
	"sort"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dstar"
	dstarrx "github.com/MattCheramie/GopherTrunk/internal/radio/dstar/receiver"
)

// D-STAR call gate (issue #1261). A `dstar` scan-list channel used to
// squelch on in-channel power alone, so any carrier on the frequency
// opened a "dstar" call that recorded nothing — the hole the ysf, nxdn and
// dmr channels had. It could not be closed until the D-STAR air format was
// rebuilt: GT's old sync words matched nothing a radio sends, so a gate on
// them would have blocked every real transmission.
//
// This gate runs the production D-STAR receiver on the channel's own
// narrowband IQ and is present only within dstarCallHold of D-STAR
// evidence, in either polarity:
//   - a header whose CRC matches (the start of a transmission), or
//   - a slow-data sync one or two 21-frame cycles (2016 bits) after the
//     previous one, in the same polarity (joining a transmission already
//     under way). A lone slow-data sync at the search tolerance turns up
//     in noise about once a minute; a pair at that spacing does not.
//
// The end-of-transmission pattern closes it at once.

// dstarCallHold is how long one piece of evidence keeps the gate open: a
// slow-data sync arrives every 420 ms, so this rides out one lost sync.
const dstarCallHold = 1000 * time.Millisecond

// dstarGateMinDwell is how long a dstar channel whose power squelch has
// broken may wait in its scan window for evidence. Joining a transmission
// under way takes two slow-data syncs, 420 ms apart, after the receiver has
// settled; measured on synthetic transmissions from twenty start offsets
// at the scanner's 2.4 MS/s, in both polarities (dstar_gate_test.go), the
// gate opened within 0.9 s.
const dstarGateMinDwell = 1200 * time.Millisecond

const (
	dstarGateFrameSyncTolerance = 2 // MMDVM FRAME_SYNC_ERRS
	dstarGateSlowSyncTolerance  = 1
	dstarGateEndSyncTolerance   = 1 // MMDVM END_SYNC_ERRS
	// dstarGateSlipBits is the timing slip a sync pair may carry.
	dstarGateSlipBits = 2
	// dstarSyncCycleBits is one 21-frame slow-data sync cycle.
	dstarSyncCycleBits = dstar.DVFramesPerSyncCycle * dstar.DVFrameBits
)

// DStarCallDetector is the D-STAR call gate.
//
// It runs TWO receivers, the second fed half a symbol later. The scanner
// tunes onto a carrier at an arbitrary symbol phase, and from the phase
// half a symbol off the Mueller-Müller loop's stable point (one start in
// ten, measured at 48 kHz) the loop hangs for 1–2 sync cycles before it
// pulls in, so joining a transmission took up to 1.6 s. Half a symbol away
// the other receiver starts at its best phase. D-STAR is 4800 bit/s, so the
// second receiver costs little.
type DStarCallDetector struct {
	fe    *toneFrontEnd
	lanes [2]*dstarGateLane

	samples  int64 // input samples processed since Reset
	lastHit  int64 // input-sample index of the last evidence; -1 = none
	evidence int   // pieces of evidence since Reset
	hold     int64 // dstarCallHold in input samples
}

// dstarGateLane is one receiver and the sync state of its bit stream.
type dstarGateLane struct {
	d     *DStarCallDetector
	rx    *dstarrx.Receiver
	delay int // samples of silence fed before the first chunk

	// [0] normal polarity, [1] inverted.
	frameSync, slowSync, endSync [2]*dstar.SyncDetector
	hits                         []int

	pos      int     // absolute bit index of the next bit from the receiver
	lastSlow [2]int  // bit index of the last slow-data sync; -1 = none
	header   []uint8 // header FEC section being collected
	collect  int     // bits still to collect into header
	inv      uint8   // polarity of the header being collected
}

// NewDStarCallDetector builds a D-STAR call gate for IQ at sampleHz.
// Returns nil when sampleHz is not set.
func NewDStarCallDetector(sampleHz float64) *DStarCallDetector {
	if sampleHz <= 0 {
		return nil
	}
	d := &DStarCallDetector{
		fe:   newToneFrontEnd(sampleHz),
		hold: int64(dstarCallHold.Seconds() * sampleHz),
	}
	halfSymbol := int(d.fe.rate / dstarrx.SymbolRate / 2)
	for l := range d.lanes {
		ln := &dstarGateLane{d: d, delay: l * halfSymbol}
		for p, inv := range []bool{false, true} {
			pat := func(b []uint8) []uint8 {
				if inv {
					for i := range b {
						b[i] ^= 1
					}
				}
				return b
			}
			ln.frameSync[p] = dstar.NewSyncDetector(pat(dstar.FrameSyncBitsSlice()), dstarGateFrameSyncTolerance)
			ln.slowSync[p] = dstar.NewSyncDetector(pat(dstar.SlowDataSyncBitsSlice()), dstarGateSlowSyncTolerance)
			ln.endSync[p] = dstar.NewSyncDetector(pat(dstar.EndSyncBitsSlice()), dstarGateEndSyncTolerance)
		}
		d.lanes[l] = ln
	}
	d.Reset()
	return d
}

// reset clears the lane and gives it a fresh receiver (the next channel
// must not start from the previous one's symbol phase), primed with its
// delay.
func (ln *dstarGateLane) reset() {
	ln.rx = dstarrx.New(dstarrx.Options{
		// The voice chain's configuration (composer.newDStarVoiceReceiver).
		SampleRateHz:        ln.d.fe.rate,
		EnableAFC:           true,
		EnableChannelFilter: true,
		BitSink:             ln.onBits,
	})
	if ln.delay > 0 {
		ln.rx.Process(make([]complex64, ln.delay))
	}
	for p := 0; p < 2; p++ {
		ln.frameSync[p].Reset()
		ln.slowSync[p].Reset()
		ln.endSync[p].Reset()
	}
	ln.pos = 0
	ln.lastSlow = [2]int{-1, -1}
	ln.header = ln.header[:0]
	ln.collect = 0
}

// Process feeds one IQ chunk and reports whether a D-STAR transmission is
// on the air.
func (d *DStarCallDetector) Process(iq []complex64) bool {
	ch := d.fe.channel(iq)
	for _, ln := range d.lanes {
		ln.rx.Process(ch)
	}
	d.samples += int64(len(iq))
	return d.Present()
}

func (ln *dstarGateLane) onBits(bits []byte, _ int) {
	base := ln.pos
	ln.pos += len(bits)
	// Header collection runs bit by bit, so find every sync first.
	type ev struct {
		at   int
		kind uint8 // 0 frame sync, 1 slow-data sync, 2 end
		pol  uint8
	}
	var evs []ev
	for p := 0; p < 2; p++ {
		for k, det := range []*dstar.SyncDetector{ln.frameSync[p], ln.slowSync[p], ln.endSync[p]} {
			ln.hits, _ = det.Process(ln.hits[:0], bits, base)
			for _, at := range ln.hits {
				evs = append(evs, ev{at, uint8(k), uint8(p)})
			}
		}
	}
	sort.Slice(evs, func(i, j int) bool { return evs[i].at < evs[j].at })
	next := 0
	for i, b := range bits {
		at := base + i
		if ln.collect > 0 {
			ln.header = append(ln.header, (b^ln.inv)&1)
			ln.collect--
			if ln.collect == 0 {
				if _, ok := dstar.DecodeHeaderBurst(ln.header); ok {
					ln.d.confirm()
				}
			}
		}
		for ; next < len(evs) && evs[next].at == at; next++ {
			e := evs[next]
			switch e.kind {
			case 0:
				if ln.collect == 0 {
					ln.collect = dstar.FECOnHeaderBits
					ln.header = ln.header[:0]
					ln.inv = e.pol
				}
			case 1:
				if last := ln.lastSlow[e.pol]; last >= 0 && syncCycleGap(at-last) {
					ln.d.confirm()
				}
				ln.lastSlow[e.pol] = at
			case 2:
				// The end pattern: the transmission is over.
				ln.d.lastHit = -1
				for _, o := range ln.d.lanes {
					o.lastSlow = [2]int{-1, -1}
				}
			}
		}
	}
}

// syncCycleGap reports whether two slow-data syncs gap bits apart are one
// or two cycles apart, within dstarGateSlipBits: a sync lost while the
// receiver pulls in (measured: one start phase in ten) costs the next one
// only. A chance sync at the search tolerance turns up about once a minute
// in noise (25/2^24 per bit per polarity); one landing in these 10 bit
// positions after it, about once in 10^5 of those.
func syncCycleGap(gap int) bool {
	for k := 1; k <= 2; k++ {
		if d := gap - k*dstarSyncCycleBits; d >= -dstarGateSlipBits && d <= dstarGateSlipBits {
			return true
		}
	}
	return false
}

func (d *DStarCallDetector) confirm() {
	d.evidence++
	d.lastHit = d.samples
}

// Present reports whether D-STAR evidence was heard within dstarCallHold.
func (d *DStarCallDetector) Present() bool {
	return d.lastHit >= 0 && d.samples-d.lastHit <= d.hold
}

// Evidence is the number of confirmed headers and sync pairs since the
// last Reset.
func (d *DStarCallDetector) Evidence() int { return d.evidence }

// Reset clears all state; the scanner calls it on every retune.
func (d *DStarCallDetector) Reset() {
	d.fe.reset()
	for _, ln := range d.lanes {
		ln.reset()
	}
	d.samples = 0
	d.lastHit = -1
	d.evidence = 0
}

func (d *DStarCallDetector) gateName() string        { return "D-STAR" }
func (d *DStarCallDetector) minDwell() time.Duration { return dstarGateMinDwell }
