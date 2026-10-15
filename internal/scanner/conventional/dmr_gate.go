package conventional

import (
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	dmrrx "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/receiver"
)

// dmrCallHold is how long one piece of call evidence keeps the DMR gate
// open. A voice transmission carries a voice sync once per superframe per
// slot (burst A, every 360 ms) and opens with a train of Voice LC Headers,
// so 1 s rides out two lost superframe syncs without letting a finished
// call hold the channel much past its terminator.
const dmrCallHold = time.Second

// dmrGateMinDwell is how long a DMR channel whose power squelch has broken
// may stay in its scan window waiting for call evidence: the receiver
// settles (~100 ms) and the next voice sync or Voice LC Header arrives (one
// superframe is 360 ms). An idle beacon never produces either, so the scan
// moves on after this instead of dwelling on the beacon (issue #1261).
const dmrGateMinDwell = 500 * time.Millisecond

// dmrGateSyncTolerance is the 24-dibit sync mismatch the gate accepts, the
// Tier II slicer's value. A false sync almost never also yields a clean slot
// type, which is what locks the polarity and what headers are judged by.
const dmrGateSyncTolerance = 2

// Burst geometry around a sync match (the match index is the LAST sync dibit),
// as in the Tier II slicer.
const (
	dmrGateLookback  = dmr.HalfPayloadDibits + dmr.SlotTypeDibits + dmr.SyncDibits - 1
	dmrGateLookahead = dmr.SlotTypeDibits + dmr.HalfPayloadDibits
	dmrGateKeep      = dmrGateLookback + dmrGateLookahead + 32
)

// dmrPolarityLockVotes is how many clean (zero-correction) slot types at one
// polarity fix the stream's discriminator polarity. An idle beacon supplies
// them within a few tens of milliseconds.
const dmrPolarityLockVotes = 2

// DMRCallDetector is the DMR analogue of the P25 NAC gate (issue #1261): it
// runs the production DMR receiver on the channel's own narrowband IQ and
// reports present only while a CALL is on the air: a voice-burst sync, or a
// data burst whose slot type is a Voice LC Header or PI header, heard within
// dmrCallHold. A repeater's idle beacon (data bursts of slot type Idle, or
// CSBKs) therefore does not open or hold the scanner's squelch, where the
// in-channel power alone did.
//
// DMR's voice and data sync words are each other's discriminator-polarity
// flip, so on a spectrum-inverted front end an idle beacon's data sync reads
// as a voice sync. The gate fixes the polarity the way the Tier II slicer
// does, from data bursts whose slot type decodes cleanly at one polarity, and
// classifies every sync at that polarity.
type DMRCallDetector struct {
	fe *toneFrontEnd
	rx *dmrrx.Receiver

	det      *dmr.SyncDetector
	buf      []uint8
	bufStart int // absolute dibit index of buf[0]
	pending  []dmr.Match
	pos      int // absolute dibit index of the next dibit from the receiver

	polarity int    // -1 until locked, else 0 / dmr.PolarityFlip
	votes    [4]int // clean slot types seen per polarity, until locked
	samples  int64  // input samples processed since Reset
	lastCall int64  // input-sample index of the last call evidence; -1 = none
	// preLockVoice is the input-sample index of an unconfirmed voice-looking
	// sync seen before the polarity locked; -1 = none.
	preLockVoice int64
	hold         int64   // dmrCallHold in input samples
	sampleHz     float64 // input rate
}

// NewDMRCallDetector builds a DMR call gate for IQ at sampleHz. Returns nil
// when sampleHz is not set.
func NewDMRCallDetector(sampleHz float64) *DMRCallDetector {
	if sampleHz <= 0 {
		return nil
	}
	d := &DMRCallDetector{
		fe:           newToneFrontEnd(sampleHz),
		det:          dmr.NewSyncDetector(nil, dmrGateSyncTolerance),
		polarity:     -1,
		lastCall:     -1,
		preLockVoice: -1,
		hold:         int64(dmrCallHold.Seconds() * sampleHz),
		sampleHz:     sampleHz,
	}
	d.rx = dmrrx.New(dmrrx.Options{
		SampleRateHz:        d.fe.rate,
		DeviationHz:         1944.0,
		EnableChannelFilter: true,
		ClockGain:           0.015, // the Tier II pipeline's value
		DibitSink:           d.onDibits,
	})
	// The scanner tunes onto a repeater carrier that is usually already up,
	// at an arbitrary symbol phase: acquire it feed-forward instead of
	// letting the loop pull in from wherever it starts (seconds from near a
	// half symbol, measured: a 2-sample offset decoded no sync at all).
	d.rx.ArmTimingAcquisition()
	return d
}

