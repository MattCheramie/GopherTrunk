package api

import (
	"errors"
	"strings"
	"testing"
)

type fakeGroupResolver struct {
	lo, hi uint32
}

func (f *fakeGroupResolver) ResolveCaptureGroup(name string, lo, hi uint32) (string, bool, error) {
	if name != "south" {
		return "", false, nil
	}
	f.lo, f.hi = lo, hi
	if hi > 853_000_000 {
		return "", true, errors.New("straddles the seam")
	}
	return "M1", true, nil
}

func TestResolveCaptureGroupSerial(t *testing.T) {
	r := &fakeGroupResolver{}
	serial, isGroup, err := resolveCaptureGroupSerial(r, captureRequest{Serial: "south", CentersHz: []uint32{851_100_000, 851_200_000}, BandwidthHz: 25_000})
	if err != nil || !isGroup || serial != "M1" {
		t.Fatalf("resolve = %q %v %v", serial, isGroup, err)
	}
	if r.lo != 851_087_500 || r.hi != 851_212_500 {
		t.Errorf("band asked = %d..%d, want the slices' union", r.lo, r.hi)
	}
	if _, isGroup, err := resolveCaptureGroupSerial(r, captureRequest{Serial: "south"}); !isGroup || err == nil || !strings.Contains(err.Error(), "no single full-band stream") {
		t.Errorf("full-band group capture: %v %v", isGroup, err)
	}
	if _, _, err := resolveCaptureGroupSerial(r, captureRequest{Serial: "south", CenterHz: 853_500_000, BandwidthHz: 25_000}); err == nil {
		t.Error("seam-straddling slice accepted")
	}
	if _, isGroup, err := resolveCaptureGroupSerial(r, captureRequest{Serial: "mock-00"}); isGroup || err != nil {
		t.Errorf("plain serial: %v %v", isGroup, err)
	}
}
