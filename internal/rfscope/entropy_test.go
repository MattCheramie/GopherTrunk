package rfscope

import (
	"context"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/stats"
	"github.com/MattCheramie/GopherTrunk/internal/survey"
)

func randomBytes(n int, seed int64) []byte {
	rng := rand.New(rand.NewSource(seed))
	b := make([]byte, n)
	rng.Read(b)
	return b
}

func TestClassifyPayloadStrongEncrypted(t *testing.T) {
	t.Parallel()
	r := classifyPayload(randomBytes(4096, 7))
	if r.Class != "strong-encrypted" {
		t.Fatalf("random bytes should be strong-encrypted, got %q (entropy %.3f, looksRandom %v)", r.Class, r.EntropyBits, r.LooksRandom)
	}
	if !r.LooksRandom {
		t.Fatalf("random bytes should pass the randomness battery")
	}
}

func TestClassifyPayloadRepeatingXOR(t *testing.T) {
	t.Parallel()
	// Non-periodic letter plaintext XOR'd with a 5-byte repeating key — the only
	// structure is the key period, so repeating-xor must win over periodic-scrambler.
	rng := rand.New(rand.NewSource(99))
	const alphabet = "abcdefghijklmnopqrstuvwxyz "
	pt := make([]byte, 3000)
	for i := range pt {
		pt[i] = alphabet[rng.Intn(len(alphabet))]
	}
	key := []byte{0x13, 0x37, 0x42, 0x99, 0x05}
	ct := make([]byte, len(pt))
	for i := range pt {
		ct[i] = pt[i] ^ key[i%len(key)]
	}
	r := classifyPayload(ct)
	if r.Class != "repeating-xor" {
		t.Fatalf("repeating-XOR ciphertext misclassified as %q", r.Class)
	}
	if r.KeyLen == 0 || r.KeyLen%len(key) != 0 {
		t.Fatalf("recovered key length %d not a multiple of %d", r.KeyLen, len(key))
	}
}

func TestClassifyPayloadMatchesEngines(t *testing.T) {
	t.Parallel()
	data := randomBytes(2048, 11)
	r := classifyPayload(data)
	if r.EntropyBits != stats.ShannonEntropy(data) {
		t.Fatalf("entropy mismatch with engine: %v vs %v", r.EntropyBits, stats.ShannonEntropy(data))
	}
	if r.IndexOfCoincidence != stats.IndexOfCoincidence(data) {
		t.Fatalf("IC mismatch with engine")
	}
	if r.ChiSquareUniform != stats.ChiSquareUniform(data) {
		t.Fatalf("chi-square mismatch with engine")
	}
}

func TestEntropyAnalyzerViaRunner(t *testing.T) {
	t.Parallel()
	// A digital emitter carrying random bits ⇒ a strong-encrypted triage result.
	bitsN := 4096
	bits := make([]int, bitsN)
	rng := rand.New(rand.NewSource(3))
	for i := range bits {
		bits[i] = rng.Intn(2)
	}
	iq := synthFSK(bits, 48000, 4800, 1200)
	dur := float64(len(iq)) / 48000

	sc := &Scene{
		SpanHz: 1_000_000, WindowSec: 10,
		Bursts: []Burst{{
			ID: 1, FreqHz: 450_000_000, StartSec: 0, EndSec: dur,
			PeakDbFS: -40, Class: survey.ClassFSK, OccupiedBwHz: 12_500,
			Features: survey.ClassFeatures{BaudHz: 4800, IFModality: 2},
		}},
	}
	in := &Input{
		SampleRateHz: 48000, Window: 10,
		BurstIQ: func(b Burst) ([]complex64, float64, error) { return iq, 48000, nil },
	}
	if err := Run(context.Background(), sc, in, "entropy"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	res := entropyResults(sc)
	if len(res) != 1 {
		t.Fatalf("want 1 entropy result, got %d", len(res))
	}
	if res[0].EmitterID == 0 || res[0].Bytes < minPayloadBytes {
		t.Fatalf("entropy result incomplete: %+v", res[0])
	}
}
