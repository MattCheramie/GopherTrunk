package nxdn

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// buildTrafficFrame assembles a full NXDN frame preceded by warm-up
// filler: warmup(8) + FSW(8) + LICH wire(8) + SACCH(32, zeros) +
// Info(144). The warm-up primes the SyncDetector (it needs FSWDibits of
// history before it can match), mirroring a live stream where the FSW is
// never the very first dibit. rfch selects the LICH RF-channel type so
// the same builder makes traffic and control frames.
func buildTrafficFrame(t *testing.T, rfch RFChannelType, info []uint8) []uint8 {
	t.Helper()
	lichByte := AssembleLICH(LICH{RFCh: rfch, FCT: FCTNSACCH, Direction: DirectionOutbound})
	lichWire := framing.BitsToDibits(EncodeLICHWire(lichByte))
	if len(lichWire) != LICHWireDibits {
		t.Fatalf("lich wire = %d dibits, want %d", len(lichWire), LICHWireDibits)
	}
	frame := make([]uint8, 0, FSWDibits+FrameDibits)
	frame = append(frame, make([]uint8, FSWDibits)...) // warm-up filler (primes the detector)
	frame = append(frame, FSWDibitsOutbound...)
	frame = append(frame, lichWire...)
	frame = append(frame, make([]uint8, SACCHDibits)...) // SACCH (skipped)
	frame = append(frame, info...)
	if len(frame) != FSWDibits+FrameDibits {
		t.Fatalf("frame = %d dibits, want %d", len(frame), FSWDibits+FrameDibits)
	}
	return frame
}

func TestTrafficChannelDecodesVCH(t *testing.T) {
	payloads := [4][]byte{mkInfo(11), mkInfo(22), mkInfo(33), mkInfo(44)}
	info := buildVCHInfoDibits(t, payloads)
	frame := buildTrafficFrame(t, RFChTraffic, info)

	var got [][]VCHFrame
	tc := NewTrafficChannel(func(frames []VCHFrame) {
		got = append(got, frames)
	})
	tc.Process(frame, 0)

	if len(got) != 1 {
		t.Fatalf("sink called %d times, want 1", len(got))
	}
	if len(got[0]) != VCHVoiceFramesPerFrame {
		t.Fatalf("got %d voice frames, want %d", len(got[0]), VCHVoiceFramesPerFrame)
	}
	for i, f := range got[0] {
		for j := range payloads[i] {
			if f.Payload[j] != payloads[i][j] {
				t.Fatalf("voice frame %d bit %d: got %d want %d", i, j, f.Payload[j], payloads[i][j])
			}
		}
	}
}

func TestTrafficChannelIgnoresControlFrame(t *testing.T) {
	payloads := [4][]byte{mkInfo(1), mkInfo(2), mkInfo(3), mkInfo(4)}
	info := buildVCHInfoDibits(t, payloads)
	frame := buildTrafficFrame(t, RFChControl, info) // control LICH → not voice

	called := false
	tc := NewTrafficChannel(func(frames []VCHFrame) { called = true })
	tc.Process(frame, 0)

	if called {
		t.Error("sink should not fire for a control-LICH frame")
	}
}

func TestTrafficChannelSurvivesChunkedFeed(t *testing.T) {
	payloads := [4][]byte{mkInfo(5), mkInfo(6), mkInfo(7), mkInfo(8)}
	info := buildVCHInfoDibits(t, payloads)
	frame := buildTrafficFrame(t, RFChTraffic, info)

	var got [][]VCHFrame
	tc := NewTrafficChannel(func(frames []VCHFrame) { got = append(got, frames) })
	// Feed one dibit at a time to exercise the cross-call countdown.
	base := 0
	for i := 0; i < len(frame); i++ {
		tc.Process(frame[i:i+1], base)
		base++
	}
	if len(got) != 1 || len(got[0]) != VCHVoiceFramesPerFrame {
		t.Fatalf("chunked feed decoded %v frame-batches", got)
	}
}
