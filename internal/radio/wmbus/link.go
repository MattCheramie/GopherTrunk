package wmbus

import (
	"errors"
	"sort"
)

// CRC16 is the EN 13757 link-layer CRC (CRC-16/EN-13757 in the reveng
// catalogue): polynomial 0x3D65, initial value 0, not reflected, final
// value complemented. It is transmitted most-significant byte first.
func CRC16(data []byte) uint16 {
	var crc uint16
	for _, b := range data {
		crc ^= uint16(b) << 8
		for range 8 {
			if crc&0x8000 != 0 {
				crc = crc<<1 ^ 0x3D65
			} else {
				crc <<= 1
			}
		}
	}
	return ^crc
}

// Mode is the radio mode a frame was received in.
type Mode uint8

const (
	// ModeT1 is meter-to-other T mode: 100 kchip/s, 3-of-6 coded.
	ModeT1 Mode = iota + 1
	// ModeC1 is meter-to-other C mode: 100 kchip/s NRZ.
	ModeC1
)

func (m Mode) String() string {
	switch m {
	case ModeT1:
		return "T1"
	case ModeC1:
		return "C1"
	}
	return "?"
}

// Format is the link-layer frame format.
type Format uint8

const (
	// FormatA: a 10-byte first block, then 16-byte blocks, each followed
	// by its own CRC. The L-field excludes the CRC bytes.
	FormatA Format = iota + 1
	// FormatB: one CRC over the first 126 bytes (blocks 1 and 2), then one
	// over the remainder. The L-field includes the CRC bytes. C mode only.
	FormatB
)

func (f Format) String() string {
	switch f {
	case FormatA:
		return "A"
	case FormatB:
		return "B"
	}
	return "?"
}

// minLField is the smallest L-field a telegram can carry: C, M (2) and
// A (6) follow L in every frame.
const minLField = 9

// frameLength is the number of bytes on the air for a frame with the given
// L-field and format, CRC bytes included, L itself included. It returns 0
// for an L-field too short to hold the data link header.
func frameLength(l byte, f Format) int {
	n := int(l)
	switch f {
	case FormatA:
		if n < minLField {
			return 0
		}
		// Block 1 holds L plus 9 header bytes; every further block holds
		// up to 16 data bytes. Each block carries a 2-byte CRC.
		blocks := 1 + (n-minLField+15)/16
		return 1 + n + 2*blocks
	case FormatB:
		if n < minLField+2 {
			return 0
		}
		return 1 + n
	}
	return 0
}

// errShort is returned for a frame shorter than its own L-field promises.
var errShort = errors.New("wmbus: frame shorter than its L-field")

// stripCRC checks every block CRC of a received frame and returns the data
// with the CRC bytes removed, L first. In the result the L-field always
// counts the data bytes after it (format A's convention), so callers parse
// both formats alike. ok is false when any block CRC fails; data is still
// returned so a marginal frame can be inspected.
func stripCRC(raw []byte, f Format) (data []byte, ok bool, err error) {
	data, _, failed, err := checkBlocks(raw, f)
	return data, err == nil && len(failed) == 0, err
}

// checkBlocks is stripCRC that also reports how many CRC blocks the frame
// has and which of them (1-based, in air order) failed. Block 1 always
// carries the link header (L, C, M, A), so a failed block 1 means the
// sender fields cannot be trusted (issue #1256: a CRC-failed "electricity
// meter" whose manufacturer code was not a valid letter triple).
func checkBlocks(raw []byte, f Format) (data []byte, blocks int, failed []int, err error) {
	if len(raw) == 0 || len(raw) < frameLength(raw[0], f) || frameLength(raw[0], f) == 0 {
		return nil, 0, nil, errShort
	}
	raw = raw[:frameLength(raw[0], f)]
	for _, sp := range blockSpans(raw, f) {
		block, crc := raw[sp[0]:sp[1]-2], raw[sp[1]-2:sp[1]]
		blocks++
		data = append(data, block...)
		if CRC16(block) != uint16(crc[0])<<8|uint16(crc[1]) {
			failed = append(failed, blocks)
		}
	}
	if f == FormatB {
		// Block 2's CRC covers blocks 1 and 2 together (up to 126 bytes);
		// an optional block 3 has its own. The L-field counts the frame
		// without its CRC bytes.
		data[0] = byte(len(data) - 1)
	}
	return data, blocks, failed, nil
}

