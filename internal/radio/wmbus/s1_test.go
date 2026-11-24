package wmbus

import (
	"bytes"
	"math/rand"
	"testing"
)

// The S1 chip encoding: the access code's literal chips (rtl-wmbus
// ACCESS_CODE_S1 = 0x547696) and Manchester with 01 = one, 10 = zero
// (MANCHESTER_IEEE_802_3, and the inverse table encoding_tab_inv_machester
// in mode_s_util.h, which codes nibble 0x0 as 0xAA and 0xF as 0x55).
func TestS1ChipEncodingIsTheReferences(t *testing.T) {
	chips := encodeS1Chips([]byte{0x0F, 0xA5}, 0)
	want := []uint8{
		0, 1, 0, 1, 0, 1, 0, 0, 0, 1, 1, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, // 0x547696
		1, 0, 1, 0, 1, 0, 1, 0, 0, 1, 0, 1, 0, 1, 0, 1, // 0x0F: AA 55
		0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, // 0xA5: 66 99
		0, 1, 0, 1,
	}
	if !bytes.Equal(chips, want) {
		t.Fatalf("chips\n got %v\nwant %v", chips, want)
	}
}

// S1 telegrams decode, at the S1 channel's own centre and from an input
// centred on 868.625 MHz between S1 and T1 / C1, where one stream carries
// both and the T / C receiver keeps decoding beside it.
func TestS1ReceiverDecodes(t *testing.T) {
	data := mustHex(t, "2E4401062020202105077A130040852F2F0F6D4C389300020043840210351F040075012C0B040048D6")
	data[0] = byte(len(data) - 1)
	for _, c := range []struct {
		name          string
		rate          float64
		s1Off, tcOff  float64
		carrierOffset float64
	}{
		{"on 868.3", 1_600_000, 0, 0, 0},
		{"on 868.3, tuner 20 kHz off", 1_000_000, 0, 0, 20_000},
		{"868.625 centre", 1_600_000, -325_000, 325_000, -15_000},
	} {
		var got []Frame
		rx, err := NewReceiver(ReceiverOptions{SampleRateHz: c.rate, OffsetHz: c.tcOff, S1: true, S1OffsetHz: c.s1Off,
			OnFrame: func(f Frame, _ FrameInfo) { got = append(got, f) }})
		if err != nil {
			t.Fatal(err)
		}
		iq := Synthesize(data, SynthOptions{Mode: ModeS1, SampleRateHz: c.rate, OffsetHz: c.s1Off + c.carrierOffset, NoiseRMS: 0.03, Seed: 3})
		tc := Synthesize(data, SynthOptions{Mode: ModeC1, SampleRateHz: c.rate, OffsetHz: c.tcOff + c.carrierOffset, NoiseRMS: 0.03, Seed: 4})
		if c.tcOff != 0 {
			iq = append(iq, tc...)
		}
		for i := 0; i < len(iq); i += 4096 {
			rx.Process(iq[i:min(i+4096, len(iq))])
		}
		rx.Flush()
		modes := map[Mode]int{}
		for _, f := range got {
			if f.CRCOK && bytes.Equal(f.Data, data) {
				modes[f.Mode]++
			}
		}
		if modes[ModeS1] != 1 || (c.tcOff != 0 && modes[ModeC1] != 1) {
			t.Errorf("%s: decoded %v from %d frames; stats %+v", c.name, modes, len(got), rx.Stats().S1Framer)
		}
	}
}

// Weak S1 telegrams: the decode cliff, and a CRC repair on soft
// Manchester decisions.
func TestS1ReceiverNoise(t *testing.T) {
	data := mustHex(t, "1E44010620202021050778046D2E121032")
	data[0] = byte(len(data) - 1)
	ok := 0
	for seed := int64(0); seed < 20; seed++ {
		n := 0
		rx, _ := NewReceiver(ReceiverOptions{SampleRateHz: 1_000_000, S1: true, OnFrame: func(f Frame, _ FrameInfo) {
			if f.Mode == ModeS1 && f.CRCOK && bytes.Equal(f.Data, data) {
				n++
			}
		}})
		iq := Synthesize(data, SynthOptions{Mode: ModeS1, SampleRateHz: 1_000_000, NoiseRMS: 0.5, Seed: seed})
		rx.Process(iq)
		rx.Flush()
		ok += n
	}
	if ok < 15 {
		t.Errorf("only %d of 20 S1 telegrams at noise 0.5 (-3 dB in 1 MHz, ~+8 dB in the channel)", ok)
	}
}

// Noise alone never frames an S1 telegram that passes its CRC.
func TestS1ReceiverQuietOnNoise(t *testing.T) {
	var frames int
	rx, _ := NewReceiver(ReceiverOptions{SampleRateHz: 1_000_000, S1: true, OnFrame: func(f Frame, _ FrameInfo) {
		if f.Mode == ModeS1 && f.CRCOK {
			frames++
		}
	}})
	r := rand.New(rand.NewSource(1))
	iq := make([]complex64, 1_000_000)
	for i := range iq {
		iq[i] = complex(float32(r.NormFloat64()*0.1), float32(r.NormFloat64()*0.1))
	}
	rx.Process(iq)
	rx.Flush()
	if frames != 0 {
		t.Errorf("%d CRC-valid S1 frames from noise", frames)
	}
}

func TestS1RejectsAnOffsetThatDoesNotFit(t *testing.T) {
	if _, err := NewReceiver(ReceiverOptions{SampleRateHz: 1_000_000, S1: true, S1OffsetHz: -650_000}); err == nil {
		t.Error("S1 at -650 kHz accepted at 1 MS/s")
	}
}
