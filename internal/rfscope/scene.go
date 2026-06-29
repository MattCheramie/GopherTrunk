// Package rfscope is protocol-agnostic RF network analysis — "Wireshark for the
// RF physical layer". Pointed at any band (a recorded IQ capture or a live SDR)
// with no prior knowledge of the technology, modulation, framing, or
// encryption, it produces a structured Scene describing what is on the air and
// how it behaves: an RF protocol hierarchy, per-channel activity/occupancy, burst
// timing and periodicity, an emitter/conversation graph, and an entropy/
// encryption triage that bridges into the cryptolab toolkit.
//
// rfscope sits at the top of the analysis lattice and imports only downward —
// survey, carriers, siglab, hunt, the cryptolab engines, and the dsp
// primitives. None of those import rfscope, so there is no cycle. It adds no DSP
// of its own; it orchestrates the existing primitives and accumulates their
// output into one shared, JSON-exportable Scene (the protocol-agnostic peer of
// hunt.DiscoveredSystem).
package rfscope

import (
	"math"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/survey"
)

// Scene is the accumulating, protocol-agnostic model of an RF environment over a
// capture or a live window. Bursts are the raw atom; Channels, Emitters,
// Conversations, Hierarchy and Anomalies are derived views populated by the
// registered Analyzers. AnalyzerOutputs carries each analyzer's own structured
// detail keyed by analyzer name (mirroring siglab.Result.Detail), so an analyzer
// can emit more than the shared model captures.
type Scene struct {
	Source     string    `json:"source"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	CenterHz   uint32    `json:"center_hz"`
	SpanHz     uint32    `json:"span_hz"`
	// WindowSec is the observation window: how much signal time the scene covers.
	WindowSec float64 `json:"window_sec"`
	// NoiseFloorDbFS is the scene-wide robust noise floor (spectrum.Percentile).
	NoiseFloorDbFS float32 `json:"noise_floor_dbfs"`

	Bursts        []Burst           `json:"bursts,omitempty"`
	Channels      []Channel         `json:"channels,omitempty"`
	Emitters      []Emitter         `json:"emitters,omitempty"`
	Conversations []Conversation    `json:"conversations,omitempty"`
	Hierarchy     ProtocolHierarchy `json:"hierarchy"`
	Anomalies     []Anomaly         `json:"anomalies,omitempty"`

	AnalyzerOutputs map[string]any `json:"analyzer_outputs,omitempty"`
}

// Burst is one onset→offset activity segment on one frequency — the atom every
// downstream statistic is built from. The spectral fields (OccupiedBwHz,
// ChannelPowerDBFS, ACPR*, SpectralFlatness) come from spectrum.ComputeOccupancy
// (PR #831); Class/Features come from survey.Classify.
type Burst struct {
	ID       uint64  `json:"id"`
	FreqHz   uint32  `json:"freq_hz"`
	StartSec float64 `json:"start_sec"`
	EndSec   float64 `json:"end_sec"`
	PeakDbFS float32 `json:"peak_dbfs"`
	SNRDb    float32 `json:"snr_db"`

	OccupiedBwHz     uint32  `json:"occupied_bw_hz"`
	ChannelPowerDBFS float64 `json:"channel_power_dbfs"`
	ACPRUpperDB      float64 `json:"acpr_upper_db"`
	ACPRLowerDB      float64 `json:"acpr_lower_db"`
	// SpectralFlatness is the Wiener entropy 0..1: ≈1 noise-like/spread, ≪1 a
	// tone or narrow carrier — a high-value protocol-agnostic discriminator.
	SpectralFlatness float64 `json:"spectral_flatness"`

	Class    survey.SignalClass   `json:"class"`
	Features survey.ClassFeatures `json:"features"`

	// EmitterID back-references the owning emitter once clustered (0 = none).
	EmitterID uint64 `json:"emitter_id,omitempty"`
}

// DurationSec is the burst's on-air time.
func (b Burst) DurationSec() float64 {
	d := b.EndSec - b.StartSec
	if d < 0 {
		return 0
	}
	return d
}

// Channel is the per-frequency, time-aggregated view — the I/O-graph row.
type Channel struct {
	FreqHz          uint32             `json:"freq_hz"`
	OccupiedBwHz    uint32             `json:"occupied_bw_hz"`
	DominantClass   survey.SignalClass `json:"dominant_class"`
	BurstCount      int                `json:"burst_count"`
	AirtimeSec      float64            `json:"airtime_sec"`
	DutyCycle       float64            `json:"duty_cycle"`
	OccupancyPct    float64            `json:"occupancy_pct"`
	BurstRatePerMin float64            `json:"burst_rate_per_min"`
	MedianPowerDbFS float32            `json:"median_power_dbfs"`
	MedianFlatness  float64            `json:"median_flatness"`

	Timeline []Bin `json:"timeline,omitempty"`

	BurstLenHist     Hist    `json:"burst_len_hist"`
	InterArrivalHist Hist    `json:"inter_arrival_hist"`
	PeriodSec        float64 `json:"period_sec"`
	PeriodConfidence float64 `json:"period_confidence"`
}

// Bin is one time-bucket of a channel's activity timeline.
type Bin struct {
	TSec       float64 `json:"t_sec"`
	PowerDbFS  float32 `json:"power_dbfs"`
	Occupied   bool    `json:"occupied"`
	BurstCount int     `json:"burst_count"`
}

// Hist is a simple histogram pairing bin counts with their edges, matching the
// (counts, edges) shape dsp/stats.Histogram returns.
type Hist struct {
	Counts []int     `json:"counts,omitempty"`
	Edges  []float64 `json:"edges,omitempty"`
}

// Emitter is a cluster of bursts sharing an RF fingerprint — the protocol-
// agnostic generalization of hunt's per-site/per-unit identity.
type Emitter struct {
	ID              uint64      `json:"id"`
	Label           string      `json:"label"`
	Fingerprint     Fingerprint `json:"fingerprint"`
	BurstIDs        []uint64    `json:"burst_ids,omitempty"`
	FirstSec        float64     `json:"first_sec"`
	LastSec         float64     `json:"last_sec"`
	TotalAirtimeSec float64     `json:"total_airtime_sec"`
	// HopSet holds >1 frequency when the emitter is a frequency hopper.
	HopSet []uint32 `json:"hop_set,omitempty"`
}

// Fingerprint is the feature vector emitters are clustered on.
type Fingerprint struct {
	CenterHz          uint32  `json:"center_hz"`
	OccupiedBwHz      uint32  `json:"occupied_bw_hz"`
	ClassFamily       string  `json:"class_family"`
	MedianPowerDbFS   float32 `json:"median_power_dbfs"`
	MedianBurstLenSec float64 `json:"median_burst_len_sec"`
	SpectralFlatness  float64 `json:"spectral_flatness"`
	PeriodSec         float64 `json:"period_sec"`
}

// Conversation links temporally correlated emitters — the generic
// conversation/endpoint graph (request/response, paired TDMA slots, a hop
// sequence, or consistently co-active emitters).
type Conversation struct {
	ID          uint64   `json:"id"`
	EmitterIDs  []uint64 `json:"emitter_ids"`
	Kind        string   `json:"kind"`
	Correlation float64  `json:"correlation"`
	Exchanges   int      `json:"exchanges"`
}

// ProtocolHierarchy is the class→bandwidth→protocol tree, flattened depth-first
// with a Depth field for rendering.
type ProtocolHierarchy struct {
	Total HierStats  `json:"total"`
	Nodes []HierNode `json:"nodes,omitempty"`
}

// HierNode is one row of the flattened hierarchy tree.
type HierNode struct {
	Depth    int                `json:"depth"` // 0=class, 1=bandwidth bucket, 2=protocol candidate
	Label    string             `json:"label"`
	Class    survey.SignalClass `json:"class,omitempty"`
	Protocol string             `json:"protocol,omitempty"`
	Stats    HierStats          `json:"stats"`
}

// HierStats are the aggregate figures attached to a hierarchy node.
type HierStats struct {
	EmitterCount int     `json:"emitter_count"`
	BurstCount   int     `json:"burst_count"`
	AirtimeSec   float64 `json:"airtime_sec"`
	SpectrumPct  float64 `json:"spectrum_pct"`
	TimePct      float64 `json:"time_pct"`
	SymbolVolume int64   `json:"symbol_volume"`
	ByteVolume   int64   `json:"byte_volume"`
}

// Anomaly is one RF expert-info finding.
type Anomaly struct {
	Severity  string `json:"severity"` // "note" | "warn" | "alert"
	Kind      string `json:"kind"`     // "intermittent" | "hopper" | "wide-carrier" | …
	FreqHz    uint32 `json:"freq_hz,omitempty"`
	EmitterID uint64 `json:"emitter_id,omitempty"`
	Detail    string `json:"detail"`
}

// Sanitize clamps every non-finite float in the scene to a safe value so a
// marshal (and the SSE/WS wire that carries it) never sees NaN/±Inf. Call before
// any JSON encode or stream send.
func (s *Scene) Sanitize() {
	s.NoiseFloorDbFS = finite32(s.NoiseFloorDbFS)
	s.WindowSec = finite(s.WindowSec)

	for i := range s.Bursts {
		b := &s.Bursts[i]
		b.StartSec = finite(b.StartSec)
		b.EndSec = finite(b.EndSec)
		b.PeakDbFS = finite32(b.PeakDbFS)
		b.SNRDb = finite32(b.SNRDb)
		b.ChannelPowerDBFS = finite(b.ChannelPowerDBFS)
		b.ACPRUpperDB = finite(b.ACPRUpperDB)
		b.ACPRLowerDB = finite(b.ACPRLowerDB)
		b.SpectralFlatness = finite(b.SpectralFlatness)
		sanitizeFeatures(&b.Features)
	}

	for i := range s.Channels {
		c := &s.Channels[i]
		c.AirtimeSec = finite(c.AirtimeSec)
		c.DutyCycle = finite(c.DutyCycle)
		c.OccupancyPct = finite(c.OccupancyPct)
		c.BurstRatePerMin = finite(c.BurstRatePerMin)
		c.MedianPowerDbFS = finite32(c.MedianPowerDbFS)
		c.MedianFlatness = finite(c.MedianFlatness)
		c.PeriodSec = finite(c.PeriodSec)
		c.PeriodConfidence = finite(c.PeriodConfidence)
		for j := range c.Timeline {
			c.Timeline[j].TSec = finite(c.Timeline[j].TSec)
			c.Timeline[j].PowerDbFS = finite32(c.Timeline[j].PowerDbFS)
		}
		sanitizeHist(&c.BurstLenHist)
		sanitizeHist(&c.InterArrivalHist)
	}

	for i := range s.Emitters {
		e := &s.Emitters[i]
		e.FirstSec = finite(e.FirstSec)
		e.LastSec = finite(e.LastSec)
		e.TotalAirtimeSec = finite(e.TotalAirtimeSec)
		e.Fingerprint.MedianPowerDbFS = finite32(e.Fingerprint.MedianPowerDbFS)
		e.Fingerprint.MedianBurstLenSec = finite(e.Fingerprint.MedianBurstLenSec)
		e.Fingerprint.SpectralFlatness = finite(e.Fingerprint.SpectralFlatness)
		e.Fingerprint.PeriodSec = finite(e.Fingerprint.PeriodSec)
	}

	for i := range s.Conversations {
		s.Conversations[i].Correlation = finite(s.Conversations[i].Correlation)
	}

	sanitizeHierStats(&s.Hierarchy.Total)
	for i := range s.Hierarchy.Nodes {
		sanitizeHierStats(&s.Hierarchy.Nodes[i].Stats)
	}
}

func sanitizeFeatures(f *survey.ClassFeatures) {
	f.SNRDb = finite(f.SNRDb)
	f.EnvelopeCV = finite(f.EnvelopeCV)
	f.IFStd = finite(f.IFStd)
	f.IFKurtosis = finite(f.IFKurtosis)
	f.BaudHz = finite(f.BaudHz)
	f.BaudProminence = finite(f.BaudProminence)
}

func sanitizeHist(h *Hist) {
	for i := range h.Edges {
		h.Edges[i] = finite(h.Edges[i])
	}
}

func sanitizeHierStats(h *HierStats) {
	h.AirtimeSec = finite(h.AirtimeSec)
	h.SpectrumPct = finite(h.SpectrumPct)
	h.TimePct = finite(h.TimePct)
}

// finite returns f when finite, otherwise 0.
func finite(f float64) float64 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	return f
}

func finite32(f float32) float32 {
	if math.IsNaN(float64(f)) || math.IsInf(float64(f), 0) {
		return 0
	}
	return f
}
