package dmr

import (
	"strings"
	"sync"
	"time"
	"unicode/utf16"
	"unicode/utf8"
)

// DMR talker-alias reassembly. A talker alias — a radio's human-readable
// display name — is carried in the voice superframe's embedded Link
// Control as a short numbered sequence of Full LC PDUs: a header
// (FLCOTalkerAlias / 0x04) that declares the text encoding and total
// length, followed by up to three blocks (0x05, 0x06, 0x07) that carry
// the remaining characters. TalkerAliasAssembler buffers the fragments
// per source radio and emits the completed name once enough blocks have
// arrived to cover the declared length.
//
// This mirrors the P25 Phase 1 assembler
// (internal/radio/p25/phase1/talker_alias.go): same per-source buffering,
// staleness eviction, and printable-only cleanup. The DMR wire layout
// differs (a header-declared length rather than a per-fragment block
// count), so the reassembly bookkeeping is DMR-specific.
//
// Wire layout (the 72-bit / 9-octet embedded-LC info block, bit indices
// MSB-first as recovered by ReassembleEmbeddedLCInfo), per ETSI
// TS 102 361-2 §7.2.18 and cross-checked against the OK-DMR ok-dmrlib
// reference decoder (okdmr/dmrlib Full Link Control):
//
//	Header (FLCO 0x04):
//	  bits 16..17 : Talker Alias Data Format (2 bits)
//	  bits 18..22 : Talker Alias length — total character count (5 bits, 0..31)
//	  bit  23     : reserved / format-defined MSB (not treated as text here)
//	  bits 24..71 : first 48 bits (6 octets) of alias text
//	Blocks (FLCO 0x05/0x06/0x07):
//	  bits 16..71 : 56 bits (7 octets) of alias text
//
// Like the DMR embedded-LC EMB/CRC constants (see emb.go / docs/status.md),
// the exact bit boundary of the header's text field is a working model
// pending validation against a real capture — the 48-bit header / 56-bit
// block split matches ok-dmrlib; a capture that decodes to garbage should
// re-check bit 23's inclusion first.

// TalkerAliasFormat is the 2-bit Talker Alias Data Format from the header.
type TalkerAliasFormat uint8

// Talker Alias Data Format values per ETSI TS 102 361-2 §7.2.18.
const (
	TalkerAliasFormat7Bit  TalkerAliasFormat = 0 // 7-bit packed characters
	TalkerAliasFormat8Bit  TalkerAliasFormat = 1 // 8-bit ISO 8859-1
	TalkerAliasFormatUTF8  TalkerAliasFormat = 2 // UTF-8
	TalkerAliasFormatUTF16 TalkerAliasFormat = 3 // UTF-16BE
)

// aliasHeaderTextOctets is the count of alias-text octets carried in the
// header PDU (bits 24..71); aliasBlockTextOctets is the count in each of
// the three continuation blocks (bits 16..71).
const (
	aliasHeaderTextOctets = 6
	aliasBlockTextOctets  = 7
)

// TalkerAliasFragment is one parsed piece of a talker alias: the header
// (Header true, carrying Format and Length) or a continuation block. Text
// holds the fragment's raw alias octets.
type TalkerAliasFragment struct {
	Header bool
	Index  uint8 // 0 for the header, 1..3 for blocks
	Format TalkerAliasFormat
	Length uint8 // total character count, valid only when Header
	Text   []byte
}

// TalkerAliasFLCO reports whether an FLCO is one of the four talker-alias
// opcodes (header 0x04 or block 0x05/0x06/0x07).
func TalkerAliasFLCO(o FLCO) bool {
	return o == FLCOTalkerAlias || o == FLCOTalkerAlias1 ||
		o == FLCOTalkerAlias2 || o == FLCOTalkerAlias3
}

// ParseTalkerAliasFragment decodes a talker-alias fragment from the
// leading 9 octets of an embedded-LC info block whose FLCO is a
// talker-alias opcode. It returns ok=false for a non-alias FLCO or a
// short buffer.
func ParseTalkerAliasFragment(info []byte) (TalkerAliasFragment, bool) {
	if len(info) < 9 {
		return TalkerAliasFragment{}, false
	}
	flco := FLCO(info[0] & 0x3F)
	if !TalkerAliasFLCO(flco) {
		return TalkerAliasFragment{}, false
	}
	if flco == FLCOTalkerAlias {
		return TalkerAliasFragment{
			Header: true,
			Index:  0,
			Format: TalkerAliasFormat((info[2] >> 6) & 0x03),
			Length: (info[2] >> 1) & 0x1F,
			Text:   append([]byte(nil), info[3:3+aliasHeaderTextOctets]...),
		}, true
	}
	return TalkerAliasFragment{
		Index: uint8(flco - FLCOTalkerAlias), // 0x05→1, 0x06→2, 0x07→3
		Text:  append([]byte(nil), info[2:2+aliasBlockTextOctets]...),
	}, true
}

