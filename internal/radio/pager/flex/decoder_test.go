package flex

import (
	"bufio"
	"encoding/hex"
	"os"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// realAirFrames loads testdata/p2000_1600x2_frames.hex: five 1600 bps frames
// off the air (the Dutch P2000 network, sliced from the FLEX recording in
// multimon-ng's test suite). Each line is one frame's 2904 symbols after the
// 64-bit frame sync — sync tail, frame information word, second sync, data —
// one bit per symbol, 1 = the high deviation, packed MSB first.
func realAirFrames(t *testing.T) [][]byte {
	t.Helper()
	f, err := os.Open("testdata/p2000_1600x2_frames.hex")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var frames [][]byte
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 1<<16)
	for sc.Scan() {
		raw, err := hex.DecodeString(sc.Text())
		if err != nil {
			t.Fatal(err)
		}
		syms := make([]byte, 0, len(raw)*8)
		for _, b := range raw {
			for k := 7; k >= 0; k-- {
				syms = append(syms, 3*((b>>uint(k))&1))
			}
		}
		frames = append(frames, syms)
	}
	return frames
}

// syncSymbols is the 64-bit frame sync of a mode, as symbols.
func syncSymbols(m Mode) []byte {
	v := uint64(m.Code)<<48 | uint64(SyncMarker)<<16 | uint64(^m.Code)
	out := make([]byte, 64)
	for i := range out {
		if (v>>uint(63-i))&1 == 0 {
			out[i] = 3
		}
	}
	return out
}

type wantPage struct {
	capcode      uint64
	cycle, frame int
	text         string
}

const proefalarm = "Test: Proefalarm Ochtend Brandweer Veiligheidsregio Rotterdam Rijnmond."

