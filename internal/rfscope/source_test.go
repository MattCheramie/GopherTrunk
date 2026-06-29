package rfscope

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

func makeIQ(n int) []complex64 {
	iq := make([]complex64, n)
	for i := range iq {
		iq[i] = complex(float32(i)*1e-3, float32(-i)*1e-3)
	}
	return iq
}

func TestFileSourceRoundTrip(t *testing.T) {
	t.Parallel()
	const n = 5000
	want := makeIQ(n)

	dir := t.TempDir()
	path := filepath.Join(dir, "cap.cfile")
	if err := os.WriteFile(path, siglab.EncodeCapture(want, siglab.FormatF32), 0o644); err != nil {
		t.Fatalf("write capture: %v", err)
	}

	src, err := OpenFile(path, siglab.FormatF32, 451_000_000, 2_400_000)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	defer src.Close()

	if src.CenterHz() != 451_000_000 || src.SampleRateHz() != 2_400_000 {
		t.Fatalf("header mismatch: %d %g", src.CenterHz(), src.SampleRateHz())
	}

	// Read in odd-sized chunks to exercise the partial-tail path.
	var got []complex64
	for {
		c, err := src.Next(context.Background(), 777)
		got = append(got, c...)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
	}
	if len(got) != n {
		t.Fatalf("sample count: got %d want %d", len(got), n)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sample %d: got %v want %v", i, got[i], want[i])
		}
	}
}

func TestFileSourceUnknownFile(t *testing.T) {
	t.Parallel()
	if _, err := OpenFile(filepath.Join(t.TempDir(), "nope.cfile"), siglab.FormatU8, 0, 1e6); err == nil {
		t.Fatalf("want error opening missing file")
	}
	if _, err := OpenFile("x", siglab.FormatU8, 0, 0); err == nil {
		t.Fatalf("want error on non-positive rate")
	}
}

func TestMemSourceChunksThenEOF(t *testing.T) {
	t.Parallel()
	iq := makeIQ(100)
	src := NewMemSource(iq, 100_000_000, 48_000)

	c, err := src.Next(context.Background(), 30)
	if err != nil || len(c) != 30 {
		t.Fatalf("first chunk: len=%d err=%v", len(c), err)
	}
	all, err := Drain(context.Background(), src, 40, 0)
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if len(all) != 70 { // 100 total minus the 30 already read
		t.Fatalf("drained %d, want 70", len(all))
	}
	if _, err := src.Next(context.Background(), 10); err != io.EOF {
		t.Fatalf("want EOF at end, got %v", err)
	}
}

func TestDrainRespectsCap(t *testing.T) {
	t.Parallel()
	src := NewMemSource(makeIQ(1000), 0, 1e6)
	got, err := Drain(context.Background(), src, 128, 256)
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if len(got) != 256 {
		t.Fatalf("cap not respected: got %d want 256", len(got))
	}
}

func TestSourceContextCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	src := NewMemSource(makeIQ(10), 0, 1e6)
	if _, err := src.Next(ctx, 4); err == nil {
		t.Fatalf("want context error")
	}
}
