package api

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
)

// Bounds on the silence restored between two consecutive segment recordings.
// The recorder drops the dead air between overs (each file holds only decoded
// speech), so the gap is re-inserted from the segments' own timestamps: the
// real gap when it is short, capped so a call that idled for a minute between
// overs does not play a minute of nothing, and a small default when the
// timestamps cannot be trusted (an unset EndedAt, or overlapping spans).
const (
	segmentGapDefault = 250 * time.Millisecond
	segmentGapMax     = 1500 * time.Millisecond
)

// concatRecordingSegments decodes every segment (WAV or FLAC, sniffed by
// content) and returns one WAV holding them back to back in order, with the
// inter-over silence restored. All segments must share a sample rate (they
// do: one call, one vocoder). An undecodable segment (e.g. an MP3 transcode)
// is an error — the caller degrades to serving the first file.
func concatRecordingSegments(segs []RecordingSegment) ([]byte, error) {
	if len(segs) == 0 {
		return nil, errors.New("no segments")
	}
	var pcm []int16
	var rate uint32
	for i, seg := range segs {
		samples, sr, err := voice.ReadAudioSamples(seg.Path)
		if err != nil {
			return nil, fmt.Errorf("segment %d (%s): %w", seg.Seq, seg.Path, err)
		}
		if sr == 0 {
			return nil, fmt.Errorf("segment %d (%s): zero sample rate", seg.Seq, seg.Path)
		}
		if rate == 0 {
			rate = sr
		} else if sr != rate {
			return nil, fmt.Errorf("segment %d (%s): sample rate %d != %d", seg.Seq, seg.Path, sr, rate)
		}
		if i > 0 {
			gap := segmentGap(segs[i-1], seg)
			pcm = append(pcm, make([]int16, int(gap.Seconds()*float64(rate)))...)
		}
		pcm = append(pcm, samples...)
	}
	return encodeWAV(pcm, rate), nil
}

// segmentGap is the silence to insert between prev and next.
func segmentGap(prev, next RecordingSegment) time.Duration {
	if prev.EndedAt.IsZero() || next.StartedAt.IsZero() {
		return segmentGapDefault
	}
	gap := next.StartedAt.Sub(prev.EndedAt)
	if gap < 0 {
		return segmentGapDefault
	}
	if gap > segmentGapMax {
		return segmentGapMax
	}
	return gap
}

// encodeWAV renders 16-bit mono PCM as a canonical 44-byte-header WAV.
func encodeWAV(pcm []int16, rate uint32) []byte {
	data := len(pcm) * 2
	buf := make([]byte, 44+data)
	le := binary.LittleEndian
	copy(buf[0:], "RIFF")
	le.PutUint32(buf[4:], uint32(36+data))
	copy(buf[8:], "WAVE")
	copy(buf[12:], "fmt ")
	le.PutUint32(buf[16:], 16)
	le.PutUint16(buf[20:], 1) // PCM
	le.PutUint16(buf[22:], 1) // mono
	le.PutUint32(buf[24:], rate)
	le.PutUint32(buf[28:], rate*2)
	le.PutUint16(buf[32:], 2)
	le.PutUint16(buf[34:], 16)
	copy(buf[36:], "data")
	le.PutUint32(buf[40:], uint32(data))
	for i, s := range pcm {
		le.PutUint16(buf[44+2*i:], uint16(s))
	}
	return buf
}
