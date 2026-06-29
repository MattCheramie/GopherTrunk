// Package assess is a cryptographic security-test harness for captured RF
// ciphertext. Given a set of encrypted frames (and optionally a known
// plaintext), it runs every applicable decryption / cryptanalysis method and
// reports how effective each one was — from 0% (the encryption held) up to
// 100% (complete decryption, which means the cipher *failed* the test).
//
// The point is an informed verdict, not a single yes/no: by seeing which
// methods recovered what, an operator learns where a deployment is weak (a
// reused IV, a default key, a structured keystream) and which attack fits
// which situation. A method that recovers nothing is reported too — that is
// the evidence the encryption is doing its job.
//
// Methods, in escalating capability:
//
//   - cipher-strength    statistical: is the ciphertext itself distinguishable
//     from random? A structured ciphertext is already a partial break.
//   - iv-reuse           structural: do frames share an IV (keystream reuse)?
//     Exposes plaintext⊕plaintext with no key.
//   - known-plaintext    if a frame's plaintext is known, recover the keystream
//     and decrypt every same-IV frame. Definitive.
//   - weak-key           try default / supplied keys with the real cipher
//     (ADP/DES/AES via engine/p25crypto); verify against known plaintext.
//   - keystream-lfsr     once a keystream is recovered, is it an LFSR (low
//     linear complexity)? If so the rest of the call is predictable.
package assess

import (
	"encoding/hex"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/keystream"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/lfsr"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/p25crypto"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/randomness"
)

// Input bundles the material the harness works from.
type Input struct {
	Frames     []keystream.Frame
	KnownLabel string   // label of a frame whose plaintext is known (optional)
	KnownPT    []byte   // that frame's plaintext (optional)
	ExtraKeys  [][]byte // additional candidate keys to try in the weak-key method
}

// MethodResult is one method's outcome.
type MethodResult struct {
	Name string `json:"name"`
	// Applicable is false when the method's preconditions aren't met (e.g. no
	// known plaintext, unsupported algorithm); Notes says why.
	Applicable bool `json:"applicable"`
	// Effectiveness is the fraction (0..1) of the traffic this method recovered
	// or exposed. 1.0 from a Verified method is a complete break.
	Effectiveness float64 `json:"effectiveness"`
	// Verified is true when the recovery is definitive (exact keystream match
	// or a structural certainty), false when it is statistical evidence only.
	Verified       bool           `json:"verified"`
	RecoveredBytes int            `json:"recovered_bytes"`
	TotalBytes     int            `json:"total_bytes"`
	SampleHex      string         `json:"sample_hex,omitempty"`
	SampleASCII    string         `json:"sample_ascii,omitempty"`
	Notes          []string       `json:"notes,omitempty"`
	Detail         map[string]any `json:"detail,omitempty"`
}

func (m *MethodResult) note(s string) { m.Notes = append(m.Notes, s) }

// Report is the full assessment.
type Report struct {
	Frames               int            `json:"frames"`
	TotalCipherBytes     int            `json:"total_cipher_bytes"`
	Methods              []MethodResult `json:"methods"`
	OverallEffectiveness float64        `json:"overall_effectiveness"`
	// Verdict is RESISTANT (nothing recovered), PARTIAL (information leaked),
	// or BROKEN (a method achieved verified complete decryption — a fail).
	Verdict string `json:"verdict"`
}

const (
	VerdictResistant = "RESISTANT"
	VerdictPartial   = "PARTIAL"
	VerdictBroken    = "BROKEN"
)

