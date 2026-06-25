package config

import (
	"testing"
	"time"
)

func TestDisplayConfigLocation(t *testing.T) {
	tests := []struct {
		name string
		tz   string
		want string // *time.Location.String()
	}{
		{"empty defaults to local", "", time.Local.String()},
		{"Local keyword", "Local", time.Local.String()},
		{"lowercase local", "local", time.Local.String()},
		{"UTC", "UTC", "UTC"},
		{"lowercase utc", "utc", "UTC"},
		{"IANA name", "America/New_York", "America/New_York"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, err := DisplayConfig{Timezone: tt.tz}.LocationStrict()
			if err != nil {
				t.Fatalf("LocationStrict(%q) error = %v", tt.tz, err)
			}
			if loc == nil {
				t.Fatalf("LocationStrict(%q) returned nil location", tt.tz)
			}
			if loc.String() != tt.want {
				t.Errorf("LocationStrict(%q) = %q, want %q", tt.tz, loc.String(), tt.want)
			}
		})
	}
}

func TestDisplayConfigLocationBadNameFallsBack(t *testing.T) {
	d := DisplayConfig{Timezone: "Not/AZone"}
	loc, err := d.LocationStrict()
	if err == nil {
		t.Fatal("LocationStrict on a bad zone: want error, got nil")
	}
	if loc != time.Local {
		t.Errorf("LocationStrict on a bad zone = %v, want time.Local fallback", loc)
	}
	// Location() swallows the error and still yields the fallback.
	if got := d.Location(); got != time.Local {
		t.Errorf("Location() on a bad zone = %v, want time.Local fallback", got)
	}
}
