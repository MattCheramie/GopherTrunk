package siglab

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/blind"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/siglab/sigref"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// IdentifyConfig configures a signal-identification scan over a capture.
type IdentifyConfig struct {
	SampleRateHz float64
	Format       SampleFormat
	AutoTune     bool
	Conjugate    bool
	IQCorrect    bool
	// MaxSamples caps the input samples each candidate processes (0 ⇒ whole
	// capture). Set it to scan a fast prefix when identifying.
	MaxSamples int64
	// Candidates restricts the protocols tried. nil ⇒ every registered
	// protocol whose channel rate fits the capture sample rate.
	Candidates []trunking.Protocol
	Log        *slog.Logger
}

// CandidateScore is one protocol's identification evidence and composite score.
type CandidateScore struct {
	Protocol       string  `json:"protocol"`
	Locked         bool    `json:"locked"`
	LockLatencySec float64 `json:"lock_latency_sec"`
	SyncHits       int     `json:"sync_hits"`
	SyncVariant    string  `json:"sync_variant"`
	ModalSpacing   int     `json:"modal_spacing"`
	FECPassRate    float64 `json:"fec_pass_rate"`
	Score          float64 `json:"score"`

	result *Result // the candidate's run result, reused by callers
}

// IdentifyResult ranks the candidates and names the most likely protocol.
type IdentifyResult struct {
	Source       string  `json:"source"`
	SampleRateHz float64 `json:"sample_rate_hz"`
	Winner       string  `json:"winner"`
	Confidence   float64 `json:"confidence"`
	Inconclusive bool    `json:"inconclusive"`
	// TuneHz is the carrier offset the winning candidate locked at — the one
	// the decode should reuse (under -auto-tune this may be an off-centre,
	// non-dominant carrier, not the band's loudest). 0 when no shift was used.
	TuneHz     float64          `json:"tune_hz"`
	Candidates []CandidateScore `json:"candidates"`

	// Blind / ReferenceMatches are the offline-signal-ID fallback, populated
	// only when the decode-based identification is Inconclusive: the blind
	// symbol-rate/modulation estimate and the reference-database candidates it
	// ranks (so an undecodable carrier is still named a best guess).
	Blind            *blind.BlindEstimate `json:"blind,omitempty"`
	ReferenceMatches []sigref.Match       `json:"reference_matches,omitempty"`
}

// maxIdentifyCarrierCandidates caps how many carrier offsets the auto-tune
// identify loop tries before giving up. Each carrier costs one prefix run per
// protocol, so this bounds the worst case; the loop also early-exits as soon as
// a carrier clears the confidence floor.
const maxIdentifyCarrierCandidates = 6

// identifyConfidenceFloor is the minimum confidence below which an
// identification is reported as inconclusive — a winner with weak,
// uncorroborated evidence (a bare cross-protocol false lock, or a protocol
// whose sync GopherTrunk can't see) lands here, surfaced as a best guess.
const identifyConfidenceFloor = 0.40

// Identify scans path against candidate protocols and returns a ranked result.
// Each candidate is run through the engine — over a bounded prefix when
// MaxSamples is set — and scored on lock + sync-landscape evidence + FEC pass
// rate. The winner's run result is retained for WinnerResult.
//
// It is a thin wrapper over IdentifyReader: the file is opened once and each
// candidate seeks back to the start, so the capture is read from disk a single
// time rather than re-opened per protocol.
func Identify(path string, cfg IdentifyConfig) (*IdentifyResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	return IdentifyReader(f, path, cfg)
}

// IdentifyIQ identifies an in-memory IQ buffer without any disk round-trip. It
// encodes the samples once in cfg.Format and delegates to IdentifyReader. This
// is the path the live hunt sweeper uses: capture a short slice off the SDR at
// a candidate frequency, then identify it in memory.
func IdentifyIQ(iq []complex64, source string, cfg IdentifyConfig) (*IdentifyResult, error) {
	buf := EncodeCapture(iq, cfg.Format)
	return IdentifyReader(bytes.NewReader(buf), source, cfg)
}

