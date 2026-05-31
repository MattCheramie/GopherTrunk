package m17

import "github.com/MattCheramie/GopherTrunk/internal/radio/framing"

// LICHBits is the Link Information CHannel size in a stream frame: four
// Golay(24,12) codewords = 96 coded bits carrying a 48-bit chunk.
const LICHBits = 96

// lichChunks is the number of LICH chunks that reassemble one LSF: each
// chunk carries 40 of the 240 LSF bits.
const lichChunks = 6

// LICHAssembler reassembles the Link Setup Frame from the LICH chunks
// embedded in successive stream frames. Each chunk carries 40 LSF bits
// plus a modulo-6 counter naming its position; once all six positions
// are seen the full 240-bit LSF is parsed and its CRC checked.
type LICHAssembler struct {
	lsf  [LSFBytes]byte
	have [lichChunks]bool

	golayErrors uint64
}

// NewLICHAssembler returns an empty assembler.
func NewLICHAssembler() *LICHAssembler { return &LICHAssembler{} }

// Push feeds one stream frame's 96-bit LICH (one byte per bit, 0/1,
// MSB-first as received). It Golay-decodes the four codewords, extracts
// the 40-bit LSF fragment and its counter, and — once all six fragments
// are present — returns the parsed LSF with ok=true. A LICH with an
// uncorrectable Golay codeword is dropped.
func (a *LICHAssembler) Push(lich []byte) (LSF, bool) {
	if len(lich) < LICHBits {
		return LSF{}, false
	}
	var chunk uint64 // 48-bit reassembled chunk
	for i := 0; i < 4; i++ {
		var cw uint32
		for j := 0; j < 24; j++ {
			cw = cw<<1 | uint32(lich[i*24+j]&1)
		}
		data, errs := framing.GolayDecode24_12(cw)
		if errs < 0 {
			a.golayErrors++
			return LSF{}, false // uncorrectable codeword
		}
		chunk = chunk<<12 | uint64(data&0x0FFF)
	}

	// chunk is 48 bits: top 40 = LSF fragment, next 3 = counter.
	frag := chunk >> 8 // top 40 bits
	cnt := int((chunk >> 5) & 0x07)
	if cnt >= lichChunks {
		return LSF{}, false
	}
	base := cnt * 5
	for i := 0; i < 5; i++ {
		a.lsf[base+i] = byte((frag >> uint(32-i*8)) & 0xFF)
	}
	a.have[cnt] = true

	for _, h := range a.have {
		if !h {
			return LSF{}, false
		}
	}
	lsf := ParseLSF(a.lsf[:])
	a.reset()
	return lsf, true
}

// reset clears the collected fragments after delivering an LSF so the
// next transmission starts fresh.
func (a *LICHAssembler) reset() {
	a.have = [lichChunks]bool{}
}
