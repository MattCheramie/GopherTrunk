// Package dpmr decodes dPMR (ETSI TS 102 658 V2.6.1): 4-level FSK at 2400
// symbols a second on 6.25 kHz channels. The standard defines three modes:
//
//	Mode 1   peer-to-peer, no infrastructure (dPMR446 handhelds)
//	Mode 2   repeater or base station
//	Mode 3   trunked: a beacon channel sets calls up and sends the
//	         radios to traffic channels
//
// Everything here follows the standard's air format and has been checked
// against two decoders that receive dPMR (dsdcc and DSD-FME) where they
// cover it. No dPMR transmission has been decoded through it yet.
//
// The traffic channel, voice:
//
//	airframe.go   The sync words FS1..FS4, the payload frame (FS2 or
//	              channel code, control channel, four TCH blocks), and the
//	              control channel's coding: CRC7, Hamming (12,8), the 12 x 6
//	              interleave and the scrambler.
//	traffic.go    TrafficChannel: locks onto the superframe grid at an FS2,
//	              follows a transmission to its end frame and hands out the
//	              vocoder frames of a voice call.
//	voice.go, voice_ambe.go
//	              The four 72-bit TCH blocks of a payload frame and the
//	              AMBE+2 vocoder frame in each (DMR's layout bit for bit).
//
// The Mode 3 beacon channel, call set-up (clauses 4.2.3.7, 5.2, 5.5.32):
//
//	beacon_air.go The message field (CRC8, Hamming, 12 x 10 interleave,
//	              scrambler) and the SYScast block (three CRC7 fields, each
//	              sent twice, 12 x 6 interleave, one 216-bit scramble).
//	beacon_msg.go Message_Type, the Aloha and Goto Channel broadcasts, the
//	              SYScast fields, the frequency coding (BAND + SEP x field)
//	              and the B2 address mapping of annex A.
//	beacon.go     BeaconFramer: FS1, then 264-dibit framesets of a message
//	              slot and a SYScast (or appended-data) slot.
//	control.go    ControlChannel: locks on the beacon's System Identity
//	              Code and publishes each Goto Channel for voice as a
//	              `trunking.Grant` with Protocol "dpmr".
//
// The internal/scanner/conventional package's dpmr scan-list mode reaches
// the same voice chain without a beacon, for Mode 1 and Mode 2.
//
// receiver/ is a plain IQ to dibit chain without carrier correction; the
// production paths use the NXDN receiver at dPMR's rate instead.
package dpmr

// DibitSink consumes the dibit stream a receiver decodes from IQ. baseIdx
// is the absolute dibit index of dibits[0], monotonically increasing.
type DibitSink func(dibits []uint8, baseIdx int)
