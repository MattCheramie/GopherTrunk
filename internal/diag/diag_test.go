package diag

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestFormatBannerFields(t *testing.T) {
	t.Setenv(QuietBannerEnv, "") // ensure not suppressed
	si := SysInfo{
		Version:    "v1.2.3 (sha=abc1234)",
		GOOS:       "linux",
		GOARCH:     "amd64",
		KernelOS:   "Linux 6.18.5",
		Hostname:   "radio-pi",
		NumCPU:     4,
		GoVersion:  "go1.25",
		MemTotalMB: 3942,
		MemInUseMB: 12,
		Dongles: []DongleInfo{
			{Driver: "rtlsdr", Index: 0, Serial: "00000001", Tuner: "R820T", Product: "RTL2838"},
		},
	}
	out := FormatBanner(si)
	for _, want := range []string{"v1.2.3", "linux/amd64", "Linux 6.18.5", "radio-pi", "cpus=4", "go=go1.25", "3942", "rtlsdr[0]", "serial=00000001", "tuner=R820T", "RTL2838"} {
		if !strings.Contains(out, want) {
			t.Errorf("banner missing %q\n%s", want, out)
		}
	}
}

func TestFormatBannerOmitsEmptyAndNoDongles(t *testing.T) {
	si := SysInfo{GOOS: "darwin", GOARCH: "arm64", NumCPU: 8, GoVersion: "go1.25"}
	out := FormatBanner(si)
	if strings.Contains(out, "(") && strings.Contains(out, "kernel") {
		t.Errorf("expected no kernel parenthetical, got:\n%s", out)
	}
	if !strings.Contains(out, "dongles : none detected") {
		t.Errorf("expected 'none detected', got:\n%s", out)
	}
	// MemTotalMB == 0 → no "/NNNN MB" total form.
	if strings.Contains(out, "/0 MB") {
		t.Errorf("zero mem total should be omitted, got:\n%s", out)
	}
}

func TestFormatBannerQuietEnv(t *testing.T) {
	t.Setenv(QuietBannerEnv, "1")
	if got := FormatBanner(SysInfo{GOOS: "linux"}); got != "" {
		t.Errorf("expected empty banner under quiet env, got %q", got)
	}
	if got := FormatBannerPlain(SysInfo{GOOS: "linux"}); got != "" {
		t.Errorf("expected empty plain banner under quiet env, got %q", got)
	}
}

func TestChainUnwrap(t *testing.T) {
	err := fmt.Errorf("a: %w", fmt.Errorf("b: %w", errors.New("c")))
	got := Chain(err)
	if len(got) != 3 {
		t.Fatalf("want 3 layers, got %d: %v", len(got), got)
	}
	if !strings.HasPrefix(got[0], "a: ") || got[2] != "c" {
		t.Errorf("chain order wrong: %v", got)
	}
	if got := Chain(nil); len(got) != 0 {
		t.Errorf("nil error should yield empty chain, got %v", got)
	}
}

func TestChainJoin(t *testing.T) {
	err := errors.Join(errors.New("first"), errors.New("second"))
	got := Chain(err)
	// The joined error itself plus each branch.
	joined := strings.Join(got, "|")
	if !strings.Contains(joined, "first") || !strings.Contains(joined, "second") {
		t.Errorf("join branches missing: %v", got)
	}
}

func TestVerboseTrace(t *testing.T) {
	err := fmt.Errorf("outer: %w", errors.New("inner"))
	stack := []byte("goroutine 1 [running]:\nmain.main()")
	out := VerboseTrace(err, stack)
	for _, want := range []string{"outer: inner", "inner", "goroutine"} {
		if !strings.Contains(out, want) {
			t.Errorf("trace missing %q:\n%s", want, out)
		}
	}
}

func TestCollectorMemoizesAndInjects(t *testing.T) {
	calls := 0
	c := &Collector{enumerate: func() ([]DongleInfo, []string) {
		calls++
		return []DongleInfo{{Driver: "mock"}}, nil
	}}
	for i := 0; i < 3; i++ {
		si := c.SysInfo()
		if len(si.Dongles) != 1 || si.Dongles[0].Driver != "mock" {
			t.Fatalf("dongles not injected: %+v", si.Dongles)
		}
	}
	if calls != 1 {
		t.Errorf("enumerate should run once, ran %d times", calls)
	}
}

func TestCollectDonglesSkipEnv(t *testing.T) {
	t.Setenv("GOPHERTRUNK_DIAG_NO_ENUMERATE", "1")
	d, errs := CollectDongles()
	if d != nil || errs != nil {
		t.Errorf("expected no dongles when enumeration skipped, got %v / %v", d, errs)
	}
}

func TestNewCollectorWithDonglesCopies(t *testing.T) {
	src := []DongleInfo{{Driver: "rtlsdr"}}
	c := NewCollectorWithDongles(src, nil)
	src[0].Driver = "mutated"
	if got := c.SysInfo().Dongles[0].Driver; got != "rtlsdr" {
		t.Errorf("collector should copy input, got %q", got)
	}
}
