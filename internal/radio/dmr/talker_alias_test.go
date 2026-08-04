package dmr

import (
	"testing"
	"time"
)

// fixedNow returns a now() that advances by a fixed step on each call so
// staleness eviction is deterministic in tests.
func fixedNow(start time.Time, step time.Duration) func() time.Time {
	t := start
	return func() time.Time {
		cur := t
		t = t.Add(step)
		return cur
	}
}

func TestTalkerAliasRoundTripFormats(t *testing.T) {
	cases := []struct {
		name   string
		format TalkerAliasFormat
		alias  string
	}{
		{"7bit-short", TalkerAliasFormat7Bit, "KD2ABC"},
		{"7bit-multiblock", TalkerAliasFormat7Bit, "DISPATCH-CENTRAL-01"},
		{"8bit-iso", TalkerAliasFormat8Bit, "Engine 12"},
		{"utf8", TalkerAliasFormatUTF8, "Café Unit"},
		{"utf16", TalkerAliasFormatUTF16, "Rescue-7"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			frags := AssembleTalkerAliasFragments(tc.format, tc.alias)
			if len(frags) == 0 {
				t.Fatal("no fragments produced")
			}
			a := NewTalkerAliasAssembler(fixedNow(time.Unix(0, 0), time.Millisecond))
			const src = uint32(0x1234)
			var got string
			var done bool
			for _, info := range frags {
				f, ok := ParseTalkerAliasFragment(info)
				if !ok {
					t.Fatalf("ParseTalkerAliasFragment rejected a fragment: %x", info)
				}
				got, done = a.Add(src, f)
			}
			if !done {
				t.Fatalf("alias never completed after %d fragments", len(frags))
			}
			if got != tc.alias {
				t.Fatalf("round-trip mismatch: got %q want %q", got, tc.alias)
			}
		})
	}
}

func TestTalkerAliasHeaderFieldExtraction(t *testing.T) {
	// A synthetic header: FLCO 0x04, format UTF-8 (2), length 9.
	frags := AssembleTalkerAliasFragments(TalkerAliasFormatUTF8, "Truck 349")
	f, ok := ParseTalkerAliasFragment(frags[0])
	if !ok || !f.Header {
		t.Fatalf("header fragment not parsed as header: ok=%v hdr=%v", ok, f.Header)
	}
	if f.Format != TalkerAliasFormatUTF8 {
		t.Errorf("format: got %d want %d", f.Format, TalkerAliasFormatUTF8)
	}
	if f.Length != 9 {
		t.Errorf("length: got %d want 9", f.Length)
	}
}

func TestTalkerAliasRejectsNonAliasFLCO(t *testing.T) {
	info := AssembleFLC(FLC{FLCO: FLCOGroupVoiceUser, DstAddr: 100, SrcAddr: 200})
	if _, ok := ParseTalkerAliasFragment(info); ok {
		t.Fatal("group-voice FLCO wrongly parsed as talker alias")
	}
}

func TestTalkerAliasOutOfOrderReassembly(t *testing.T) {
	frags := AssembleTalkerAliasFragments(TalkerAliasFormat7Bit, "DISPATCH-CENTRAL-01")
	if len(frags) < 3 {
		t.Fatalf("expected a multi-block alias, got %d fragments", len(frags))
	}
	a := NewTalkerAliasAssembler(fixedNow(time.Unix(0, 0), time.Millisecond))
	const src = uint32(0x55)
	// Feed blocks before the header, and the header last.
	order := append([]int{}, indicesReversed(len(frags))...)
	var got string
	var done bool
	for _, i := range order {
		f, ok := ParseTalkerAliasFragment(frags[i])
		if !ok {
			t.Fatalf("fragment %d rejected", i)
		}
		got, done = a.Add(src, f)
	}
	if !done {
		t.Fatal("alias never completed with out-of-order fragments")
	}
	if got != "DISPATCH-CENTRAL-01" {
		t.Fatalf("out-of-order mismatch: got %q", got)
	}
}

func TestTalkerAliasIncompleteStaysPending(t *testing.T) {
	frags := AssembleTalkerAliasFragments(TalkerAliasFormat7Bit, "DISPATCH-CENTRAL-01")
	a := NewTalkerAliasAssembler(fixedNow(time.Unix(0, 0), time.Millisecond))
	const src = uint32(0x99)
	// Feed only the header — the declared length needs more blocks.
	f, _ := ParseTalkerAliasFragment(frags[0])
	if _, done := a.Add(src, f); done {
		t.Fatal("alias completed from the header alone")
	}
}

// indicesReversed returns [n-1, n-2, ..., 0].
func indicesReversed(n int) []int {
	out := make([]int, n)
	for i := 0; i < n; i++ {
		out[i] = n - 1 - i
	}
	return out
}
