package survey

import (
	"github.com/MattCheramie/GopherTrunk/internal/scanner/conventional"
)

// AnalogReport summarises a conventional analog carrier: whether it is keyed up
// (carrier present) and any sub-audible squelch tone/code identifying the
// system. It reuses the conventional scanner's primitives — the same
// IQ-power squelch and CTCSS/DCS detectors that gate the live analog scanner —
// so the survey's analog verdict matches what the scanner would do.
type AnalogReport struct {
	Active   bool    `json:"active"`            // carrier power above squelch
	PowerDbFS float64 `json:"power_dbfs"`        // measured RMS power
	CTCSSHz  float64 `json:"ctcss_hz,omitempty"` // detected CTCSS tone, 0 if none
	DCSCode  string  `json:"dcs_code,omitempty"` // detected DCS code, "" if none
}

// analogSquelchDbFS is the carrier-present threshold (dBFS) for the survey. It
// matches the conventional scanner's default working point: a unity-amplitude
// tone is 0 dBFS, and a keyed FM carrier sits well above the noise floor.
const analogSquelchDbFS = -30

// standardCTCSSTones is the 38-tone EIA CTCSS set, in Hz. Scanning all of them
// blind is cheap (one Goertzel each) and identifies the repeater's tone.
var standardCTCSSTones = []float64{
	67.0, 69.3, 71.9, 74.4, 77.0, 79.7, 82.5, 85.4, 88.5, 91.5,
	94.8, 97.4, 100.0, 103.5, 107.2, 110.9, 114.8, 118.8, 123.0, 127.3,
	131.8, 136.5, 141.3, 146.2, 151.4, 156.7, 162.2, 167.9, 173.8, 179.9,
	186.2, 192.8, 203.5, 210.7, 218.1, 225.7, 233.6, 241.8,
}

// standardDCSCodes is the common EIA DCS (DPL) code set, as 3-digit octal
// strings. The blind scan tries each; a match identifies the digital squelch.
var standardDCSCodes = []string{
	"023", "025", "026", "031", "032", "043", "047", "051", "054", "065",
	"071", "072", "073", "074", "114", "115", "116", "125", "131", "132",
	"134", "143", "152", "155", "156", "162", "165", "172", "174", "205",
	"223", "226", "243", "244", "245", "251", "261", "263", "265", "271",
	"306", "311", "315", "331", "343", "346", "351", "364", "365", "371",
	"411", "412", "413", "423", "431", "432", "445", "464", "465", "466",
	"503", "506", "516", "532", "546", "565", "606", "612", "624", "627",
	"631", "632", "654", "662", "664", "703", "712", "723", "731", "732",
	"734", "743", "754",
}

// AnalyzeAnalogFM measures a baseband analog-FM carrier: carrier-present power
// plus a blind CTCSS-tone and DCS-code scan. iq is the channelised baseband
// capture and inputRateHz its sample rate. Returns the report (Active=false
// when no carrier clears squelch).
func AnalyzeAnalogFM(iq []complex64, inputRateHz uint32) *AnalogReport {
	rep := &AnalogReport{PowerDbFS: conventional.PowerDbFS(iq)}
	rep.Active = rep.PowerDbFS >= analogSquelchDbFS
	if !rep.Active {
		return rep
	}
	rep.CTCSSHz = scanCTCSS(iq, float64(inputRateHz))
	rep.DCSCode = scanDCS(iq, float64(inputRateHz))
	return rep
}

// scanCTCSS runs the standard CTCSS tones against the buffer and returns the
// first that locks (0 when none). Detectors are independent, so the first
// Present() wins; ties are rare because the reverse-bin rejection in
// CTCSSDetector keeps adjacent tones from co-triggering.
func scanCTCSS(iq []complex64, rateHz float64) float64 {
	for _, hz := range standardCTCSSTones {
		det := conventional.NewCTCSSDetector(conventional.CTCSSConfig{SampleHz: rateHz, TargetHz: hz})
		if det == nil {
			continue
		}
		if det.Process(iq) {
			return hz
		}
	}
	return 0
}

// scanDCS runs the common DCS codes against the buffer and returns the first
// that locks ("" when none).
func scanDCS(iq []complex64, rateHz float64) string {
	for _, code := range standardDCSCodes {
		det := conventional.NewDCSDetector(conventional.DCSConfig{SampleHz: rateHz, Code: code})
		if det == nil {
			continue
		}
		if det.Process(iq) {
			return det.Code()
		}
	}
	return ""
}
