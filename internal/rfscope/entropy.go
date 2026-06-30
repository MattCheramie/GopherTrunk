package rfscope

import (
	"context"
	"fmt"
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/lfsr"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/randomness"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/stats"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

func init() { Register(entropyAnalyzer{}) }

const (
	// minPayloadBytes is the shortest recovered payload worth triaging.
	minPayloadBytes = 32
	// maxEntropyEmitters caps how many emitters are triaged per scene.
	maxEntropyEmitters = 16
)

// EntropyResult is the byte-level triage of one emitter's recovered payload. It
// mirrors the cryptolab `classify auto` taxonomy so an operator can take the
// recommended next command straight into the cryptolab toolkit. The raw payload
// is carried (unserialized) for the cryptolab bridge.
type EntropyResult struct {
	EmitterID          uint64  `json:"emitter_id"`
	FreqHz             uint32  `json:"freq_hz"`
	Bytes              int     `json:"bytes"`
	Class              string  `json:"class"`
	Confidence         float64 `json:"confidence"`
	EntropyBits        float64 `json:"entropy_bits"`
	IndexOfCoincidence float64 `json:"index_of_coincidence"`
	ChiSquareUniform   float64 `json:"chi_square_uniform"`
	KeyLen             int     `json:"key_len,omitempty"`
	LooksRandom        bool    `json:"looks_random"`
	Recommended        string  `json:"recommended"`

	payload []byte // raw recovered bytes, for the bridge; not serialized
}

// entropyAnalyzer triages every unidentified digital emitter's payload: it
// blind-demodulates a representative burst, then runs the cryptolab randomness
// battery and the same measurements `classify auto` uses to label the payload
// plaintext / substitution / repeating-xor / periodic-scrambler /
// lfsr-or-keyless-scrambler / strong-encrypted. It is the entropy-based
// encryption detector — Wireshark's "is this flow encrypted?" for RF — built on
// #832's engines rather than hand-rolled thresholds. Results land in
// AnalyzerOutputs["entropy"]; expert.go turns the interesting ones into anomalies
// and bridge.go hands the payloads to the cryptolab toolkit.
type entropyAnalyzer struct{}

func (entropyAnalyzer) Name() string { return "entropy" }
func (entropyAnalyzer) Synopsis() string {
	return "bitstream entropy / encryption triage (cryptolab bridge)"
}
func (entropyAnalyzer) DependsOn() []string { return []string{"topology"} }

func (entropyAnalyzer) Analyze(ctx context.Context, sc *Scene, in *Input) error {
	if in == nil || in.BurstIQ == nil || len(sc.Emitters) == 0 {
		return nil
	}
	var results []EntropyResult
	triaged := 0
	for _, e := range sc.Emitters {
		if triaged >= maxEntropyEmitters {
			break
		}
		if e.Fingerprint.ClassFamily != "digital" {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		bid := longestBurstID(sc, e)
		if bid == 0 {
			continue
		}
		bidx := burstIndexByID(sc, bid)
		iq, rate, err := in.BurstIQ(sc.Bursts[bidx])
		if err != nil || len(iq) == 0 || rate <= 0 {
			continue
		}
		// Skip emitters that decode as a known protocol — triage is for unknowns.
		if ident, err := siglab.IdentifyIQ(iq, "rfscope-burst", siglab.IdentifyConfig{SampleRateHz: rate, Format: siglab.FormatF32, Log: in.Log}); err == nil && ident != nil && !ident.Inconclusive {
			continue
		}
		payload := recoverPayload(iq, rate, sc.Bursts[bidx].Features)
		if len(payload) < minPayloadBytes {
			continue
		}
		triaged++
		r := classifyPayload(payload)
		r.EmitterID = e.ID
		r.FreqHz = e.Fingerprint.CenterHz
		results = append(results, r)
	}
	if len(results) == 0 {
		return nil
	}
	if sc.AnalyzerOutputs == nil {
		sc.AnalyzerOutputs = map[string]any{}
	}
	sc.AnalyzerOutputs["entropy"] = results
	return nil
}

// classifyPayload runs the cryptolab measurements over a recovered payload and
// returns the highest-confidence verdict, using the exact thresholds and
// recommendations cryptolab's `classify auto` applies (so rfscope and the
// toolkit agree). It calls the untagged engine packages directly, so it works in
// the default binary without the cryptolab build tag.
func classifyPayload(data []byte) EntropyResult {
	ent := stats.ShannonEntropy(data)
	ic := stats.IndexOfCoincidence(data)
	chi := stats.ChiSquareUniform(data)
	printable, letters := charProfileBytes(data)
	keylens := stats.GuessXORKeyLength(data, 40)
	periods := stats.DetectPeriod(data, 256, 4)

	bits := lfsr.BitsFromBytes(data)
	var rep randomness.Report
	if len(data) >= 2048 {
		rep = randomness.Battery(bits, randomness.DefaultAlpha)
	} else {
		rep = randomness.Quick(bits, randomness.DefaultAlpha)
	}
	lcFailed := randomnessFailed(rep, "linear_complexity")
	spectralFailed := randomnessFailed(rep, "spectral_dft")

	type verdict struct {
		class       string
		confidence  float64
		recommended string
		keyLen      int
	}
	var vs []verdict
	if printable > 0.95 && ent < 4.8 {
		vs = append(vs, verdict{"plaintext", 0.6 + 0.4*printable, "already plaintext — no decryption needed", 0})
	}
	if ic > 0.045 && letters > 0.6 {
		vs = append(vs, verdict{"substitution-or-shift", clampUnit(ic / 0.066), "cryptolab brute substitution -in <file>", 0})
	}
	if len(keylens) > 0 && keylens[0].Length > 1 && keylens[0].Score < 3.6 {
		vs = append(vs, verdict{"repeating-xor", clampUnit((3.6 - keylens[0].Score) / 3.6),
			fmt.Sprintf("cryptolab brute xor -in <file> -keylen %d", keylens[0].Length), keylens[0].Length})
	}
	if len(periods) > 0 {
		vs = append(vs, verdict{"periodic-scrambler", clampUnit(periods[0].ZScore / 12),
			fmt.Sprintf("cryptolab stats period -in <file> (lag≈%d) then descramble rolling or ks mtp", periods[0].Lag), 0})
	}
	if ent > 7.0 && (lcFailed || spectralFailed) {
		vs = append(vs, verdict{"lfsr-or-keyless-scrambler", 0.7, "cryptolab lfsr bm -in <file>", 0})
	}
	if ent > 7.9 && rep.LooksRandom() {
		vs = append(vs, verdict{"strong-encrypted", clampUnit((ent - 7.9) / 0.1),
			"no weak structure — if frames reuse an IV/MI try cryptolab ks reuse, else key material is required", 0})
	}
	if len(vs) == 0 {
		vs = append(vs, verdict{"inconclusive", 0.2, "cryptolab stats scan -in <file>", 0})
	}
	sort.SliceStable(vs, func(i, j int) bool { return vs[i].confidence > vs[j].confidence })
	top := vs[0]

	return EntropyResult{
		Bytes:              len(data),
		Class:              top.class,
		Confidence:         top.confidence,
		EntropyBits:        ent,
		IndexOfCoincidence: ic,
		ChiSquareUniform:   chi,
		KeyLen:             top.keyLen,
		LooksRandom:        rep.LooksRandom(),
		Recommended:        top.recommended,
		payload:            data,
	}
}

func charProfileBytes(data []byte) (printable, letters float64) {
	if len(data) == 0 {
		return 0, 0
	}
	var p, l int
	for _, b := range data {
		if b >= 0x20 && b <= 0x7e {
			p++
		}
		if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') {
			l++
		}
	}
	n := float64(len(data))
	return float64(p) / n, float64(l) / n
}

func randomnessFailed(r randomness.Report, name string) bool {
	for _, t := range r.Tests {
		if t.Name == name && t.Applicable && !t.Passed {
			return true
		}
	}
	return false
}

func clampUnit(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// entropyResults returns the triage results from a populated scene, or nil.
func entropyResults(sc *Scene) []EntropyResult {
	if sc.AnalyzerOutputs == nil {
		return nil
	}
	r, _ := sc.AnalyzerOutputs["entropy"].([]EntropyResult)
	return r
}
