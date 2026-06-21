package sdr

import "testing"

func TestMatchHintSerialsExact(t *testing.T) {
	devices := []string{"AAA", "BBB"}
	hints := []string{"BBB"}
	got, partial := matchHintSerials(devices, hints)
	if got[0] != -1 {
		t.Errorf("device 0 (AAA) matched hint %d, want -1", got[0])
	}
	if got[1] != 0 {
		t.Errorf("device 1 (BBB) matched hint %d, want 0", got[1])
	}
	if partial[1] {
		t.Error("BBB should be an exact match, not partial")
	}
}

// TestMatchHintSerialsLegacyTruncatedPrefix is the regression for the
// downstream effect of the HackRF serial-truncation bug: a config that
// captured the old 16-zero prefix must still bind the device whose full
// serial starts with it.
func TestMatchHintSerialsLegacyTruncatedPrefix(t *testing.T) {
	devices := []string{"0000000000000000457863c8284a625f"}
	hints := []string{"0000000000000000"}
	got, partial := matchHintSerials(devices, hints)
	if got[0] != 0 {
		t.Fatalf("legacy-prefix hint did not bind device: got %v", got)
	}
	if !partial[0] {
		t.Error("legacy-prefix bind should be flagged partial")
	}
}

func TestMatchHintSerialsDistinctiveTail(t *testing.T) {
	devices := []string{"0000000000000000457863c8284a625f"}
	hints := []string{"457863c8284a625f"}
	got, _ := matchHintSerials(devices, hints)
	if got[0] != 0 {
		t.Fatalf("tail hint did not bind device: got %v", got)
	}
}

// TestMatchHintSerialsAmbiguousPartialSkipped guards against the fallback
// grabbing the wrong dongle: two HackRFs share the all-zero prefix, so the
// prefix must NOT bind either one.
func TestMatchHintSerialsAmbiguousPartialSkipped(t *testing.T) {
	devices := []string{
		"0000000000000000457863c8284a625f",
		"0000000000000000aabbccddeeff0011",
	}
	hints := []string{"0000000000000000"}
	got, _ := matchHintSerials(devices, hints)
	if got[0] != -1 || got[1] != -1 {
		t.Errorf("ambiguous prefix should bind nothing, got %v", got)
	}
}

// TestMatchHintSerialsShortNeverPartial confirms short serials only ever
// match exactly — a fragment below minPartialSerialLen must not substring-
// match a longer device serial.
func TestMatchHintSerialsShortNeverPartial(t *testing.T) {
	devices := []string{"S1X9"}
	hints := []string{"S1"} // 2 chars, below the partial floor
	got, _ := matchHintSerials(devices, hints)
	if got[0] != -1 {
		t.Errorf("short fragment should not partial-match, got %v", got)
	}
}

func TestMatchHintSerialsExactWinsOverPartial(t *testing.T) {
	// Device 0's serial contains hint "abcdef" as a substring, but device 1
	// matches it exactly — exact must win and claim device 1, leaving 0 free.
	devices := []string{"zzabcdefzz", "abcdef"}
	hints := []string{"abcdef"}
	got, partial := matchHintSerials(devices, hints)
	if got[1] != 0 || partial[1] {
		t.Errorf("exact device should win: got=%v partial=%v", got, partial)
	}
	if got[0] != -1 {
		t.Errorf("substring device should be left unbound once exact wins, got %v", got)
	}
}
