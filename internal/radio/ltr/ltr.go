// Package ltr decodes Logic Trunked Radio (LTR) — the legacy
// distributed-trunking system invented by E.F. Johnson in the 1970s
// and still in deployment for utility / industrial fleets. LTR is
// architecturally different from the centrally-coordinated trunked
// systems (P25, DMR, NXDN, Motorola Type II, EDACS): every repeater
// transmits its own 41-bit status word at 300 bps, on top of the
// in-band voice. There is no central control channel; an LTR scanner
// follows calls by watching every repeater's status word and tuning
// to whichever one currently announces the talkgroup of interest.
//
// Files:
//
//	status.go    41-bit Status word: Sync + Area + Group flag +
//	             Channel + Home repeater + Group ID + Free + FCS.
//	             Round-trip assemble / parse helpers and bit
//	             accessors.
//	bandplan.go  Channel → Hz resolver. LTR uses 4-bit channel
//	             numbers (1..20 typically); the operator-supplied
//	             band plan maps each to the repeater's transmit
//	             frequency. Same Resolver shape as the
//	             motorola / edacs packages.
//	control.go   Per-repeater state machine. Ingest each decoded
//	             Status word and republish it as a
//	             events.KindGrant when the status indicates an
//	             active call on this repeater, plus a one-shot
//	             events.KindCCLocked the first time we see a valid
//	             status from a given repeater (so the hunter can
//	             confirm we're tuned to the right place).
//
// What's NOT yet wired (honest deferrals):
//
//   - The 300-baud sub-audible status-word demodulator. It rides
//     under the voice on most LTR repeaters and uses its own
//     baseband decoding; the Status parser here assumes the upstream
//     caller has already delivered 41 clean bits.
//   - Manchester encoding / decoding of the on-air bit stream.
//   - Repeater-pair coordination (LTR-Net) where status-word
//     references can hop between physical sites.
package ltr
