package phase1

import (
	"sync"
	"time"
)

// P25 Phase 1 talker-alias reassembly. A talker alias — a radio's
// human-readable display name — is too long for one TSBK, so a system
// sends it as a numbered sequence of vendor TSBK fragments.
// TalkerAliasAssembler buffers the fragments per source unit and emits
// the completed name once every block has arrived.
//
// This mirrors the Phase 2 assembler (internal/radio/p25/phase2/
// talker_alias.go); the two packages keep independent copies rather
// than cross-importing, as phase1/phase2 already do for BandPlan and
// other shared shapes. The fragment wire layout is the project's
// working model — see tsbk_vendor.go.

// TalkerAliasFragment is one numbered piece of a radio's talker alias.
type TalkerAliasFragment struct {
	SourceID   uint32
	BlockIndex uint8
	BlockCount uint8
	Data       []byte
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
// otherwise ("", 0, false). Out-of-range or malformed fragments are
// ignored. Add tolerates fragments arriving in any order.
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

// cleanAlias renders the alias bytes as a printable ASCII string,
// dropping control and non-ASCII characters.
func cleanAlias(raw []byte) string {
	out := make([]byte, 0, len(raw))
	for _, b := range raw {
		if b >= 0x20 && b < 0x7F {
			out = append(out, b)
		}
	}
	return string(out)
}
