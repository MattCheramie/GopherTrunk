package trunking

import (
	"sync"
	"time"
)

// PatchGroup associates a P25 super-group / dynamic-regroup talkgroup
// with the member talkgroups merged into it. A patch makes the member
// groups share one RF channel, so a call on the super-group physically
// IS the members' traffic — "following" a patch means attributing the
// call to every member, not retuning.
type PatchGroup struct {
	SuperGroup uint32
	Members    []uint32
	Vendor     string // "motorola" | "harris"
	UpdatedAt  time.Time
}

// Patch is the events.KindPatch payload — a patch add or cancel a
// trunked system announced.
type Patch struct {
	System     string
	Protocol   string
	SuperGroup uint32
	Members    []uint32
	Vendor     string
	Add        bool // true = patch now active, false = patch cancelled
	At         time.Time
}

// PatchRegistry is a thread-safe live table of active patch groups
// keyed by super-group. The engine maintains one and consults it when
// dispatching grants so a call on a patched super-group is attributed
// to its member talkgroups.
type PatchRegistry struct {
	mu     sync.Mutex
	groups map[uint32]PatchGroup
}

// NewPatchRegistry returns an empty registry.
func NewPatchRegistry() *PatchRegistry {
	return &PatchRegistry{groups: make(map[uint32]PatchGroup)}
}

// Apply records (or replaces) a patch group.
func (r *PatchRegistry) Apply(pg PatchGroup) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.groups[pg.SuperGroup] = pg
}

// Delete removes a patch group by its super-group address.
func (r *PatchRegistry) Delete(superGroup uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.groups, superGroup)
}

// MembersOf returns a copy of the member talkgroups of the patch keyed
// by group, or nil if group is not an active super-group.
func (r *PatchRegistry) MembersOf(group uint32) []uint32 {
	r.mu.Lock()
	defer r.mu.Unlock()
	pg, ok := r.groups[group]
	if !ok {
		return nil
	}
	return append([]uint32(nil), pg.Members...)
}

// Active returns a snapshot of every active patch group.
func (r *PatchRegistry) Active() []PatchGroup {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]PatchGroup, 0, len(r.groups))
	for _, pg := range r.groups {
		out = append(out, pg)
	}
	return out
}
