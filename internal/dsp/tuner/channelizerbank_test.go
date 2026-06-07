package tuner

import (
	"errors"
	"math"
	"testing"
)

func TestChannelizerBankSingleTapLandsAtDC(t *testing.T) {
	const (
		inRate  = 2_400_000.0
		outRate = 48_000.0
		toneAt  = 600_000.0
		M       = 16
	)
	b := NewChannelizerBank(inRate, outRate, 0.05, M, 16, 9.0)
	var got []complex64
	if err := b.AddTap(toneAt, func(out []complex64) {
		got = append(got, out...)
	}); err != nil {
		t.Fatalf("AddTap: %v", err)
	}

	gen := newToneGen(inRate, 0.5, toneAt)
	for i := 0; i < 16; i++ {
		b.Process(gen.Next(4096))
	}
	if len(got) < 1024 {
		t.Fatalf("not enough output samples: %d", len(got))
	}
	settled := got[len(got)/2:]
	if frac := powerNearDC(settled, outRate, 500); frac < 0.95 {
		t.Errorf("only %.1f%% of power within ±500 Hz of DC after tuning to %.0f Hz",
			frac*100, toneAt)
	}
}

// TestChannelizerBankWidebandRateLandsAtDC is a regression test for issue #550.
// The report singled out the polyphase channelizer path as breaking on wideband
// rates; this confirms a tone is cleanly extracted to baseband at 10 MHz (binRate
// = 625 kHz, which reduces to an exact 48/625 resampler ratio).
func TestChannelizerBankWidebandRateLandsAtDC(t *testing.T) {
	const (
		inRate  = 10_000_000.0
		outRate = 48_000.0
		M       = 16
		toneAt  = 625_000.0 // bin 1 centre at this rate (binRate = 625 kHz)
	)
	b := NewChannelizerBank(inRate, outRate, 0.05, M, 16, 9.0)
	var got []complex64
	if err := b.AddTap(toneAt, func(out []complex64) {
		got = append(got, out...)
	}); err != nil {
		t.Fatalf("AddTap: %v", err)
	}
	gen := newToneGen(inRate, 0.5, toneAt)
	for i := 0; i < 8192 && len(got) < 2200; i++ {
		b.Process(gen.Next(4096))
	}
	if len(got) < 1024 {
		t.Fatalf("not enough output samples: %d", len(got))
	}
	settled := got[len(got)/2:]
	if frac := powerNearDC(settled, outRate, 500); frac < 0.95 {
		t.Errorf("wideband %.0f Hz: only %.1f%% of power within ±500 Hz of DC", inRate, frac*100)
	}
}

func TestChannelizerBankResidualOffsetIsCorrected(t *testing.T) {
	const (
		inRate  = 2_400_000.0
		outRate = 48_000.0
		M       = 16 // bin width = 150 kHz
	)
	// 180 kHz is 30 kHz away from the nearest bin (bin 1 at +150 kHz).
	// Residual = 30 kHz, well within the channelizer prototype's flat
	// passband (~±60 kHz of the bin centre with M=16, beta=9). The
	// fine-tune DDC must clean it up to land at DC.
	const toneAt = 180_000.0
	b := NewChannelizerBank(inRate, outRate, 0.05, M, 16, 9.0)
	var got []complex64
	if err := b.AddTap(toneAt, func(out []complex64) {
		got = append(got, out...)
	}); err != nil {
		t.Fatalf("AddTap: %v", err)
	}

	gen := newToneGen(inRate, 0.5, toneAt)
	for i := 0; i < 24; i++ {
		b.Process(gen.Next(4096))
	}
	if len(got) < 1024 {
		t.Fatalf("not enough output samples: %d", len(got))
	}
	settled := got[len(got)/2:]
	if frac := powerNearDC(settled, outRate, 500); frac < 0.90 {
		t.Errorf("residual not corrected: only %.1f%% of power within ±500 Hz of DC", frac*100)
	}
}

func TestChannelizerBankMultipleTapsSeparateTones(t *testing.T) {
	const (
		inRate  = 2_400_000.0
		outRate = 48_000.0
		M       = 32 // 75 kHz bin width - generous separation
	)
	offsets := []float64{-750_000, -300_000, +150_000, +900_000}
	b := NewChannelizerBank(inRate, outRate, 0.05, M, 16, 9.0)
	collected := make(map[float64]*[]complex64, len(offsets))
	for _, off := range offsets {
		off := off
		buf := &[]complex64{}
		collected[off] = buf
		if err := b.AddTap(off, func(out []complex64) {
			*buf = append(*buf, out...)
		}); err != nil {
			t.Fatalf("AddTap(%.0f): %v", off, err)
		}
	}

	gen := newToneGen(inRate, 0.2, offsets...)
	for i := 0; i < 32; i++ {
		b.Process(gen.Next(4096))
	}

	for off, bufPtr := range collected {
		buf := *bufPtr
		if len(buf) < 1024 {
			t.Fatalf("tap %.0f: too few samples (%d)", off, len(buf))
		}
		settled := buf[len(buf)/2:]
		// 4 simultaneous tones; per-tap ≥ 80 % of energy near DC.
		if frac := powerNearDC(settled, outRate, 500); frac < 0.80 {
			t.Errorf("tap %.0f Hz: only %.1f%% of power within ±500 Hz of DC", off, frac*100)
		}
	}
}

func TestChannelizerBankRejectsCollidingBins(t *testing.T) {
	const M = 8 // bin width = 300 kHz
	b := NewChannelizerBank(2_400_000, 48_000, 0.05, M, 16, 9.0)
	if err := b.AddTap(0, func([]complex64) {}); err != nil {
		t.Fatalf("first tap: %v", err)
	}
	// 50 kHz away from 0 - same bin.
	if err := b.AddTap(50_000, func([]complex64) {}); !errors.Is(err, ErrBinAlreadyClaimed) {
		t.Errorf("expected ErrBinAlreadyClaimed, got %v", err)
	}
}

func TestChannelizerBankRejectsOutOfBandOffset(t *testing.T) {
	b := NewChannelizerBank(2_400_000, 48_000, 0.05, 16, 16, 9.0)
	if err := b.AddTap(1_200_000, func([]complex64) {}); !errors.Is(err, ErrOffsetOutOfBand) {
		t.Errorf("expected ErrOffsetOutOfBand, got %v", err)
	}
}

func TestBinForOffsetWrapsNegativeFrequencies(t *testing.T) {
	const M = 16
	b := NewChannelizerBank(2_400_000, 48_000, 0.05, M, 16, 9.0)
	// Bin width = 150 kHz. +150 kHz → bin 1. -150 kHz → bin M-1 = 15.
	if got := b.binForOffset(150_000); got != 1 {
		t.Errorf("binForOffset(+150_000) = %d, want 1", got)
	}
	if got := b.binForOffset(-150_000); got != 15 {
		t.Errorf("binForOffset(-150_000) = %d, want 15", got)
	}
	// Centre frequencies should round-trip.
	if got := b.binCenterHz(1); math.Abs(got-150_000) > 1 {
		t.Errorf("binCenterHz(1) = %.1f, want 150000", got)
	}
	if got := b.binCenterHz(15); math.Abs(got+150_000) > 1 {
		t.Errorf("binCenterHz(15) = %.1f, want -150000", got)
	}
}
