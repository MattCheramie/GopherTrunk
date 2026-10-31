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
//   - Sample packing is set OFF on every open (SET_PACKING 26, wIndex 0), as
//     libhydrasdr's hydrasdr_generic_device_init does. The firmware keeps the
//     packing state while powered, so a unit last used by an application
//     that turned packing on would otherwise stream 12-bit packed samples
//     this driver reads as unpacked uint16, i.e. noise.
//   - The IF filter bandwidth: GET_BANDWIDTHS (31) lists the firmware's
//     bandwidths (old firmware answers with none); after RX mode is switched
//     on, SET_BANDWIDTH (30, wIndex = table index) selects the narrowest one
//     at least as wide as the sample rate. The firmware applies its own
//     default when RX starts, so libhydrasdr's start_rx overrides it at that
//     point; this driver does the same, in StreamIQ.
//
// What is NOT used: RF port selection (the firmware default, port 0 = ANT,
// the only port with the bias tee), the unified SET_GAIN (33; libhydrasdr's
// RFOne gain calls still use the per-stage commands this driver sends), and
// the firmware's data_format flag (libhydrasdr reads it only to pick its ADC
// lookup table and runs its real-to-IQ converter whatever it says; this
// driver does the same).
//
// Verified against the reference's wire constants only. No RFOne has been
// on this driver yet, so per #764/#771 it is unverified on air.

const (
	vidHydraSDR uint16 = 0x38af
	pidHydraSDR uint16 = 0x0001

	hydraDriverName = "hydrasdr"

	reqSetRFBias              uint8 = 20  // HYDRASDR_SET_RF_BIAS_CMD
	reqSetPacking             uint8 = 26  // HYDRASDR_SET_PACKING
	reqGetCapabilities        uint8 = 29  // HYDRASDR_GET_CAPABILITIES
	reqSetBandwidth           uint8 = 30  // HYDRASDR_SET_BANDWIDTH
	reqGetBandwidths          uint8 = 31  // HYDRASDR_GET_BANDWIDTHS
	hydraMaxListLen                 = 100 // MAX_SUPPORTED_RATE_COUNT
	hydraLNAMaxGain                 = 14  // RFONE_LNA_MAX_GAIN
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

// hydraOpenInit runs the RFOne-only steps of opening, after the rate table
// has been read: the per-rate ADC width, the bandwidth table, and packing
// off. All are best effort, like the rest of open: a failure leaves the
// device on the defaults old firmware streams with.
func (d *Device) hydraOpenInit() {
	d.fetchHydraRateInfo()
	d.fetchHydraBandwidths()
	// Packing off, so the stream is the unpacked uint16 this driver
	// converts, whatever the last application left the firmware in.
	_, _ = d.t.ControlIn(reqSetPacking, 0, 0, 1, controlTimeoutMs)
}

// fetchHydraBandwidths reads the IF filter bandwidths (Hz) the firmware
// offers: a uint32 count, then that many uint32s, as libhydrasdr reads them.
// Firmware without the request leaves the table empty and the filter on the
// firmware's own default.
func (d *Device) fetchHydraBandwidths() {
	b, err := d.t.ControlIn(reqGetBandwidths, 0, 0, 4, controlTimeoutMs)
	if err != nil || len(b) < 4 {
		return
	}
	n := int(binary.LittleEndian.Uint32(b))
	if n <= 0 || n >= hydraMaxListLen {
		return
	}
	list, err := d.t.ControlIn(reqGetBandwidths, 0, uint16(n), 4*n, controlTimeoutMs)
	if err != nil || len(list) < 4*n {
		return
	}
	bws := make([]uint32, n)
	for i := range bws {
		bws[i] = binary.LittleEndian.Uint32(list[4*i:])
	}
	d.mu.Lock()
	d.bandwidths = bws
	d.mu.Unlock()
}

// hydraBandwidthIndex picks the narrowest advertised bandwidth at least as
// wide as iqHz (libhydrasdr's auto-bandwidth rule), or -1 when the firmware
// listed none or none is wide enough.
func hydraBandwidthIndex(bws []uint32, iqHz uint32) int {
	best := -1
	for i, bw := range bws {
		if bw >= iqHz && (best < 0 || bw < bws[best]) {
			best = i
		}
	}
	return best
}

// applyHydraBandwidth sets the IF filter for the programmed sample rate. It
// runs after RX mode is on, because the firmware sets its default bandwidth
// when RX starts. Best effort: on failure the firmware default stays.
func (d *Device) applyHydraBandwidth() {
	d.mu.Lock()
	bws, last := d.bandwidths, d.lastReqRate
	d.mu.Unlock()
	if len(bws) == 0 || last == 0 {
		return
	}
	idx := hydraBandwidthIndex(bws, d.resolvedIQRate(last))
	if idx < 0 {
		return
	}
	_, _ = d.t.ControlIn(reqSetBandwidth, 0, uint16(idx), 1, controlTimeoutMs)
}
