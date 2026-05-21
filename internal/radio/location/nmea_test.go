package location

import (
	"errors"
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-3 }

func TestParseGGA(t *testing.T) {
	p, err := ParseNMEA("$GPGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,*47")
	if err != nil {
		t.Fatalf("ParseNMEA GGA: %v", err)
	}
	if !approx(p.Latitude, 48.1173) {
		t.Errorf("latitude = %f, want ~48.1173", p.Latitude)
	}
	if !approx(p.Longitude, 11.516667) {
		t.Errorf("longitude = %f, want ~11.5167", p.Longitude)
	}
	if !p.Valid() {
		t.Error("fix should be valid")
	}
}

func TestParseRMC(t *testing.T) {
	p, err := ParseNMEA("$GPRMC,123519,A,4807.038,N,01131.000,E,022.4,084.4,230394,003.1,W*6A")
	if err != nil {
		t.Fatalf("ParseNMEA RMC: %v", err)
	}
	if !approx(p.Latitude, 48.1173) || !approx(p.Longitude, 11.516667) {
		t.Errorf("coords = %f,%f", p.Latitude, p.Longitude)
	}
	if !p.HasSpeed || !approx(p.SpeedKnots, 22.4) {
		t.Errorf("speed = %f (has=%v)", p.SpeedKnots, p.HasSpeed)
	}
	if !p.HasHeading || !approx(p.HeadingDeg, 84.4) {
		t.Errorf("heading = %f (has=%v)", p.HeadingDeg, p.HasHeading)
	}
}

func TestParseRMCSouthWest(t *testing.T) {
	// Same magnitudes, southern + western hemispheres → negative.
	p, err := ParseNMEA("$GPRMC,123519,A,4807.038,S,01131.000,W,000.0,000.0,230394,,*12")
	if err != nil {
		t.Fatalf("ParseNMEA: %v", err)
	}
	if p.Latitude >= 0 || p.Longitude >= 0 {
		t.Errorf("S/W hemispheres should be negative: %f,%f", p.Latitude, p.Longitude)
	}
}

func TestParseRMCVoidStatus(t *testing.T) {
	_, err := ParseNMEA("$GPRMC,,V,,,,,,,,,,N")
	if !errors.Is(err, ErrNoFix) {
		t.Fatalf("void RMC error = %v, want ErrNoFix", err)
	}
}

func TestParseNMEAChecksumMismatch(t *testing.T) {
	// Valid body, wrong checksum byte.
	_, err := ParseNMEA("$GPGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,*00")
	if err == nil {
		t.Fatal("a bad checksum should be rejected")
	}
}

func TestParseNMEARejects(t *testing.T) {
	cases := []string{
		"GPGGA,no,dollar",
		"$GPXYZ,unsupported",
		"",
		"$",
	}
	for _, c := range cases {
		if _, err := ParseNMEA(c); err == nil {
			t.Errorf("ParseNMEA(%q) should error", c)
		}
	}
}

func TestPositionValid(t *testing.T) {
	if (Position{}).Valid() {
		t.Error("zero position (Null Island) must be invalid")
	}
	if (Position{Latitude: 91, Longitude: 0}).Valid() {
		t.Error("out-of-range latitude must be invalid")
	}
	if !(Position{Latitude: 40.7, Longitude: -74}).Valid() {
		t.Error("a real fix must be valid")
	}
}
