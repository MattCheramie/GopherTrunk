package siglab

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Acceptance is the expected-outcome contract a capture is graded against by
// the `test` harness. A nil field means "don't check this dimension". It
// generalizes the per-test assertions the integration_cc_*_test.go files
// and samples/README.md acceptance criteria describe into one schema.
type Acceptance struct {
	// Lock, when non-nil, requires the run to (true) or to not (false) lock
	// the control channel.
	Lock *bool `json:"lock,omitempty" yaml:"lock,omitempty"`
	// LockLatencyMaxSec caps the time-to-first-lock (0 ⇒ unchecked).
	LockLatencyMaxSec float64 `json:"lock_latency_max_sec,omitempty" yaml:"lock_latency_max_sec,omitempty"`
	// LockFields are expected lock-payload fields (NAC, ColorCode,
	// SystemID, …). Values may be given as numbers or hex strings
	// ("0x293"); comparison is hex-tolerant. Matched as a subset of the
	// observed lock fields (and the common frequency_hz).
	LockFields map[string]any `json:"lock_fields,omitempty" yaml:"lock_fields,omitempty"`
	// MinGrants requires at least this many grants (0 ⇒ unchecked).
	MinGrants int `json:"min_grants,omitempty" yaml:"min_grants,omitempty"`
	// BaudTolerancePct caps |effective − expected| / expected (0 ⇒ unchecked).
	BaudTolerancePct float64 `json:"baud_tolerance_pct,omitempty" yaml:"baud_tolerance_pct,omitempty"`
	// MaxDecodeErrorRate caps decode errors per 1000 symbols (0 ⇒ unchecked).
	MaxDecodeErrorRate float64 `json:"max_decode_error_rate,omitempty" yaml:"max_decode_error_rate,omitempty"`
	// MaxEVMPct caps the demod error-vector magnitude percentage (0 ⇒
	// unchecked). Grades demod quality directly — a capture whose eye is too
	// closed fails even if it happens to lock.
	MaxEVMPct float64 `json:"max_evm_pct,omitempty" yaml:"max_evm_pct,omitempty"`
	// MinSNRdB requires the estimated demod SNR to be at least this many dB
	// (0 ⇒ unchecked).
	MinSNRdB float64 `json:"min_snr_db,omitempty" yaml:"min_snr_db,omitempty"`
}

// Verdict is the pass/fail outcome of grading a Result against an Acceptance.
type Verdict struct {
	Pass     bool     `json:"pass" yaml:"pass"`
	Failures []string `json:"failures" yaml:"failures"`
	Checks   []string `json:"checks" yaml:"checks"`
}

