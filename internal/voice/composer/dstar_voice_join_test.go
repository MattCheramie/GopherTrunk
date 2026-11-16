package composer

import (
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dstar"
)

// dstarJoinFixture is 3 s of a D-STAR transmission under way (no header):
// DV frames of distinct random AMBE payloads with the slow-data sync every
// 21st, GMSK at 48 kHz with a little noise. want maps each payload to its
// frame number.
func dstarJoinFixture(t *testing.T, invert bool) (iq []complex64, want map[string]int) {
	t.Helper()
	r := rand.New(rand.NewSource(71))
	want = map[string]int{}
	var bits []uint8
	for f := 0; f < 150; f++ {
		info := make([]byte, 49)
		for i := range info {
			info[i] = byte(r.Intn(2))
		}
		want[string(info)] = f
		voice, err := dstar.EncodeDVVoiceBits(info)
		if err != nil {
			t.Fatal(err)
		}
		bits = append(bits, dstar.EncodeDVFrame(voice, f, [3]byte{byte(r.Intn(256)), 'G', 'T'})...)
	}
	if invert {
		for i := range bits {
			bits[i] ^= 1
		}
	}
	iq = demod.ModulateGFSK(bits, 10, 4, 0.5, 48_000, 1200)
	for i := range iq {
		iq[i] = iq[i]*0.5 + complex(float32(r.NormFloat64()*0.01), float32(r.NormFloat64()*0.01))
	}
	return iq, want
}

// dstarJoinResult is what a voice receiver made of a transmission joined
// at sample start: how long after the join the first decoded frame was on
// the air, and the frame numbers it emitted, in order.
type dstarJoinResult struct {
	lostSeconds float64
	frames      []int
	unknown     int
}

func runDStarJoin(iq []complex64, want map[string]int, start, chunk int, lanes bool) dstarJoinResult {
	var res dstarJoinResult
	emit := func(f dstar.DVFrame) {
		if n, ok := want[string(f.Payload)]; ok {
			res.frames = append(res.frames, n)
		} else {
			res.unknown++
		}
	}
	var process func([]complex64)
	if lanes {
		process = newDStarVoiceLanes(48_000, emit).Process
	} else {
		vc := dstar.NewVoiceChannel(emit)
		process = newDStarVoiceReceiver(48_000, func(b []byte, base int) { vc.Process(b, base) }).Process
	}
	in := iq[start:]
	for i := 0; i < len(in); i += chunk {
		process(in[i:min(i+chunk, len(in))])
	}
	res.lostSeconds = 3
	if len(res.frames) > 0 {
		res.lostSeconds = float64(res.frames[0]*960-start) / 48_000
	}
	return res
}

// Joining a transmission under way must not cost a second of speech. One
// receiver alone hangs for 1-2 sync cycles from the start phase half a
// symbol off its timing loop's stable point; the chain's receiver pair
// decodes from every phase (issue #1261).
func TestDStarVoiceLanesJoinFromEveryPhase(t *testing.T) {
	for _, inv := range []bool{false, true} {
		iq, want := dstarJoinFixture(t, inv)
		worstOne, worstPair := 0.0, 0.0
		unkOne, unkPair := 0, 0
		for off := 0; off < 40; off++ {
			start := off * 1013 // ~21 ms further in each time, every sub-symbol phase four times
			one := runDStarJoin(iq, want, start, 4096, false)
			pair := runDStarJoin(iq, want, start, 4096, true)
			worstOne = max(worstOne, one.lostSeconds)
			worstPair = max(worstPair, pair.lostSeconds)
			if pair.lostSeconds > 0.1 {
				t.Errorf("inverted=%v offset %d (phase %d): first decoded frame %.2f s after the join, want within 0.1 s",
					inv, off, start%10, pair.lostSeconds)
			}
			// A frame with a bit error in AMBE's unprotected bits decodes to
			// a payload that was never sent; the first frame after a join
			// can be one. More than that is a receiver emitting garbage.
			if pair.unknown > 1 {
				t.Errorf("inverted=%v offset %d: %d frames that were never sent", inv, off, pair.unknown)
			}
			unkOne += one.unknown
			unkPair += pair.unknown
			for i := 1; i < len(pair.frames); i++ {
				if pair.frames[i] <= pair.frames[i-1] {
					t.Errorf("inverted=%v offset %d: frame %d emitted after frame %d (duplicate or out of order)",
						inv, off, pair.frames[i], pair.frames[i-1])
					break
				}
			}
			if avail := 150 - (start+959)/960; len(pair.frames) < avail-6 {
				t.Errorf("inverted=%v offset %d: decoded %d of %d frames on the air", inv, off, len(pair.frames), avail)
			}
		}
		t.Logf("inverted=%v: worst speech lost at the join: one receiver %.2f s, receiver pair %.2f s; damaged frames over 40 joins: %d, %d",
			inv, worstOne, worstPair, unkOne, unkPair)
		if unkPair > unkOne {
			t.Errorf("inverted=%v: the pair emitted %d damaged frames, the lone receiver %d", inv, unkPair, unkOne)
		}
		// The failing-first half: the lone receiver really does hang, so
		// the pair is what closes it (if this stops failing, the pair can go).
		if worstOne < 0.5 {
			t.Errorf("inverted=%v: a lone receiver lost only %.2f s; the fixture no longer reproduces the hang", inv, worstOne)
		}
	}
}

// The hand-over rule counts frames, so the output must not depend on how
// the IQ is chunked: one lane processed far ahead of the other inside a
// big chunk must not look like the owner going quiet.
func TestDStarVoiceLanesChunkSizeInvariant(t *testing.T) {
	iq, want := dstarJoinFixture(t, false)
	ref := runDStarJoin(iq, want, 5*1013, 480, true)
	for _, chunk := range []int{37, 4096, 48_000, len(iq)} {
		got := runDStarJoin(iq, want, 5*1013, chunk, true)
		if len(got.frames) != len(ref.frames) {
			t.Fatalf("chunk %d: %d frames, want %d", chunk, len(got.frames), len(ref.frames))
		}
		for i := range got.frames {
			if got.frames[i] != ref.frames[i] {
				t.Fatalf("chunk %d: frame %d is %d, want %d", chunk, i, got.frames[i], ref.frames[i])
			}
		}
	}
}
