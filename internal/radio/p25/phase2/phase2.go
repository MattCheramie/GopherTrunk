// Package phase2 decodes P25 Phase 2 traffic-channel framing per
// TIA-102.BBAB / BCKB. Phase 2 introduces 2-slot TDMA on top of the
// existing P25 Phase 1 control-channel infrastructure: the control
// channel itself stays Phase 1 (FDMA, 4800 sym/sec, C4FM), but voice
// grants direct subscribers to a Phase 2 traffic channel that runs
// 6000 sym/sec H-DQPSK (or H-CPM in some references) carrying two
// concurrent timeslots.
//
// What this package gives you:
//
//   sync.go        Phase 2 outbound + inbound sync constants.
//   superframe.go  Superframe + slot-type layout constants.
//   mac.go         MAC PDU structure + opcode enum + payload
//                  accessors. MAC PDUs carry late-grant signalling
//                  (GroupVoiceChannelGrantUpdate, MAC_PTT, etc.)
//                  on top of the Phase 2 voice frames.
//   control.go     State machine that ingests MAC PDUs and
//                  publishes events.KindGrant on the bus, with the
//                  trunking.Grant.Protocol tag set to
//                  "p25-phase2".
//
// What's NOT yet wired (honest deferrals):
//
//   - The H-DQPSK / H-CPM symbol decoder for the Phase 2 traffic
//     channel. internal/dsp/demod/dqpsk.go is the closest fit;
//     wiring it through a TDMA superframe-sync stage is the gating
//     piece for live captures.
//   - TDMA superframe / sub-slot synchronisation. Phase 2
//     superframes are scheduled in 12 sub-frames per 360 ms, with
//     the two timeslots interleaved at the symbol level.
//   - Voice frame extraction → AMBE+2 vocoder. The frame layout is
//     here; the AMBE+2 decode lives behind the `mbelib` build tag
//     (see internal/voice/mbelib).
//   - Reed-Solomon / Trellis FEC over the MAC PDU bits. The
//     parsing here assumes the upstream caller has already
//     corrected errors.
package phase2
