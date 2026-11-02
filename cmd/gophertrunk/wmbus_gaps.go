package main

import "github.com/MattCheramie/GopherTrunk/internal/radio/wmbus"

// wmbusGapLog remembers where in a live stream the SDR dropped IQ, so a
// telegram that failed its CRC can say whether a hole cut it.
//
// Issue #1256: on the 1 Nov walk the phone starved the decoder (122 298
// chunks, 626 s of IQ, dropped in a 21.6-minute run), and the reporter could
// not tell a water meter that was too weak from one the phone cut in half.
// One such snippet holds 3.3 ms of a 73-byte telegram that takes 6.7 ms on
// air, at -27.7 dBFS, then nothing but noise: header OK, blocks 3-5 failed.
//
// Where the hole falls: both drivers a phone uses here (RTL-SDR, Airspy)
// drop the NEWEST chunk when their output queue is full. So when the drop
// count has grown since the previous chunk, the hole sits right after the
// chunks still queued, i.e. after the current chunk plus len(stream) more.
// A chunk the driver queues between the drop and the read can push that one
// chunk late, so a frame is matched against the hole with one chunk of
// slack either side.
type wmbusGapLog struct {
	gaps  []int64 // input-sample positions of holes, oldest first
	chunk int64   // the latest chunk length: the position's uncertainty
	// failed counts CRC-failed telegrams that overlapped a hole.
	failed int
}

// wmbusGapKeep bounds the log: holes older than a few seconds can no
// longer touch a telegram still being decoded.
const wmbusGapKeep = 256

// wmbusGapPosition is where a hole sits in input samples when the drop
// count is seen to grow on receiving a chunk of chunkLen samples, with
// processed samples decoded before it and queued chunks still waiting
// behind it.
func wmbusGapPosition(processed int64, chunkLen, queued int) int64 {
	return processed + int64(chunkLen)*int64(1+queued)
}

func (g *wmbusGapLog) mark(at int64, chunkLen int) {
	g.chunk = int64(chunkLen)
	if len(g.gaps) == wmbusGapKeep {
		copy(g.gaps, g.gaps[1:])
		g.gaps = g.gaps[:wmbusGapKeep-1]
	}
	g.gaps = append(g.gaps, at)
}

// near reports whether a hole lies within [start-chunk, end+chunk].
func (g *wmbusGapLog) near(start, end int64) bool {
	for _, at := range g.gaps {
		if at >= start-g.chunk && at <= end+g.chunk {
			return true
		}
	}
	return false
}

// wmbusFrameSpan is the stretch of input samples a frame that ended at
// info.Sample occupied on air (the longest mode's chips per byte, plus
// preamble and sync).
func wmbusFrameSpan(f wmbus.Frame, info wmbus.FrameInfo, rateHz float64) (start, end int64) {
	frameSec := float64(len(f.Raw)*wmbusChipsPerByte+wmbusSyncChips) / wmbus.ChipRateHz
	return info.Sample - int64(frameSec*rateHz), info.Sample
}

// cutByGap reports, and counts, a CRC-failed frame that a hole overlapped.
// A nil log (decoding a file) never has holes.
func (g *wmbusGapLog) cutByGap(f wmbus.Frame, info wmbus.FrameInfo, rateHz float64) bool {
	if g == nil || f.CRCOK {
		return false
	}
	if g.near(wmbusFrameSpan(f, info, rateHz)) {
		g.failed++
		return true
	}
	return false
}
