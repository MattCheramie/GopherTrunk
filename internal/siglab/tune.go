package siglab

import (
	"errors"
	"fmt"
	"io"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
)

// readCarrierPrefix reads a prefix of rs into decoded IQ, then rewinds rs to
// the start so the prefix is still decoded by the main read loop. It is the
// shared front-end of the auto-tune estimators; it takes an io.ReadSeeker so
// it works equally over an *os.File capture and an in-memory *bytes.Reader
// (the synthesized-signal path).
func readCarrierPrefix(rs io.ReadSeeker, decode SampleDecoder, bytesPerSample int) ([]complex64, error) {
	const prefixSamples = 262144
	buf := make([]byte, prefixSamples*bytesPerSample)
	n, err := io.ReadFull(rs, buf)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if _, serr := rs.Seek(0, io.SeekStart); serr != nil {
		return nil, serr
	}
	pairs := n / bytesPerSample
	if pairs == 0 {
		return nil, fmt.Errorf("capture has no samples to estimate from")
	}
	iq := make([]complex64, pairs)
	decode(buf[:pairs*bytesPerSample], iq)
	return iq, nil
}

// estimateCaptureCarrierHz estimates the dominant carrier offset across the
// full band. It is the shared implementation behind single-carrier -auto-tune.
func estimateCaptureCarrierHz(rs io.ReadSeeker, decode SampleDecoder, bytesPerSample int, sampleRateHz float64) (float64, error) {
	iq, err := readCarrierPrefix(rs, decode, bytesPerSample)
	if err != nil {
		return 0, err
	}
	return dsp.EstimateCarrierOffsetHz(iq, sampleRateHz, sampleRateHz*0.5), nil
}

// estimateCaptureCarrierCandidates estimates up to maxCandidates carrier
// offsets across the full band, strongest first. It backs the multi-candidate
// auto-tune: a control channel that is off-centre AND not the loudest carrier
// in a wideband grab is still surfaced as a candidate to try.
func estimateCaptureCarrierCandidates(rs io.ReadSeeker, decode SampleDecoder, bytesPerSample int, sampleRateHz float64, maxCandidates int) ([]dsp.CarrierCandidate, error) {
	iq, err := readCarrierPrefix(rs, decode, bytesPerSample)
	if err != nil {
		return nil, err
	}
	return dsp.EstimateCarrierCandidatesHz(iq, sampleRateHz, sampleRateHz*0.5, 0, maxCandidates), nil
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
