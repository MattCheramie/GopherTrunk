package log

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
)

func TestNewWithSwapDefaultsToInfoText(t *testing.T) {
	logger, sw := NewWithSwap("", "")
	var buf bytes.Buffer
	sw.Redirect(&buf)

	logger.Debug("debug-line")
	if buf.Len() != 0 {
		t.Errorf("debug line emitted at default level: %q", buf.String())
	}

	logger.Info("info-line")
	if !strings.Contains(buf.String(), "info-line") {
		t.Errorf("info line missing: %q", buf.String())
	}
	if strings.HasPrefix(strings.TrimSpace(buf.String()), "{") {
		t.Errorf("default format should be text, got JSON-looking: %q", buf.String())
	}
}

func TestNewWithSwapHonoursLevelStrings(t *testing.T) {
	cases := []struct {
		level     string
		emitDebug bool
		emitInfo  bool
		emitWarn  bool
		emitError bool
	}{
		{"debug", true, true, true, true},
		{"DEBUG", true, true, true, true},
		{"info", false, true, true, true},
		{"warn", false, false, true, true},
		{"WARN", false, false, true, true},
		{"error", false, false, false, true},
		{"unknown", false, true, true, true}, // default → info
		{"", false, true, true, true},        // empty → info
	}
	for _, tc := range cases {
		t.Run(tc.level, func(t *testing.T) {
			logger, sw := NewWithSwap(tc.level, "")
			var buf bytes.Buffer
			sw.Redirect(&buf)

			logger.Debug("d")
			logger.Info("i")
			logger.Warn("w")
			logger.Error("e")
			out := buf.String()
			if got := strings.Contains(out, "msg=d"); got != tc.emitDebug {
				t.Errorf("debug emit = %v, want %v (out=%q)", got, tc.emitDebug, out)
			}
			if got := strings.Contains(out, "msg=i"); got != tc.emitInfo {
				t.Errorf("info emit = %v, want %v (out=%q)", got, tc.emitInfo, out)
			}
			if got := strings.Contains(out, "msg=w"); got != tc.emitWarn {
				t.Errorf("warn emit = %v, want %v (out=%q)", got, tc.emitWarn, out)
			}
			if got := strings.Contains(out, "msg=e"); got != tc.emitError {
				t.Errorf("error emit = %v, want %v (out=%q)", got, tc.emitError, out)
			}
		})
	}
}

func TestNewWithSwapJSONFormat(t *testing.T) {
	logger, sw := NewWithSwap("info", "json")
	var buf bytes.Buffer
	sw.Redirect(&buf)

	logger.Info("payload")
	out := strings.TrimSpace(buf.String())
	if !strings.HasPrefix(out, "{") || !strings.HasSuffix(out, "}") {
		t.Errorf("expected JSON-shaped output, got %q", out)
	}
	if !strings.Contains(out, `"msg":"payload"`) {
		t.Errorf("expected JSON msg field, got %q", out)
	}
}

func TestNewWithSwapJSONFormatCaseInsensitive(t *testing.T) {
	// strings.EqualFold at log.go:85 means "JSON" / "Json" / "jSoN"
	// all resolve to the JSON handler, not the text fallback.
	for _, format := range []string{"JSON", "Json", "jSoN"} {
		logger, sw := NewWithSwap("info", format)
		var buf bytes.Buffer
		sw.Redirect(&buf)
		logger.Info("payload")
		out := strings.TrimSpace(buf.String())
		if !strings.HasPrefix(out, "{") {
			t.Errorf("format=%q did not produce JSON output: %q", format, out)
		}
	}
}

func TestNew_NonSwapVariantBuildsLogger(t *testing.T) {
	logger := New("debug", "json")
	if logger == nil {
		t.Fatal("New returned nil")
	}
	// Smoke: emitting a record should not panic.
	logger.Debug("smoke")
}

