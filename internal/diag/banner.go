package diag

import (
	"fmt"
	"os"
	"strings"
)

// QuietBannerEnv suppresses banner rendering when set (matches the
// existing GOPHERTRUNK_QUIET_BANNER used by the launcher for CI/tests).
const QuietBannerEnv = "GOPHERTRUNK_QUIET_BANNER"

// FormatBanner renders the boxed diagnostic banner shown at the top of
// an error on a terminal. Returns "" when GOPHERTRUNK_QUIET_BANNER is
// set so CI logs stay clean.
func FormatBanner(si SysInfo) string {
	if os.Getenv(QuietBannerEnv) != "" {
		return ""
	}
	lines := bannerLines(si)
	const top = "┌─ GopherTrunk diagnostics ──────────────────────────────"
	const bot = "└─────────────────────────────────────────────────────────"
	var b strings.Builder
	b.WriteString(top + "\n")
	for _, ln := range lines {
		b.WriteString("│ " + ln + "\n")
	}
	b.WriteString(bot)
	return b.String()
}

// FormatBannerPlain renders the same fields without box-drawing
// characters, for embedding in JSON error envelopes, structured logs,
// or non-UTF-8 terminals. Honors GOPHERTRUNK_QUIET_BANNER.
func FormatBannerPlain(si SysInfo) string {
	if os.Getenv(QuietBannerEnv) != "" {
		return ""
	}
	return "GopherTrunk diagnostics\n" + strings.Join(bannerLines(si), "\n")
}

// bannerLines is the shared field rendering for both banner formats.
// Empty/zero fields are omitted so a platform that can't supply a value
// never produces a broken line.
func bannerLines(si SysInfo) []string {
	var lines []string
	lines = append(lines, "version : "+orNA(si.Version))

	osLine := si.GOOS + "/" + si.GOARCH
	if si.KernelOS != "" {
		osLine += "  (" + si.KernelOS + ")"
	}
	lines = append(lines, "os      : "+osLine)

	host := orNA(si.Hostname)
	hostLine := fmt.Sprintf("%s  cpus=%d  go=%s", host, si.NumCPU, si.GoVersion)
	if si.MemTotalMB > 0 {
		hostLine += fmt.Sprintf("  mem=%d/%d MB", si.MemInUseMB, si.MemTotalMB)
	} else {
		hostLine += fmt.Sprintf("  mem=%d MB used", si.MemInUseMB)
	}
	lines = append(lines, "host    : "+hostLine)

	if len(si.Dongles) == 0 {
		lines = append(lines, "dongles : none detected")
	} else {
		lines = append(lines, fmt.Sprintf("dongles : %d detected", len(si.Dongles)))
		for _, d := range si.Dongles {
			detail := fmt.Sprintf("  - %s[%d]", d.Driver, d.Index)
			if d.Serial != "" {
				detail += " serial=" + d.Serial
			}
			if d.Tuner != "" {
				detail += " tuner=" + d.Tuner
			}
			if d.Product != "" {
				detail += " product=" + d.Product
			}
			lines = append(lines, detail)
		}
	}
	for _, e := range si.DongleErrs {
		lines = append(lines, "  ! enumerate: "+e)
	}
	return lines
}

func orNA(s string) string {
	if s == "" {
		return "(unknown)"
	}
	return s
}
