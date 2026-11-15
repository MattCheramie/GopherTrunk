// Package dstar decodes D-STAR (Digital Smart Technology for Amateur
// Radio) signalling per the JARL D-STAR specification, freely
// published by the Japanese Amateur Radio League. D-STAR is an
// amateur-radio digital voice + data mode using GMSK at 4800 bps with
// AMBE (the original variant — not AMBE+2) for voice.
//
// D-STAR is *not* a trunked protocol in the cellular / TETRA sense:
// each repeater is its own conventional channel and there's no
// dedicated control channel that grants traffic onto a separate
// frequency. What it does have is a structured Header frame at call
// setup time that carries the full call identity (calling / called
// callsigns + repeater routing), and embedded sync / framing so a
// receiver can lock and demarcate voice frames cleanly.
//
// What this package gives you:
//
//	sync.go     Frame sync, slow-data sync and end patterns in on-air
//	            time order, and a tolerant SyncDetector; every
//	            detector also finds the complement (Polarity).
//	header.go   The 41-byte header (flags, RPT2, RPT1, UR, MY1, MY2)
//	            and its CRC-16/X-25, carried low byte first.
//	control.go  State machine that ingests Header frames and
//	            publishes events.KindCCLocked + events.KindGrant on
//	            the bus with `trunking.Grant.Protocol = "dstar"`.
//	            cc.locked fires on the first valid header (the
//	            receiver has locked onto the repeater); grant fires
//	            on every header whose UR field is a group call ("CQCQCQ"
//	            or a "/repeater" routing tag). Same shape as the other
//	            protocol packages so the engine + recorder + composer
//	            don't need to know D-STAR is conventional.
//
// Also wired now:
//
//	receiver/     The IQ → 4800 bps GMSK (BT=0.5) bit chain.
//	framing.DecodeDStarHeaderFEC  The header FEC (descramble →
//	              deinterleave → K=3 Viterbi), driven by Process.
//	encode.go     On-air encoders for fixtures and synthetic signals.
//	voice.go      Slow-Data-sync-anchored 96-bit DV cadence tracker
//	              (VoiceChannel), independent of the header decode.
//	voice_ambe.go The AMBE FEC to the 49-bit vocoder payload. D-STAR
//	              carries the original AMBE 3600×2400 — which IS the
//	              base decoder internal/voice/ambe2.New() implements
//	              (params.go ports mbelib's ambe3600x2400.c), so no
//	              separate vocoder is needed; the composer's dstar
//	              voice chain renders it via the "ambe2" mapping.
//
// The air format (#1261) follows the MMDVM firmware (DStarTX.cpp /
// DStarRX.cpp, the code D-STAR hotspots and repeaters run) and DSD's
// voice interleave, pinned by literal output of MMDVM's own encoder and
// MMDVM's null AMBE frame. The format GT carried before was invented and
// agreed only with its own encoder. ⚠️ Still UNVERIFIED ON A REAL D-STAR
// CAPTURE (#764/#771).
package dstar
