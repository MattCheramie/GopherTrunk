package airspy

import (
	"encoding/binary"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// HydraSDR RFOne support (#1325).
//
// The RFOne is the Airspy R2's open-hardware successor by the same designer:
// a real-sampling R828D + LPC4370 receiver whose firmware keeps the libairspy
// vendor-request numbering and streams the same bare ADC samples (unpacked
// little-endian uint16, 12-bit) at twice the IQ rate, so the Airspy driver's
// stream path and real-to-IQ converter carry over unchanged. The protocol
// facts below are taken from the vendor's BSD/MIT host library,
// hydrasdr-host (libhydrasdr/src/hydrasdr_commands.h, hydrasdr_shared.c,
// hydrasdr_rfone.c, hydrasdr.c, hydrasdr-tools/51-hydrasdr.rules); nothing is
// ported, only the wire constants. Where the RFOne differs from the Airspy:
//
//   - USB IDs: 0x38af:0x0001 (the official VID), and the historic
//     0x1d50:0x60a1 it shares with the Airspy on older units.
//   - SET_FREQ carries a uint64 (8 bytes), not the Airspy's uint32.
//   - The bias tee is SET_RF_BIAS_CMD (20, wIndex = on/off), not the
//     Airspy's GPIO 1/13 write.
//   - GET_CAPABILITIES (29) reports a feature mask; with
//     HYDRASDR_CAP_EXTENDED_SAMPLERATES, GET_SAMPLERATES with wValue=1 returns
//     an 8-byte {rate, adc_bits, data_format, reserved} record per rate, and
//     SET_SAMPLERATE replies with 4 bytes instead of 1. Firmware without the
//     capability is the 12-bit raw ADC at every rate, as libhydrasdr assumes.
//   - LNA gain tops out at 14 (mixer and VGA at 15, like the Airspy).
//
// What is NOT used: sample packing (left off, its default, so the stream
// stays unpacked uint16), RF port selection (the firmware default, port 0 =
// ANT, the only port with the bias tee), and the firmware's data_format flag
// (libhydrasdr runs its real-to-IQ converter whatever the flag says; this
// driver does the same).
//
// Verified against the reference's wire constants only. No RFOne has been
// on this driver yet, so per #764/#771 it is unverified on air.

const (
	vidHydraSDR uint16 = 0x38af
	pidHydraSDR uint16 = 0x0001

	hydraDriverName = "hydrasdr"

	reqSetRFBias              uint8 = 20 // HYDRASDR_SET_RF_BIAS_CMD
	reqGetCapabilities        uint8 = 29 // HYDRASDR_GET_CAPABILITIES
	hydraLNAMaxGain                 = 14 // RFONE_LNA_MAX_GAIN
	hydraCapExtendedRates           = 1 << 21
	hydraRateInfoLen                = 8 // sizeof(hydrasdr_samplerate_info_t)
	hydraSetRateExtReplyLen         = 4 // result, adc_bits, data_format, reserved
	hydraTunerName                  = "R828D (HydraSDR RFOne)"
	hydraLegacyProductKeyword       = "HYDRASDR"
)

// NewHydraSDR returns the HydraSDR RFOne driver backed by enum (nil →
// platform default). It is the Airspy driver with the RFOne protocol
// differences switched on; see the file comment.
func NewHydraSDR(enum usb.Enumerator) *Driver {
	if enum == nil {
		enum = usb.DefaultEnumerator()
	}
	return &Driver{enum: enum, hydra: true}
}

func (d *Driver) driverName() string {
	if d.hydra {
		return hydraDriverName
	}
	return driverName
}

func (d *Driver) tunerName(product string) string {
	if d.hydra {
		return hydraTunerName
	}
	return tunerNameFor(product)
}

// isHydraProduct reports whether a 0x1d50:0x60a1 device is really a HydraSDR
// on the historic Airspy IDs. Its USB Product string is the only thing that
// tells the two apart before opening it.
func isHydraProduct(product string) bool {
	return strings.Contains(strings.ToUpper(product), hydraLegacyProductKeyword)
}

// listDescriptors returns the devices this driver owns. The two drivers split
// the shared 0x1d50:0x60a1 pair by Product string so one device is never
// listed (and opened) twice: the HydraSDR driver takes the official IDs plus
// any legacy-ID device naming itself HydraSDR, the Airspy driver the rest.
func (d *Driver) listDescriptors() ([]usb.Descriptor, error) {
	legacy, err := d.enum.List(vidAirspy, pidAirspy)
	if err != nil {
		return nil, err
	}
	var out []usb.Descriptor
	if d.hydra {
		official, err := d.enum.List(vidHydraSDR, pidHydraSDR)
		if err != nil {
			return nil, err
		}
		out = append(out, official...)
	}
	for _, desc := range legacy {
		if isHydraProduct(desc.Product) == d.hydra {
			out = append(out, desc)
		}
	}
	return out, nil
}

// hydraFreqPayload is SET_FREQ's 8-byte little-endian uint64 frequency.
func hydraFreqPayload(hz uint32) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(hz))
	return b
}

// setSamplerateReplyLen is how many bytes SET_SAMPLERATE is read with:
// libhydrasdr asks extended-rate firmware for 4, everything else for 1.
func (d *Device) setSamplerateReplyLen() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.hydra && d.rateADCBits != nil {
		return hydraSetRateExtReplyLen
	}
	return 1
}

// adcBitsForRateLocked returns the ADC width the firmware streams for the
// requested IQ rate (the table entry it snaps to), or 0 for the 12-bit
// default when no extended rate info was read. Callers hold d.mu.
func (d *Device) adcBitsForRateLocked(iqHz uint32) uint8 {
	if d.rateADCBits == nil || len(d.rates) == 0 {
		return 0
	}
	idx := -1
	for i, r := range d.rates {
		if r == iqHz {
			idx = i
			break
		}
	}
	if idx < 0 {
		best, bestDiff := 0, ^uint32(0)
		for i, r := range d.rates {
			diff := r - iqHz
			if iqHz > r {
				diff = iqHz - r
			}
			if diff < bestDiff {
				best, bestDiff = i, diff
			}
		}
		idx = best
	}
	if idx >= len(d.rateADCBits) {
		return 0
	}
	return d.rateADCBits[idx]
}

// fetchHydraRateInfo reads GET_CAPABILITIES and, when the firmware has
// extended sample-rate info, the per-rate ADC width. Any failure leaves the
// device on the 12-bit default, which is what pre-capability firmware streams.
func (d *Device) fetchHydraRateInfo() {
	d.mu.Lock()
	rates := d.rates
	d.mu.Unlock()
	if len(rates) == 0 {
		return
	}
	caps, err := d.t.ControlIn(reqGetCapabilities, 0, 0, 4, controlTimeoutMs)
	if err != nil || len(caps) < 4 {
		return
	}
	if binary.LittleEndian.Uint32(caps)&hydraCapExtendedRates == 0 {
		return
	}
	n := len(rates)
	info, err := d.t.ControlIn(reqGetSamplerates, 1, uint16(n), n*hydraRateInfoLen, controlTimeoutMs)
	if err != nil || len(info) < n*hydraRateInfoLen {
		return
	}
	bits := make([]uint8, n)
	for i := range bits {
		rec := info[i*hydraRateInfoLen:]
		if binary.LittleEndian.Uint32(rec) != rates[i] {
			// The extended table must describe the same rates in the same
			// order; if it doesn't, trust neither and stay on the default.
			return
		}
		bits[i] = rec[4]
	}
	d.mu.Lock()
	d.rateADCBits = bits
	d.mu.Unlock()
}
