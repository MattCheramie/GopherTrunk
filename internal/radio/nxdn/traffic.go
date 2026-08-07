package nxdn

import (
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// TrafficChannel decodes NXDN VCH (voice) traffic frames from the raw
// dibit stream, in parallel to (and independent of) the
// control-channel ControlChannel. The CC path (process.go / control.go)
// is hardwired to CAC and gates on RFCh==RFChControl, so voice needs its
// own adapter: this one runs its own outbound-FSW SyncDetector, slices
// each frame's LICH + Information field, and — for RFChTraffic frames —
// routes the 144-dibit Info field through ExtractVCHFrames, delivering
// the decoded AMBE+2 voice frames to a sink callback.
//
// The dibit stream + baseIdx contract matches ControlChannel.Process
// (continuous stream, monotonic baseIdx). Attach a TrafficChannel to the
// same receiver DibitSink used for the CC, or a dedicated voice-tap
// receiver.
//
// ⚠️ UNVERIFIED ON AIR — see voice_ambe.go / voice.go.
type TrafficChannel struct {
	det          *SyncDetector
	remaining    int
	frame        []uint8
	matchScratch []Match
	sink         func(frames []VCHFrame)
}

// postSyncDibitsTraffic is the count of dibits collected after the
// 8-dibit FSW match for a full traffic frame: 8 LICH wire + 32 SACCH +
// 144 Information = 184 dibits (the whole 192-dibit frame minus the FSW).
const postSyncDibitsTraffic = LICHWireDibits + SACCHDibits + InfoFieldDibits

// NewTrafficChannel builds a traffic-frame decoder. sink receives the
// decoded voice frames of each RFChTraffic frame (never called with an
// empty slice).
func NewTrafficChannel(sink func(frames []VCHFrame)) *TrafficChannel {
	return &TrafficChannel{
		det:   NewSyncDetector([][]uint8{FSWDibitsOutbound}, 1),
		frame: make([]uint8, 0, postSyncDibitsTraffic),
		sink:  sink,
	}
}

// Process consumes a window of raw dibits (same contract as
// ControlChannel.Process) and returns baseIdx+len(dibits). It mirrors
// the process.go countdown: on each outbound FSW match it collects the
// next postSyncDibitsTraffic dibits, then decodes the frame.
func (t *TrafficChannel) Process(dibits []uint8, baseIdx int) int {
	t.matchScratch, _ = t.det.Process(t.matchScratch[:0], dibits, baseIdx)
	matchIdx := 0
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
		for matchIdx < len(t.matchScratch) && t.matchScratch[matchIdx].Index == absPos {
			if !t.matchScratch[matchIdx].Inbound {
				t.remaining = postSyncDibitsTraffic
				t.frame = t.frame[:0]
			}
			matchIdx++
		}
	}
	return baseIdx + len(dibits)
}

// handleFrame slices a collected post-sync traffic frame: LICH (8 wire
// dibits), SACCH (32, currently skipped), Information (144). Only
// RFChTraffic frames with a valid LICH parity are routed to the VCH
// decoder; everything else (control frames, corrupt LICH) is dropped.
func (t *TrafficChannel) handleFrame(frame []uint8) {
	if len(frame) != postSyncDibitsTraffic {
		return
	}
	lichBits := framing.DibitsToBits(frame[0:LICHWireDibits])
	lichByte, _ := DecodeLICHWire(lichBits)
	lich := ParseLICH(lichByte)
	if !lich.ParityOK || lich.RFCh != RFChTraffic {
		return
	}
	infoStart := LICHWireDibits + SACCHDibits
	info := frame[infoStart : infoStart+InfoFieldDibits]
	frames := ExtractVCHFrames(info)
	if len(frames) > 0 && t.sink != nil {
		t.sink(frames)
	}
}

// Reset clears the sync-detection + partial-frame state; call on retune.
func (t *TrafficChannel) Reset() {
	t.det.Reset()
	t.remaining = 0
	t.frame = t.frame[:0]
}
