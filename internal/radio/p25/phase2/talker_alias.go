package phase2

import (
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/motorola"
)

// P25 Phase 2 talker-alias reassembly.
//
// A talker alias — the human-readable display name of a radio — is too
// long for one MAC PDU, so a system sends it as a numbered sequence of
// fragment PDUs. TalkerAliasAssembler buffers the fragments per source
// unit and emits the completed name once every block has arrived.
//
// Layout note: the talker-alias MAC opcode, the fragment header, and
// the character encoding are not in the repo's spec PDFs — Motorola and
// Harris each have their own talker-alias format. This file is the
// project's working model: a single OpVendorTalkerAlias opcode whose
// payload is SourceID(3) + BlockIndex(1) + BlockCount(1) + Data, ASCII
// in the data. All wire detail is confined here; the assembler logic
// (per-source buffering, completion, staleness eviction) is
// encoding-independent and tolerant of reordered or missing fragments.
//
// CORRECTION PENDING (#376): the working model above does not match the
// real Motorola Phase 2 alias observed on-air (SDRTrunk, Victorian MMR).
// The real form rides on FACCH-S during hangtime with a HEADER opcode
// 0x91 + DATA opcodes 0x95 (see mac_vendor.go), the fragments are run
// through the Motorola alias cipher (port phase1/motorola_alias_cipher.go)
// not read as plain ASCII, there is a CRC-16 tail to validate, and the
// source RID is carried inline in the header rather than supplied
// separately. The encoding-independent assembler below should survive
// the correction; AsTalkerAliasFragment + cleanAlias are what change.
// Implement against the fixtures before trusting the output.

// TalkerAliasFragment is one numbered piece of a radio's talker alias.
type TalkerAliasFragment struct {
	SourceID   uint32
	BlockIndex uint8
	BlockCount uint8
	Data       []byte
}

// AsTalkerAliasFragment returns the fragment if the PDU opcode is
// OpVendorTalkerAlias, otherwise (zero, false). It is MFID-agnostic —
// both Motorola and Harris alias PDUs decode through it.
func (p MACPDU) AsTalkerAliasFragment() (TalkerAliasFragment, bool) {
	if p.Opcode != OpVendorTalkerAlias {
		return TalkerAliasFragment{}, false
	}
	if len(p.Payload) < 5 {
		return TalkerAliasFragment{}, false
	}
	f := TalkerAliasFragment{
		SourceID:   uint32(p.Payload[0])<<16 | uint32(p.Payload[1])<<8 | uint32(p.Payload[2]),
		BlockIndex: p.Payload[3],
		BlockCount: p.Payload[4],
	}
	f.Data = append([]byte(nil), p.Payload[5:]...)
	return f, true
}

// Talker-alias assembler bounds.
const (
	// aliasStaleAfter is how long an incomplete alias is kept before a
	// later Add evicts it — a lost final fragment must not leak memory.
	aliasStaleAfter = 10 * time.Second
	// aliasMaxPending caps the number of distinct source units buffered
	// at once; the oldest is dropped when the cap is reached.
	aliasMaxPending = 64
	// aliasMaxBlocks caps the block count an alias may claim.
	aliasMaxBlocks = 16
)

type aliasBuf struct {
	count   uint8
	blocks  map[uint8][]byte
	updated time.Time
}

// TalkerAliasAssembler reassembles talker-alias fragments per source
// unit. It is safe for concurrent use; construct one per ControlChannel.
type TalkerAliasAssembler struct {
	now     func() time.Time
	mu      sync.Mutex
	pending map[uint32]*aliasBuf
}

// NewTalkerAliasAssembler returns a ready assembler. now is injectable
// for tests; nil defaults to time.Now.
func NewTalkerAliasAssembler(now func() time.Time) *TalkerAliasAssembler {
	if now == nil {
		now = time.Now
	}
	return &TalkerAliasAssembler{now: now, pending: make(map[uint32]*aliasBuf)}
}

// Add feeds one fragment to the assembler. When the fragment completes
// an alias it returns (alias, sourceID, true) and forgets the source;
// otherwise (\"\", 0, false). Out-of-range or malformed fragments are
// ignored. Add is tolerant of fragments arriving in any order.
func (a *TalkerAliasAssembler) Add(f TalkerAliasFragment) (string, uint32, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.evictStaleLocked()
	if f.BlockCount == 0 || f.BlockCount > aliasMaxBlocks || f.BlockIndex >= f.BlockCount {
		return "", 0, false
	}
	buf := a.pending[f.SourceID]
	if buf == nil {
		if len(a.pending) >= aliasMaxPending {
			a.evictOldestLocked()
		}
		buf = &aliasBuf{blocks: make(map[uint8][]byte)}
		a.pending[f.SourceID] = buf
	}
	buf.count = f.BlockCount
	buf.blocks[f.BlockIndex] = append([]byte(nil), f.Data...)
	buf.updated = a.now()

	if len(buf.blocks) < int(buf.count) {
		return "", 0, false
	}
	var raw []byte
	for i := uint8(0); i < buf.count; i++ {
		b, ok := buf.blocks[i]
		if !ok {
			return "", 0, false // a duplicate filled the count but a gap remains
		}
		raw = append(raw, b...)
	}
	delete(a.pending, f.SourceID)
	return cleanAlias(raw), f.SourceID, true
}

