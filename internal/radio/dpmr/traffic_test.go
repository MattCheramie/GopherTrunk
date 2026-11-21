package dpmr

import (
	"math/rand"
	"testing"

	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
)

// airTx assembles a dPMR transmission the way a radio sends it (TS 102 658
// clauses 4.2.2 and 5.2.1.1), out of pieces that do not come from the code
// under test: the sync words as the standard prints them, control channels
// the reference decoder has decoded (airCCHVectors), and vocoder frames
// from the DMR package's AMBE encoder.
type airTx struct {
	t       *testing.T
	r       *rand.Rand
	dibits  []uint8
	voice   [][]byte // the 49-bit payloads sent, in order
	cch     [4]int   // index into airCCHVectors for frames 1..4
	code    uint32   // channel code
	silence bool     // send the vocoder's silence frame in every TCH block
}

const (
	airTxFS1 = "0101 0111 1111 1111 0101 1111 0111 0101 1101 0101 0111 0111"
	airTxFS2 = "0101 1111 1111 0111 0111 1101"
	airTxFS3 = "0111 1101 1101 1111 1111 0101"
)

func newAirTx(t *testing.T, seed int64) *airTx {
	// A voice call from 0xABCDEF to 0x123456.
	return &airTx{t: t, r: rand.New(rand.NewSource(seed)), cch: [4]int{0, 1, 2, 3}, code: 0x575F77}
}

func (tx *airTx) random(n int) {
	for i := 0; i < n; i++ {
		tx.dibits = append(tx.dibits, uint8(tx.r.Intn(4)))
	}
}

// header sends the preamble and the header frame: P | FS1 | HI0 | CC | HI1.
// The header information is not decoded by the framer, so it is filler.
func (tx *airTx) header() {
	for i := 0; i < 9; i++ { // 72 bits of 5F
		tx.dibits = append(tx.dibits, 1, 1, 3, 3)
	}
	tx.dibits = append(tx.dibits, dibitsOfBinary(airTxFS1)...)
	tx.random(60)
	tx.dibits = append(tx.dibits, wordToDibits(uint64(tx.code), 12)...)
	tx.random(60)
}

func (tx *airTx) tch() {
	info := make([]byte, 49)
	if tx.silence {
		for i, c := range ambeSilencePayload {
			info[i] = byte(c - '0')
		}
	} else {
		for i := range info {
			info[i] = byte(tx.r.Intn(2))
		}
	}
	bits, err := dmrvoice.EncodeAMBEFrame(info)
	if err != nil {
		tx.t.Fatal(err)
	}
	for i := 0; i < 36; i++ {
		tx.dibits = append(tx.dibits, bits[2*i]<<1|bits[2*i+1])
	}
	tx.voice = append(tx.voice, info)
}

// superframe sends four payload frames.
func (tx *airTx) superframe() {
	for f := 0; f < 4; f++ {
		if f%2 == 0 {
			tx.dibits = append(tx.dibits, dibitsOfBinary(airTxFS2)...)
		} else {
			tx.dibits = append(tx.dibits, wordToDibits(uint64(tx.code), 12)...)
		}
		tx.dibits = append(tx.dibits, dibitsOf(airCCHVectors[tx.cch[f]].onAir)...)
		for i := 0; i < 4; i++ {
			tx.tch()
		}
	}
}

// end sends the end frame: FS3 | END.
func (tx *airTx) end() {
	tx.dibits = append(tx.dibits, dibitsOfBinary(airTxFS3)...)
	tx.random(36)
}

// rx runs dibits through a TrafficChannel in chunks and collects what it
// hands out.
type airRx struct {
	tc   *TrafficChannel
	got  [][]byte
	errs int
	ends int
}

func runAirRx(dibits []uint8, chunk int) *airRx {
	rx := &airRx{}
	rx.tc = NewTrafficChannel(func(frames []TCHFrame) {
		for _, f := range frames {
			rx.got = append(rx.got, f.Payload)
			rx.errs += f.Errors
		}
	})
	rx.tc.OnEnd = func() { rx.ends++ }
	base := 0
	for i := 0; i < len(dibits); i += chunk {
		j := i + chunk
		if j > len(dibits) {
			j = len(dibits)
		}
		base = rx.tc.Process(dibits[i:j], base)
	}
	return rx
}

