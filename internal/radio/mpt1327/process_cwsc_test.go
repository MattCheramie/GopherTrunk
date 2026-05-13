package mpt1327

import (
	"log/slog"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// cwscBitsSlice returns the 16-bit CWSC pattern (`1100010011010111`)
// as a byte slice with one bit per element — the format the Process
// adapter expects from the receiver.
func cwscBitsSlice() []byte {
	out := make([]byte, cwscBits)
	for i := 0; i < cwscBits; i++ {
		out[i] = cwscPattern[i]
	}
	return out
}

// TestFindCWSCFindsExactPattern verifies the bit-by-bit matcher
// locates the 16-bit sync sequence anywhere in the buffer.
func TestFindCWSCFindsExactPattern(t *testing.T) {
	for _, lead := range []int{0, 1, 7, 31, 63} {
		buf := make([]byte, lead+cwscBits+8)
		// Fill the leader with 0s and 1s alternating so it doesn't
		// accidentally match.
		for i := 0; i < lead; i++ {
			buf[i] = byte(i & 1)
		}
		copy(buf[lead:], cwscBitsSlice())
		got, ok := findCWSC(buf, 0, 0)
		if !ok {
			t.Errorf("lead=%d: findCWSC returned !ok", lead)
			continue
		}
		if got != lead {
			t.Errorf("lead=%d: findCWSC = %d, want %d", lead, got, lead)
		}
	}
}

// TestFindCWSCWithinTolerance is the table-driven replacement for the
// old exact-match-rejection check. With the matcher now accepting up
// to maxErrors bit flips, the spec-compliant 1-2-bit-error tolerance
// is exercised across the boundary: 0/1/2 flips must match under
// tolerance 2; 3+ flips must reject. Exact-match (tolerance 0) only
// accepts the unmodified pattern.
func TestFindCWSCWithinTolerance(t *testing.T) {
	cases := []struct {
		name      string
		flips     []int // bit positions inside the 16-bit window to invert
		tolerance int
		wantMatch bool
	}{
		{"zero_flips_tol_0", nil, 0, true},
		{"zero_flips_tol_2", nil, 2, true},
		{"one_flip_tol_0", []int{7}, 0, false},
		{"one_flip_tol_1", []int{7}, 1, true},
		{"one_flip_tol_2", []int{7}, 2, true},
		{"two_flips_tol_1", []int{3, 11}, 1, false},
		{"two_flips_tol_2", []int{3, 11}, 2, true},
		{"three_flips_tol_2", []int{3, 7, 11}, 2, false},
		{"three_flips_tol_3", []int{3, 7, 11}, 3, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := cwscBitsSlice()
			for _, idx := range tc.flips {
				buf[idx] ^= 1
			}
			_, ok := findCWSC(buf, 0, tc.tolerance)
			if ok != tc.wantMatch {
				t.Errorf("findCWSC(flips=%v, tol=%d) = %v, want %v",
					tc.flips, tc.tolerance, ok, tc.wantMatch)
			}
		})
	}
}

// TestFindCWSCFalsePositiveControl asserts the default tolerance of 2
// keeps random 16-bit-window false-positive rate within the
// combinatorial bound — C(16, 0..2) / 2^16 ≈ 0.21%. Sample size large
// enough that 0.5% is comfortably above statistical noise.
func TestFindCWSCFalsePositiveControl(t *testing.T) {
	const trials = 65536
	const tolerance = cwscDefaultMaxErrors
	rng := rand.New(rand.NewSource(0xC4D7C4D7))
	hits := 0
	buf := make([]byte, cwscBits)
	for i := 0; i < trials; i++ {
		for j := 0; j < cwscBits; j++ {
			buf[j] = byte(rng.Intn(2))
		}
		if _, ok := findCWSC(buf, 0, tolerance); ok {
			hits++
		}
	}
	rate := float64(hits) / float64(trials)
	// Theoretical bound: (1 + 16 + 120) / 65536 ≈ 0.0021.
	// Allow generous headroom (0.005) to absorb sampling variance.
	if rate > 0.005 {
		t.Errorf("false-positive rate %.4f exceeds 0.005 ceiling (hits=%d, trials=%d)",
			rate, hits, trials)
	}
}

// TestFindCWSCRespectsFromOffset confirms the scan begins at the
// supplied start index, so a CWSC that appears before the cursor
// position isn't re-matched.
func TestFindCWSCRespectsFromOffset(t *testing.T) {
	first := cwscBitsSlice()
	second := cwscBitsSlice()
	gap := make([]byte, 8)
	buf := append(append(first, gap...), second...)
	got, ok := findCWSC(buf, len(first), 0)
	if !ok {
		t.Fatalf("findCWSC didn't find the second copy")
	}
	want := len(first) + len(gap)
	if got != want {
		t.Errorf("findCWSC = %d, want %d", got, want)
	}
}

