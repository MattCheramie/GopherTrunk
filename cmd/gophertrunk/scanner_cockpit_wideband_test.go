package main

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/api"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/widebandt2"
)

// fakeWidebandHealth is a stand-in for a widebandt2.Engine's health surface.
type fakeWidebandHealth struct{ snap []widebandt2.SystemHealth }

func (f fakeWidebandHealth) SystemHealthSnapshot() []widebandt2.SystemHealth { return f.snap }

func findSystem(systems []api.SystemHuntStatusDTO, name string) *api.SystemHuntStatusDTO {
	for i := range systems {
		if systems[i].Name == name {
			return &systems[i]
		}
	}
	return nil
}

// TestScannerCockpitSurfacesWidebandSystems pins the reported DMR IPSC signal
// meter bug: systems hosted on a `role: wideband` device are stripped from the
// cchunt supervisor, so the scanner status (GET /api/v1/scanner) listed none of
// them — the web meter read "decode: —" with no dBFS while a single-channel
// TETRA control SDR read clean. The cockpit must now fold each wideband
// engine's per-system health into Status().Systems with the decode verdict,
// carrier offset and front-end level the meter needs.
func TestScannerCockpitSurfacesWidebandSystems(t *testing.T) {
	lockedAt := time.Unix(1_700_000_000, 0).UTC()
	c := scannerCockpit{
		wideband: []widebandHealthProvider{
			fakeWidebandHealth{snap: []widebandt2.SystemHealth{
				{
					Name:            "Fire2",
					Protocol:        "dmr-tier2",
					FreqHz:          442_387_500,
					Locked:          true,
					LockedAt:        lockedAt,
					DecodeQuality:   "clean",
					HasDecodeHealth: true,
					CarrierOffsetHz: -120,
					SignalDbFS:      -48.3,
					HasSignal:       true,
				},
				{
					// A still-hunting system: no lock, no decode health, but a
					// level reading is still useful.
					Name:       "Fire",
					Protocol:   "dmr-tier2",
					FreqHz:     441_012_500,
					Locked:     false,
					SignalDbFS: -71.0,
					HasSignal:  true,
				},
			}},
		},
	}

	st := c.Status()

	fire2 := findSystem(st.Systems, "Fire2")
	if fire2 == nil {
		t.Fatal("locked wideband system Fire2 is missing from scanner status")
	}
	if fire2.State != "locked" {
		t.Errorf("Fire2 state = %q, want locked", fire2.State)
	}
	if !fire2.HasDecodeHealth || fire2.DecodeQuality != "clean" {
		t.Errorf("Fire2 decode = %q (has=%v), want clean/true", fire2.DecodeQuality, fire2.HasDecodeHealth)
	}
	if !fire2.HasSignal || fire2.SignalDbFS != -48.3 {
		t.Errorf("Fire2 signal = %v (has=%v), want -48.3/true", fire2.SignalDbFS, fire2.HasSignal)
	}
	if fire2.LockedFreqHz != 442_387_500 {
		t.Errorf("Fire2 locked_freq_hz = %d, want 442387500", fire2.LockedFreqHz)
	}
	if fire2.CarrierOffsetHz != -120 {
		t.Errorf("Fire2 carrier_offset_hz = %d, want -120", fire2.CarrierOffsetHz)
	}
	if !fire2.LockedAt.Equal(lockedAt) {
		t.Errorf("Fire2 locked_at = %v, want %v", fire2.LockedAt, lockedAt)
	}

	fire := findSystem(st.Systems, "Fire")
	if fire == nil {
		t.Fatal("hunting wideband system Fire is missing from scanner status")
	}
	if fire.State != "hunting" {
		t.Errorf("Fire state = %q, want hunting", fire.State)
	}
	if fire.HasDecodeHealth {
		t.Error("Fire has_decode_health = true, want false (not locked)")
	}
	if fire.LockedFreqHz != 0 {
		t.Errorf("Fire locked_freq_hz = %d, want 0 (not locked)", fire.LockedFreqHz)
	}
	if !fire.HasSignal || fire.SignalDbFS != -71.0 {
		t.Errorf("Fire signal = %v (has=%v), want -71.0/true", fire.SignalDbFS, fire.HasSignal)
	}
}

// TestScannerCockpitWidebandDedupesByName guards against a system that is
// somehow present on BOTH paths being double-listed: the cchunt entry wins and
// the wideband duplicate is skipped.
func TestScannerCockpitWidebandDedupesByName(t *testing.T) {
	c := scannerCockpit{
		wideband: []widebandHealthProvider{
			fakeWidebandHealth{snap: []widebandt2.SystemHealth{
				{Name: "Dup", Protocol: "dmr-tier2", Locked: true, HasSignal: true, SignalDbFS: -50},
				{Name: "Dup", Protocol: "dmr-tier2", Locked: true, HasSignal: true, SignalDbFS: -50},
			}},
		},
	}
	st := c.Status()
	count := 0
	for _, s := range st.Systems {
		if s.Name == "Dup" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("system Dup listed %d times, want 1 (deduped)", count)
	}
}