func samePayloads(t *testing.T, got, want [][]byte) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%d vocoder frames, want %d", len(got), len(want))
	}
	for i := range want {
		if bitString(got[i]) != bitString(want[i]) {
			t.Fatalf("vocoder frame %d differs", i)
		}
	}
}

// TestTrafficChannelDecodesAVoiceTransmission: noise, a header, three
// superframes, an end frame, noise. Every vocoder frame comes out, in
// order, whatever the chunking.
func TestTrafficChannelDecodesAVoiceTransmission(t *testing.T) {
	tx := newAirTx(t, 1)
	tx.random(300)
	tx.header()
	for i := 0; i < 3; i++ {
		tx.superframe()
	}
	tx.end()
	tx.random(600)

	for _, chunk := range []int{1, 7, 100, 4096, len(tx.dibits)} {
		rx := runAirRx(tx.dibits, chunk)
		samePayloads(t, rx.got, tx.voice)
		if rx.errs != 0 {
			t.Errorf("chunk %d: %d Golay corrections on a clean signal", chunk, rx.errs)
		}
		st := rx.tc.Stats()
		if st.Locks != 1 || st.Drops != 0 || st.EndFrames != 1 || rx.ends != 1 {
			t.Errorf("chunk %d: locks %d drops %d end frames %d OnEnd %d, want 1 0 1 1",
				chunk, st.Locks, st.Drops, st.EndFrames, rx.ends)
		}
		if st.Frames != 12 || st.CCHValid != 12 || st.CCHInvalid != 0 || st.VoiceFrames != 48 {
			t.Errorf("chunk %d: frames %d CCH %d/%d voice %d", chunk, st.Frames, st.CCHValid, st.CCHInvalid, st.VoiceFrames)
		}
		if !st.HaveCalled || st.CalledID != 0x123456 || !st.HaveCalling || st.CallingID != 0xABCDEF {
			t.Errorf("chunk %d: called %06X (%v) calling %06X (%v)", chunk, st.CalledID, st.HaveCalled, st.CallingID, st.HaveCalling)
		}
		if !st.HaveChannelCode || st.ChannelCode != 0x575F77 || st.Inverted {
			t.Errorf("chunk %d: channel code %06X (%v) inverted %v", chunk, st.ChannelCode, st.HaveChannelCode, st.Inverted)
		}
		if rx.tc.Locked() {
			t.Errorf("chunk %d: still locked after the end frame", chunk)
		}
	}
}

// TestTrafficChannelSilenceFrames: a transmission of nothing but the
// vocoder's silence frame, which is what a radio sends from PTT release to
// the end of the superframe.
func TestTrafficChannelSilenceFrames(t *testing.T) {
	tx := newAirTx(t, 2)
	tx.silence = true
	tx.header()
	tx.superframe()
	tx.end()
	tx.random(450)
	rx := runAirRx(tx.dibits, 512)
	samePayloads(t, rx.got, tx.voice)
	if rx.errs != 0 {
		t.Errorf("%d Golay corrections", rx.errs)
	}
}

// TestTrafficChannelInvertedSpectrum: every sign bit flipped. FS1 turns
// into FS4 and says nothing; FS2 settles it.
func TestTrafficChannelInvertedSpectrum(t *testing.T) {
	tx := newAirTx(t, 3)
	tx.random(100)
	tx.header()
	tx.superframe()
	tx.superframe()
	tx.end()
	tx.random(450)
	inv := append([]uint8(nil), tx.dibits...)
	for i := range inv {
		inv[i] ^= 2
	}
	rx := runAirRx(inv, 333)
	samePayloads(t, rx.got, tx.voice)
	st := rx.tc.Stats()
	if !st.Inverted || st.EndFrames != 1 || st.ChannelCode != 0x575F77 {
		t.Errorf("inverted %v end frames %d channel code %06X", st.Inverted, st.EndFrames, st.ChannelCode)
	}
}