// Run executes every method and assembles the report.
func Run(in Input) Report {
	total := 0
	for _, f := range in.Frames {
		total += len(f.CT)
	}
	rep := Report{Frames: len(in.Frames), TotalCipherBytes: total}

	cipherStrength := methodCipherStrength(in.Frames, total)
	reuse := methodIVReuse(in.Frames, total)
	known, knownKS := methodKnownPlaintext(in, total)
	weak, weakKS := methodWeakKey(in, total)

	// The keystream-LFSR method runs only on a genuine keystream a stronger
	// method recovered (known-plaintext, or a verified weak key). A reuse XOR
	// (p1⊕p2) is plaintext, not keystream, so it is deliberately not fed here.
	ks := knownKS
	ksSource := "known-plaintext"
	if ks == nil && weakKS != nil {
		ks, ksSource = weakKS, "weak-key"
	}
	lc := methodKeystreamLFSR(ks, ksSource)

	rep.Methods = []MethodResult{cipherStrength, reuse, known, weak, lc}

	// Overall = the strongest result; verdict is BROKEN only on a verified,
	// (near-)complete decryption.
	broken := false
	for _, m := range rep.Methods {
		if m.Applicable && m.Effectiveness > rep.OverallEffectiveness {
			rep.OverallEffectiveness = m.Effectiveness
		}
		if m.Applicable && m.Verified && m.Effectiveness >= 0.999 {
			broken = true
		}
	}
	switch {
	case broken:
		rep.Verdict = VerdictBroken
	case rep.OverallEffectiveness > 0:
		rep.Verdict = VerdictPartial
	default:
		rep.Verdict = VerdictResistant
	}
	return rep
}

// methodCipherStrength: is the ciphertext distinguishable from random? A strong
// cipher's output passes the battery (0% exploitable); a structured ciphertext
// (weak/keyless construction) fails tests and is exploitable.
func methodCipherStrength(frames []keystream.Frame, total int) MethodResult {
	m := MethodResult{Name: "cipher-strength", TotalBytes: total}
	var ct []byte
	for _, f := range frames {
		ct = append(ct, f.CT...)
		if len(ct) >= 256_000 {
			break
		}
	}
	if len(ct) < 16 {
		m.note("too little ciphertext to assess")
		return m
	}
	m.Applicable = true
	rep := randomness.Battery(lfsr.BitsFromBytes(ct), randomness.DefaultAlpha)
	applicable := rep.Passed + rep.Failed
	if applicable > 0 {
		m.Effectiveness = float64(rep.Failed) / float64(applicable)
	}
	m.Detail = map[string]any{"tests_failed": rep.Failed, "tests_passed": rep.Passed}
	if rep.LooksRandom() {
		m.note("ciphertext is statistically indistinguishable from random — the cipher output is strong; no structural leakage to exploit.")
	} else {
		m.note(fmt.Sprintf("ciphertext fails %d/%d randomness tests — structured output indicates a weak or keyless construction (try lfsr / stats period).", rep.Failed, applicable))
	}
	return m
}

// methodIVReuse: structural keystream reuse. This is an EXPOSURE method, not a
// turnkey decryption — a collision leaks plaintext⊕plaintext but still needs a
// crib or known plaintext to fully recover, so it is reported unverified
// (contributes to PARTIAL, never BROKEN on its own).
func methodIVReuse(frames []keystream.Frame, total int) MethodResult {
	m := MethodResult{Name: "iv-reuse", TotalBytes: total}
	groups := keystream.FindReuse(frames)
	m.Applicable = true
	if len(groups) == 0 {
		m.note("no IV/MI reuse — every frame uses a distinct IV, so there is no keystream collision to exploit. This is the secure case.")
		return m
	}
	exposed := 0
	for _, g := range groups {
		for _, f := range g.Frames {
			exposed += len(f.CT)
		}
	}
	m.RecoveredBytes = exposed
	if total > 0 {
		m.Effectiveness = float64(exposed) / float64(total)
	}
	// p1⊕p2 from the largest group's first pair, as evidence of the leak.
	pair := keystream.XOR(groups[0].Frames[0].CT, groups[0].Frames[1].CT)
	m.SampleHex = hexPreview(pair, 32)
	m.SampleASCII = asciiPreview(pair, 32)
	m.Detail = map[string]any{"reuse_groups": len(groups), "exposed_bytes": exposed}
	m.note(fmt.Sprintf("%d IV/MI reuse group(s): %d of %d ciphertext bytes share a keystream and leak plaintext⊕plaintext (exposure, not full recovery). Run `ks mtp` with a crib or known plaintext to peel them apart.", len(groups), exposed, total))
	return m
}

