package receiver

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/lora"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// shift multiplies a baseband signal by exp(j*2*pi*offHz*n/rate).
func shift(in []complex64, offHz, rate float64) []complex64 {
	out := make([]complex64, len(in))
	w := 2 * math.Pi * offHz / rate
	for n, s := range in {
		ph := w * float64(n)
		out[n] = s * complex(float32(math.Cos(ph)), float32(math.Sin(ph)))
	}
	return out
}

func add(dst, src []complex64) []complex64 {
	if len(src) > len(dst) {
		grown := make([]complex64, len(src))
		copy(grown, dst)
		dst = grown
	}
	for i := range src {
		dst[i] += src[i]
	}
	return dst
}

// TestWidebandTwoChannels synthesises two LoRa packets at different
// frequency offsets within one wide-band capture, pushes the summed IQ
// through the real tuner bank + streaming demods, and asserts both frames
// are recovered on the bus.
func TestWidebandTwoChannels(t *testing.T) {
	const (
		inRate = 1_000_000.0 // wide-band SDR rate
		bw     = lora.BW125
		// Generate each packet at the wide-band rate directly: osfWide =
		// inRate/BW so the modulator output needs no resampling. The bank
		// decimates back to DefaultOversample*BW for the demod.
		osfWide = int(inRate) / int(bw) // 8
	)
	plA := []byte("alpha-channel")
	plB := []byte("bravo-channel")

	baseA := lora.LoRaModulate(plA, 7, 1, osfWide, bw, true, 0x12)
	baseB := lora.LoRaModulate(plB, 9, 4, osfWide, bw, true, 0x12)

	wide := make([]complex64, 0)
	wide = add(wide, shift(baseA, -200_000, inRate))
	wide = add(wide, shift(baseB, +180_000, inRate))
	// Trailing silence so each frame sits comfortably inside the capture
	// (the decimated frame must not butt up against the buffer end).
	wide = append(wide, make([]complex64, 40_000)...)

	bus := events.NewBus(256)
	sub := bus.Subscribe()
	defer sub.Close()

	rx, err := New(Options{
		InputRateHz: uint32(inRate),
		BW:          bw,
		Oversample:  lora.DefaultOversample,
		Channels: []ChannelConfig{
			{OffsetHz: -200_000, FrequencyHz: 915_000_000, SF: 7, SyncWord: 0x12},
			{OffsetHz: +180_000, FrequencyHz: 915_400_000, SF: 9, SyncWord: 0x12},
		},
		Bus: bus,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Feed the capture as ~10 ms chunks, then close to flush.
	in := make(chan []complex64)
	go func() {
		const chunk = 10_000
		for off := 0; off < len(wide); off += chunk {
			end := off + chunk
			if end > len(wide) {
				end = len(wide)
			}
			in <- wide[off:end]
		}
		close(in)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := rx.Process(ctx, in); err != nil {
		t.Fatalf("Process: %v", err)
	}

	// Collect emitted frames.
	got := map[string]storage.LoRaFrame{}
	for {
		select {
		case ev := <-sub.C:
			if f, ok := ev.Payload.(storage.LoRaFrame); ok {
				got[f.PayloadHex] = f
			}
			continue
		default:
		}
		break
	}

	wantA := hexOf(plA)
	wantB := hexOf(plB)
	fa, okA := got[wantA]
	fb, okB := got[wantB]
	if !okA {
		t.Errorf("channel A frame %q not recovered (got %d frames)", plA, len(got))
	} else if fa.SF != 7 || !fa.CRCOK || fa.FrequencyHz != 915_000_000 {
		t.Errorf("channel A frame wrong: %+v", fa)
	}
	if !okB {
		t.Errorf("channel B frame %q not recovered (got %d frames)", plB, len(got))
	} else if fb.SF != 9 || !fb.CRCOK || fb.FrequencyHz != 915_400_000 {
		t.Errorf("channel B frame wrong: %+v", fb)
	}
	if t.Failed() {
		t.Logf("frames emitted: %d", rx.Stats().FramesEmitted)
	}
}

func hexOf(b []byte) string {
	const hexd = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, c := range b {
		out[i*2] = hexd[c>>4]
		out[i*2+1] = hexd[c&0x0f]
	}
	return string(out)
}
