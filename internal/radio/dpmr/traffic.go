package dpmr

// TrafficChannel decodes dPMR Mode 3 voice traffic frames from the raw
// dibit stream, in parallel to (and independent of) the control-channel
// ControlChannel. The CC path (process.go) is hardwired to FS3 → CSBK,
// so voice needs its own adapter: this one runs FS1 + FS2 sync
// detectors (the superframe-start and mid-superframe voice syncs,
// defined in sync.go but unused by the CC path), slices each frame's
// CCH + TCH fields, and routes the 144-dibit TCH payload through
// ExtractTCHFrames, delivering the decoded AMBE+2 voice frames to a
// sink callback. FS3/CSBK signalling bursts never match FS1/FS2 within
// tolerance, so they are ignored here by construction.
//
// The dibit stream + baseIdx contract matches ControlChannel.Process
// (continuous stream, monotonic baseIdx). Attach a TrafficChannel to a
// dedicated voice-tap receiver tuned to the granted traffic channel.
//
// ⚠️ UNVERIFIED ON AIR — see voice_ambe.go / voice.go.
type TrafficChannel struct {
	fs1 *SyncDetector
	fs2 *SyncDetector
	// remaining > 0: collecting post-sync frame dibits; counts down
	// to 0 as Process feeds dibits forward.
	remaining int
	frame     []uint8
	// scratch slices reused across calls so the detectors don't
	// allocate per Process call.
	fs1Scratch []int
	fs2Scratch []int
	sink       func(frames []TCHFrame)
}

// NewTrafficChannel builds a voice-traffic-frame decoder. sink receives
// the decoded voice frames of each FS1/FS2 frame (never called with an
// empty slice).
func NewTrafficChannel(sink func(frames []TCHFrame)) *TrafficChannel {
	return &TrafficChannel{
		fs1:   NewSyncDetector(FS1Dibits(), 1),
		fs2:   NewSyncDetector(FS2Dibits(), 1),
		frame: make([]uint8, 0, postSyncDibitsTraffic),
		sink:  sink,
	}
}

// Process consumes a window of raw dibits (same contract as
// ControlChannel.Process) and returns baseIdx+len(dibits). It mirrors
// the process.go countdown: on each FS1 or FS2 match it collects the
// next postSyncDibitsTraffic dibits, then decodes the frame. Collecting
// happens BEFORE the sync-match check for the same reason as in
// process.go: a match's absolute index is the LAST dibit of the
// 24-dibit sync, so the frame starts one iteration later.
func (t *TrafficChannel) Process(dibits []uint8, baseIdx int) int {
	t.fs1Scratch, _ = t.fs1.Process(t.fs1Scratch[:0], dibits, baseIdx)
	t.fs2Scratch, _ = t.fs2.Process(t.fs2Scratch[:0], dibits, baseIdx)
	fs1Idx, fs2Idx := 0, 0
	for i, d := range dibits {
		absPos := baseIdx + i
		if t.remaining > 0 {
			t.frame = append(t.frame, d)
			t.remaining--
			if t.remaining == 0 {
				t.handleFrame(t.frame)
				t.frame = t.frame[:0]
			}
		}
		matched := false
		for fs1Idx < len(t.fs1Scratch) && t.fs1Scratch[fs1Idx] == absPos {
			matched = true
			fs1Idx++
		}
		for fs2Idx < len(t.fs2Scratch) && t.fs2Scratch[fs2Idx] == absPos {
			matched = true
			fs2Idx++
		}
		if matched {
			t.remaining = postSyncDibitsTraffic
			t.frame = t.frame[:0]
		}
	}
	return baseIdx + len(dibits)
}

// handleFrame slices a collected post-sync voice frame: CCH (24 dibits,
// currently skipped — like NXDN's SACCH, its call-control content is
// not needed to render audio) then the 144-dibit TCH payload.
func (t *TrafficChannel) handleFrame(frame []uint8) {
	if len(frame) != postSyncDibitsTraffic {
		return
	}
	tch := frame[CCHDibits : CCHDibits+TCHFieldDibits]
	frames := ExtractTCHFrames(tch)
	if len(frames) > 0 && t.sink != nil {
		t.sink(frames)
	}
}

// Reset clears the sync-detection + partial-frame state; call on retune.
func (t *TrafficChannel) Reset() {
	t.fs1.Reset()
	t.fs2.Reset()
	t.remaining = 0
	t.frame = t.frame[:0]
}