// Soft-decision CRC repair (issue #1256). C mode data is plain NRZ, so a
// marginal chip is a bit error and fails its whole CRC block. The EN 13757
// CRC detects every error of up to five bits in a block of format A's
// length (at most 16 data bytes + 2 CRC bytes: minimum distance 6), so a
// block that one or two flipped bits make pass has exactly one such repair.
// It is not safe to search every pair, though: a block that is garbage (a
// slipped chip clock leaves the rest of a frame garbage) lands within two
// bits of some valid block about one time in six. The search is therefore
// restricted to the block's repairCandidates least reliable bits, which is
// where a weak signal's errors fall once the receiver clips FM clicks (see
// clickClipRad): repairCandidates singles plus their pairs is 136 candidate
// patterns, about the 145 a plain one-bit search of an 18-byte block tries,
// so a garbage block passes about one time in 500. Every block of a slipped
// frame has to, so in practice the risk is a frame with one bad block; the
// repaired frame says how many bits it took (Frame.CorrectedBits).
//
// On the reporter's walk the repaired telegrams were meters heard cleanly
// elsewhere on the same walk (or, once, a water meter numbered in the
// middle of a run of its heard neighbours), and 60 s of noise gave none.
const (
	repairCandidates = 16
	// repairMaxBlockBytes is the longest block (data + CRC) the distance-6
	// argument holds for; format B's 126-byte block 2 is not repaired.
	repairMaxBlockBytes = 18
)

// blockSpans returns each CRC block's [start, end) in raw, CRC bytes
// included, in air order. raw must hold a whole frame.
func blockSpans(raw []byte, f Format) [][2]int {
	var spans [][2]int
	switch f {
	case FormatA:
		spans = append(spans, [2]int{0, 12})
		for p := 12; p < len(raw); {
			n := min(16, len(raw)-p-2)
			spans = append(spans, [2]int{p, p + n + 2})
			p += n + 2
		}
	case FormatB:
		for p := 0; p < len(raw); {
			n := min(126, len(raw)-p-2)
			spans = append(spans, [2]int{p, p + n + 2})
			p += n + 2
		}
	}
	return spans
}

// repairBlocks tries to repair each failed block (1-based numbers, as
// checkBlocks reports them) of raw in place, using rel, one reliability per
// bit of raw (MSB first). It returns how many bits it flipped in total.
func repairBlocks(raw []byte, rel []float32, f Format, failed []int) int {
	raw = raw[:frameLength(raw[0], f)]
	spans := blockSpans(raw, f)
	flipped := 0
	for _, b := range failed {
		if b < 1 || b > len(spans) {
			continue
		}
		sp := spans[b-1]
		if sp[1]-sp[0] > repairMaxBlockBytes {
			continue
		}
		flipped += repairBlock(raw[sp[0]:sp[1]], rel[8*sp[0]:8*sp[1]])
	}
	return flipped
}

// repairBlock repairs one block (data + 2 CRC bytes) in place when one or
// two of its repairCandidates least reliable bits make it pass, and returns
// the number of bits flipped (0 when no candidate passes).
func repairBlock(block []byte, rel []float32) int {
	idx := make([]int, len(rel))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return rel[idx[a]] < rel[idx[b]] })
	idx = idx[:min(repairCandidates, len(idx))]
	flip := func(i int) { block[i/8] ^= 0x80 >> (i % 8) }
	n := len(block) - 2
	ok := func() bool { return CRC16(block[:n]) == uint16(block[n])<<8|uint16(block[n+1]) }
	for _, i := range idx {
		if flip(i); ok() {
			return 1
		}
		flip(i)
	}
	for a, i := range idx {
		for _, j := range idx[a+1:] {
			flip(i)
			if flip(j); ok() {
				return 2
			}
			flip(i)
			flip(j)
		}
	}
	return 0
}
