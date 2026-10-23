package conventional

import "testing"

// streamPos names where the current transmission began (issue #1288): the
// first above-squelch chunk of a run, kept across gaps up to onsetGap (one
// DMR simplex over's TDMA gaps), and reset by a longer silence.
func TestStreamPosOnset(t *testing.T) {
	p := &streamPos{next: 1000, gap: 100}
	p.advance(50, false) // 1000..1049 noise
	if p.inRun {
		t.Fatal("noise started a run")
	}
	p.advance(50, true) // 1050..1099 carrier
	p.advance(80, false)
	p.advance(50, true) // 80-sample gap ≤ 100: same transmission
	if !p.inRun || p.onset != 1050 {
		t.Fatalf("onset %d (inRun %v), want 1050 across a short gap", p.onset, p.inRun)
	}
	p.advance(200, false)
	p.advance(10, true) // 200-sample silence > 100: a new transmission
	if p.onset != 1430 {
		t.Fatalf("onset %d after a long silence, want 1430", p.onset)
	}
	if p.next != 1440 {
		t.Fatalf("next %d, want 1440", p.next)
	}
}
