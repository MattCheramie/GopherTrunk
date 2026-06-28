package lang

import "testing"

func TestCleanDropsNonLetters(t *testing.T) {
	t.Parallel()
	got := Clean([]byte("Ab! 9zZ"))
	want := []int{0, 1, 25, 25}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestTrigramScoresEnglishAboveGibberish(t *testing.T) {
	t.Parallel()
	m := EnglishTrigram()
	eng := m.Score([]byte("the patient analyst studies the common patterns of the language"))
	gib := m.Score([]byte("xqzj kwvbq mxptz qfxw jjkq zzxv qwxk vbmn pqrx zzqj kwxv"))
	if eng <= gib {
		t.Fatalf("english score %.2f should beat gibberish %.2f", eng, gib)
	}
}

func TestFreqOrderHasCommonLettersFirst(t *testing.T) {
	t.Parallel()
	order := FreqOrder()
	if len(order) != 26 {
		t.Fatalf("FreqOrder len = %d", len(order))
	}
	// 'E' (index 4) and 'T' (19) are the two most common English letters and
	// should land in the top handful of a real English corpus.
	top := map[int]bool{}
	for _, v := range order[:5] {
		top[v] = true
	}
	if !top[4] || !top[19] {
		t.Fatalf("expected E and T near the front, got top5 %v", order[:5])
	}
}
