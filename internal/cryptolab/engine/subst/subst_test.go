package subst

import (
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/lang"
)

// encrypt applies a substitution permutation (enc[plain]=cipher), preserving
// case and non-letters.
func encrypt(pt []byte, enc [26]byte) []byte {
	out := make([]byte, len(pt))
	for i, b := range pt {
		li := lang.LetterIndex(b)
		if li < 0 {
			out[i] = b
			continue
		}
		c := enc[li]
		if b >= 'a' && b <= 'z' {
			out[i] = 'a' + c
		} else {
			out[i] = 'A' + c
		}
	}
	return out
}

const plaintext = `It is a truth universally acknowledged that a careful reader will recover the
meaning of a scrambled message when enough of the text survives intact. The
patient analyst studies the frequency of each symbol, compares the common
patterns of the language, and slowly rebuilds the hidden alphabet one letter
at a time. With every correct guess the surrounding words grow clearer, until
the whole passage stands revealed and the original sentence can be read aloud
without the slightest doubt about its honest and ordinary meaning.`

func TestSolveSubstitutionRecoversPlaintext(t *testing.T) {
	t.Parallel()
	// A fixed random encryption permutation.
	encR := rand.New(rand.NewSource(7))
	var enc [26]byte
	for i := range enc {
		enc[i] = byte(i)
	}
	encR.Shuffle(26, func(i, j int) { enc[i], enc[j] = enc[j], enc[i] })

	ct := encrypt([]byte(plaintext), enc)

	solveR := rand.New(rand.NewSource(42))
	_, got, _ := Solve(ct, lang.EnglishTrigram(), solveR, Options{})

	match, total := 0, 0
	for i := range plaintext {
		if lang.LetterIndex(plaintext[i]) < 0 {
			continue
		}
		total++
		if got[i] == plaintext[i] {
			match++
		}
	}
	ratio := float64(match) / float64(total)
	if ratio < 0.9 {
		t.Fatalf("recovered only %.1f%% of letters:\n%s", ratio*100, got)
	}
}

func TestDecryptPreservesNonLetters(t *testing.T) {
	t.Parallel()
	var k Key
	for i := range k {
		k[i] = byte(25 - i) // atbash-like
	}
	got := k.Decrypt([]byte("AB! cd."))
	// A(0)->25=Z, B(1)->24=Y, c(2)->23=x, d(3)->22=w; punctuation/space kept.
	if string(got) != "ZY! xw." {
		t.Fatalf("got %q", got)
	}
}
