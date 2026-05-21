package phase2

import (
	"sync"
	"time"
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