// TestTrafficChannelLateEntry: the receiver arrives in the middle of a
// transmission and picks it up at the next FS2.
func TestTrafficChannelLateEntry(t *testing.T) {
	tx := newAirTx(t, 4)
	tx.header()
	for i := 0; i < 4; i++ {
		tx.superframe()
	}
	tx.end()
	tx.random(450)
	// Cut in 100 dibits into the first superframe's first frame: the first
	// pair is lost, the other seven are whole.
	const headerDibits = 36 + 24 + 60 + 12 + 60
	rx := runAirRx(tx.dibits[headerDibits+100:], 256)
	samePayloads(t, rx.got, tx.voice[8:])
	if st := rx.tc.Stats(); st.Locks != 1 || st.EndFrames != 1 {
		t.Errorf("locks %d end frames %d", st.Locks, st.EndFrames)
	}
}

// TestTrafficChannelNeverLocksOnNoise: a quarter of an hour of random
// dibits. FS2 matches by chance several times in that; nothing may come of
// it.
func TestTrafficChannelNeverLocksOnNoise(t *testing.T) {
	r := rand.New(rand.NewSource(5))
	noise := make([]uint8, 2_160_000) // 15 minutes at 2400 symbols a second
	for i := range noise {
		noise[i] = uint8(r.Intn(4))
	}
	chance := 0
	for i := 0; i+AirSyncDibits <= len(noise); i++ {
		if p, _ := findFS2(noise[i : i+AirSyncDibits]); p == 0 {
			chance++
		}
	}
	if chance == 0 {
		t.Fatalf("the noise never matched FS2: the test proves nothing")
	}
	rx := runAirRx(noise, 4096)
	st := rx.tc.Stats()
	if st.Locks != 0 || len(rx.got) != 0 || rx.ends != 0 {
		t.Fatalf("on noise: %d chance sync matches, %d locks, %d vocoder frames, %d end frames",
			chance, st.Locks, len(rx.got), rx.ends)
	}
	t.Logf("%d chance FS2 matches in 15 minutes of noise, no lock", chance)
}

// TestTrafficChannelDataTransmissionYieldsNoVoice: the control channel
// names a data mode, so the TCH blocks are not vocoder frames.
func TestTrafficChannelDataTransmissionYieldsNoVoice(t *testing.T) {
	tx := newAirTx(t, 6)
	tx.cch = [4]int{6, 7, 6, 7} // mode 2: data without FEC
	tx.header()
	tx.superframe()
	tx.superframe()
	tx.end()
	tx.random(450)
	rx := runAirRx(tx.dibits, 500)
	st := rx.tc.Stats()
	if len(rx.got) != 0 || st.VoiceFrames != 0 {
		t.Fatalf("%d vocoder frames out of a data transmission", len(rx.got))
	}
	if st.Locks != 1 || st.DataFrames != 8 || st.EndFrames != 1 {
		t.Errorf("locks %d data frames %d end frames %d", st.Locks, st.DataFrames, st.EndFrames)
	}
}

