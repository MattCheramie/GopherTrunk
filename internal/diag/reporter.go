package diag

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// verboseHint is the one-line nudge appended on a non-interactive,
// non-verbose error so the operator knows how to get the full trace
// without us blocking on a prompt that nothing will answer.
const verboseHint = "(set diagnostics.verbose_errors: true, GOPHERTRUNK_VERBOSE_ERRORS=1, or pass -verbose-errors for the full trace)"

// Reporter centralizes CLI/launcher error reporting: it prints the
// diagnostic banner, a concise message, and then either the full
// verbose trace (when Verbose) or — on an interactive terminal — an
// offer to show it. On a non-TTY it prints a hint instead of prompting
// so service managers and pipes never hang.
type Reporter struct {
	Out         io.Writer  // defaults to os.Stderr
	In          *os.File   // defaults to os.Stdin (prompt source)
	Collector   *Collector // banner source; lazily falls back to NewCollector()
	Verbose     bool       // diagnostics.verbose_errors (config/env/flag resolved)
	Prefix      string     // message prefix, e.g. "config", "import-pdf"; "" for none
	interactive func() bool
}

// NewReporter builds a Reporter for a CLI command. prefix is prepended
// to the concise message ("<prefix>: <err>"); pass "" for no prefix.
func NewReporter(prefix string, verbose bool, c *Collector) *Reporter {
	return &Reporter{
		Out:       os.Stderr,
		In:        os.Stdin,
		Collector: c,
		Verbose:   verbose,
		Prefix:    prefix,
	}
}

// Report prints the banner, the concise error, and then the verbose
// trace (Verbose) or an interactive offer / hint. The goroutine stack
// is captured here so it reflects the report site.
func (r *Reporter) Report(err error) {
	if err == nil {
		return
	}
	out := r.Out
	if out == nil {
		out = os.Stderr
	}
	stack := CaptureStack()

	if r.Collector == nil {
		r.Collector = NewCollector()
	}
	if banner := FormatBanner(r.Collector.SysInfo()); banner != "" {
		fmt.Fprintln(out, banner)
	}

	fmt.Fprintln(out, r.concise(err))

	switch {
	case r.Verbose:
		fmt.Fprintln(out, VerboseTrace(err, stack))
	case r.isInteractive():
		r.prompt(out, err, stack)
	default:
		fmt.Fprintln(out, verboseHint)
	}
}

// Fatal reports err and exits with the given code (preserving the
// caller's existing exit semantics).
func (r *Reporter) Fatal(code int, err error) {
	r.Report(err)
	os.Exit(code)
}

// Fatalf is the fmt-string convenience used by the many call sites that
// build their message inline. Reports and exits with code.
func (r *Reporter) Fatalf(code int, format string, a ...any) {
	r.Report(fmt.Errorf(format, a...))
	os.Exit(code)
}

// concise renders "<prefix>: <outermost message>".
func (r *Reporter) concise(err error) string {
	if r.Prefix == "" {
		return err.Error()
	}
	return r.Prefix + ": " + err.Error()
}

// prompt offers the verbose trace on an interactive terminal. The
// default answer is No (Enter / EOF / anything but y[es]).
func (r *Reporter) prompt(out io.Writer, err error, stack []byte) {
	in := r.In
	if in == nil {
		in = os.Stdin
	}
	fmt.Fprint(out, "Show full diagnostic trace? [y/N]: ")
	line, _ := bufio.NewReader(in).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		fmt.Fprintln(out, VerboseTrace(err, stack))
	}
}

func (r *Reporter) isInteractive() bool {
	if r.interactive != nil {
		return r.interactive()
	}
	in := r.In
	if in == nil {
		in = os.Stdin
	}
	return IsInteractive(in, os.Stdout)
}

// IsInteractive reports whether both stdin and stdout are attached to a
// character device (an interactive terminal). Shared so the launcher
// and the reporter agree on what "interactive" means.
func IsInteractive(stdin, stdout *os.File) bool {
	return isCharDevice(stdin) && isCharDevice(stdout)
}

func isCharDevice(f *os.File) bool {
	if f == nil {
		return false
	}
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
