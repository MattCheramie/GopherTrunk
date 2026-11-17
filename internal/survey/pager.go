package survey

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	flexrx "github.com/MattCheramie/GopherTrunk/internal/radio/pager/flex/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/pager/pocsag"
)

// oversample is the resampled samples per bit the integrate-and-dump slicer
// expects. It mirrors the live POCSAG receiver (receiver.Oversample = 8) so a
// buffer decoded here behaves identically to the streaming path.
const oversample = 8

// pocsagBauds are the POCSAG signalling rates tried, fastest-first is not
// important — the syncer rejects a wrong-rate stream (no sync codewords found),
// so the rate that actually yields pages wins.
var pocsagBauds = []uint32{1200, 512, 2400}

// PageRef is a protocol-neutral summary of one decoded pager message, suitable
// for the survey inventory and JSON. It flattens the pocsag.Page / flex.Message
// shapes the receivers publish to the bus.
type PageRef struct {
	Protocol  string `json:"protocol"`  // "pocsag" | "flex"
	BaudHz    uint32 `json:"baud_hz"`   // signalling rate that decoded it
	Capcode   uint32 `json:"capcode"`   // RIC (POCSAG) / capcode (FLEX)
	Encoding  string `json:"encoding"`  // "alpha" | "numeric" | FLEX type name
	Text      string `json:"text"`      // decoded payload
	Corrected int    `json:"corrected"` // FEC bit corrections (signal-quality proxy)
}

// DecodePOCSAG runs the POCSAG decode pipeline over a baseband IQ buffer at each
// candidate baud and returns the pages from the rate that decoded the most. It
// reuses the exact FM→resample→slice→syncer chain the live receiver uses
// (internal/radio/pager/pocsag/receiver), just over a fixed buffer instead of a
// channel, so the protocol layer is unchanged.
func DecodePOCSAG(iq []complex64, inputRateHz uint32) []PageRef {
	var best []PageRef
	for _, baud := range pocsagBauds {
		bits := demodBits(iq, inputRateHz, baud)
		if bits == nil {
			continue
		}
		syncer := pocsag.NewSyncer()
		var pages []pocsag.Page
		for _, b := range bits {
			pages = append(pages, syncer.Push(b)...)
		}
		if p := syncer.Flush(); p != nil {
			pages = append(pages, *p)
		}
		if len(pages) <= len(best) {
			continue
		}
		refs := make([]PageRef, 0, len(pages))
		for _, p := range pages {
			enc := "alpha"
			if p.Encoding == pocsag.EncodingNumeric {
				enc = "numeric"
			}
			refs = append(refs, PageRef{
				Protocol: "pocsag", BaudHz: baud, Capcode: p.RIC,
				Encoding: enc, Text: p.Text, Corrected: p.Corrected,
			})
		}
		best = refs
	}
	return best
}

// DecodeFLEX runs the FLEX decode pipeline over a baseband IQ buffer and
// returns the decoded messages. It is the live FLEX receiver's front end
// (internal/radio/pager/flex/receiver) over a fixed buffer, so every mode —
// 1600, 3200 and 6400 bps — is followed; BaudHz reports the bit rate of the
// frame a page came in. A frame is 1.875 s long and only complete frames
// decode, so a buffer shorter than two frames may hold none.
func DecodeFLEX(iq []complex64, inputRateHz uint32) []PageRef {
	dm, err := flexrx.NewIQDemod(inputRateHz)
	if err != nil || len(iq) == 0 {
		return nil
	}
	var refs []PageRef
	for _, m := range dm.ProcessIQ(iq) {
		if m.Capcode > math.MaxUint32 {
			continue
		}
		refs = append(refs, PageRef{
			Protocol: "flex", BaudHz: uint32(m.Mode.BitRate()), Capcode: uint32(m.Capcode),
			Encoding: m.Type.TypeName(), Text: m.Text, Corrected: m.Corrected,
		})
	}
	return refs
}

// demodBits FM-demodulates iq, resamples to baud×oversample sps, and slices to
// a bit stream with an integrate-and-dump + slow-mean slicer — the identical
// front-end the pager receivers run. Returns nil when the input rate can't be
// resampled to the requested bit rate.
func demodBits(iq []complex64, inputRateHz uint32, baud uint32) []byte {
	if len(iq) == 0 || inputRateHz == 0 || baud == 0 {
		return nil
	}
	out := baud * oversample
	g := gcdU32(out, inputRateHz)
	L := int(out / g)
	M := int(inputRateHz / g)
	if L == 0 || M == 0 {
		return nil
	}
	d := demod.NewFM().Process(nil, iq)
	samples := dsp.NewRealResampler(L, M, 16, 7.0).Process(nil, d)

	bits := make([]byte, 0, len(samples)/oversample)
	var intAcc, meanEMA float32
	var intCount int
	var meanReady bool
	for _, s := range samples {
		intAcc += s
		intCount++
		if intCount < oversample {
			continue
		}
		avg := intAcc / float32(oversample)
		intAcc, intCount = 0, 0
		if !meanReady {
			meanEMA, meanReady = avg, true
		} else {
			meanEMA += (avg - meanEMA) * (1.0 / 64.0)
		}
		var bit byte
		if avg > meanEMA {
			bit = 1
		}
		bits = append(bits, bit)
	}
	return bits
}

// gcdU32 is Euclid's algorithm (mirrors the receivers' local gcd).
func gcdU32(a, b uint32) uint32 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
