package conventional

import (
	"context"
	"testing"
	"time"
)

// TestConvScannerUsesExplicitTalkgroupID pins the #1105 talkgroup_id override:
// a channel with TalkgroupID set surfaces its synthetic calls under that fixed
// ID instead of the positional 0x80000000|index default — so the ID survives
// scan-list reordering and any talkgroup_file roster rows stay valid. Fails
// against the pre-override scanner, which always emitted 0x80000001 for the
// second channel.
func TestConvScannerUsesExplicitTalkgroupID(t *testing.T) {
	tuner := &fakeTuner{}
	iq := &fakeIQ{
		tuner: tuner,
		chunks: map[uint32][][]complex64{
			200_000_000: {
				loudChunk(256), loudChunk(256), loudChunk(256),
			},
		},
	}
	eng := &fakeEngine{}
	s, err := New(Options{
		Tuner: tuner, IQ: iq, Engine: eng, Recorder: fakeRecorder{},
		DeviceSerial: "CONV-1",
		SystemName:   "test",
		Channels: []Channel{
			{Label: "A", FrequencyHz: 100_000_000, SquelchDbFS: -10},
			{Label: "B", FrequencyHz: 200_000_000, SquelchDbFS: -10,
				Hangtime: 50 * time.Millisecond, TalkgroupID: 2_147_500_000},
		},
		MinDwellPerChannel: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = s.Run(ctx)

	if eng.startCount() == 0 {
		t.Fatal("squelch never fired (no HandleSyntheticCall)")
	}
	if eng.starts[0].GroupID != 2_147_500_000 {
		t.Errorf("group id = %d, want the explicit talkgroup_id 2147500000 (positional default would be %#x)",
			eng.starts[0].GroupID, 0x80000001)
	}
	if eng.starts[0].GroupLabel != "B" {
		t.Errorf("group label = %q, want B", eng.starts[0].GroupLabel)
	}
}
