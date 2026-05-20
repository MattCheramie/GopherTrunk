package voice

import "github.com/MattCheramie/GopherTrunk/internal/radio/dmr"

// VoiceSuperframe is one decoded DMR voice superframe: the 18 on-air
// AMBE+2 frames carried by bursts A–F. Each frame is 72 bits, one bit
// per byte MSB-first; AMBE forward-error-correction has not yet been
// applied.
type VoiceSuperframe struct {
	// Frames holds the 18 AMBE frames in transmission order — bursts
	// A..F, three frames per burst.
	Frames [FramesPerSuperframe][]byte
	// SyncName is the voice sync word that framed burst A:
	// "BS-Voice", "MS-Voice", "DM-Voice-TS1" or "DM-Voice-TS2".
	SyncName string
	// StartDibit is the absolute dibit index of burst A's first dibit.
	StartDibit int
}

// voiceSyncs are the sync words that frame burst A of a voice
// superframe. Bursts B–F carry embedded signalling instead, so they
// produce no sync match and are located by TDMA cadence.
var voiceSyncs = []dmr.SyncPattern{
	dmr.BSVoice, dmr.MSVoice, dmr.DMVoice1, dmr.DMVoice2,
}

// burstLookback is the dibit distance from a sync match (the index of
// the last sync dibit) back to the burst's first dibit: 54 payload
// dibits + 24 sync dibits − 1.
const burstLookback = VoiceHalfDibits + 24 - 1 // 77

// superframeDibits is the dibit span of a full A–F superframe.
const superframeDibits = dmr.BurstDibits * BurstsPerSuperframe // 792

// bufKeep retains enough dibits that two pending burst-A anchors can
// each still slice a complete superframe once their trailing bursts
// arrive. Dibits are one byte each, so the buffer cost is trivial.
const bufKeep = 2*superframeDibits + dmr.BurstDibits

// Decoder extracts DMR voice superframes from a dibit stream. It is
// the voice-burst counterpart of the tier2 / tier3 control-channel
// Process adapters: those slice a burst on every sync match, but
// voice bursts B–F carry no sync, so the Decoder locks onto burst A
// via its voice sync word and slices B–F at the fixed 132-dibit TDMA
// cadence.
//
// A Decoder is stateful and not safe for concurrent use; construct
// one per voice-call decode chain.
type Decoder struct {
	det      *dmr.SyncDetector
	buf      []uint8
	bufStart int // absolute dibit index of buf[0]
	pending  []dmr.Match
}

// NewDecoder returns a Decoder ready to consume dibits.
func NewDecoder() *Decoder {
	return &Decoder{det: dmr.NewSyncDetector(voiceSyncs, 2)}
}

// Reset clears all buffered state. Call on a stream re-sync so a stale
// burst-A anchor does not slice across the discontinuity.
func (d *Decoder) Reset() {
	d.buf = d.buf[:0]
	d.bufStart = 0
	d.pending = d.pending[:0]
	d.det = dmr.NewSyncDetector(voiceSyncs, 2)
}

// Process consumes a window of dibits and returns every voice
// superframe that completed within it. baseIdx is the absolute dibit
// index of dibits[0]; it must be monotonically non-decreasing across
// calls. Superframes are returned in stream order.
func (d *Decoder) Process(dibits []uint8, baseIdx int) []VoiceSuperframe {
	if len(d.buf) == 0 {
		d.bufStart = baseIdx
	}
	d.buf = append(d.buf, dibits...)

	matches, _ := d.det.Process(nil, dibits, baseIdx)
	d.pending = append(d.pending, matches...)

	var out []VoiceSuperframe
	bufEnd := d.bufStart + len(d.buf)
	keep := d.pending[:0]
	for _, m := range d.pending {
		start := m.Index - burstLookback
		if start+superframeDibits > bufEnd {
			keep = append(keep, m) // trailing bursts not buffered yet
			continue
		}
		if start < d.bufStart {
			continue // anchor fell off the front of the buffer
		}
		out = append(out, d.sliceSuperframe(start, m.Pattern.Name))
	}
	d.pending = keep

	if len(d.buf) > bufKeep {
		drop := len(d.buf) - bufKeep
		copy(d.buf, d.buf[drop:])
		d.buf = d.buf[:bufKeep]
		d.bufStart += drop
	}
	return out
}

// sliceSuperframe cuts the six 132-dibit bursts starting at absolute
// dibit index start and extracts their 18 AMBE frames. The caller has
// already confirmed the full span is buffered.
func (d *Decoder) sliceSuperframe(start int, syncName string) VoiceSuperframe {
	sf := VoiceSuperframe{SyncName: syncName, StartDibit: start}
	off := start - d.bufStart
	frame := 0
	for b := 0; b < BurstsPerSuperframe; b++ {
		var burst dmr.Burst
		copy(burst.Dibits[:], d.buf[off+b*dmr.BurstDibits:off+(b+1)*dmr.BurstDibits])
		for _, f := range AMBEFrames(&burst) {
			sf.Frames[frame] = f
			frame++
		}
	}
	return sf
}