// evaluateAcceptance grades r against a, returning a Verdict. Every checked
// dimension contributes a line to Checks; failures also append to Failures.
func evaluateAcceptance(r *Result, a *Acceptance) *Verdict {
	v := &Verdict{Pass: true}
	fail := func(msg string) {
		v.Pass = false
		v.Failures = append(v.Failures, msg)
		v.Checks = append(v.Checks, "FAIL: "+msg)
	}
	pass := func(msg string) { v.Checks = append(v.Checks, "ok: "+msg) }

	if a.Lock != nil {
		want := *a.Lock
		if r.Locked != want {
			fail(fmt.Sprintf("lock: want %v, got %v", want, r.Locked))
		} else {
			pass(fmt.Sprintf("lock = %v", want))
		}
	}
	if a.LockLatencyMaxSec > 0 {
		switch {
		case !r.Locked:
			fail(fmt.Sprintf("lock latency: required ≤ %.2fs but never locked", a.LockLatencyMaxSec))
		case r.LockLatencySec > a.LockLatencyMaxSec:
			fail(fmt.Sprintf("lock latency: %.2fs > %.2fs", r.LockLatencySec, a.LockLatencyMaxSec))
		default:
			pass(fmt.Sprintf("lock latency %.2fs ≤ %.2fs", r.LockLatencySec, a.LockLatencyMaxSec))
		}
	}
	for k, want := range a.LockFields {
		got, ok := lookupLockField(r.Lock, k)
		if !ok {
			fail(fmt.Sprintf("lock field %q: missing (no lock or field absent)", k))
			continue
		}
		if !fieldMatches(want, got) {
			fail(fmt.Sprintf("lock field %q: want %v, got %v", k, want, got))
		} else {
			pass(fmt.Sprintf("lock field %q = %v", k, want))
		}
	}
	if a.MinGrants > 0 {
		if len(r.Grants) < a.MinGrants {
			fail(fmt.Sprintf("grants: %d < %d", len(r.Grants), a.MinGrants))
		} else {
			pass(fmt.Sprintf("grants %d ≥ %d", len(r.Grants), a.MinGrants))
		}
	}
	if a.BaudTolerancePct > 0 {
		if r.ExpectedBaud <= 0 {
			pass("baud tolerance: skipped (no nominal baud for protocol)")
		} else if math.Abs(r.BaudDeviationPct) > a.BaudTolerancePct {
			fail(fmt.Sprintf("baud deviation: %.2f%% > %.2f%%", math.Abs(r.BaudDeviationPct), a.BaudTolerancePct))
		} else {
			pass(fmt.Sprintf("baud deviation %.2f%% ≤ %.2f%%", math.Abs(r.BaudDeviationPct), a.BaudTolerancePct))
		}
	}
	if a.MaxDecodeErrorRate > 0 {
		rate := 0.0
		if r.Signal != nil {
			rate = r.Signal.DecodeErrorRate
		}
		if rate > a.MaxDecodeErrorRate {
			fail(fmt.Sprintf("decode-error rate: %.2f/ksym > %.2f/ksym", rate, a.MaxDecodeErrorRate))
		} else {
			pass(fmt.Sprintf("decode-error rate %.2f/ksym ≤ %.2f/ksym", rate, a.MaxDecodeErrorRate))
		}
	}
	if a.MaxEVMPct > 0 {
		switch {
		case r.Signal == nil || r.Signal.Demod == nil:
			fail(fmt.Sprintf("EVM: required ≤ %.1f%% but no demod metrics were produced", a.MaxEVMPct))
		case r.Signal.Demod.EVMPct > a.MaxEVMPct:
			fail(fmt.Sprintf("EVM: %.1f%% > %.1f%%", r.Signal.Demod.EVMPct, a.MaxEVMPct))
		default:
			pass(fmt.Sprintf("EVM %.1f%% ≤ %.1f%%", r.Signal.Demod.EVMPct, a.MaxEVMPct))
		}
	}
	if a.MinSNRdB > 0 {
		switch {
		case r.Signal == nil || r.Signal.Demod == nil:
			fail(fmt.Sprintf("SNR: required ≥ %.1f dB but no demod metrics were produced", a.MinSNRdB))
		case r.Signal.Demod.SNREstimateDB < a.MinSNRdB:
			fail(fmt.Sprintf("SNR: %.1f dB < %.1f dB", r.Signal.Demod.SNREstimateDB, a.MinSNRdB))
		default:
			pass(fmt.Sprintf("SNR %.1f dB ≥ %.1f dB", r.Signal.Demod.SNREstimateDB, a.MinSNRdB))
		}
	}
	return v
}

// lookupLockField finds a lock field by name, tolerating the snake_case the
// metadata schema uses vs the Go field names the flattener emits (e.g.
// "system_id" ↔ "SystemID", "frequency_hz" ↔ the dedicated FrequencyHz).
func lookupLockField(lock *LockInfo, key string) (any, bool) {
	if lock == nil {
		return nil, false
	}
	if normKey(key) == "frequencyhz" {
		return lock.FrequencyHz, true
	}
	for k, v := range lock.Fields {
		if normKey(k) == normKey(key) {
			return v, true
		}
	}
	return nil, false
}

func normKey(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, "_", ""))
}

// fieldMatches compares an expected acceptance value (number or hex/decimal
// string) to an observed value (typically a reflected integer), coercing
// both to uint64 when possible so 0x293, "659" and uint16(659) all match.
// Falls back to string equality for non-numeric values.
func fieldMatches(expected, got any) bool {
	en, eok := toUint64(expected)
	gn, gok := toUint64(got)
	if eok && gok {
		return en == gn
	}
	return fmt.Sprintf("%v", expected) == fmt.Sprintf("%v", got)
}

// toUint64 coerces numbers and hex/decimal strings to uint64.
func toUint64(v any) (uint64, bool) {
	switch n := v.(type) {
	case uint64:
		return n, true
	case uint:
		return uint64(n), true
	case uint32:
		return uint64(n), true
	case uint16:
		return uint64(n), true
	case uint8:
		return uint64(n), true
	case int:
		return uint64(n), true
	case int64:
		return uint64(n), true
	case int32:
		return uint64(n), true
	case int16:
		return uint64(n), true
	case int8:
		return uint64(n), true
	case float64:
		return uint64(n), true
	case float32:
		return uint64(n), true
	case string:
		s := strings.TrimSpace(n)
		if strings.HasPrefix(strings.ToLower(s), "0x") {
			if u, err := strconv.ParseUint(s[2:], 16, 64); err == nil {
				return u, true
			}
			return 0, false
		}
		if u, err := strconv.ParseUint(s, 10, 64); err == nil {
			return u, true
		}
		return 0, false
	default:
		return 0, false
	}
}