// evictStaleLocked drops incomplete aliases older than aliasStaleAfter.
func (a *TalkerAliasAssembler) evictStaleLocked() {
	cutoff := a.now().Add(-aliasStaleAfter)
	for src, buf := range a.pending {
		if buf.updated.Before(cutoff) {
			delete(a.pending, src)
		}
	}
}

// evictOldestLocked drops the single least-recently-updated alias.
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

// cleanAlias trims trailing NULs and renders the alias bytes as a
// printable ASCII string, dropping control characters.
func cleanAlias(raw []byte) string {
	out := make([]byte, 0, len(raw))
	for _, b := range raw {
		if b >= 0x20 && b < 0x7F {
			out = append(out, b)
		}
	}
	return string(out)
}

// --- Real Motorola Phase 2 talker alias (FACCH-S, #376) ------------
//
// The on-air form (SDRTrunk ground truth, Victorian MMR) is a HEADER
// MAC PDU (opcode 0x91, MFID 0x90) followed by N DATA MAC PDUs (opcode
// 0x95). Both reassemble — header fragment first, then data blocks in
// order — into one Motorola message with the same framing as Phase 1:
// WACN|System|RadioID|cipher-alias|CRC-16. The source RID therefore
// falls out of the reassembled prefix (the motorola package decodes it).
//
// The per-PDU offsets and the nibble alignment below are anchored to the
// SDRTrunk-decoded FRAGMENT fields posted in #376 (the same bytes SDRTrunk
// concatenates into the message it decodes), so reassembly is no longer a
// guess — assembleMotorolaAliasMessage reproduces SDRTrunk's stream exactly
// (see TestMotorolaAliasReassemblesSDRTrunkFragmentStream). What remains
// air-unverified (#376/#773) is the cipher LUT and hence the final decoded
// alias *string*: no plaintext fixture for a known RID is committed. A
// wrong cipher still fails safe — DecodeMessage finds no coherent
// SUID/length, or DecodeAlias flags the result unreliable, and no alias is
// published.
//
// Header payload layout (after ParseMACPDU strips opcode+MFID), from
// SDRTrunk MSG 9190114EF002010006BEE00164030D7E24… / FRAGMENT BEE00164030D7E24:
//
//	[0]    length
//	[1:3]  talkgroup
//	[3]    blocks to follow
//	[4]    format (1 = Unicode)
//	[5]    sequence (low nibble)
//	[6]    reserved
//	[7:]   first cipher fragment (SUID-bearing, byte-aligned), trailing FACCH CRC ignored
//
// Data payload layout, from SDRTrunk MSG 95901101044F6FF2…63C /
// FRAGMENT 44F6FF2FA9AC3EC34432FA63C:
//
//	[0]    length
//	[1]    block number (1-based)
//	[2]    sequence (high nibble) + first cipher nibble (low nibble)
//	[3:]   cipher fragment continues, trailing FACCH CRC ignored
//
// The data fragment is therefore NIBBLE-aligned: its low nibble of [2]
// leads, and fragments concatenate as a nibble stream, not whole bytes.
const (
	aliasHeaderFragOffset = 7
	aliasDataFragOffset   = 3
)

// MotorolaAliasHeader is a decoded FACCH-S talker-alias header PDU.
type MotorolaAliasHeader struct {
	TalkgroupID uint16
	Sequence    uint8
	BlockCount  uint8
	Fragment    []byte // first cipher fragment (carries the SUID prefix)
}

// AsMotorolaAliasHeader returns the header if the PDU is a Motorola
// FACCH-S alias header (opcode 0x91, MFID 0x90), otherwise (zero, false).
func (p MACPDU) AsMotorolaAliasHeader() (MotorolaAliasHeader, bool) {
	if p.Opcode != OpMotorolaAliasHeader || p.MFID != MFIDMotorola {
		return MotorolaAliasHeader{}, false
	}
	if len(p.Payload) <= aliasHeaderFragOffset {
		return MotorolaAliasHeader{}, false
	}
	return MotorolaAliasHeader{
		TalkgroupID: uint16(p.Payload[1])<<8 | uint16(p.Payload[2]),
		BlockCount:  p.Payload[3],
		Sequence:    p.Payload[5] & 0x0F,
		Fragment:    append([]byte(nil), p.Payload[aliasHeaderFragOffset:]...),
	}, true
}

