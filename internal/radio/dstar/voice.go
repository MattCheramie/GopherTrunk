package dstar

// DV (voice) frame tracking for D-STAR.
//
// After the 660-bit FEC-encoded PCH header, a D-STAR DV transmission is
// a free-running cadence of 96-bit DV frames: 72 voice bits (one AMBE
// 3600×2400 frame, voice_ambe.go) followed by 24 slow-data bits. DV
// frame 0 — and every 21st frame after it — carries the 24-bit Slow
// Data sync (sync.go SlowDataSyncHex) in its data field, so the voice
// cadence can be anchored purely on that sync with NO dependency on
// the header decode: a sync match ending at bit p means bits
// [p−95, p−24] were that frame's voice payload, and the next frame
// starts at p+1.
//
// VoiceChannel free-runs the 96-bit cadence between syncs, re-anchors
// unconditionally on every Slow Data sync match (so a symbol slip
// mid-transmission recovers within one 21-frame cycle), and drops the
// anchor after two sync-less cycles so noise after the transmission
// ends doesn't keep emitting garbage frames.
//
// ⚠️ UNVERIFIED ON AIR — see voice_ambe.go. The 96-bit [72 voice |
// 24 data] carve and the 21-frame sync cadence are transcribed from
// the JARL DV spec and validated only synthetically; a real D-STAR
// capture is needed to confirm them.

const (
	// DVFrameBits is one DV frame: 72 voice + 24 slow-data bits.
	DVFrameBits = 96
	// DVVoiceBits is the AMBE voice payload at the start of each DV
	// frame.
	DVVoiceBits = 72
	// DVDataBits is the slow-data field that closes each DV frame.
	DVDataBits = DVFrameBits - DVVoiceBits
	// DVFramesPerSyncCycle is the Slow Data sync cadence: frame 0 of
	// every 21-frame cycle carries the sync in its data field.
	DVFramesPerSyncCycle = 21
)

// DVFrame is one decoded AMBE voice frame carved from the DV stream.
type DVFrame struct {
	// Payload is the 49-bit vocoder payload, one bit per byte, in the
	// order DecodeDVVoiceBits emits (C0:12 + C1:12 + C2:11 + C3:14).
	// Pack it MSB-first to 7 bytes (framing.PackBitsMSB) for the base
	// ambe2 decoder.
	Payload []byte
	// Errors is the Golay-corrected bit count across the C0/C1
	// sub-vectors — a decode-quality signal to feed the vocoder's
	// error-aware smoothing.
	Errors int
}

// VoiceChannel tracks the DV voice cadence on the raw bit stream, in
// parallel to (and independent of) the header-decoding ControlChannel.
// The bit stream + baseIdx contract matches ControlChannel.Process
// (continuous stream, monotonic baseIdx).
type VoiceChannel struct {
	det *SyncDetector
	// recent is a rolling window of the most recent DVFrameBits bits,
	// so a sync match can decode the voice payload of the frame the
	// sync itself closes.
	recent []uint8
	// anchored: free-running the 96-bit cadence. remaining counts
	// down the bits of the frame being collected.
	anchored  bool
	remaining int
	frame     []uint8
	// framesSinceSync counts free-run frames since the last sync
	// match; at 2 cycles without a sync the anchor is dropped.
	framesSinceSync int
	// lastFrameEnd is the absolute bit index at which the free-run
	// last emitted a frame. When the cadence is healthy, the Slow
	// Data sync's last bit lands exactly on a frame boundary — the
	// free-run has already emitted that frame, so the sync branch
	// must not emit it a second time from the rolling window.
	lastFrameEnd int
	matchScratch []int
	sink         func(DVFrame)
}

// NewVoiceChannel builds a DV voice tracker. sink receives each
// FEC-decoded voice frame.
func NewVoiceChannel(sink func(DVFrame)) *VoiceChannel {
	return &VoiceChannel{
		det:          NewSyncDetector(SlowDataSyncBitsSlice(), 1),
		recent:       make([]uint8, 0, DVFrameBits),
		frame:        make([]uint8, 0, DVFrameBits),
		lastFrameEnd: -1,
		sink:         sink,
	}
}

// Process consumes a window of raw bits (same contract as
// ControlChannel.Process) and returns baseIdx+len(bits).
func (v *VoiceChannel) Process(bits []byte, baseIdx int) int {
	v.matchScratch, _ = v.det.Process(v.matchScratch[:0], bits, baseIdx)
	matchIdx := 0
	for i, b := range bits {
		absPos := baseIdx + i

		// Rolling history for the sync-frame voice decode.
		if len(v.recent) == DVFrameBits {
			copy(v.recent, v.recent[1:])
			v.recent = v.recent[:DVFrameBits-1]
		}
		v.recent = append(v.recent, b&1)

		// Free-run collection (before the sync check, for the same
		// reason as the other Process adapters: a match's index is the
		// LAST bit of the sync, so the next frame starts one iteration
		// later).
		if v.anchored && v.remaining > 0 {
			v.frame = append(v.frame, b&1)
			v.remaining--
			if v.remaining == 0 {
				v.emitVoice(v.frame[:DVVoiceBits])
				v.lastFrameEnd = absPos
				v.frame = v.frame[:0]
				v.framesSinceSync++
				if v.framesSinceSync >= 2*DVFramesPerSyncCycle {
					// Two full cycles without a Slow Data sync: the
					// transmission is over (or we slipped hopelessly) —
					// stop free-running until the next sync.
					v.anchored = false
				} else {
					v.remaining = DVFrameBits
				}
			}
		}

		for matchIdx < len(v.matchScratch) && v.matchScratch[matchIdx] == absPos {
			// A Slow Data sync just ended here: the 96 bits in the
			// rolling window are the sync's own DV frame — decode its
			// voice payload (unless the free-run already emitted this
			// exact frame at this bit), then (re-)anchor the cadence
			// unconditionally so a bit slip re-locks.
			if len(v.recent) == DVFrameBits && v.lastFrameEnd != absPos {
				v.emitVoice(v.recent[:DVVoiceBits])
				v.lastFrameEnd = absPos
			}
			v.anchored = true
			v.remaining = DVFrameBits
			v.frame = v.frame[:0]
			v.framesSinceSync = 0
			matchIdx++
		}
	}
	return baseIdx + len(bits)
}

// emitVoice FEC-decodes 72 voice bits and hands the result to the sink.
// Frames whose AMBE FEC hard-fails are dropped silently.
func (v *VoiceChannel) emitVoice(voiceBits []uint8) {
	if v.sink == nil {
		return
	}
	payload, errs, err := DecodeDVVoiceBits(voiceBits)
	if err != nil {
		return
	}
	v.sink(DVFrame{Payload: payload, Errors: errs})
}

// Reset clears the sync-detection + cadence state; call on retune.
func (v *VoiceChannel) Reset() {
	v.det.Reset()
	v.recent = v.recent[:0]
	v.anchored = false
	v.remaining = 0
	v.frame = v.frame[:0]
	v.framesSinceSync = 0
	v.lastFrameEnd = -1
}
