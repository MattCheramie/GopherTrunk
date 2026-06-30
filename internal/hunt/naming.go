package hunt

import (
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/blind"
	"github.com/MattCheramie/GopherTrunk/internal/siglab/sigref"
	"github.com/MattCheramie/GopherTrunk/internal/survey"
)

// nameRefFloor is the minimum reference-catalog match score at which an
// undecoded carrier is named a best guess (matching the wideband survey's
// keep floor). Below it the carrier is named by its modulation class instead.
const nameRefFloor = 0.40

// nameSignal fills the consolidated inventory fields — Name, Service, Purpose —
// for any signal, decoded or not. Service/Purpose come from the frequency
// allocation; Name prefers the decoded protocol, then a best-guess from the
// reference catalog, then the modulation class. It never overrides a value the
// router already set and is safe to call once per signal before sanitize.
func nameSignal(ds *DetectedSignal) {
	if a, ok := sigref.Lookup(ds.FreqHz); ok {
		ds.Service = a.Service
		ds.Purpose = a.Purpose
	}

	switch {
	case ds.Trunking != nil && ds.Trunking.Protocol != "":
		if e, ok := sigref.ByProtocol(ds.Trunking.Protocol); ok {
			ds.Name = e.DisplayName
		} else {
			ds.Name = strings.ToUpper(ds.Trunking.Protocol)
		}
	case len(ds.Pages) > 0:
		ds.Name = ds.Pages[0].Protocol
	default:
		obs := sigref.Observation{
			SymbolRateHz: ds.BaudHz,
			ModClass:     modClassFor(ds.Class),
			Levels:       ds.Features.IFModality,
			ChannelBWHz:  float64(ds.OccupiedBwHz),
			CenterFreqHz: ds.FreqHz,
		}
		if m := sigref.Rank(obs, 1); len(m) > 0 && m[0].Score >= nameRefFloor {
			ds.Name = m[0].Entry.DisplayName
			if ds.Confidence == 0 { // fill confidence for an undecoded/wideband row
				ds.Confidence = m[0].Score
			}
		} else if ds.Wideband {
			ds.Name = "wideband signal"
		} else {
			ds.Name = string(ds.Class)
		}
	}
}

// modClassFor maps a survey signal class to the blind modulation class sigref
// scores against. Classes with no clean blind analog (analog families, data,
// wideband) map to "" so they don't constrain the reference match.
func modClassFor(c survey.SignalClass) string {
	switch c {
	case survey.ClassFSK:
		return blind.ModFSK2
	case survey.ClassC4FM:
		return blind.ModFSK4
	case survey.ClassPSK:
		return blind.ModPSK
	default:
		return ""
	}
}

// widebandSignal builds the inventory row for a stitched wideband-occupancy
// candidate: it is named (cellular/WiFi/…) and captured, never decoded.
// OccupiedBwHz is the stitched bandwidth; SNRDb the span's level over the floor.
func widebandSignal(cand Candidate) DetectedSignal {
	ds := DetectedSignal{
		FreqHz:       cand.FreqHz,
		SNRDb:        cand.SNRDb,
		OccupiedBwHz: cand.BwHz,
		Class:        survey.ClassWideband,
		Wideband:     true,
	}
	nameSignal(&ds)
	ds.sanitize()
	return ds
}