// MotorolaAliasData is a decoded FACCH-S talker-alias data PDU. The
// cipher fragment is nibble-aligned: LeadNibble is the low nibble of
// payload[2] (the first cipher nibble) and Fragment is the byte-aligned
// remainder. Reassembly must concatenate LeadNibble + Fragment as a
// nibble stream, not whole bytes (see assembleMotorolaAliasMessage).
type MotorolaAliasData struct {
	BlockNumber uint8
	Sequence    uint8
	LeadNibble  uint8 // low nibble of payload[2]; first cipher nibble of the block
	Fragment    []byte
}

// AsMotorolaAliasData returns the data block if the PDU is a Motorola
// FACCH-S alias data block (opcode 0x95, MFID 0x90), otherwise
// (zero, false).
func (p MACPDU) AsMotorolaAliasData() (MotorolaAliasData, bool) {
	if p.Opcode != OpMotorolaAliasData || p.MFID != MFIDMotorola {
		return MotorolaAliasData{}, false
	}
	if len(p.Payload) <= aliasDataFragOffset {
		return MotorolaAliasData{}, false
	}
	return MotorolaAliasData{
		BlockNumber: p.Payload[1],
		Sequence:    p.Payload[2] >> 4,
		LeadNibble:  p.Payload[2] & 0x0F,
		Fragment:    append([]byte(nil), p.Payload[aliasDataFragOffset:]...),
	}, true
}

// MotorolaAliasResult is the outcome of feeding a fragment to a
// MotorolaAliasAssembler. Complete is true once the header and every
// data block of the same sequence have arrived and the message
// reassembled; the other fields are meaningful only then.
type MotorolaAliasResult struct {
	// Alias is the decoded talker-alias string (best-effort printable
	// ASCII). While the proprietary cipher is unverified it is typically
	// empty or garbage — see Reliable.
	Alias string
	// SourceID is the radio (subscriber) ID the alias belongs to,
	// recovered from the byte-aligned message prefix.
	SourceID uint32
	// TalkgroupID is the talkgroup carried in the alias header PDU.
	TalkgroupID uint16
	// Encoded is the reassembled cipher region (the 2n encoded-alias
	// bytes, CRC stripped) — chosen-plaintext / known-RID ground truth
	// for the cipher cryptanalysis (#773), surfaced regardless of whether
	// the cipher decodes.
	Encoded []byte
	// CRCOK reports whether the reassembled message's trailing CRC-16
	// matched. Advisory (#376).
	CRCOK bool
	// Reliable is true only when the decode is clean printable ASCII AND
	// the cipher is verified; false while the cipher is gated (#773).
	Reliable bool
	// Complete is true once a full alias message reassembled.
	Complete bool
}

// MotorolaAliasAssembler reassembles the real Motorola FACCH-S talker
// alias for one active call. Construct one per voice chain; it is
// single-goroutine like phase1.MotorolaTalkerAliasBuf. AddHeader /
// AddData return a MotorolaAliasResult whose Complete is true once the
// header and every data block of the same sequence have arrived. Reliable
// is false when the decoded alias holds non-ASCII-printable characters
// (bit-error corruption surviving the CRC, #711) or the cipher is still
// gated (#773).
type MotorolaAliasAssembler struct {
	now func() time.Time

	haveHeader bool
	sequence   uint8
	blockCount uint8
	talkgroup  uint16
	header     []byte
	blocks     map[uint8][]byte
	leads      map[uint8]uint8 // per-block leading cipher nibble
	updated    time.Time
}

// NewMotorolaAliasAssembler returns a ready assembler. now is injectable
// for tests; nil defaults to time.Now.
func NewMotorolaAliasAssembler(now func() time.Time) *MotorolaAliasAssembler {
	if now == nil {
		now = time.Now
	}
	return &MotorolaAliasAssembler{
		now:    now,
		blocks: make(map[uint8][]byte),
		leads:  make(map[uint8]uint8),
	}
}

// AddHeader feeds a decoded alias header. A header with a new sequence
// resets any in-flight data blocks.
func (a *MotorolaAliasAssembler) AddHeader(h MotorolaAliasHeader) MotorolaAliasResult {
	a.evictStale()
	if h.BlockCount == 0 || h.BlockCount > aliasMaxBlocks {
		return MotorolaAliasResult{}
	}
	if !a.haveHeader || h.Sequence != a.sequence {
		a.blocks = make(map[uint8][]byte)
		a.leads = make(map[uint8]uint8)
	}
	a.haveHeader = true
	a.sequence = h.Sequence
	a.blockCount = h.BlockCount
	a.talkgroup = h.TalkgroupID
	a.header = append([]byte(nil), h.Fragment...)
	a.updated = a.now()
	return a.tryComplete()
}

