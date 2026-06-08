package siglab

import (
	"errors"
	"fmt"
	"io"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
)

// estimateCaptureCarrierHz reads a prefix of rs, estimates the dominant
// carrier offset across the full band, then rewinds rs to the start so the
// prefix is still decoded by the main read loop. It is the shared
// implementation behind -auto-tune (lifted verbatim from replay.go so the
// replay subcommand and the engine share one estimator). It takes an
// io.ReadSeeker so it works equally over an *os.File capture and an
// in-memory *bytes.Reader (the synthesized-signal path).
func estimateCaptureCarrierHz(rs io.ReadSeeker, decode SampleDecoder, bytesPerSample int, sampleRateHz float64) (float64, error) {
	const prefixSamples = 262144
	buf := make([]byte, prefixSamples*bytesPerSample)
	n, err := io.ReadFull(rs, buf)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return 0, err
	}
	if _, serr := rs.Seek(0, io.SeekStart); serr != nil {
		return 0, serr
	}
	pairs := n / bytesPerSample
	if pairs == 0 {
		return 0, fmt.Errorf("capture has no samples to estimate from")
	}
	iq := make([]complex64, pairs)
	decode(buf[:pairs*bytesPerSample], iq)
	return dsp.EstimateCarrierOffsetHz(iq, sampleRateHz, sampleRateHz*0.5), nil
}

// ratioOrZero returns num/den, or 0 when den is too small to divide safely.
// Shared with the replay state log (avoids a divide-by-zero / +Inf on a
// not-yet-seeded value).
func ratioOrZero(num, den float64) float64 {
	if den < 1e-12 && den > -1e-12 {
		return 0
	}
	return num / den
}
