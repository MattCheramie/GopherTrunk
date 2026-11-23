package dpmr

import (
	"math/rand"
	"testing"
)

// testBeacon builds a beacon downlink: random dibits, then FS1 and n
// framesets alternating an Aloha and a Goto Channel, each with SYScast1
// (the system code) and the calling party, then random dibits.
func testBeacon(r *rand.Rand, n int) []uint8 {
	var d []uint8
	random := func(k int) {
		for i := 0; i < k; i++ {
			d = append(d, uint8(r.Intn(4)))
		}
	}
	random(500)
	d = append(d, AirFS1Dibits()...)
	sy1 := EncodeSYScast1(SYScast1{SystemCode: 0x0ABC})
	for i := 0; i < n; i++ {
		var m BeaconMessage
		var sy [3]uint32
		if i%2 == 0 {
			m = EncodeAloha(Aloha{MSID: 0xFFFFFF, SystemCode: 0x0ABC, Active: true})
			sy = [3]uint32{sy1, EncodeSYScast2(SYTypeFrameCounter, uint16(i)), EncodeSYScast2(SYTypeFrameCounter, uint16(i))}
		} else {
			m = EncodeGotoChannel(GotoChannel{Called: 0x02C00A, RxField: 14722, TxField: 14722, Band: 5, Mode: AirModeVoice})
			a, b := EncodeCallingParty(0x1B91FD)
			sy = [3]uint32{sy1, a, b}
		}
		d = append(d, EncodeBeaconFrameset(m, BeaconChannelCode0, sy)...)
	}
	random(500)
	return d
}

func frameBeacon(d []uint8, chunk int) (*BeaconFramer, []BeaconFrameset) {
	var got []BeaconFrameset
	f := NewBeaconFramer(func(fs BeaconFrameset) { got = append(got, fs) })
	base := 0
	for i := 0; i < len(d); i += chunk {
		base = f.Process(d[i:min(i+chunk, len(d))], base)
	}
	return f, got
}

// A clean beacon frames every frameset, messages and SYScasts, from any
// chunking.
func TestBeaconFramerFramesACleanBeacon(t *testing.T) {
	d := testBeacon(rand.New(rand.NewSource(1)), 20)
	for _, chunk := range []int{1, 37, 264, 5000} {
		f, got := frameBeacon(d, chunk)
		st := f.Stats()
		if st.Locks != 1 || st.Messages < 20 {
			t.Errorf("chunk %d: locks %d messages %d", chunk, st.Locks, st.Messages)
			continue
		}
		var alohas, gotos, callers int
		for _, fs := range got {
			if a, ok := fs.Message.Aloha(); fs.MessageOK && ok && a.SystemCode == 0x0ABC {
				alohas++
			}
			if g, ok := fs.Message.GotoChannel(); fs.MessageOK && ok && g.Called == 0x02C00A {
				gotos++
				if id, ok := CallingParty(fs.SYScast[1].Bits, fs.SYScast[2].Bits); ok && id == 0x1B91FD {
					callers++
				}
			}
			if fs.MessageOK && fs.ChannelCode != BeaconChannelCode0 {
				t.Errorf("chunk %d: channel code %06X", chunk, fs.ChannelCode)
			}
		}
		if alohas != 10 || gotos != 10 || callers != 10 {
			t.Errorf("chunk %d: %d alohas, %d gotos, %d calling parties; want 10 each", chunk, alohas, gotos, callers)
		}
	}
}

// An inverted spectrum flips every sign bit; the framer locks on the
// inverted FS1 and reads the same messages.
func TestBeaconFramerInverted(t *testing.T) {
	d := testBeacon(rand.New(rand.NewSource(2)), 10)
	airInvert(d)
	f, _ := frameBeacon(d, 400)
	if st := f.Stats(); st.Locks != 1 || !st.Inverted || st.Messages < 10 {
		t.Fatalf("locks %d inverted %v messages %d", st.Locks, st.Inverted, st.Messages)
	}
}

// Joined anywhere inside the beacon, the framer is locked within two
// framesets (220 ms).
func TestBeaconFramerJoinsMidStream(t *testing.T) {
	d := testBeacon(rand.New(rand.NewSource(3)), 12)
	for cut := 500; cut < 500+BeaconFramesetDibits; cut += 29 {
		var first = -1
		f := NewBeaconFramer(nil)
		base := 0
		for i := cut; i < len(d); i += 12 {
			base = f.Process(d[i:min(i+12, len(d))], base)
			if f.Locked() && first < 0 {
				first = i - cut
			}
		}
		if first < 0 || first > 2*BeaconFramesetDibits+BeaconFS1Dibits {
			t.Errorf("joined %d dibits in: locked after %d dibits", cut-500, first)
		}
	}
}

// A frameset whose SYScast slot carries appended data, with no FS1 before
// the next message, keeps the lock.
func TestBeaconFramerKeepsLockThroughAppendedData(t *testing.T) {
	r := rand.New(rand.NewSource(4))
	var d []uint8
	for i := 0; i < 300; i++ {
		d = append(d, uint8(r.Intn(4)))
	}
	d = append(d, AirFS1Dibits()...)
	aloha := EncodeAloha(Aloha{SystemCode: 7})
	sy := [3]uint32{EncodeSYScast1(SYScast1{SystemCode: 7}), 0, 0}
	var app [72]uint8
	for i := range app {
		app[i] = uint8(r.Intn(2))
	}
	for i := 0; i < 6; i++ {
		if i == 2 || i == 3 {
			d = append(d, EncodeBeaconAppendedFrameset(aloha, BeaconChannelCode0, app)...)
		} else {
			d = append(d, EncodeBeaconFrameset(aloha, BeaconChannelCode0, sy)...)
		}
	}
	f, got := frameBeacon(d, 100)
	st := f.Stats()
	if st.Locks != 1 || st.Drops != 0 || st.AppendedFrames != 2 {
		t.Fatalf("locks %d drops %d appended %d", st.Locks, st.Drops, st.AppendedFrames)
	}
	if !got[2].AppendedOK || got[2].Appended != app {
		t.Fatal("appended data not returned")
	}
}

// A stream that loses dibits moves the grid; the framer drops the lost
// grid and locks onto the new one.
func TestBeaconFramerRelocksAfterLostDibits(t *testing.T) {
	d := testBeacon(rand.New(rand.NewSource(5)), 30)
	cut := 500 + 24 + 10*BeaconFramesetDibits + 100
	d = append(d[:cut:cut], d[cut+37:]...) // 37 dibits lost mid-frameset
	f, got := frameBeacon(d, 64)
	st := f.Stats()
	if st.Locks != 2 || st.Drops < 1 {
		t.Fatalf("locks %d drops %d", st.Locks, st.Drops)
	}
	ok := 0
	for _, fs := range got {
		if fs.MessageOK {
			ok++
		}
	}
	// One frameset is cut by the hole, and one may go to the re-hunt.
	if ok < 27 {
		t.Fatalf("%d of 30 messages framed across the hole", ok)
	}
}

// Random dibits never lock a framer.
func TestBeaconFramerIgnoresNoise(t *testing.T) {
	r := rand.New(rand.NewSource(6))
	d := make([]uint8, 2400*120) // two minutes
	for i := range d {
		d[i] = uint8(r.Intn(4))
	}
	if f, _ := frameBeacon(d, 4096); f.Stats().Locks != 0 {
		t.Fatalf("%d locks on noise", f.Stats().Locks)
	}
}
