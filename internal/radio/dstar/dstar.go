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
//	sync.go     Frame Sync and Slow Data sync constants and a
//	            tolerant SyncDetector matching the shape used by the
//	            other protocol packages.
//	header.go   PCH (Preamble + Header) parser — the 660-bit
//	            packet that opens a transmission carrying the eight
//	            callsign fields (RPT2, RPT1, UR, MY1) plus the
//	            Repeater / Personal flags (RF1).
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
// What's NOT yet wired (honest deferrals):
//
//   - The 4800 bps GMSK demodulator + bit-clock recovery.
//     internal/dsp/demod is the closest fit; D-STAR uses BT=0.5
//     filtering with a slightly different matched-filter shape than
//     the other protocols here.
//   - The PCH FEC: convolutional rate-1/2 inner + scrambler outer +
//     interleaver. Parsing here assumes upstream FEC has corrected
//     errors.
//   - Voice frame extraction → original AMBE vocoder (note: original
//     AMBE, not AMBE+2 — separate algorithm, same DVSI patent family).
//     IMBE lives in internal/voice/imbe and AMBE+2 in
//     internal/voice/ambe2; plain AMBE is a future deferral with no
//     pure-Go decoder shipped yet.
//
// As with the other protocol packages: ship a clean structured
// surface now, leave the analogue / FEC / vocoder pieces as named
// follow-ups so the trunking engine can consume the events
// end-to-end against fixtures.
package dstar