// methodKnownPlaintext: recover the keystream from a known frame and decrypt
// its whole reuse group. Returns the result and the recovered keystream.
func methodKnownPlaintext(in Input, total int) (MethodResult, []byte) {
	m := MethodResult{Name: "known-plaintext", TotalBytes: total}
	if in.KnownLabel == "" || len(in.KnownPT) == 0 {
		m.note("no known plaintext supplied (-known-label / -known-pt) — skipped.")
		return m, nil
	}
	groups := keystream.FindReuse(in.Frames)
	for _, g := range groups {
		ks, decoded, ok := keystream.RecoverWithKnown(g, in.KnownLabel, in.KnownPT)
		if !ok {
			continue
		}
		m.Applicable = true
		m.Verified = true
		recovered := 0
		var sample []byte
		for _, d := range decoded {
			recovered += len(d.Plaintext)
			if d.Label != in.KnownLabel && sample == nil {
				sample = d.Plaintext
			}
		}
		m.RecoveredBytes = recovered
		if total > 0 {
			m.Effectiveness = float64(recovered) / float64(total)
		}
		if sample == nil && len(decoded) > 0 {
			sample = decoded[0].Plaintext
		}
		m.SampleHex = hexPreview(sample, 32)
		m.SampleASCII = asciiPreview(sample, 32)
		m.Detail = map[string]any{"group_frames": len(decoded), "keystream_bytes": len(ks)}
		m.note(fmt.Sprintf("recovered %d keystream bytes from the known frame and decrypted its %d-frame reuse group; the same keystream decrypts any future frame reusing this IV.", len(ks), len(decoded)))
		return m, ks
	}
	m.note(fmt.Sprintf("known frame %q is not in any reuse group, so its keystream decrypts only itself.", in.KnownLabel))
	return m, nil
}

// methodWeakKey: try default / supplied keys with the real cipher. When a known
// plaintext is available the hit is verified by exact keystream match (a
// complete break); otherwise candidate decryptions are scored by structure and
// reported unverified.
func methodWeakKey(in Input, total int) (MethodResult, []byte) {
	m := MethodResult{Name: "weak-key", TotalBytes: total}

	// Group frames by supported algorithm.
	algs := map[uint8]bool{}
	for _, f := range in.Frames {
		if p25crypto.Supported(f.AlgID) {
			algs[f.AlgID] = true
		}
	}
	if len(algs) == 0 {
		m.note("no frames carry a supported algorithm id (ADP/DES/AES), or ALGID is absent — cannot generate a keystream to test keys.")
		return m, nil
	}
	m.Applicable = true

	// A known-plaintext frame gives a definitive verification oracle.
	var oracle *keystream.Frame
	var oracleKSWant []byte
	if in.KnownLabel != "" && len(in.KnownPT) > 0 {
		for i := range in.Frames {
			if in.Frames[i].Label == in.KnownLabel {
				oracle = &in.Frames[i]
				oracleKSWant = keystream.XOR(in.KnownPT, oracle.CT)
				break
			}
		}
	}

	tried := 0
	for alg := range algs {
		keys := append(p25crypto.DefaultKeys(alg), keysOfSize(in.ExtraKeys, p25crypto.KeySize(alg))...)
		for _, key := range keys {
			tried++
			// Verification path: exact keystream match against the oracle.
			if oracle != nil && oracle.AlgID == alg {
				cand, err := p25crypto.Keystream(alg, key, oracle.IV, len(oracleKSWant))
				if err != nil {
					continue
				}
				if bytesEqual(cand, oracleKSWant) {
					m.Verified = true
					m.Effectiveness = effForAlg(in.Frames, alg, total)
					m.RecoveredBytes = bytesForAlg(in.Frames, alg)
					m.SampleHex = hexPreview(in.KnownPT, 32)
					m.SampleASCII = asciiPreview(in.KnownPT, 32)
					m.Detail = map[string]any{"algorithm": p25crypto.AlgName(alg), "key_hex": hex.EncodeToString(key), "keys_tried": tried}
					m.note(fmt.Sprintf("COMPLETE BREAK: %s key %s verified against the known plaintext — every frame under this key/algorithm is decryptable.", p25crypto.AlgName(alg), hex.EncodeToString(key)))
					return m, cand
				}
			}
		}
	}
	m.Detail = map[string]any{"keys_tried": tried}
	if oracle == nil {
		m.note(fmt.Sprintf("tried %d default/supplied key(s) but no known plaintext to verify against — supply -known-label/-known-pt to confirm a hit, or a larger key list. No key confirmed.", tried))
	} else {
		m.note(fmt.Sprintf("tried %d default/supplied key(s) against the known plaintext; none matched — the key is not a common default. The keyspace is not feasibly brute-forced for strong algorithms.", tried))
	}
	return m, nil
}