// AddData feeds a decoded alias data block. Blocks whose sequence
// doesn't match the current header are ignored.
func (a *MotorolaAliasAssembler) AddData(d MotorolaAliasData) MotorolaAliasResult {
	a.evictStale()
	if !a.haveHeader || d.Sequence != a.sequence || d.BlockNumber == 0 ||
		d.BlockNumber > a.blockCount {
		return MotorolaAliasResult{}
	}
	a.blocks[d.BlockNumber] = append([]byte(nil), d.Fragment...)
	a.leads[d.BlockNumber] = d.LeadNibble
	a.updated = a.now()
	return a.tryComplete()
}

// tryComplete reassembles and decodes once the header and every data
// block are present.
func (a *MotorolaAliasAssembler) tryComplete() MotorolaAliasResult {
	if !a.haveHeader || len(a.blocks) < int(a.blockCount) {
		return MotorolaAliasResult{}
	}
	blocks := make([][]byte, 0, a.blockCount)
	leads := make([]uint8, 0, a.blockCount)
	for i := uint8(1); i <= a.blockCount; i++ {
		b, ok := a.blocks[i]
		if !ok {
			return MotorolaAliasResult{}
		}
		blocks = append(blocks, b)
		leads = append(leads, a.leads[i])
	}
	talkgroup := a.talkgroup
	decoded, ok := motorola.DecodeMessage(assembleMotorolaAliasMessage(a.header, blocks, leads))
	if !ok {
		return MotorolaAliasResult{}
	}
	a.reset()
	// An empty alias (all-non-printable decode) still completes so the
	// source RID and the reassembled ciphertext are reported; the publish
	// path drops the empty alias string, but the ciphertext is surfaced as
	// cryptanalysis ground truth regardless (#773).
	return MotorolaAliasResult{
		Alias:       decoded.Alias,
		SourceID:    decoded.RadioID,
		TalkgroupID: talkgroup,
		Encoded:     decoded.Encoded,
		CRCOK:       decoded.CRCOK,
		Reliable:    decoded.AliasReliable,
		Complete:    true,
	}
}

// assembleMotorolaAliasMessage rebuilds the packed Motorola alias message
// from the byte-aligned header fragment and the nibble-aligned data
// fragments. Each data block contributes a leading cipher nibble (the low
// nibble of its sequence octet) followed by its byte-aligned fragment, so
// the fragments must be concatenated as a nibble stream and only then
// packed back to bytes — matching SDRTrunk's reassembled FRAGMENT stream
// (#376/#773). A whole-byte concatenation drops each block's leading
// nibble and shifts the entire cipher region by 4 bits, which is what made
// the decode fail safe to an empty alias.
func assembleMotorolaAliasMessage(header []byte, blocks [][]byte, leads []uint8) []byte {
	nibs := bytesToNibbles(header)
	for i, b := range blocks {
		nibs = append(nibs, leads[i]&0x0F)
		nibs = append(nibs, bytesToNibbles(b)...)
	}
	return packNibbles(nibs)
}

// bytesToNibbles expands each byte into its high then low nibble.
func bytesToNibbles(b []byte) []uint8 {
	nibs := make([]uint8, 0, len(b)*2)
	for _, v := range b {
		nibs = append(nibs, v>>4, v&0x0F)
	}
	return nibs
}

// packNibbles packs a nibble stream MSB-first into bytes. A trailing odd
// nibble (the message should be byte-aligned once every block is present)
// is left-justified into a final byte so no cipher data is silently lost.
func packNibbles(nibs []uint8) []byte {
	out := make([]byte, 0, (len(nibs)+1)/2)
	for i := 0; i < len(nibs); i += 2 {
		hi := nibs[i] & 0x0F
		var lo uint8
		if i+1 < len(nibs) {
			lo = nibs[i+1] & 0x0F
		}
		out = append(out, hi<<4|lo)
	}
	return out
}

func (a *MotorolaAliasAssembler) evictStale() {
	if !a.haveHeader {
		return
	}
	if a.now().Sub(a.updated) > aliasStaleAfter {
		a.reset()
	}
}

func (a *MotorolaAliasAssembler) reset() {
	a.haveHeader = false
	a.sequence = 0
	a.blockCount = 0
	a.talkgroup = 0
	a.header = nil
	a.blocks = make(map[uint8][]byte)
	a.leads = make(map[uint8]uint8)
	a.updated = time.Time{}
}
