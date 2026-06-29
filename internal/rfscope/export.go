package rfscope

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Format is an rfscope scene serialization. It mirrors the multi-format export
// convention the siglab and hunt subcommands use.
type Format int

const (
	// FormatJSON is one indented JSON object for the whole scene.
	FormatJSON Format = iota
	// FormatJSONL is one JSON record per line: a scene header, then each burst,
	// channel, emitter, conversation, and anomaly — a stream-friendly form.
	FormatJSONL
	// FormatYAML is a single YAML document.
	FormatYAML
	// FormatCSV is the burst table (one row per burst).
	FormatCSV
	// FormatSummary is the human-readable hierarchy + channel table report.
	FormatSummary
)

// ParseFormat maps a -format flag value to a Format.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "json", "":
		return FormatJSON, nil
	case "jsonl", "ndjson":
		return FormatJSONL, nil
	case "yaml", "yml":
		return FormatYAML, nil
	case "csv":
		return FormatCSV, nil
	case "summary", "txt", "text", "report":
		return FormatSummary, nil
	default:
		return FormatJSON, fmt.Errorf("rfscope: unknown format %q (want json, jsonl, yaml, csv, or summary)", s)
	}
}

func (f Format) String() string {
	switch f {
	case FormatJSONL:
		return "jsonl"
	case FormatYAML:
		return "yaml"
	case FormatCSV:
		return "csv"
	case FormatSummary:
		return "summary"
	default:
		return "json"
	}
}

// FileExtension returns the conventional file extension for the format.
func (f Format) FileExtension() string {
	switch f {
	case FormatJSONL:
		return "jsonl"
	case FormatYAML:
		return "yaml"
	case FormatCSV:
		return "csv"
	case FormatSummary:
		return "txt"
	default:
		return "json"
	}
}

// Write serializes sc to w in the requested format. It Sanitizes the scene first
// so a stray NaN/Inf never reaches the encoder (or the SSE/WS wire).
func Write(w io.Writer, sc *Scene, f Format) error {
	sc.Sanitize()
	switch f {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(sc)
	case FormatYAML:
		enc := yaml.NewEncoder(w)
		defer enc.Close()
		return enc.Encode(sc)
	case FormatJSONL:
		return writeJSONL(w, sc)
	case FormatCSV:
		return writeCSV(w, sc)
	case FormatSummary:
		return writeSummary(w, sc)
	default:
		return fmt.Errorf("rfscope: unsupported format %v", f)
	}
}

type jsonlRec struct {
	Kind string `json:"kind"`
	Data any    `json:"data"`
}

