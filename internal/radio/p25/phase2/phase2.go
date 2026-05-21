// Package phase2 decodes P25 Phase 2 traffic-channel framing per
// TIA-102.BBAB / BBAC. Phase 2 introduces 2-slot TDMA on top of the
// existing P25 Phase 1 control-channel infrastructure: the control
// channel itself stays Phase 1 (FDMA, 4800 sym/sec, C4FM), but voice
// grants direct subscribers to a Phase 2 traffic channel that runs
// 6000 sym/sec H-DQPSK carrying two concurrent timeslots.
//
// The live decode path is structured: the receiver
// (internal/radio/p25/phase2/receiver) recovers the H-DQPSK dibit
// stream, SuperframeDecoder (superframe_decoder.go) locks the 360 ms
// TDMA superframe and slices its 12 sub-frames, isch.go decodes each
// sub-frame's SlotType, and IngestSuperframe (superframe_ingest.go)
// routes the MAC-bearing sub-frames into the control-channel state
// machine while the composer voice chain extracts voice.
//
// What this package gives you:
//
//	sync.go               Phase 2 outbound + inbound sync constants.
//	superframe.go         Superframe + slot-type layout constants.
//	superframe_decoder.go SuperframeDecoder: dibit stream → 360 ms
//	                      superframes of 12 SlotType-tagged sub-frames.
//	isch.go               ISCH (Inter-Slot Channel) SlotType decode.
//	process.go            Flat sync-window MAC PDU adapter + the
//	                      decodeMACPDUDibits FEC chain (trellis, RS,
//	                      PN44 descramble, block deinterleave).
//	superframe_ingest.go  IngestSuperframe: route MAC sub-frames into
//	                      the state machine.
//	voice.go              AMBE+2 voice-frame extraction (4V / 2V slots).
//	mac.go / mac_standard.go / mac_vendor.go
//	                      MAC PDU structure, opcode enum, and per-opcode
//	                      accessors — standard plus Motorola/Harris
//	                      vendor messages (patch/regroup, talker alias),
//	                      encryption identification.
//	talker_alias.go       Multi-fragment talker-alias reassembly.
//	identifier.go         Band-plan accumulator: resolves a grant's
//	                      (ChannelID, ChannelNumber) to a frequency.
//	control.go            State machine that ingests MAC PDUs and
//	                      publishes events.KindGrant / KindCCLocked /
//	                      KindPatch / KindAffiliation / KindTalkerAlias,
//	                      with trunking.Grant.Protocol set to
//	                      "p25-phase2".
//	encode.go             Synthesized-fixture encoders (test scaffolding).
//
// Honest non-goals and working models:
//
//   - Decryption. Like SDRtrunk, this package identifies encryption
//     (algorithm ID, key ID) but does not decrypt voice.
//   - Inbound (MS → BS) uplink decoding. Only the outbound downlink is
//     decoded — the inbound sync constant in sync.go is unused.
//   - Several TIA-102.BBAB/BBAC wire details — the superframe sync
//     cadence, the ISCH and voice-slot bit layouts, the encryption-sync
//     and talker-alias MAC opcodes, the per-burst block interleaver —
//     are not in the repo's spec PDFs. Each is implemented as a
//     documented working model confined to one file with a symmetric
//     encoder, so a real-capture calibration is a local change.
package phase2