// TestParseCWSCTolerance verifies the user-facing string parser the
// ccdecoder connector uses to translate the mpt1327_cwsc_tolerance
// system field into a numeric threshold.
func TestParseCWSCTolerance(t *testing.T) {
	cases := []struct {
		in      string
		want    int
		wantOK  bool
	}{
		{"", cwscDefaultMaxErrors, true},
		{"0", 0, true},
		{"exact", 0, true},
		{"off", 0, true},
		{"OFF", 0, true},
		{"  exact  ", 0, true},
		{"1", 1, true},
		{"2", 2, true},
		{"4", 4, true},
		{"15", 15, true},  // cwscBits-1 is the upper bound
		{"16", cwscDefaultMaxErrors, false}, // >= cwscBits is invalid
		{"-1", cwscDefaultMaxErrors, false},
		{"banana", cwscDefaultMaxErrors, false},
	}
	for _, tc := range cases {
		got, ok := ParseCWSCTolerance(tc.in)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("ParseCWSCTolerance(%q) = (%d, %v), want (%d, %v)",
				tc.in, got, ok, tc.want, tc.wantOK)
		}
	}
}

// TestSetCWSCToleranceRoundTrips covers the setter + getter pair the
// ccdecoder connector uses after parsing the config field, and the
// boundary clamps for negative and oversized values.
func TestSetCWSCToleranceRoundTrips(t *testing.T) {
	cc := New(Options{
		Log:         slog.Default(),
		SystemName:  "Sys",
		FrequencyHz: 169_212_500,
	})
	// New() zero-values cwscTolerance to 0 (exact-match) so in-package
	// fixture tests stay unaffected; the ccdecoder connector raises it
	// to cwscDefaultMaxErrors via ParseCWSCTolerance("").
	if got := cc.CWSCTolerance(); got != 0 {
		t.Errorf("default CWSCTolerance = %d, want 0", got)
	}
	cc.SetCWSCTolerance(3)
	if got := cc.CWSCTolerance(); got != 3 {
		t.Errorf("after SetCWSCTolerance(3) = %d, want 3", got)
	}
	cc.SetCWSCTolerance(-5)
	if got := cc.CWSCTolerance(); got != 0 {
		t.Errorf("negative input not clamped: got %d, want 0", got)
	}
	cc.SetCWSCTolerance(1000)
	if got := cc.CWSCTolerance(); got != cwscBits-1 {
		t.Errorf("oversized input not clamped: got %d, want %d", got, cwscBits-1)
	}
}

// TestProcessLocksOnCWSCBeforeRecognisedCodeword exercises the
// two-stage alignment: a CWSC-prefixed Aloha codeword must lock the
// state machine at the byte immediately after the sync pattern, even
// when the preceding bits in the buffer are noise that would
// otherwise drive the fallback search.
func TestProcessLocksOnCWSCBeforeRecognisedCodeword(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "Sys",
		FrequencyHz: 169_212_500,
	})

	// Noise prefix that's deliberately not a parseable codeword and
	// doesn't contain CWSC.
	noise := make([]byte, 40)
	for i := range noise {
		noise[i] = byte(i&1) ^ 1
	}

	aloha := alohaCodeword(0x5)
	stream := append([]byte{}, noise...)
	stream = append(stream, cwscBitsSlice()...)
	stream = append(stream, CodewordBits(aloha)...)

	cc.Process(stream, 0)

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked {
			t.Fatalf("first event = %v, want cc.locked", ev.Kind)
		}
		ls, ok := ev.Payload.(LockState)
		if !ok {
			t.Fatalf("payload = %T, want LockState", ev.Payload)
		}
		if ls.Prefix != 0x5 {
			t.Errorf("LockState.Prefix = %#x, want 0x5", ls.Prefix)
		}
	default:
		t.Fatal("no cc.locked event after CWSC-prefixed Aloha")
	}
}

// TestProcessCWSCPrefixedGTCEmitsGrant confirms a grant publishes
// when an Aloha + GoToChan pair follows the sync.
func TestProcessCWSCPrefixedGTCEmitsGrant(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "Sys",
		FrequencyHz: 169_212_500,
	})

	aloha := alohaCodeword(0x5)
	gtc := gtcCodeword(0x5, 0x123, 7)
	stream := append([]byte{}, cwscBitsSlice()...)
	stream = append(stream, CodewordBits(aloha)...)
	stream = append(stream, CodewordBits(gtc)...)

	cc.Process(stream, 0)

	var sawLock, sawGrant bool
	for i := 0; i < 4; i++ {
		select {
		case ev := <-sub.C:
			switch ev.Kind {
			case events.KindCCLocked:
				sawLock = true
			case events.KindGrant:
				sawGrant = true
				g, ok := ev.Payload.(trunking.Grant)
				if !ok {
					t.Errorf("grant payload = %T, want trunking.Grant", ev.Payload)
					continue
				}
				if g.ChannelNum != 7 {
					t.Errorf("grant.ChannelNum = %d, want 7", g.ChannelNum)
				}
			}
		default:
		}
	}
	if !sawLock {
		t.Error("did not see cc.locked")
	}
	if !sawGrant {
		t.Error("did not see grant")
	}
}

// TestProcessFallsBackWhenNoCWSC confirms the legacy "first parseable
// codeword wins" path still works when the stream doesn't include the
// sync sequence — protects synthesized-fixture tests that pre-date
// this PR.
func TestProcessFallsBackWhenNoCWSC(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "Sys",
		FrequencyHz: 169_212_500,
	})

	aloha := alohaCodeword(0x5)
	// No CWSC prefix; the existing fallback path should still
	// recognise the Aloha codeword and lock.
	cc.Process(CodewordBits(aloha), 0)

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked {
			t.Fatalf("first event = %v, want cc.locked", ev.Kind)
		}
	default:
		t.Fatal("fallback path did not lock on bare Aloha codeword")
	}
}