// AssembleTalkerAliasFragments packs an alias string into the ordered
// sequence of 9-octet embedded-LC info blocks (header + as many blocks as
// the length needs) for the given format. It is the inverse of
// ParseTalkerAliasFragment / TalkerAliasAssembler and is used to build
// synthetic test vectors. Length is clamped to the 5-bit header field.
func AssembleTalkerAliasFragments(format TalkerAliasFormat, alias string) [][]byte {
	text, charLen := encodeAliasText(format, alias)
	if charLen > 0x1F {
		charLen = 0x1F
	}
	// Distribute text octets: 6 into the header, 7 into each block.
	var out [][]byte
	header := make([]byte, 9)
	header[0] = byte(FLCOTalkerAlias)
	header[2] = byte(format)<<6 | byte(charLen&0x1F)<<1
	n := copy(header[3:9], text)
	out = append(out, header)
	text = text[n:]
	for idx := 1; len(text) > 0 && idx <= 3; idx++ {
		blk := make([]byte, 9)
		blk[0] = byte(FLCOTalkerAlias) + byte(idx) // 0x05/0x06/0x07
		m := copy(blk[2:9], text)
		text = text[m:]
		out = append(out, blk)
	}
	return out
}

// encodeAliasText renders an alias string to its on-air octet stream for
// the given format and returns the octets plus the character count that
// belongs in the header Length field.
func encodeAliasText(format TalkerAliasFormat, alias string) ([]byte, uint8) {
	switch format {
	case TalkerAliasFormat7Bit:
		runes := []rune(alias)
		bits := make([]byte, 0, len(runes)*7)
		for _, r := range runes {
			c := byte(r) & 0x7F
			for b := 6; b >= 0; b-- {
				bits = append(bits, (c>>uint(b))&1)
			}
		}
		return packBits(bits), uint8(len(runes))
	case TalkerAliasFormatUTF16:
		u := utf16.Encode([]rune(alias))
		out := make([]byte, 0, len(u)*2)
		for _, w := range u {
			out = append(out, byte(w>>8), byte(w))
		}
		return out, uint8(len(u))
	default: // 8-bit ISO 8859-1 and UTF-8 are byte streams
		b := []byte(alias)
		if format == TalkerAliasFormat8Bit {
			return b, uint8(len(alias)) // ISO-8859-1: 1 octet == 1 char
		}
		return b, uint8(utf8.RuneCountInString(alias))
	}
}

// packBits packs an MSB-first 0/1 bit slice into octets, zero-padding the
// final octet.
func packBits(bits []byte) []byte {
	out := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b&1 != 0 {
			out[i>>3] |= 1 << uint(7-(i&7))
		}
	}
	return out
}

// decodeAliasText renders a reassembled alias octet stream back to a
// string for the given format, truncated to length characters.
func decodeAliasText(format TalkerAliasFormat, text []byte, length uint8) string {
	switch format {
	case TalkerAliasFormat7Bit:
		var sb []rune
		for i := 0; i < int(length); i++ {
			bit := i * 7
			var c byte
			for b := 0; b < 7; b++ {
				idx := bit + b
				if idx/8 >= len(text) {
					break
				}
				if text[idx/8]&(1<<uint(7-(idx%8))) != 0 {
					c |= 1 << uint(6-b)
				}
			}
			sb = append(sb, rune(c))
		}
		return string(sb)
	case TalkerAliasFormatUTF16:
		u := make([]uint16, 0, len(text)/2)
		for i := 0; i+1 < len(text); i += 2 {
			u = append(u, uint16(text[i])<<8|uint16(text[i+1]))
		}
		r := utf16.Decode(u)
		if int(length) < len(r) {
			r = r[:length]
		}
		return string(r)
	case TalkerAliasFormat8Bit:
		r := make([]rune, 0, len(text))
		for _, b := range text { // ISO 8859-1: octet == code point
			r = append(r, rune(b))
		}
		if int(length) < len(r) {
			r = r[:length]
		}
		return string(r)
	default: // UTF-8
		s := string(text)
		if !utf8.ValidString(s) {
			s = strings.ToValidUTF8(s, "")
		}
		r := []rune(s)
		if int(length) < len(r) {
			r = r[:length]
		}
		return string(r)
	}
}

