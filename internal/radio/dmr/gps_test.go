package dmr

import (
	"math"
	"testing"
)

func TestGPSInfoRoundTrip(t *testing.T) {
	cases := []struct {
		name     string
		lat, lon float64
	}{
		{"melbourne", -37.8136, 144.9631},
		{"nyc", 40.7128, -74.0060},
		{"equator-prime", 0.0001, 0.0001},
		{"southwest", -45.5, -170.25},
		{"near-limits", 89.9, 179.9},
	}
	// One quantisation step is the coarser of the two axes.
	const tol = 360.0 / (1 << 25)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := AssembleGPSInfo(GPSInfo{Latitude: tc.lat, Longitude: tc.lon})
			g, ok := ParseGPSInfo(info)
			if !ok {
				t.Fatal("ParseGPSInfo rejected a GPS info block")
			}
			if math.Abs(g.Latitude-tc.lat) > tol {
				t.Errorf("latitude: got %.6f want %.6f (Δ %.2g)", g.Latitude, tc.lat, g.Latitude-tc.lat)
			}
			if math.Abs(g.Longitude-tc.lon) > tol {
				t.Errorf("longitude: got %.6f want %.6f (Δ %.2g)", g.Longitude, tc.lon, g.Longitude-tc.lon)
			}
			if !g.Valid() {
				t.Errorf("fix %.4f,%.4f reported invalid", g.Latitude, g.Longitude)
			}
		})
	}
}

func TestGPSInfoSignedHemispheres(t *testing.T) {
	// A southern + western fix must sign-extend to negative degrees.
	info := AssembleGPSInfo(GPSInfo{Latitude: -12.5, Longitude: -60.75})
	g, _ := ParseGPSInfo(info)
	if g.Latitude >= 0 || g.Longitude >= 0 {
		t.Fatalf("sign extension failed: lat=%.4f lon=%.4f", g.Latitude, g.Longitude)
	}
}

func TestGPSInfoRejectsNonGPSFLCO(t *testing.T) {
	info := AssembleFLC(FLC{FLCO: FLCOGroupVoiceUser, DstAddr: 1, SrcAddr: 2})
	if _, ok := ParseGPSInfo(info); ok {
		t.Fatal("group-voice FLCO wrongly parsed as GPS info")
	}
}

func TestGPSInfoValidRejectsNullIsland(t *testing.T) {
	if (GPSInfo{Latitude: 0, Longitude: 0}).Valid() {
		t.Fatal("(0,0) placeholder treated as a valid fix")
	}
	if (GPSInfo{Latitude: 200, Longitude: 0}).Valid() {
		t.Fatal("out-of-range latitude treated as valid")
	}
}

func TestGPSInfoPositionError(t *testing.T) {
	info := AssembleGPSInfo(GPSInfo{Latitude: 1, Longitude: 1, PositionError: 5})
	g, _ := ParseGPSInfo(info)
	if g.PositionError != 5 {
		t.Fatalf("position error: got %d want 5", g.PositionError)
	}
}
