package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/gtbundle"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/baseband"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// baseName is filepath.Base; stemOf strips the final extension from a leaf.
func baseName(p string) string { return filepath.Base(p) }
func stemOf(leaf string) string {
	return strings.TrimSuffix(leaf, filepath.Ext(leaf))
}

// sliceSeconds is the narrowband-slice length carved from a full capture, by
// intent. The slice is the small, shareable channelized stream; the full
// wideband IQ remains the source of truth in the bundle. These mirror the
// capture-intent policy documented for the bundle format.
func sliceSeconds(intent gtbundle.CaptureIntent) float64 {
	switch intent {
	case gtbundle.IntentCrypto:
		return 60
	case gtbundle.IntentCCMap:
		return 30
	case gtbundle.IntentSurvey:
		return 10
	default:
		return 15
	}
}

// carveSlice reads a raw IQ capture, downconverts it to the per-protocol
// channel rate (48 kHz C4FM family / 144 kHz TETRA — the exact stream the
// decoder sees), trims it to the intent-driven length, and writes it as a
// 2-channel s16 baseband WAV. It reuses the production single-channel
// Downconverter and the baseband IQWriter, so the slice replays identically via
// `siglab -format wav`. protoName may be empty (undecoded carrier), in which
// case the C4FM-family target rate is used. Returns the achieved output rate.
func carveSlice(capturePath string, format siglab.SampleFormat, sampleRateHz, tuneHz float64, protoName string, intent gtbundle.CaptureIntent, outWav string) (uint32, error) {
	raw, err := os.ReadFile(capturePath)
	if err != nil {
		return 0, err
	}
	decode, bytesPerSample := format.Decoder()
	if bytesPerSample <= 0 {
		return 0, fmt.Errorf("carve slice: unknown sample format %q", format)
	}
	nSamples := len(raw) / bytesPerSample
	if nSamples == 0 {
		return 0, fmt.Errorf("carve slice: capture %s has no samples", capturePath)
	}
	iq := make([]complex64, nSamples)
	decode(raw[:nSamples*bytesPerSample], iq)

	proto := trunking.ProtocolP25 // default target family; TETRA needs the wide rate
	if protoName != "" {
		if p, perr := trunking.ParseProtocol(protoName); perr == nil {
			proto = p
		}
	}
	target := ccdecoder.DDCTargetForProtocol(proto)
	ddc := ccdecoder.NewDownconverterWithOffset(sampleRateHz, target, tuneHz)
	outRate := uint32(ddc.OutRateHz() + 0.5)

	w, err := baseband.NewIQWriter(outWav, outRate)
	if err != nil {
		return 0, err
	}

	maxOut := int(sliceSeconds(intent) * float64(outRate))
	const chunk = 65536
	var dst []complex64
	written := 0
	for off := 0; off < len(iq) && written < maxOut; off += chunk {
		end := off + chunk
		if end > len(iq) {
			end = len(iq)
		}
		out := ddc.Process(dst[:0], iq[off:end])
		if len(out) == 0 {
			continue
		}
		if written+len(out) > maxOut {
			out = out[:maxOut-written]
		}
		if err := w.Write(out); err != nil {
			w.Close()
			return 0, err
		}
		written += len(out)
	}
	if err := w.Close(); err != nil {
		return 0, err
	}
	return outRate, nil
}

// captureBundleParams carries the inputs for writing a capture straight into a
// new bundle (the `capture -bundle` path).
type captureBundleParams struct {
	bundlePath   string
	name         string
	intent       gtbundle.CaptureIntent
	capturePath  string
	format       siglab.SampleFormat
	sampleRateHz float64
	centerHz     uint32
	tuneHz       float64
	protocol     string
	source       string
	meta         *siglab.Metadata
	emitSigMF    bool
}

// writeCaptureBundle assembles a bundle from a just-recorded capture: the raw
// IQ, the metadata sidecar (as YAML), and a carved narrowband slice. Slice
// carving failures are non-fatal (the full IQ still lands) so a bundle is always
// produced.
func writeCaptureBundle(p captureBundleParams) error {
	f, err := os.Create(p.bundlePath)
	if err != nil {
		return err
	}
	defer f.Close()

	w, err := gtbundle.NewWriter(f, gtbundle.WriterOptions{
		Name:          p.name,
		CaptureIntent: p.intent,
		Provenance: gtbundle.Provenance{
			Source:       p.source,
			Protocol:     p.protocol,
			CenterFreqHz: p.centerHz,
			SampleRateHz: p.sampleRateHz,
		},
	})
	if err != nil {
		return err
	}

	src, err := os.Open(p.capturePath)
	if err != nil {
		return err
	}
	iqLeaf := baseName(p.capturePath)
	if _, err := w.Add(gtbundle.RoleCaptureIQ, iqLeaf, src, "gophertrunk capture"); err != nil {
		src.Close()
		return err
	}
	src.Close()

	if p.meta != nil {
		if _, err := w.AddYAML(gtbundle.RoleCaptureMeta, stemOf(iqLeaf)+".meta.yaml", p.meta, "gophertrunk capture"); err != nil {
			return err
		}
		if p.emitSigMF {
			if doc, ok, serr := gtbundle.MetadataToSigMF(p.meta); serr != nil {
				return serr
			} else if ok {
				if _, err := w.AddBytes(gtbundle.RoleCaptureSigMF, stemOf(iqLeaf)+".sigmf-meta", doc, "gophertrunk bundle"); err != nil {
					return err
				}
			}
		}
	}

	// Carve a narrowband slice next to the capture, then fold it in. Best-effort.
	sliceTmp := p.capturePath + ".slice.wav"
	if rate, cerr := carveSlice(p.capturePath, p.format, p.sampleRateHz, p.tuneHz, p.protocol, p.intent, sliceTmp); cerr == nil {
		if sf, oerr := os.Open(sliceTmp); oerr == nil {
			_, aerr := w.Add(gtbundle.RoleCaptureSlice, stemOf(iqLeaf)+".slice.wav", sf, "gophertrunk bundle")
			sf.Close()
			os.Remove(sliceTmp)
			if aerr != nil {
				return aerr
			}
			w.SetNote(fmt.Sprintf("narrowband DDC slice @ %d Hz (%gs)", rate, sliceSeconds(p.intent)))
		}
	} else {
		fmt.Fprintf(os.Stderr, "capture: note — slice carving skipped (%v); full IQ still bundled\n", cerr)
		os.Remove(sliceTmp)
	}

	return w.Close()
}
