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
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/cipherinfo"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/keystream"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/lfsr"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/p25crypto"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/randomness"
)

// Input bundles the material the harness works from.
type Input struct {
	Frames     []keystream.Frame
	Protocol   string   // protocol whose algorithm-id namespace applies (default "p25")
	KnownLabel string   // label of a frame whose plaintext is known (optional)
	KnownPT    []byte   // that frame's plaintext (optional)
	ExtraKeys  [][]byte // additional candidate keys to try in the weak-key method
	// BruteBits enables the reduced-keyspace brute method: search the low
	// BruteBits of the key against a known-plaintext oracle. 0 disables it.
	BruteBits int
	// BaseKey is the fixed remainder of the key for the brute (the high bits);
	// nil means all-zero.
	BaseKey []byte
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
	weakness, weaknessFloor := methodKnownWeakness(in)
	reuse := methodIVReuse(in.Frames, total)
	known, knownKS := methodKnownPlaintext(in, total)
	weak, weakKS := methodWeakKey(in, total)
	brute, bruteKS := methodKeyBrute(in, total)

	// The keystream-LFSR method runs only on a genuine keystream a stronger
	// method recovered (known-plaintext, a verified weak key, or a brute hit).
	// A reuse XOR (p1⊕p2) is plaintext, not keystream, so it is deliberately
	// not fed here.
	ks, ksSource := knownKS, "known-plaintext"
	if ks == nil && weakKS != nil {
		ks, ksSource = weakKS, "weak-key"
	}
	if ks == nil && bruteKS != nil {
		ks, ksSource = bruteKS, "key-brute"
	}
	lc := methodKeystreamLFSR(ks, ksSource)

	rep.Methods = []MethodResult{cipherStrength, weakness, reuse, known, weak, brute, lc}

	// Overall = the strongest result; verdict is BROKEN only on a verified,
	// (near-)complete decryption. A published-weakness floor lifts the verdict
	// off RESISTANT when the deployment uses an algorithm with a known break,
	// even if our harness did not carry the cipher to exploit it.
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
	case rep.OverallEffectiveness > 0 || weaknessFloor:
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

// methodKnownWeakness: consult the published-weakness knowledge base for every
// algorithm id present and report its design weaknesses. This is the advisory
// layer — it recovers no bytes, but it tells the operator when a deployment is
// using an algorithm with a known break (e.g. TETRA TEA1's 32-bit backdoor)
// that this harness may not carry the cipher to exploit. Returns the method
// result and whether a broken/brute-forceable algorithm was seen (the verdict
// floor).
func methodKnownWeakness(in Input) (MethodResult, bool) {
	m := MethodResult{Name: "known-weakness"}
	proto := in.Protocol
	if proto == "" {
		proto = "p25"
	}
	// Distinct algorithm ids, in stable order.
	seen := map[uint8]bool{}
	var algs []uint8
	for _, f := range in.Frames {
		if !seen[f.AlgID] {
			seen[f.AlgID] = true
			algs = append(algs, f.AlgID)
		}
	}
	sort.Slice(algs, func(i, j int) bool { return algs[i] < algs[j] })

	var weakest cipherinfo.Strength = cipherinfo.StrengthUnknown
	floor := false
	details := map[string]any{"protocol": proto}
	known := 0
	for _, alg := range algs {
		info, ok := cipherinfo.Lookup(proto, alg)
		if !ok {
			m.note(fmt.Sprintf("algorithm 0x%02X: no entry in the %s knowledge base (unrecognised).", alg, proto))
			continue
		}
		known++
		m.Applicable = true
		details[fmt.Sprintf("0x%02X", alg)] = map[string]any{
			"name": info.Name, "family": info.Family,
			"key_bits": info.KeyBits, "effective_key_bits": info.EffectiveKeyBits,
			"strength": string(info.Strength), "brute_forceable": info.BruteForceable,
			"bundled": info.Bundled, "reference": info.Reference,
		}
		switch info.Strength {
		case cipherinfo.StrengthBroken, cipherinfo.StrengthWeak:
			floor = true
		}
		if info.BruteForceable {
			floor = true
		}
		if rank(info.Strength) < rank(weakest) {
			weakest = info.Strength
		}
		msg := fmt.Sprintf("%s (%s, %d-bit", info.Name, info.Family, info.KeyBits)
		if info.EffectiveKeyBits != info.KeyBits {
			msg += fmt.Sprintf(", %d-bit effective", info.EffectiveKeyBits)
		}
		msg += fmt.Sprintf("): %s", strengthVerb(info.Strength))
		if info.Weakness != "" {
			msg += " — " + info.Weakness
		}
		if info.Reference != "" {
			msg += " [" + info.Reference + "]"
		}
		m.note(msg)
	}
	if known == 0 {
		m.note("no recognised algorithm ids on the frames (set -protocol or check the capture's algid field).")
		return m, false
	}
	details["weakest_strength"] = string(weakest)
	m.Detail = details
	return m, floor
}

// methodKeyBrute: reduced-keyspace brute force (the hashcat-analog, and the
// shape of the TETRA TEA1 backdoor attack). With a known-plaintext oracle it
// searches the low BruteBits of the key for the bundled cipher, verified by an
// exact keystream match. For an unbundled but brute-forceable algorithm (e.g.
// TEA1) it reports what the brute *would* do, so the finding is actionable.
func methodKeyBrute(in Input, total int) (MethodResult, []byte) {
	m := MethodResult{Name: "key-brute"}
	if in.BruteBits <= 0 {
		m.note("disabled — set -brute-bits N to search the low N bits of the key against a known-plaintext oracle.")
		return m, nil
	}
	if in.KnownLabel == "" || len(in.KnownPT) == 0 {
		m.note("needs a known-plaintext oracle (-known-label / -known-pt) to verify candidate keys.")
		return m, nil
	}
	var oracle *keystream.Frame
	for i := range in.Frames {
		if in.Frames[i].Label == in.KnownLabel {
			oracle = &in.Frames[i]
			break
		}
	}
	if oracle == nil {
		m.note(fmt.Sprintf("known frame %q not found.", in.KnownLabel))
		return m, nil
	}
	proto := in.Protocol
	if proto == "" {
		proto = "p25"
	}
	if !p25crypto.Supported(oracle.AlgID) {
		// Unbundled cipher: report the brute that would apply (e.g. TEA1).
		if info, ok := cipherinfo.Lookup(proto, oracle.AlgID); ok && info.BruteForceable {
			m.note(fmt.Sprintf("%s has a %d-bit effective keyspace (%s) — a 2^%d brute is feasible, but its keystream function is not bundled here; supply an implementation to run it.",
				info.Name, info.EffectiveKeyBits, info.Reference, info.EffectiveKeyBits))
		} else {
			m.note(fmt.Sprintf("the oracle's algorithm 0x%02X is not bundled, so its keystream cannot be brute-forced here.", oracle.AlgID))
		}
		return m, nil
	}
	if in.BruteBits > 32 {
		m.note(fmt.Sprintf("brute-bits %d capped at 32 (2^%d is the feasibility ceiling).", in.BruteBits, 32))
		in.BruteBits = 32
	}
	m.Applicable = true
	sz := p25crypto.KeySize(oracle.AlgID)
	base := make([]byte, sz)
	copy(base, in.BaseKey)
	want := keystream.XOR(in.KnownPT, oracle.CT)
	if len(want) == 0 {
		m.note("oracle plaintext/ciphertext overlap is empty.")
		return m, nil
	}
	prefix := 8
	if prefix > len(want) {
		prefix = len(want)
	}
	space := uint64(1) << uint(in.BruteBits)
	for i := uint64(0); i < space; i++ {
		key := keyWithLowBits(base, i, in.BruteBits)
		cand, err := p25crypto.Keystream(oracle.AlgID, key, oracle.IV, prefix)
		if err != nil {
			m.note(err.Error())
			return m, nil
		}
		if !bytesEqual(cand, want[:prefix]) {
			continue
		}
		full, _ := p25crypto.Keystream(oracle.AlgID, key, oracle.IV, len(want))
		if !bytesEqual(full, want) {
			continue
		}
		m.Verified = true
		m.Effectiveness = effForAlg(in.Frames, oracle.AlgID, total)
		m.RecoveredBytes = bytesForAlg(in.Frames, oracle.AlgID)
		m.SampleHex = hexPreview(in.KnownPT, 32)
		m.SampleASCII = asciiPreview(in.KnownPT, 32)
		m.Detail = map[string]any{"algorithm": p25crypto.AlgName(oracle.AlgID), "key_hex": hex.EncodeToString(key), "searched": i + 1, "brute_bits": in.BruteBits}
		m.note(fmt.Sprintf("COMPLETE BREAK: brute-forced %s key %s in the low %d bits after %d candidates — the keyspace is too small.", p25crypto.AlgName(oracle.AlgID), hex.EncodeToString(key), in.BruteBits, i+1))
		return m, full
	}
	m.Detail = map[string]any{"algorithm": p25crypto.AlgName(oracle.AlgID), "searched": space, "brute_bits": in.BruteBits}
	m.note(fmt.Sprintf("searched all 2^%d low-bit keys for %s; none matched. The key lies outside this slice of the keyspace.", in.BruteBits, p25crypto.AlgName(oracle.AlgID)))
	return m, nil
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

// keyWithLowBits returns base with its low `bits` overwritten by val (the
// trailing bytes, big-endian). base is not mutated.
func keyWithLowBits(base []byte, val uint64, bits int) []byte {
	k := append([]byte(nil), base...)
	nbytes := (bits + 7) / 8
	for b := 0; b < nbytes && b < len(k); b++ {
		k[len(k)-1-b] = byte(val >> uint(8*b))
	}
	return k
}

// rank orders strengths weakest-first for picking the worst in a set.
func rank(s cipherinfo.Strength) int {
	switch s {
	case cipherinfo.StrengthBroken:
		return 0
	case cipherinfo.StrengthWeak:
		return 1
	case cipherinfo.StrengthLegacy:
		return 2
	case cipherinfo.StrengthStrong:
		return 3
	default:
		return 4
	}
}

func strengthVerb(s cipherinfo.Strength) string {
	switch s {
	case cipherinfo.StrengthBroken:
		return "BROKEN by design / published attack"
	case cipherinfo.StrengthWeak:
		return "WEAK — feasibly brute-forced"
	case cipherinfo.StrengthLegacy:
		return "LEGACY — aging, exhaustible with effort"
	case cipherinfo.StrengthStrong:
		return "no known practical break with a well-managed key"
	default:
		return "clear / unknown"
	}
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