func TestSwappableWriterRedirectAndRestore(t *testing.T) {
	base := &bytes.Buffer{}
	sw := NewSwappableWriter(base)

	mustWrite(t, sw, "A")
	if base.String() != "A" {
		t.Fatalf("base after A = %q, want %q", base.String(), "A")
	}

	tmp := &bytes.Buffer{}
	sw.Redirect(tmp)
	mustWrite(t, sw, "B")
	if tmp.String() != "B" {
		t.Fatalf("tmp after B = %q, want %q", tmp.String(), "B")
	}
	if base.String() != "A" {
		t.Fatalf("base mutated after redirect: %q", base.String())
	}

	sw.Restore()
	mustWrite(t, sw, "C")
	if base.String() != "AC" {
		t.Fatalf("base after restore+C = %q, want %q", base.String(), "AC")
	}
	if tmp.String() != "B" {
		t.Fatalf("tmp mutated after restore: %q", tmp.String())
	}
}

func TestSwappableWriterRestoreWithoutRedirectIsNoOp(t *testing.T) {
	base := &bytes.Buffer{}
	sw := NewSwappableWriter(base)
	sw.Restore() // saved is nil; must be a no-op

	mustWrite(t, sw, "X")
	if base.String() != "X" {
		t.Fatalf("base after Restore-then-Write = %q, want %q", base.String(), "X")
	}
}

func TestSwappableWriterRedirectChainRestoresOneLevel(t *testing.T) {
	base := &bytes.Buffer{}
	a := &bytes.Buffer{}
	b := &bytes.Buffer{}

	sw := NewSwappableWriter(base)
	sw.Redirect(a) // saved = base, out = a
	sw.Redirect(b) // saved = a,    out = b

	sw.Restore() // out = a, saved cleared
	mustWrite(t, sw, "P")
	if a.String() != "P" {
		t.Fatalf("after one restore, expected write to land in a, got base=%q a=%q b=%q",
			base.String(), a.String(), b.String())
	}

	// A second Restore is now a no-op (saved was cleared by the
	// first restore); writes keep landing in a.
	sw.Restore()
	mustWrite(t, sw, "Q")
	if a.String() != "PQ" {
		t.Fatalf("second restore should be no-op, got a=%q", a.String())
	}
	if base.String() != "" {
		t.Fatalf("base should be untouched, got %q", base.String())
	}
}

func TestSwappableWriterNilTargetIsSink(t *testing.T) {
	sw := NewSwappableWriter(nil)
	n, err := sw.Write([]byte("dropped"))
	if err != nil {
		t.Fatalf("nil-target Write returned err: %v", err)
	}
	if n != len("dropped") {
		t.Fatalf("nil-target Write returned n=%d, want %d", n, len("dropped"))
	}
}

func TestSwappableWriterConcurrentWritesNoInterleave(t *testing.T) {
	const goroutines = 64
	const linesPerG = 32

	var mu sync.Mutex
	captured := []string{}
	sink := writerFunc(func(p []byte) (int, error) {
		mu.Lock()
		defer mu.Unlock()
		captured = append(captured, string(p))
		return len(p), nil
	})

	sw := NewSwappableWriter(sink)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		g := g
		go func() {
			defer wg.Done()
			payload := []byte{byte('A' + (g % 26))}
			for i := 0; i < linesPerG; i++ {
				if _, err := sw.Write(payload); err != nil {
					t.Errorf("goroutine %d write %d: %v", g, i, err)
					return
				}
			}
		}()
	}
	wg.Wait()

	if len(captured) != goroutines*linesPerG {
		t.Fatalf("captured %d writes, want %d", len(captured), goroutines*linesPerG)
	}
	// Each call delivered exactly one byte; if the mutex didn't
	// serialize Write delegations, the sink would have seen
	// multi-byte payloads here.
	for i, c := range captured {
		if len(c) != 1 {
			t.Fatalf("captured[%d] = %q (len %d), expected single-byte payloads", i, c, len(c))
		}
	}
}

func mustWrite(t *testing.T, w io.Writer, s string) {
	t.Helper()
	n, err := w.Write([]byte(s))
	if err != nil {
		t.Fatalf("Write %q: %v", s, err)
	}
	if n != len(s) {
		t.Fatalf("Write %q: n=%d, want %d", s, n, len(s))
	}
}

type writerFunc func(p []byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }
