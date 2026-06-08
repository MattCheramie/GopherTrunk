package m17

import "sync/atomic"

// M17 16-bit frame sync words (M17 spec, data-link layer).
const (
	SyncStream uint16 = 0xFF5D // stream (voice/data) frame
	SyncLSF    uint16 = 0x55F7 // dedicated link-setup frame
	SyncPacket uint16 = 0x75FF // packet frame
)

// Stream frame geometry (type-4 bits, after the 16-bit sync): the
// first 96 bits are the Golay-coded LICH; the remaining 272 bits carry
// the frame number + Codec2 payload (convolutionally coded — decoded by
// the voice path, skipped here).
const streamPayloadBits = 368

type m17State uint8

const (
	stHunt m17State = iota // scanning the bit stream for a frame sync
	stLICH                 // collecting the 96-bit LICH of a stream frame
	stSkip                 // skipping the rest of the stream payload
)

// Decoder turns a demodulated M17 bit stream into Link Setup Frames.
// Push one wire bit at a time (MSB-first within each dibit); a complete
// LSF is returned (ok=true) once six LICH chunks reassemble it. Not safe
// for concurrent use — one Decoder per channel.
type Decoder struct {
	st   m17State
	reg  uint16 // sliding 16-bit window for sync hunt
	lich [LICHBits]byte
	n    int // bits collected in the current field
	asm  *LICHAssembler

	syncs      atomic.Uint64
	lsfDecoded atomic.Uint64
}

// NewDecoder returns a ready Decoder.
func NewDecoder() *Decoder {
	return &Decoder{asm: NewLICHAssembler()}
}

// Push feeds one wire bit. Returns the LSF and ok=true when a stream's
// LICH chunks have reassembled a complete link setup frame.
func (d *Decoder) Push(bit byte) (LSF, bool) {
	b := bit & 1
	switch d.st {
	case stHunt:
		d.reg = (d.reg << 1) | uint16(b)
		if d.reg == SyncStream {
			d.syncs.Add(1)
			d.st = stLICH
			d.n = 0
		}
		// LSF / packet syncs are recognised but their payloads need the
		// convolutional path; the LICH route (stream frames) carries the
		// same metadata, so we wait for a stream frame.
		return LSF{}, false
	case stLICH:
		d.lich[d.n] = b
		d.n++
		if d.n >= LICHBits {
			d.st = stSkip
			d.n = LICHBits // count payload bits already consumed
			if lsf, ok := d.asm.Push(d.lich[:]); ok {
				d.lsfDecoded.Add(1)
				return lsf, true
			}
		}
		return LSF{}, false
	case stSkip:
		d.n++
		if d.n >= streamPayloadBits {
			d.st = stHunt
			d.reg = 0
		}
		return LSF{}, false
	}
	return LSF{}, false
}

// Stats reports cumulative counters.
type Stats struct {
	Syncs      uint64
	LSFDecoded uint64
}

func (d *Decoder) Stats() Stats {
	return Stats{Syncs: d.syncs.Load(), LSFDecoded: d.lsfDecoded.Load()}
}
