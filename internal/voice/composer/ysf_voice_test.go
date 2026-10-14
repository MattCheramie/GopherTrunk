package composer

import (
	"context"
	"encoding/hex"
	"math/rand"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/ysf"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// On-air FICH bytes from MMDVMHost's encoder (see ysf/airframe_test.go):
// a header, a V/D mode 2 communications frame, a terminator, and a V/D
// mode 1 communications frame.
const (
	ysfFICHHeader = "0037b4ad9d002d9f9071003fe1e3ac0025d16449000b81cc5f"
	ysfFICHVD2    = "3127cca4bacfbe90326f4b9d70efea4fd5207d0684c823065b"
	ysfFICHTerm   = "c397b4716241cd9ca0ea4c0fe395858a65d1e502c3cb8161af"
	ysfFICHVD1    = "3d0ff81103c5dc8c9b524857becc50418657352d83189b987b"
)

func ysfFrameDibits(t *testing.T, fichHex string, payload []uint8) []uint8 {
	t.Helper()
	b, err := hex.DecodeString(fichHex)
	if err != nil {
		t.Fatal(err)
	}
	out := append([]uint8(nil), ysf.FSWPattern...)
	for _, x := range b {
		for s := 6; s >= 0; s -= 2 {
			out = append(out, (x>>uint(s))&3)
		}
	}
	return append(out, payload...)
}

// TestComposerYSFConventionalCallRecordsVD2Voice drives issue #1261's YSF
// scan-list path end to end: a modulated System Fusion transmission (header,
// V/D mode 2 voice frames, terminator) granted as a conventional "ysf" call
// must reach the recorder as the transmitted AMBE+2 payloads, in order.
// Before #1261 there was no YSF voice chain: the composer bypassed the call
// and recorded nothing.
func TestComposerYSFConventionalCallRecordsVD2Voice(t *testing.T) {
	const (
		sampleRate = 48_000.0
		deviation  = 1800.0
		vd2Frames  = 8
	)
	r := rand.New(rand.NewSource(3))
	// YSF is RRC-shaped (α 0.20) like DMR, so the fixtures use the generic
	// C4FM modulator, not ModulateP25C4FM's P25 pulse shape.
	lead := make([]uint8, 480) // carrier settling before the header
	for i := range lead {
		lead[i] = uint8(r.Intn(4))
	}
	dibits := append([]uint8(nil), lead...)
	dibits = append(dibits, ysfFrameDibits(t, ysfFICHHeader, make([]uint8, ysf.PayloadDibits))...)
	var want [][]byte
	for f := 0; f < vd2Frames; f++ {
		var infos [5][]byte
		for j := range infos {
			infos[j] = make([]byte, ysf.VD2InfoBits)
			for k := range infos[j] {
				infos[j][k] = uint8(r.Intn(2))
			}
			want = append(want, framing.PackBitsMSB(infos[j]))
		}
		dibits = append(dibits, ysfFrameDibits(t, ysfFICHVD2, ysf.EncodeVD2Voice(infos))...)
	}
	dibits = append(dibits, ysfFrameDibits(t, ysfFICHTerm, make([]uint8, ysf.PayloadDibits))...)
	dibits = append(dibits, lead...)
	iq := demod.ModulateC4FM(dibits, 10, 8, 0.20, sampleRate, deviation)

	src := newFakeSource()
	sink := &recordingSink{}
	bus := events.NewBus(64)
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"CONV-1": src}},
		Sink:          sink,
		Engine:        &fakeEngine{},
		IQSampleRate:  uint32(sampleRate),
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	defer c.Close()
	defer bus.Close()

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant: trunking.Grant{
				System: "scanner", Protocol: "ysf", Conventional: true,
				GroupID: 0x80000004, GroupLabel: "C4FM Rptr", FrequencyHz: 145_500_000,
			},
			DeviceSerial: "CONV-1",
			StartedAt:    time.Now().UTC(),
		},
	})
	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	// The receiver's clock and slicer settle during the lead and header,
	// so allow the first voice frame or two to be lost; the rest must
	// arrive exactly, in order.
	minFrames := (vd2Frames - 2) * 5
	waitFor(t, 8*time.Second, func() bool { return len(sink.rawFrames("CONV-1")) >= minFrames })
	got := sink.rawFrames("CONV-1")
	start := -1
	for i, w := range want {
		if string(w) == string(got[0]) {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("first recorded frame matches no transmitted AMBE+2 payload (got %d frames)", len(got))
	}
	for i := range got {
		if start+i >= len(want) {
			t.Fatalf("recorded %d frames past the end of the transmission", len(got)-i)
		}
		if string(got[i]) != string(want[start+i]) {
			t.Fatalf("frame %d differs from transmitted frame %d", i, start+i)
		}
	}
}

// A V/D mode 1 call is recognised from its FICH and not decoded as V/D
// mode 2: recording its payload as VD2 voice would render noise.
func TestComposerYSFSkipsVDMode1(t *testing.T) {
	const sampleRate = 48_000.0
	r := rand.New(rand.NewSource(5))
	dibits := make([]uint8, 480)
	for i := range dibits {
		dibits[i] = uint8(r.Intn(4))
	}
	dibits = append(dibits, ysfFrameDibits(t, ysfFICHHeader, make([]uint8, ysf.PayloadDibits))...)
	for f := 0; f < 6; f++ {
		payload := make([]uint8, ysf.PayloadDibits)
		for i := range payload {
			payload[i] = uint8(r.Intn(4))
		}
		dibits = append(dibits, ysfFrameDibits(t, ysfFICHVD1, payload)...)
	}
	iq := demod.ModulateC4FM(dibits, 10, 8, 0.20, sampleRate, 1800)

	src := newFakeSource()
	sink := &recordingSink{}
	bus := events.NewBus(64)
	c, err := New(Options{
		Bus: bus, Devices: &fakeDevices{src: map[string]IQSource{"CONV-1": src}},
		Sink: sink, Engine: &fakeEngine{}, IQSampleRate: uint32(sampleRate),
		PCMSampleRate: 8000, TouchInterval: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	defer c.Close()
	defer bus.Close()
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: trunking.CallStart{
		Grant:        trunking.Grant{System: "scanner", Protocol: "ysf", Conventional: true, GroupID: 0x80000004},
		DeviceSerial: "CONV-1", StartedAt: time.Now().UTC(),
	}})
	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)
	time.Sleep(1500 * time.Millisecond)
	if n := len(sink.rawFrames("CONV-1")); n != 0 {
		t.Errorf("recorded %d frames from a V/D mode 1 call, want 0", n)
	}
}