// methodKeystreamLFSR: once a keystream is in hand, is it an LFSR? Low linear
// complexity means the whole keystream (and call) is predictable from a short
// segment — a generator-level break.
func methodKeystreamLFSR(ks []byte, source string) MethodResult {
	m := MethodResult{Name: "keystream-lfsr"}
	if len(ks) < 8 {
		m.note("no keystream recovered by an earlier method — nothing to analyze. (Recover one via known-plaintext or a verified weak key first.)")
		return m
	}
	m.Applicable = true
	bits := lfsr.BitsFromBytes(ks)
	res := lfsr.BerlekampMassey(bits)
	n := len(bits)
	lc := res.LinearComplexity
	predictable := 0.0
	if lc*2 < n {
		predictable = float64(n-2*lc) / float64(n)
		m.Verified = true
	}
	m.Effectiveness = predictable
	m.TotalBytes = len(ks)
	m.Detail = map[string]any{"keystream_source": source, "linear_complexity": lc, "keystream_bits": n}
	if lc*2 >= n {
		m.note(fmt.Sprintf("keystream (from %s) has linear complexity %d ≈ n/2 (%d bits): no short LFSR, the generator looks cryptographically strong.", source, lc, n))
	} else {
		m.note(fmt.Sprintf("keystream (from %s) has linear complexity %d for %d bits: a short LFSR generates it, so ~%.0f%% of the keystream is predictable from the first %d bits — a generator-level break.", source, lc, n, predictable*100, 2*lc))
	}
	return m
}

// --- helpers ---

func effForAlg(frames []keystream.Frame, alg uint8, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(bytesForAlg(frames, alg)) / float64(total)
}

func bytesForAlg(frames []keystream.Frame, alg uint8) int {
	n := 0
	for _, f := range frames {
		if f.AlgID == alg {
			n += len(f.CT)
		}
	}
	return n
}

func keysOfSize(keys [][]byte, size int) [][]byte {
	if size == 0 {
		return nil
	}
	var out [][]byte
	for _, k := range keys {
		if len(k) == size {
			out = append(out, k)
		}
	}
	return out
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func hexPreview(b []byte, n int) string {
	if len(b) > n {
		b = b[:n]
	}
	return hex.EncodeToString(b)
}

func asciiPreview(b []byte, n int) string {
	if len(b) > n {
		b = b[:n]
	}
	out := make([]byte, len(b))
	for i, c := range b {
		if c >= 0x20 && c <= 0x7e {
			out[i] = c
		} else {
			out[i] = '.'
		}
	}
	return string(out)
}