// Process feeds one IQ chunk and reports whether a call is on the air.
func (d *DMRCallDetector) Process(iq []complex64) bool {
	d.rx.Process(d.fe.channel(iq))
	d.samples += int64(len(iq))
	return d.Present()
}

func (d *DMRCallDetector) onDibits(dibits []uint8, _ int) {
	if len(d.buf) == 0 {
		d.bufStart = d.pos
	}
	d.buf = append(d.buf, dibits...)
	d.pending, d.pos = d.det.Process(d.pending, dibits, d.pos)

	bufEnd := d.bufStart + len(d.buf)
	keep := d.pending[:0]
	for _, m := range d.pending {
		start := m.Index - dmrGateLookback
		if m.Index+dmrGateLookahead+1 > bufEnd {
			keep = append(keep, m)
			continue
		}
		if start < d.bufStart {
			continue
		}
		var b dmr.Burst
		copy(b.Dibits[:], d.buf[start-d.bufStart:start-d.bufStart+dmr.BurstDibits])
		d.classify(m.Pattern, b)
	}
	d.pending = keep

	if len(d.buf) > dmrGateKeep {
		drop := len(d.buf) - dmrGateKeep
		copy(d.buf, d.buf[drop:])
		d.buf = d.buf[:dmrGateKeep]
		d.bufStart += drop
	}
}

// classify judges one burst. At each candidate polarity a sync is either a
// data sync (the burst carries a slot type) or a voice sync (burst A of a
// voice superframe).
func (d *DMRCallDetector) classify(p dmr.SyncPattern, raw dmr.Burst) {
	for _, k := range dmr.CandidatePolarities {
		if d.polarity >= 0 && uint8(d.polarity) != k {
			continue
		}
		if !dmr.SyncIsDataAtPolarity(p, k) {
			continue
		}
		b := raw
		dmr.RotateBurstDibits(&b, k)
		slot, corrected, err := dmr.ParseSlotType(b.SlotTypeBitsAll())
		if err != nil {
			continue
		}
		if corrected != 0 {
			// A slot type that needed correcting is weak evidence of either
			// kind. Before the lock it still marks this sync as a possible
			// data burst at k, so an inverted beacon heard while the
			// receiver acquires (bit errors in every burst) is not counted
			// as voice; it decides nothing else.
			if d.polarity < 0 {
				return
			}
			continue
		}
		if d.polarity < 0 {
			d.votes[k]++
			if d.votes[k] >= dmrPolarityLockVotes {
				d.polarity = int(k)
			}
		}
		switch slot.DataType {
		case dmr.DTVoiceLCHeader, dmr.DTPIHeader:
			d.markCall()
		}
		return
	}
	// No data reading at an allowed polarity: a voice sync at the locked
	// polarity (or, before the lock, at identity) is burst A of a voice
	// superframe.
	k := uint8(0)
	if d.polarity >= 0 {
		k = uint8(d.polarity)
	}
	if dmr.SyncIsDataAtPolarity(p, k) || p.Hex == dmr.MSRC.Hex {
		return
	}
	if d.polarity < 0 {
		// Before the polarity is known a lone voice-looking sync could be a
		// garbled inverted data burst; two within dmrCallHold are a call
		// (a superframe repeats its voice sync every 360 ms).
		if d.preLockVoice < 0 || d.samples-d.preLockVoice > d.hold {
			d.preLockVoice = d.samples
			return
		}
	}
	d.markCall()
}

func (d *DMRCallDetector) markCall() { d.lastCall = d.samples }

// Present reports whether call evidence was seen within dmrCallHold.
func (d *DMRCallDetector) Present() bool {
	return d.lastCall >= 0 && d.samples-d.lastCall <= d.hold
}

// Reset clears all state; the scanner calls it on every retune.
func (d *DMRCallDetector) Reset() {
	d.fe.reset()
	d.rx.Reset()
	d.rx.ArmTimingAcquisition()
	d.det = dmr.NewSyncDetector(nil, dmrGateSyncTolerance)
	d.buf = d.buf[:0]
	d.pending = d.pending[:0]
	d.pos = 0
	d.polarity = -1
	d.votes = [4]int{}
	d.samples = 0
	d.lastCall = -1
	d.preLockVoice = -1
}