// TestTrafficChannelReanchorsAfterLostDibits: the stream loses a few
// dibits mid-transmission (dropped IQ), which brings every later sync word
// early. The framer must find the grid again at once instead of cutting on
// the old one; only vocoder frames between the gap and the next frame
// boundary may suffer.
func TestTrafficChannelReanchorsAfterLostDibits(t *testing.T) {
	const headerDibits = 36 + 24 + 60 + 12 + 60
	cases := []struct {
		name string
		at   int // dibit of the third pair at which 5 dibits are lost
		good int // vocoder frames of that pair that still come out whole
		out  int // vocoder frames handed out for that pair
	}{
		// Inside the second vocoder frame of the first payload frame: the
		// first is whole, the rest of that frame is cut on the old grid,
		// and the second payload frame is given up.
		{"first frame", 100, 1, 4},
		// Inside the second vocoder frame of the second payload frame.
		{"second frame", AirFrameDibits + 100, 5, 8},
	}
	for _, c := range cases {
		tx := newAirTx(t, 7)
		tx.header()
		for i := 0; i < 6; i++ {
			tx.superframe()
		}
		tx.end()
		tx.random(450)
		cut := headerDibits + 2*airPairDibits + c.at
		holed := append(append([]uint8(nil), tx.dibits[:cut]...), tx.dibits[cut+5:]...)
		rx := runAirRx(holed, 300)

		st := rx.tc.Stats()
		if st.Drops != 1 || st.Locks != 2 || st.EndFrames != 1 {
			t.Fatalf("%s: drops %d locks %d end frames %d, want 1 2 1", c.name, st.Drops, st.Locks, st.EndFrames)
		}
		if want := 16 + c.out + 72; len(rx.got) != want {
			t.Fatalf("%s: %d vocoder frames, want %d", c.name, len(rx.got), want)
		}
		// Everything before the gap, and every pair after the damaged one.
		samePayloads(t, rx.got[:16+c.good], tx.voice[:16+c.good])
		samePayloads(t, rx.got[len(rx.got)-72:], tx.voice[24:])
	}
}

// TestTrafficChannelRidesThroughBitErrors: 3 % of the dibits wrong. The
// framer keeps its lock from the first FS2 to the end frame, and the
// vocoder frames come out with their FEC doing the work.
func TestTrafficChannelRidesThroughBitErrors(t *testing.T) {
	tx := newAirTx(t, 8)
	tx.random(200)
	tx.header()
	for i := 0; i < 10; i++ {
		tx.superframe()
	}
	tx.end()
	tx.random(450)
	r := rand.New(rand.NewSource(9))
	noisy := append([]uint8(nil), tx.dibits...)
	for i := range noisy {
		if r.Float64() < 0.03 {
			noisy[i] ^= uint8(1 + r.Intn(3))
		}
	}
	rx := runAirRx(noisy, 1000)
	st := rx.tc.Stats()
	if st.Frames < 36 {
		t.Fatalf("%d of 40 payload frames accepted", st.Frames)
	}
	exact := 0
	for i := range rx.got {
		for _, w := range tx.voice {
			if bitString(rx.got[i]) == bitString(w) {
				exact++
				break
			}
		}
	}
	// A frame survives intact when its 23 unprotected bits are untouched
	// and each Golay word has at most three errors: about a third of them
	// at this error rate.
	if exact < len(tx.voice)/5 {
		t.Errorf("%d of %d vocoder frames recovered exactly", exact, len(tx.voice))
	}
	t.Logf("3 %% dibit errors: %d locks, %d drops, %d/40 frames, CCH %d good %d bad, %d/%d vocoder frames exact",
		st.Locks, st.Drops, st.Frames, st.CCHValid, st.CCHInvalid, exact, len(tx.voice))
}

// TestTrafficChannelIgnoresTheLegacyModel: the frames the old traffic
// adapter was built around (a 48-bit sync word from sync.go, a 48-bit
// control field, the payload) are not dPMR and must not decode.
func TestTrafficChannelIgnoresTheLegacyModel(t *testing.T) {
	tx := newAirTx(t, 10)
	var d []uint8
	for i := 0; i < 20; i++ {
		d = append(d, FS1Dibits()...)
		d = append(d, make([]uint8, 24)...)
		tx.dibits = tx.dibits[:0]
		for j := 0; j < 4; j++ {
			tx.tch()
		}
		d = append(d, tx.dibits...)
	}
	rx := runAirRx(d, 512)
	if st := rx.tc.Stats(); st.Locks != 0 || len(rx.got) != 0 {
		t.Fatalf("legacy-model frames gave %d locks and %d vocoder frames", st.Locks, len(rx.got))
	}
}

func TestTrafficChannelReset(t *testing.T) {
	tx := newAirTx(t, 11)
	tx.header()
	tx.superframe()
	rx := runAirRx(tx.dibits, 4096)
	if !rx.tc.Locked() {
		t.Fatalf("not locked after a superframe")
	}
	rx.tc.Reset()
	if rx.tc.Locked() {
		t.Fatalf("still locked after Reset")
	}
}
