package gtbundle

import (
	"fmt"
	"sort"
	"strings"
)

// renderReadme builds the human-facing README.md that ships at the bundle root.
// It is generated deterministically from the manifest so a person who untars the
// archive can see what the case contains and how to open it, without any tool.
func renderReadme(m *Manifest) []byte {
	var b strings.Builder

	title := m.Title
	if title == "" {
		title = m.Name
	}
	fmt.Fprintf(&b, "# GopherTrunk Bundle — %s\n\n", title)
	b.WriteString("This is a **GopherTrunk Bundle** (`.gtb.tar.gz`): one SDR capture-to-analysis\n")
	b.WriteString("case — the raw capture, logs, signal analysis, crypto analysis, and site/network\n")
	b.WriteString("mapping — indexed by `MANIFEST.yaml`.\n\n")

	fmt.Fprintf(&b, "- **Intent:** %s\n", m.CaptureIntent)
	fmt.Fprintf(&b, "- **Created:** %s\n", m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	fmt.Fprintf(&b, "- **Generator:** %s %s", m.Generator.Tool, m.Generator.Version)
	if m.Generator.Commit != "" {
		fmt.Fprintf(&b, " (%s)", m.Generator.Commit)
	}
	b.WriteString("\n")

	p := m.Provenance
	if p.Source != "" {
		fmt.Fprintf(&b, "- **Source:** %s\n", p.Source)
	}
	if p.Protocol != "" {
		fmt.Fprintf(&b, "- **Protocol:** %s\n", p.Protocol)
	}
	if p.CenterFreqHz != 0 {
		fmt.Fprintf(&b, "- **Center:** %.4f MHz\n", float64(p.CenterFreqHz)/1e6)
	}
	if p.SampleRateHz != 0 {
		fmt.Fprintf(&b, "- **Sample rate:** %g MS/s\n", p.SampleRateHz/1e6)
	}
	if p.Location != "" {
		fmt.Fprintf(&b, "- **Location:** %s\n", p.Location)
	}
	b.WriteString("\n")

	if m.Notes != "" {
		b.WriteString("## Notes\n\n")
		b.WriteString(m.Notes)
		b.WriteString("\n\n")
	}

	b.WriteString("## Contents\n\n")
	b.WriteString("| Role | Path | Size | Produced by |\n")
	b.WriteString("|---|---|---|---|\n")
	files := append([]FileEntry(nil), m.Files...)
	sort.SliceStable(files, func(i, j int) bool {
		ri, rj := roleWriteRank(files[i].Role), roleWriteRank(files[j].Role)
		if ri != rj {
			return ri < rj
		}
		return files[i].Path < files[j].Path
	})
	for _, f := range files {
		fmt.Fprintf(&b, "| `%s` | `%s` | %s | %s |\n",
			f.Role, f.Path, humanSize(f.SizeBytes), dashIfEmpty(f.ProducedBy))
	}
	b.WriteString("\n")

	b.WriteString("## Opening this bundle\n\n")
	b.WriteString("```sh\n")
	b.WriteString("gophertrunk bundle info   -in <file>.gtb.tar.gz   # this summary\n")
	b.WriteString("gophertrunk bundle verify -in <file>.gtb.tar.gz   # check integrity\n")
	b.WriteString("gophertrunk bundle extract -in <file>.gtb.tar.gz -out ./case\n")
	b.WriteString("gophertrunk bundle commit -in <file>.gtb.tar.gz -config config.yaml -dry-run\n")
	b.WriteString("```\n\n")
	b.WriteString("Or open it in the SigLab / CryptoLab web consoles.\n")

	return []byte(b.String())
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
