package trunking

// Priority resolution per Trunk-Recorder convention:
//   - Talkgroup priority is an integer 1..10, lower = higher priority.
//   - Priority 0 (the zero value) means "unset" → treated as lowest priority.
//   - Lockout flag wins outright: a locked-out grant is dropped before
//     priority comparison.
//   - Emergency flag bumps priority to 0 (above the highest configured).

const (
	priorityEmergency = 0
	priorityUnset     = 11 // anything ≥ 10 is "lowest"
)

// EffectivePriority returns the runtime priority used by the engine when
// comparing grants. Lower number = higher priority.
func EffectivePriority(g Grant, tg *TalkGroup) int {
	if g.Emergency {
		return priorityEmergency
	}
	if tg == nil || tg.Priority <= 0 {
		return priorityUnset
	}
	return tg.Priority
}

// CanPreempt reports whether a new grant should kick an active call off a
// Voice device. The rule is strict-higher: equal priority does NOT
// preempt (so a stable call holds the device against same-priority grants).
//
// Returns false if the new grant is locked out by talkgroup policy — the
// engine handles lockout earlier in the dispatch path, but the predicate
// is defensive here so callers can compose freely.
func CanPreempt(active Grant, activeTG *TalkGroup, incoming Grant, incomingTG *TalkGroup) bool {
	if incomingTG != nil && incomingTG.Lockout {
		return false
	}
	return EffectivePriority(incoming, incomingTG) < EffectivePriority(active, activeTG)
}
