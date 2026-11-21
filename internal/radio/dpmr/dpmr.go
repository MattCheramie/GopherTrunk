// Package dpmr decodes dPMR (digital PMR446 / ETSI TS 102 658) Mode 3
// trunking signalling. dPMR is a 4-level FSK protocol designed for
// 6.25 kHz channel spacing — the digital successor to analogue PMR446
// in Europe and a sibling of NXDN in framing philosophy. Three
// operating modes are defined:
//
//	Mode 1   peer-to-peer (no infrastructure)
//	Mode 2   managed direct (no repeater, but a dispatch channel)
//	Mode 3   centralised trunking (a dedicated control channel that
//	         grants voice / data calls onto traffic channels)
//
// This package targets Mode 3 — the only mode where the standard
// "see CC opcode → retune Voice device → follow grant" loop applies.
// Modes 1 and 2 don't have a control channel for the engine to hunt.
//
// The package has two layers, and only one of them is the air format.
//
// The traffic channel (voice) follows ETSI TS 102 658 as transmitted:
//
//	airframe.go  The sync words FS1..FS4, the payload frame (FS2 or
//	             channel code, control channel, four TCH blocks), and
//	             the control channel's coding: CRC7, Hamming (12,8),
//	             the 12 x 6 interleave and the scrambler.
//	traffic.go   TrafficChannel: locks onto the superframe grid at an
//	             FS2, follows a transmission to its end frame, and hands
//	             out the vocoder frames of a voice call.
//	voice.go     The four 72-bit TCH blocks of a payload frame.
//	voice_ambe.go The AMBE+2 vocoder frame in each, which is DMR's and
//	             NXDN's layout bit for bit.
//
// It is pinned to the standard and to two decoders that receive dPMR on
// the air (dsdcc and DSD-FME); no dPMR transmission has been decoded
// through it yet.
//
// The Mode 3 control-channel side is an older model that is NOT the air
// format:
//
//	sync.go      "FS1 / FS2 / FS3" as 48-bit words. The standard's FS2
//	             and FS3 are 24 bits, and none of the three values is
//	             the standard's.
//	csbk.go      An 80-bit "CSBK". dPMR has no such unit: a Mode 3 beacon
//	             sends SYScast frames and message frames (clause 4.2.2).
//	process.go   The "FS3 then CSBK" framer built on those.
//	opcodes.go, bandplan.go, control.go
//	             The message types, channel plan and state machine that
//	             publish events.KindCCLocked / events.KindGrant with
//	             `trunking.Grant.Protocol = "dpmr"`.
//
// It decodes its own synthetic fixtures and cannot frame a real beacon, so
// nothing on the air can start a dPMR call yet: the voice chain runs only
// on a "dpmr" grant, and only this control channel issues one. Rebuilding
// it on the beacon format, or giving the conventional scanner a dPMR
// channel mode for the licence-free dPMR446 handhelds, is the next step.
//
// receiver/ is the IQ to dibit chain the control-channel pipeline uses
// (2400 symbols a second, root raised cosine 0.2). It has no carrier
// correction; the voice chain uses the NXDN receiver at the same rate
// instead (see internal/voice/composer/dpmr_voice.go).
package dpmr