func writeJSONL(w io.Writer, sc *Scene) error {
	enc := json.NewEncoder(w)
	header := *sc
	header.Bursts, header.Channels, header.Emitters, header.Conversations, header.Anomalies = nil, nil, nil, nil, nil
	recs := []jsonlRec{{Kind: "scene", Data: header}}
	for i := range sc.Bursts {
		recs = append(recs, jsonlRec{Kind: "burst", Data: sc.Bursts[i]})
	}
	for i := range sc.Channels {
		recs = append(recs, jsonlRec{Kind: "channel", Data: sc.Channels[i]})
	}
	for i := range sc.Emitters {
		recs = append(recs, jsonlRec{Kind: "emitter", Data: sc.Emitters[i]})
	}
	for i := range sc.Conversations {
		recs = append(recs, jsonlRec{Kind: "conversation", Data: sc.Conversations[i]})
	}
	for i := range sc.Anomalies {
		recs = append(recs, jsonlRec{Kind: "anomaly", Data: sc.Anomalies[i]})
	}
	for _, r := range recs {
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	return nil
}

func writeCSV(w io.Writer, sc *Scene) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{
		"id", "freq_hz", "start_sec", "end_sec", "duration_sec", "class",
		"occupied_bw_hz", "channel_power_dbfs", "spectral_flatness", "snr_db", "emitter_id",
	}); err != nil {
		return err
	}
	for _, b := range sc.Bursts {
		rec := []string{
			strconv.FormatUint(b.ID, 10),
			strconv.FormatUint(uint64(b.FreqHz), 10),
			strconv.FormatFloat(b.StartSec, 'f', 4, 64),
			strconv.FormatFloat(b.EndSec, 'f', 4, 64),
			strconv.FormatFloat(b.DurationSec(), 'f', 4, 64),
			string(b.Class),
			strconv.FormatUint(uint64(b.OccupiedBwHz), 10),
			strconv.FormatFloat(b.ChannelPowerDBFS, 'f', 2, 64),
			strconv.FormatFloat(b.SpectralFlatness, 'f', 4, 64),
			strconv.FormatFloat(float64(b.SNRDb), 'f', 2, 64),
			strconv.FormatUint(b.EmitterID, 10),
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func writeSummary(w io.Writer, sc *Scene) error {
	bw := &strings.Builder{}
	fmt.Fprintf(bw, "RF Scene")
	if sc.Source != "" {
		fmt.Fprintf(bw, " — %s", sc.Source)
	}
	fmt.Fprintf(bw, "\n")
	fmt.Fprintf(bw, "  center %.6f MHz   span %.3f MHz   window %.2fs   noise floor %.1f dBFS\n",
		float64(sc.CenterHz)/1e6, float64(sc.SpanHz)/1e6, sc.WindowSec, sc.NoiseFloorDbFS)
	fmt.Fprintf(bw, "  %d bursts   %d channels   %d emitters   %d anomalies\n",
		len(sc.Bursts), len(sc.Channels), len(sc.Emitters), len(sc.Anomalies))

	if len(sc.Hierarchy.Nodes) > 0 {
		fmt.Fprintf(bw, "\nProtocol hierarchy (bursts / airtime / time%% / spectrum%%):\n")
		for _, n := range sc.Hierarchy.Nodes {
			indent := strings.Repeat("  ", n.Depth+1)
			fmt.Fprintf(bw, "%s%-22s %4d  %7.2fs  %5.1f%%  %5.1f%%\n",
				indent, n.Label, n.Stats.BurstCount, n.Stats.AirtimeSec, n.Stats.TimePct, n.Stats.SpectrumPct)
		}
	}

	if len(sc.Channels) > 0 {
		fmt.Fprintf(bw, "\nChannels:\n")
		fmt.Fprintf(bw, "  %-13s %-8s %6s %7s %7s %9s %8s\n",
			"freq(MHz)", "class", "bursts", "duty%", "occ%", "rate/min", "period")
		for _, c := range sc.Channels {
			period := "-"
			if c.PeriodSec > 0 {
				period = fmt.Sprintf("%.3fs", c.PeriodSec)
			}
			fmt.Fprintf(bw, "  %-13.6f %-8s %6d %7.1f %7.1f %9.1f %8s\n",
				float64(c.FreqHz)/1e6, string(c.DominantClass), c.BurstCount,
				c.DutyCycle*100, c.OccupancyPct, c.BurstRatePerMin, period)
		}
	}

	if len(sc.Emitters) > 0 {
		fmt.Fprintf(bw, "\nTop talkers:\n")
		for _, e := range sc.Emitters {
			hop := ""
			if len(e.HopSet) > 1 {
				hop = fmt.Sprintf(" (hops %d)", len(e.HopSet))
			}
			fmt.Fprintf(bw, "  %-28s %.2fs airtime%s\n", e.Label, e.TotalAirtimeSec, hop)
		}
	}

	if len(sc.Anomalies) > 0 {
		fmt.Fprintf(bw, "\nExpert info:\n")
		for _, a := range sc.Anomalies {
			fmt.Fprintf(bw, "  [%s] %s: %s\n", a.Severity, a.Kind, a.Detail)
		}
	}

	_, err := io.WriteString(w, bw.String())
	return err
}
