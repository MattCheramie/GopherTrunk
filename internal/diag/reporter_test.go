package diag

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// newTestReporter wires a reporter with a buffer sink, a fixed
// collector, and an explicit interactivity decision so the prompt path
// is deterministic in tests.
func newTestReporter(buf *bytes.Buffer, verbose, interactive bool, stdin *os.File) *Reporter {
	return &Reporter{
		Out:         buf,
		In:          stdin,
		Collector:   NewCollectorWithDongles(nil, nil),
		Verbose:     verbose,
		Prefix:      "test",
		interactive: func() bool { return interactive },
	}
}

func TestReporterNonInteractiveTerse(t *testing.T) {
	t.Setenv(QuietBannerEnv, "1") // keep output focused on the message + hint
	var buf bytes.Buffer
	r := newTestReporter(&buf, false, false, nil)
	r.Report(errors.New("boom"))
	out := buf.String()
	if !strings.Contains(out, "test: boom") {
		t.Errorf("missing concise message: %q", out)
	}
	if !strings.Contains(out, "verbose_errors") {
		t.Errorf("non-interactive non-verbose should print the verbose hint: %q", out)
	}
	if strings.Contains(out, "goroutine") {
		t.Errorf("should not print a stack when not verbose: %q", out)
	}
}

func TestReporterVerbosePrintsTrace(t *testing.T) {
	t.Setenv(QuietBannerEnv, "1")
	var buf bytes.Buffer
	r := newTestReporter(&buf, true, false, nil)
	r.Report(errors.New("kaboom"))
	out := buf.String()
	if !strings.Contains(out, "kaboom") || !strings.Contains(out, "goroutine") {
		t.Errorf("verbose should print chain + stack: %q", out)
	}
	if strings.Contains(out, "[y/N]") {
		t.Errorf("verbose should not prompt: %q", out)
	}
}

func TestReporterInteractivePromptYes(t *testing.T) {
	t.Setenv(QuietBannerEnv, "1")
	rd, wr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer rd.Close()
	go func() { _, _ = wr.WriteString("y\n"); wr.Close() }()

	var buf bytes.Buffer
	r := newTestReporter(&buf, false, true, rd)
	r.Report(errors.New("explode"))
	out := buf.String()
	if !strings.Contains(out, "[y/N]") {
		t.Errorf("interactive should prompt: %q", out)
	}
	if !strings.Contains(out, "goroutine") {
		t.Errorf("'y' answer should print the trace: %q", out)
	}
}

func TestReporterInteractivePromptNo(t *testing.T) {
	t.Setenv(QuietBannerEnv, "1")
	rd, wr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer rd.Close()
	go func() { _, _ = wr.WriteString("\n"); wr.Close() }() // bare Enter → default No

	var buf bytes.Buffer
	r := newTestReporter(&buf, false, true, rd)
	r.Report(errors.New("explode"))
	out := buf.String()
	if strings.Contains(out, "goroutine") {
		t.Errorf("default-No answer should not print the trace: %q", out)
	}
}

// A terminal whose stdin another reader owns (the "type q and Enter" stop
// watcher of capture / power / wmbus) must not be prompted: the #1256
// reporter answered "y", the watcher swallowed it, nothing was printed, and
// the process sat in the prompt until their next line arrived. The reporter
// prints the hint instead and returns without reading stdin at all.
func TestReporterNoPromptWhenStdinIsOwnedElsewhere(t *testing.T) {
	t.Setenv(QuietBannerEnv, "1")
	rd, wr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer rd.Close()
	defer wr.Close() // never written: a read here would block the test

	var buf bytes.Buffer
	r := newTestReporter(&buf, false, true, rd)
	r.PromptAllowed = func() bool { return false }
	done := make(chan struct{})
	go func() { r.Report(errors.New("usb gone")); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Report read stdin although prompting was not allowed")
	}
	out := buf.String()
	if strings.Contains(out, "[y/N]") {
		t.Errorf("prompted although stdin belongs to another reader: %q", out)
	}
	if !strings.Contains(out, "test: usb gone") || !strings.Contains(out, "verbose_errors") {
		t.Errorf("want the concise error and the verbose hint: %q", out)
	}
}