// IdentifyReader scans the capture read from r against candidate protocols and
// returns a ranked result. r is read once per candidate and rewound between
// candidates via Seek, so the underlying bytes are materialized a single time
// (an *os.File or *bytes.Reader both satisfy io.ReadSeeker, and the seek keeps
// AutoTune working). It is the shared core behind Identify and IdentifyIQ.
func IdentifyReader(r io.ReadSeeker, source string, cfg IdentifyConfig) (*IdentifyResult, error) {
	if cfg.SampleRateHz <= 0 {
		return nil, fmt.Errorf("siglab: identify sample rate must be > 0")
	}
	candidates := cfg.Candidates
	if candidates == nil {
		// Try every registered protocol. A protocol whose channel rate is far
		// above the capture rate simply won't lock (too few samples/symbol
		// after the DDC pass-through), so it scores itself out rather than
		// needing an explicit rate filter — and a capture sampled at or below
		// a protocol's target (e.g. a 72 kHz TETRA grab) still decodes.
		candidates = ccdecoder.RegisteredProtocols()
	}

	// Resolve the carrier offsets to try. Without auto-tune the capture is
	// taken as already centred (one pass at 0 Hz — identical to the historical
	// single-pass behaviour). With auto-tune, detect the candidate carriers
	// once up front and try each: the loudest carrier may not be the control
	// channel, so the loop keeps going until one carrier identifies (clears the
	// confidence floor) or the candidates are exhausted.
	offsets := []float64{0}
	if cfg.AutoTune {
		decode, bytesPerSample := cfg.Format.Decoder()
		cands, err := estimateCaptureCarrierCandidates(r, decode, bytesPerSample, cfg.SampleRateHz, maxIdentifyCarrierCandidates)
		if err != nil {
			return nil, fmt.Errorf("siglab: identify auto-tune: %w", err)
		}
		if len(cands) > 0 {
			offsets = offsets[:0]
			for _, c := range cands {
				offsets = append(offsets, c.OffsetHz)
			}
		}
	}

	out := &IdentifyResult{Source: filepath.Base(source), SampleRateHz: cfg.SampleRateHz}
	var bestScores []CandidateScore
	bestConf := -1.0
	bestOffset := 0.0
	bestLocked := false
	for _, off := range offsets {
		scores, err := scanProtocolsAtOffset(r, source, cfg, candidates, off)
		if err != nil {
			return nil, err
		}
		if len(scores) == 0 {
			continue
		}
		sort.SliceStable(scores, func(i, j int) bool { return scores[i].Score > scores[j].Score })
		conf := confidenceOf(scores)
		locked := scores[0].Locked
		// Prefer a carrier whose winner actually LOCKS over one that merely
		// looks plausible by cadence. The loudest carrier in a trunked system
		// is often a traffic channel (same modulation + sync as the control
		// channel, so it clears the confidence floor) but carries no control
		// data — only the control channel locks. Among equal lock states the
		// higher confidence wins.
		better := bestScores == nil
		if locked != bestLocked {
			better = locked
		} else if conf > bestConf {
			better = true
		}
		if better {
			bestLocked, bestConf, bestScores, bestOffset = locked, conf, scores, off
		}
		// A locked winner above the floor is definitive — no weaker candidate
		// can beat a real control-channel lock, so stop searching.
		if locked && conf >= identifyConfidenceFloor {
			break
		}
	}
	if len(bestScores) == 0 {
		return nil, fmt.Errorf("siglab: identify produced no candidate scores")
	}
	out.Candidates = bestScores
	out.TuneHz = bestOffset
	out.Winner = bestScores[0].Protocol
	out.Confidence = bestConf
	out.Inconclusive = bestConf < identifyConfidenceFloor
	if out.Inconclusive {
		// Nothing decoded with confidence — fall back to the offline reference
		// DB: blindly estimate the carrier's symbol rate / modulation and name
		// the closest catalog signals. Best-effort; a read error just leaves
		// the fallback empty.
		attachReferenceFallback(r, cfg, out)
	}
	return out, nil
}