// TestRealAirFrames is the on-air pin for everything below the symbol
// slicer: sync position and bit senses, the frame information word, the block
// interleave, the BCH layout, the address / vector / header layout, and
// fragment reassembly (capcode 1400141's message starts in frame 12 and ends
// in frame 13). The pages are the ones multimon-ng's own test suite expects
// from this recording.
//
// The decoder this replaced read the mode code AFTER the marker, the frame
// information word straight after that with no BCH, and the information bits
// reversed; it decoded none of these frames.
func TestRealAirFrames(t *testing.T) {
	want := []wantPage{
		{1120103, 14, 116, "test"},
		{1180000, 0, 12, "TESTOPROEP MOB"},
		{1400521, 0, 12, proefalarm},
		{1400136, 0, 12, proefalarm},
		{1400141, 0, 13, proefalarm},
		{1400311, 0, 13, proefalarm},
		{1400321, 0, 13, proefalarm},
		{1400511, 0, 13, proefalarm},
		{1123201, 0, 38, "B2 Eindhoven Rit: 105012"},
		{2029572, 0, 40, "A2 (DIA: ja) AMBU 17133 G v Voorneweg 3233TM Oostvoorne OTVOOR bon 125033"},
	}
	d := NewDecoder()
	var got []Message
	for _, frame := range realAirFrames(t) {
		for _, s := range syncSymbols(Mode1600x2) {
			d.PushSymbol(s)
		}
		if d.Hunting() {
			t.Fatal("frame sync not recognised")
		}
		for _, s := range frame {
			got = append(got, d.PushSymbol(s)...)
		}
		if !d.Hunting() {
			t.Fatal("decoder did not finish the frame")
		}
	}
	if len(got) != len(want) {
		t.Fatalf("decoded %d pages, want %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		g := got[i]
		if g.Capcode != w.capcode || g.Cycle != w.cycle || g.Frame != w.frame || g.Text != w.text ||
			g.Type != PageAlphanumeric || g.Mode != Mode1600x2 || g.Phase != 'A' {
			t.Errorf("page %d = %v, want capcode %d at %d.%d %q", i, g, w.capcode, w.cycle, w.frame, w.text)
		}
	}
	st := d.Stats()
	if st.FramesSynced != 5 || st.Frames1600 != 5 || st.FIWErrors != 0 || st.WordsFailed != 0 {
		t.Errorf("stats = %+v, want 5 clean 1600 bps frames", st)
	}
}

// TestRealAirFrameSections pins the two sections the decoder only counts
// through — and the synthesiser therefore has to get right on its own —
// against the air: the 16 bits after the frame sync, and the second sync.
func TestRealAirFrameSections(t *testing.T) {
	frame := realAirFrames(t)[1]
	synth := FrameSymbols(Mode1600x2, 0, 12)[32+64:]
	diff := 0
	for i := 0; i < fiwSkipSymbols; i++ {
		if frame[i] != synth[i] {
			diff++
		}
	}
	if diff != 0 {
		t.Errorf("sync tail: synthesised frame differs from the air in %d of 16 symbols", diff)
	}
	for i := fiwSymbols; i < fiwSymbols+40; i++ {
		if frame[i] != synth[i] {
			diff++
		}
	}
	if diff != 0 {
		t.Errorf("second sync: synthesised frame differs from the air in %d symbols", diff)
	}
	// The on-air frame information word, decoded and re-encoded, is the word
	// that was sent: the encoder's parity bits and even-parity bit match the
	// transmitter's.
	var raw uint32
	for i := fiwSkipSymbols; i < fiwSymbols; i++ {
		raw >>= 1
		if frame[i] > 1 {
			raw |= 0x80000000
		}
	}
	info, errs := framing.FLEXBCHDecode32(raw)
	if errs != 0 || !wordChecksumOK(info) {
		t.Fatalf("on-air frame information word %08x: errs=%d checksum ok=%v", raw, errs, wordChecksumOK(info))
	}
	if cycle, frm := (info>>4)&0xF, (info>>8)&0x7F; cycle != 0 || frm != 12 {
		t.Errorf("on-air frame information word reads %d.%d, want 0.12", cycle, frm)
	}
	if enc := framing.FLEXBCHEncode21(info); enc != raw {
		t.Errorf("re-encoded frame information word = %08x, on air %08x", enc, raw)
	}
	if withChecksum(info) != info {
		t.Errorf("withChecksum changes an on-air word: %06x -> %06x", info, withChecksum(info))
	}
}

// TestRealAirWordsAreCodewords: a random 31-bit word has a zero BCH syndrome
// once in 1024, so on-air frames whose words mostly decode with no correction
// (the rest within the code's two-bit reach) settle the generator and the bit
// layout.
func TestRealAirWordsAreCodewords(t *testing.T) {
	clean, total := 0, 0
	for _, frame := range realAirFrames(t) {
		data := frame[fiwSymbols+40:]
		var words [PhaseWords]uint32
		for c, s := range data {
			if s > 1 {
				words[((c>>5)&^7)|(c&7)] |= 1 << uint((c>>3)&31)
			}
		}
		for _, w := range words {
			total++
			if _, errs := framing.FLEXBCHDecode32(w); errs == 0 {
				clean++
			} else if errs < 0 {
				t.Errorf("on-air word %08x is uncorrectable", w)
			}
		}
	}
	if clean*10 < total*8 {
		t.Errorf("only %d of %d on-air words are error-free codewords", clean, total)
	}
}

func mustPhase(t *testing.T, pages ...Page) [PhaseWords]uint32 {
	t.Helper()
	w, ok := PhaseWith(pages...)
	if !ok {
		t.Fatal("pages do not fit a phase")
	}
	return w
}

func pushAll(d *Decoder, syms []byte) []Message {
	var out []Message
	for _, s := range syms {
		out = append(out, d.PushSymbol(s)...)
	}
	return out
}

// TestEveryModeCarriesItsPhases sends a different page on every phase of
// every mode and expects each back on its own phase: the per-mode symbol
// counts, the 4-level Gray mapping and the A/B ↔ C/D symbol alternation.
func TestEveryModeCarriesItsPhases(t *testing.T) {
	texts := map[byte]string{
		'A': "PHASE A: CODE BLUE ROOM 12",
		'B': "phase b - call 555-0142",
		'C': "Phase C | unit 7 respond",
		'D': "PHASE D ~ test page {4}",
	}
	for _, mode := range Modes {
		var phases [][PhaseWords]uint32
		for _, ph := range mode.Phases() {
			phases = append(phases, mustPhase(t,
				Page{Capcode: 100000 + uint32(ph), Text: texts[ph]},
				Page{Capcode: 200000 + uint32(ph), Text: "555 0100-" + string(ph-'A'+'1'), Numeric: true}))
		}
		syms := FrameSymbols(mode, 3, 77, phases...)
		if secs := float64(32+64+fiwSymbols)/1600 + float64(len(syms)-(32+64+fiwSymbols))/float64(mode.Baud); secs != 1.875 {
			t.Errorf("%d/%d: frame lasts %v s, want 1.875", mode.Baud, mode.Levels, secs)
		}
		for _, inverted := range []bool{false, true} {
			in := syms
			if inverted {
				in = make([]byte, len(syms))
				for i, s := range syms {
					in[i] = 3 - s
				}
			}
			d := NewDecoder()
			got := pushAll(d, in)
			if len(got) != 2*len(mode.Phases()) {
				t.Fatalf("%d/%d inverted=%v: %d pages, want %d: %v",
					mode.Baud, mode.Levels, inverted, len(got), 2*len(mode.Phases()), got)
			}
			for k, ph := range mode.Phases() {
				a, n := got[2*k], got[2*k+1]
				if a.Phase != ph || a.Capcode != 100000+uint64(ph) || a.Text != texts[ph] || a.Type != PageAlphanumeric {
					t.Errorf("%d/%d phase %c alpha = %v", mode.Baud, mode.Levels, ph, a)
				}
				if n.Phase != ph || n.Capcode != 200000+uint64(ph) || n.Text != "555 0100-"+string(ph-'A'+'1') || n.Type != PageNumeric {
					t.Errorf("%d/%d phase %c numeric = %v", mode.Baud, mode.Levels, ph, n)
				}
				if a.Mode != mode || a.Cycle != 3 || a.Frame != 77 {
					t.Errorf("%d/%d phase %c: mode/cycle/frame = %v %d %d", mode.Baud, mode.Levels, ph, a.Mode, a.Cycle, a.Frame)
				}
			}
			if d.Inverted() != inverted {
				t.Errorf("%d/%d: Inverted() = %v, want %v", mode.Baud, mode.Levels, d.Inverted(), inverted)
			}
		}
	}
}

// TestMixedRateFramesBackToBack: a channel that changes mode every frame is
// followed with no state carried over — the "few networks use all of them"
// case.
func TestMixedRateFramesBackToBack(t *testing.T) {
	order := []Mode{Mode1600x2, Mode3200x4, Mode3200x2, Mode1600x4, Mode3200x4, Mode1600x2}
	d := NewDecoder()
	for i, mode := range order {
		got := pushAll(d, FrameSymbols(mode, 0, i, mustPhase(t, Page{Capcode: 4242, Text: "frame"})))
		if len(got) != 1 || got[0].Mode != mode || got[0].Frame != i {
			t.Fatalf("frame %d (%d/%d): got %v", i, mode.Baud, mode.Levels, got)
		}
	}
	st := d.Stats()
	if st.Frames1600 != 2 || st.Frames3200 != 2 || st.Frames6400 != 2 {
		t.Errorf("frames by bit rate = %d/%d/%d, want 2/2/2", st.Frames1600, st.Frames3200, st.Frames6400)
	}
}

func TestModeTable(t *testing.T) {
	for _, tc := range []struct {
		code   uint16
		bps    int
		phases string
	}{
		{0x870C, 1600, "A"}, {0x7B18, 3200, "AC"}, {0xB068, 3200, "AB"},
		{0xDEA0, 6400, "ABCD"}, {0x4C7C, 6400, "ABCD"},
	} {
		reg := uint64(tc.code)<<48 | uint64(SyncMarker)<<16 | uint64(^tc.code)
		m, ok := MatchSync(reg)
		if !ok || m.Code != tc.code || m.BitRate() != tc.bps || string(m.Phases()) != tc.phases {
			t.Errorf("code %04x: got %+v ok=%v, want %d bps phases %s", tc.code, m, ok, tc.bps, tc.phases)
		}
		// Three bit errors in the marker are tolerated, four are not.
		if _, ok := MatchSync(reg ^ 0x7<<20); !ok {
			t.Errorf("code %04x: sync with 3 marker errors rejected", tc.code)
		}
		if _, ok := MatchSync(reg ^ 0xF<<20); ok {
			t.Errorf("code %04x: sync with 4 marker errors accepted", tc.code)
		}
	}
	if _, ok := MatchSync(uint64(0x1234)<<48 | uint64(SyncMarker)<<16 | uint64(^uint16(0x1234))); ok {
		t.Error("unknown mode code accepted")
	}
}

// TestBadFrameInfoWordAbandonsFrame: a sync followed by a frame information
// word that fails its checksum must not be decoded as a frame.
func TestBadFrameInfoWordAbandonsFrame(t *testing.T) {
	syms := FrameSymbols(Mode3200x4, 1, 2, mustPhase(t, Page{Capcode: 9, Text: "nope"}))
	for i := 32 + 64 + fiwSkipSymbols; i < 32+64+fiwSkipSymbols+6; i++ {
		syms[i] = 3 - syms[i] // six bit errors: beyond the BCH
	}
	d := NewDecoder()
	if got := pushAll(d, syms); len(got) != 0 {
		t.Fatalf("decoded %v from a frame with a corrupt information word", got)
	}
	if st := d.Stats(); st.FIWErrors != 1 || st.FramesSynced != 0 {
		t.Errorf("stats = %+v, want one FIW error and no frame", st)
	}
}

// TestBitErrorsAreCorrected: two errors in every word of a 6400 bps frame
// are inside the code's reach.
func TestBitErrorsAreCorrected(t *testing.T) {
	syms := FrameSymbols(Mode3200x4, 0, 0,
		mustPhase(t, Page{Capcode: 1, Text: "alpha"}), mustPhase(t, Page{Capcode: 2, Text: "bravo"}),
		mustPhase(t, Page{Capcode: 3, Text: "charlie"}), mustPhase(t, Page{Capcode: 4, Text: "delta"}))
	head := 32 + 64 + fiwSymbols + 80
	// Symbols 2c and 2c+1 carry bit c of A/B and C/D. Flipping a symbol's
	// sign flips its phase-A (or C) bit; hit bit positions 3 and 17 of every
	// word of those two phases.
	for c := 0; c < PhaseWords*32; c++ {
		if pos := (c >> 3) & 31; pos == 3 || pos == 17 {
			syms[head+2*c] = 3 - syms[head+2*c]
			syms[head+2*c+1] = 3 - syms[head+2*c+1]
		}
	}
	got := pushAll(NewDecoder(), syms)
	if len(got) != 4 {
		t.Fatalf("got %d pages, want 4: %v", len(got), got)
	}
	for i, want := range []string{"alpha", "bravo", "charlie", "delta"} {
		if got[i].Text != want {
			t.Errorf("page %d = %q, want %q", i, got[i].Text, want)
		}
	}
	if got[0].Corrected == 0 || got[1].Corrected != 0 {
		t.Errorf("corrected counts A=%d B=%d, want A>0 and B=0", got[0].Corrected, got[1].Corrected)
	}
}

// TestLongCapcode pins the three long-address sets to values worked from the
// reference's formulas.
func TestLongCapcode(t *testing.T) {
	for _, tc := range []struct {
		w1, w2 uint32
		want   uint64
	}{
		{1, 2097150, 2101249},          // set 1: 1 + 1·32768 + 2068480
		{32768, 2064383, 1075843072},   // set 1, top
		{1, 1966081, 1075843073},       // set 2: 1 + 32769·32768 + 2068480
		{2064383, 1966081, 3223326719}, // set 3: 0 + 98305·32768 + 2068479
	} {
		got, ok := longCapcode(tc.w1, tc.w2)
		if !ok || got != tc.want {
			t.Errorf("longCapcode(%d, %d) = %d ok=%v, want %d", tc.w1, tc.w2, got, ok, tc.want)
		}
	}
	if _, ok := longCapcode(1, 5); ok {
		t.Error("long address with an out-of-set second word accepted")
	}
	if c, ok := longCapcode(2097150, 2031616); ok {
		t.Errorf("long address beyond the largest capcode accepted as %d", c)
	}
}

// TestLongAddressPage: a long address takes two address words and two vector
// words, and its message header rides in the second vector word.
func TestLongAddressPage(t *testing.T) {
	w := idlePhase()
	w[0] = withChecksum(3 << 10) // addresses at 1,2; vectors at 3,4
	w[1], w[2] = 1, 2097150
	w[3] = withChecksum(uint32(PageAlphanumeric)<<4 | 5<<7 | 2<<14) // body at 5, 2 words incl. header
	w[4] = 3 << 11                                                  // header: first fragment
	w[5] = uint32('O')<<7 | uint32('K')<<14
	got := pushAll(NewDecoder(), FrameSymbols(Mode1600x2, 0, 0, w))
	if len(got) != 1 || got[0].Capcode != 2101249 || got[0].Text != "OK" {
		t.Fatalf("got %v, want one page to capcode 2101249 reading OK", got)
	}
}
