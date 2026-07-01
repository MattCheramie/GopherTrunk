package sigref

import "sort"

// Allocation names what a slice of spectrum is *for* — the service and its
// purpose — independent of any modulation. Where the signal catalog (Entry /
// Rank) answers "what kind of signal is this" from its shape, the allocation
// table answers "what is this frequency used for" from where it sits. The
// full-spectrum survey pairs them: a carrier is named by its modulation and
// labeled by its allocation, so an undecodable emitter still reads as e.g.
// "FRS — license-free handheld" or "Cellular 700 MHz downlink".
//
// The table is a curated, US-default set (Region is carried for future
// expansion). Band edges are approximate — they are for human labeling, not
// regulatory precision.
type Allocation struct {
	LoHz    uint32 `json:"lo_hz"`
	HiHz    uint32 `json:"hi_hz"`
	Service string `json:"service"`
	Purpose string `json:"purpose"`
	Region  string `json:"region"`
}

// BwHz is the allocation's span.
func (a Allocation) BwHz() uint32 { return a.HiHz - a.LoHz }

// allocations is the curated freq→service table, US defaults. Order does not
// matter: Lookup returns the *narrowest* matching band so a specific sub-band
// (FRS) wins over a broad one (UHF public-safety).
var allocations = []Allocation{
	{530_000, 1_710_000, "AM broadcast", "commercial AM radio", "US"},
	{1_800_000, 2_000_000, "Amateur 160m", "ham radio", "US"},
	{3_500_000, 4_000_000, "Amateur 80m", "ham radio", "US"},
	{7_000_000, 7_300_000, "Amateur 40m", "ham radio", "US"},
	{14_000_000, 14_350_000, "Amateur 20m", "ham radio", "US"},
	{26_960_000, 27_410_000, "CB radio", "Citizens Band 27 MHz", "US"},
	{28_000_000, 29_700_000, "Amateur 10m", "ham radio", "US"},
	{50_000_000, 54_000_000, "Amateur 6m", "ham radio", "US"},
	{88_000_000, 108_000_000, "FM broadcast", "commercial FM radio", "US"},
	{108_000_000, 118_000_000, "Aeronautical nav", "VOR / ILS navigation", "US"},
	{118_000_000, 137_000_000, "Aircraft VHF", "air-traffic voice (AM)", "US"},
	{137_000_000, 138_000_000, "Weather satellite", "NOAA APT / metsat downlink", "US"},
	{144_000_000, 148_000_000, "Amateur 2m", "ham radio", "US"},
	{151_820_000, 154_600_000, "MURS / business", "license-free & business VHF", "US"},
	{156_000_000, 162_025_000, "Marine VHF", "maritime voice & AIS", "US"},
	{162_400_000, 162_550_000, "NOAA weather", "weather radio broadcast", "US"},
	{162_000_000, 174_000_000, "VHF public safety", "land-mobile (police/fire/EMS)", "US"},
	{225_000_000, 400_000_000, "Military UHF", "DoD air/ground (AM/PSK)", "US"},
	{420_000_000, 450_000_000, "Amateur 70cm", "ham radio", "US"},
	{450_000_000, 462_000_000, "UHF business", "land-mobile business", "US"},
	{462_000_000, 467_725_000, "FRS / GMRS", "license-free handheld", "US"},
	{470_000_000, 512_000_000, "UHF-T / public safety", "land-mobile & T-band", "US"},
	{512_000_000, 608_000_000, "TV broadcast", "UHF television (DTV)", "US"},
	{617_000_000, 652_000_000, "Cellular 600 MHz DL", "LTE/5G downlink", "US"},
	{663_000_000, 698_000_000, "Cellular 600 MHz UL", "LTE/5G uplink", "US"},
	{758_000_000, 775_000_000, "700 MHz public safety", "FirstNet / LTE public safety", "US"},
	{746_000_000, 757_000_000, "Cellular 700 MHz DL", "LTE/5G downlink", "US"},
	{776_000_000, 788_000_000, "Cellular 700 MHz UL", "LTE/5G uplink", "US"},
	{806_000_000, 824_000_000, "800 MHz SMR/public safety", "trunked land-mobile uplink", "US"},
	{824_000_000, 849_000_000, "Cellular 850 MHz UL", "LTE/5G uplink", "US"},
	{851_000_000, 869_000_000, "800 MHz SMR/public safety", "trunked land-mobile downlink", "US"},
	{869_000_000, 894_000_000, "Cellular 850 MHz DL", "LTE/5G downlink", "US"},
	{902_000_000, 928_000_000, "ISM 900 MHz", "license-free IoT/telemetry/LoRa", "US"},
	{978_000_000, 978_000_000, "ADS-B UAT", "aircraft surveillance (978 MHz)", "US"},
	{960_000_000, 1_215_000_000, "Aeronautical L-band", "DME / ADS-B / SSR", "US"},
	{1_090_000_000, 1_090_000_000, "ADS-B 1090ES", "aircraft transponder (1090 MHz)", "US"},
	{1_525_000_000, 1_559_000_000, "Satellite L-band DL", "Inmarsat / Iridium downlink", "US"},
	{1_559_000_000, 1_610_000_000, "GNSS L1", "GPS / GLONASS / Galileo", "US"},
	{1_710_000_000, 1_780_000_000, "Cellular AWS UL", "LTE/5G uplink", "US"},
	{1_850_000_000, 1_915_000_000, "Cellular PCS UL", "LTE/5G uplink", "US"},
	{1_930_000_000, 1_995_000_000, "Cellular PCS DL", "LTE/5G downlink", "US"},
	{2_110_000_000, 2_200_000_000, "Cellular AWS DL", "LTE/5G downlink", "US"},
	{2_400_000_000, 2_483_500_000, "ISM 2.4 GHz", "WiFi / Bluetooth / license-free", "US"},
	{2_500_000_000, 2_690_000_000, "Cellular 2.5 GHz (BRS)", "LTE/5G TDD", "US"},
	{3_550_000_000, 3_700_000_000, "CBRS 3.5 GHz", "shared LTE/5G", "US"},
	// Note: the U-NII 5 GHz WiFi band (5.15–5.895 GHz) is above the uint32 Hz
	// ceiling (~4.29 GHz) used throughout, and unreachable by the supported
	// front-ends, so it is intentionally omitted here.
}

// Lookup returns the narrowest allocation whose band contains centerHz, and
// whether one was found. Narrowest-wins so a specific sub-band beats a broad
// umbrella band that overlaps it.
func Lookup(centerHz uint32) (Allocation, bool) {
	var best Allocation
	found := false
	for _, a := range allocations {
		if centerHz < a.LoHz || centerHz > a.HiHz {
			continue
		}
		if !found || a.BwHz() < best.BwHz() {
			best = a
			found = true
		}
	}
	return best, found
}

// Allocations returns the full table (sorted by low edge) for display/export.
func Allocations() []Allocation {
	out := append([]Allocation(nil), allocations...)
	sort.Slice(out, func(i, j int) bool { return out[i].LoHz < out[j].LoHz })
	return out
}