// attachReferenceFallback runs the blind estimator over a prefix of the capture
// and ranks it against the signal-ID reference database, attaching the result to
// out. It is invoked only for an inconclusive identification, so the
// decode-based happy path is untouched.
func attachReferenceFallback(r io.ReadSeeker, cfg IdentifyConfig, out *IdentifyResult) {
	// The scan loop leaves r at EOF; rewind before reading the prefix.
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return
	}
	decode, bytesPerSample := cfg.Format.Decoder()
	iq, err := readCarrierPrefix(r, decode, bytesPerSample)
	if err != nil || len(iq) == 0 {
		return
	}
	est := blind.Estimate(iq, cfg.SampleRateHz)
	if est.SymbolRateHz <= 0 {
		return
	}
	out.Blind = &est
	out.ReferenceMatches = sigref.Rank(sigref.Observation{
		SymbolRateHz:   est.SymbolRateHz,
		SymbolRateConf: est.SymbolRateConf,
		ModClass:       est.ModClass,
		Levels:         est.Levels,
		ChannelBWHz:    est.OccupiedBWHz,
	}, 5)
}

// scanProtocolsAtOffset runs every candidate protocol over the capture with the
// carrier tuned to a fixed offset (auto-tune resolved once by the caller, not
// re-estimated per protocol), and returns the scored candidates.
func scanProtocolsAtOffset(r io.ReadSeeker, source string, cfg IdentifyConfig, candidates []trunking.Protocol, tuneHz float64) ([]CandidateScore, error) {
	scores := make([]CandidateScore, 0, len(candidates))
	for _, p := range candidates {
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("siglab: identify rewind: %w", err)
		}
		res, err := RunReader(r, source, Config{
			Protocol:      p,
			SystemName:    "identify",
			SampleRateHz:  cfg.SampleRateHz,
			Format:        cfg.Format,
			TuneHz:        tuneHz,
			AutoTune:      false,
			Conjugate:     cfg.Conjugate,
			IQCorrect:     cfg.IQCorrect,
			MaxSamples:    cfg.MaxSamples,
			CollectIQDiag: true,
			Log:           cfg.Log,
		})
		if err != nil {
			continue // a candidate that errors out is simply dropped
		}
		scores = append(scores, scoreCandidate(p, res))
	}
	return scores, nil
}

// WinnerResult returns the run Result for the winning candidate (the prefix
// scan), or nil. Callers wanting a full-capture analysis should re-run
// siglab.Run with MaxSamples=0 for the winning protocol.
func (r *IdentifyResult) WinnerResult() *Result {
	for i := range r.Candidates {
		if r.Candidates[i].Protocol == r.Winner {
			return r.Candidates[i].result
		}
	}
	return nil
}

// WinnerProtocol parses the winning protocol name back to a trunking.Protocol.
func (r *IdentifyResult) WinnerProtocol() (trunking.Protocol, error) {
	return trunking.ParseProtocol(r.Winner)
}