// Talker-alias assembler bounds (mirrors the P25 assembler).
const (
	dmrAliasStaleAfter = 10 * time.Second
	dmrAliasMaxPending = 64
)

type dmrAliasBuf struct {
	format  TalkerAliasFormat
	length  uint8
	haveHdr bool
	blocks  map[uint8][]byte // index (0 header .. 3) → text octets
	updated time.Time
}

// TalkerAliasAssembler reassembles DMR talker-alias fragments per source
// radio. Because the embedded-LC fragments carry no source address, the
// caller supplies the active call's source RID as the buffering key. It
// is safe for concurrent use; construct one per voice-decode chain.
type TalkerAliasAssembler struct {
	now     func() time.Time
	mu      sync.Mutex
	pending map[uint32]*dmrAliasBuf
}

// NewTalkerAliasAssembler returns a ready assembler. now is injectable
// for tests; nil defaults to time.Now.
func NewTalkerAliasAssembler(now func() time.Time) *TalkerAliasAssembler {
	if now == nil {
		now = time.Now
	}
	return &TalkerAliasAssembler{now: now, pending: make(map[uint32]*dmrAliasBuf)}
}

// Add feeds one fragment (keyed to the active call's source RID) to the
// assembler. When the fragment set covers the header-declared length it
// returns (alias, true) and forgets the source; otherwise ("", false).
// Fragments may arrive in any order. A stray block before its header is
// buffered until the header lands.
func (a *TalkerAliasAssembler) Add(sourceID uint32, f TalkerAliasFragment) (string, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.evictStaleLocked()
	buf := a.pending[sourceID]
	if buf == nil {
		if len(a.pending) >= dmrAliasMaxPending {
			a.evictOldestLocked()
		}
		buf = &dmrAliasBuf{blocks: make(map[uint8][]byte)}
		a.pending[sourceID] = buf
	}
	if f.Header {
		buf.haveHdr = true
		buf.format = f.Format
		buf.length = f.Length
	}
	buf.blocks[f.Index] = append([]byte(nil), f.Text...)
	buf.updated = a.now()

	alias, done := assembleAlias(buf)
	if !done {
		return "", false
	}
	delete(a.pending, sourceID)
	return alias, true
}

// assembleAlias concatenates the header + contiguous blocks and, once the
// available text covers the declared character length, decodes it.
func assembleAlias(buf *dmrAliasBuf) (string, bool) {
	if !buf.haveHdr || buf.length == 0 {
		return "", false
	}
	need := neededTextOctets(buf.format, buf.length)
	var text []byte
	for idx := uint8(0); ; idx++ {
		blk, ok := buf.blocks[idx]
		if !ok {
			break
		}
		text = append(text, blk...)
		if len(text) >= need {
			break
		}
	}
	if len(text) < need {
		return "", false
	}
	return decodeAliasText(buf.format, text, buf.length), true
}

// neededTextOctets is the minimum count of alias-text octets required to
// hold length characters in the given format.
func neededTextOctets(format TalkerAliasFormat, length uint8) int {
	switch format {
	case TalkerAliasFormat7Bit:
		return (int(length)*7 + 7) / 8
	case TalkerAliasFormatUTF16:
		return int(length) * 2
	default: // 8-bit / UTF-8: at least one octet per character
		return int(length)
	}
}

func (a *TalkerAliasAssembler) evictStaleLocked() {
	cutoff := a.now().Add(-dmrAliasStaleAfter)
	for src, buf := range a.pending {
		if buf.updated.Before(cutoff) {
			delete(a.pending, src)
		}
	}
}

func (a *TalkerAliasAssembler) evictOldestLocked() {
	var oldestSrc uint32
	var oldest time.Time
	first := true
	for src, buf := range a.pending {
		if first || buf.updated.Before(oldest) {
			oldestSrc, oldest, first = src, buf.updated, false
		}
	}
	if !first {
		delete(a.pending, oldestSrc)
	}
}