// scoreCandidate computes a 0..~1 composite score from a candidate's run.
// Weighting: lock (0.35) + sync evidence (0.40) + FEC pass (0.25), plus a small
// DMR-conventional tie-break so a direct-mode (DM-sync) capture prefers Tier I
// and a base-station (BS-sync) capture prefers Tier II when both lock.
func scoreCandidate(p trunking.Protocol, r *Result) CandidateScore {
	hits, syncLen, modal, variant, fecPass := detailEvidence(r)

	normHits := float64(hits) / 20.0
	if normHits > 1 {
		normHits = 1
	}
	// Sync hits only count as real evidence when they recur at a genuine
	// frame cadence — a modal spacing at least as large as the sync word
	// itself. The false-lock failure mode is many chance correlations at a
	// tiny/degenerate spacing (e.g. every 2 symbols), which is heavily
	// discounted here.
	modalGate := 0.1
	switch {
	case modal >= syncLen && syncLen > 0:
		modalGate = 1.0
	case modal > 0:
		modalGate = 0.3
	}
	syncScore := normHits * modalGate

	// Sync cadence is the most reliable, key-independent evidence (a
	// descrambled protocol like TETRA can't lock or pass CRC without its
	// colour code, but its sync still stands out); weight it highest.
	score := 0.0
	if r.Locked {
		score += 0.25
	}
	score += 0.45 * syncScore
	score += 0.30 * fecPass
	// A recovered grant is strong, higher-layer evidence the decode is real
	// (e.g. a DMR Voice LC Header that passed BPTC + RS + FLC parse) — it
	// distinguishes Tier II conventional (grants on VLC) from a Tier III
	// broadcast capture (no grant) when both lock on the shared slot type.
	if len(r.Grants) > 0 {
		score += 0.10
	}
	score += dmrVariantBonus(p, variant)

	return CandidateScore{
		Protocol:       p.String(),
		Locked:         r.Locked,
		LockLatencySec: r.LockLatencySec,
		SyncHits:       hits,
		SyncVariant:    variant,
		ModalSpacing:   modal,
		FECPassRate:    fecPass,
		Score:          score,
		result:         r,
	}
}

// detailEvidence pulls the sync-landscape + FEC evidence out of a run's Detail,
// handling both the generic *ProtocolDetail and P25 Phase 1's *P25P1Detail.
func detailEvidence(r *Result) (hits, syncLen, modal int, variant string, fecPass float64) {
	switch d := r.Detail.(type) {
	case *ProtocolDetail:
		if d.Sync != nil {
			hits, syncLen, modal, variant = d.Sync.WinnerHits, d.Sync.SyncLen, d.Sync.ModalSpacing, d.Sync.WinnerVariant
		}
		// Count only cleanly-decoded or CRC-validated frames as FEC evidence
		// — NOT "corrected". A short FEC like the DMR slot-type Hamming(20,8)
		// "corrects" almost any 20-bit input to some valid codeword, so
		// crediting corrections would inflate a cross-protocol false lock.
		var frames, ok float64
		for _, f := range d.FEC {
			frames += float64(f.Frames)
			ok += float64(f.Clean + f.CRCPass)
		}
		if frames > 0 {
			fecPass = ok / frames
		}
	case *P25P1Detail:
		// The P25 FSW landscape already filters to genuine frame hits, so a
		// non-trivial WinningHits implies a real cadence; signal that with a
		// modal ≥ syncLen sentinel so it clears the cadence gate.
		hits, syncLen, variant = d.WinningHits, p25FSWLen, fmt.Sprintf("rot%d", d.WinningRotation)
		if d.WinningHits > 2 {
			modal = p25FSWLen
		}
		if cc := d.CCStats; cc != nil {
			att := cc.NIDTrusted + cc.NIDMarginal + cc.NIDFailed
			if att > 0 {
				fecPass = float64(cc.NIDTrusted+cc.NIDMarginal) / float64(att)
			}
		}
	}
	return hits, syncLen, modal, variant, fecPass
}

// dmrVariantBonus breaks the Tier I / Tier II tie on a direct-mode capture:
// when the winning sync is a DM (direct-mode) word, prefer Tier I — Tier II's
// permissive all-sync detector also locks it, but a DM sync means direct mode.
// (Tier III vs Tier II is broken instead by the grant bonus + registration
// order, since both use the base-station sync.)
func dmrVariantBonus(p trunking.Protocol, variant string) float64 {
	if p == trunking.ProtocolDMRTier1 && strings.HasPrefix(variant, "DM-") {
		return 0.05
	}
	return 0
}

// confidenceOf blends the top score with its margin over the runner-up so a
// clear, well-separated winner reads as high confidence.
func confidenceOf(cands []CandidateScore) float64 {
	if len(cands) == 0 {
		return 0
	}
	top := clamp01(cands[0].Score)
	if len(cands) == 1 {
		return top
	}
	margin := clamp01((cands[0].Score - cands[1].Score) / 0.3)
	return clamp01(0.7*top + 0.3*margin)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
