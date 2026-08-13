package trunking

import (
	"reflect"
	"testing"
)

func TestParseProtocol(t *testing.T) {
	cases := map[string]Protocol{"p25": ProtocolP25, "dmr": ProtocolDMR, "nxdn": ProtocolNXDN}
	for in, want := range cases {
		got, err := ParseProtocol(in)
		if err != nil || got != want {
			t.Errorf("ParseProtocol(%q) = %v, %v; want %v, nil", in, got, err, want)
		}
	}
	if _, err := ParseProtocol("lte"); err == nil {
		t.Error("expected error for unknown protocol")
	}
}

func TestProtocolCampsWhenIdle(t *testing.T) {
	camp := []Protocol{ProtocolDMRTier2, ProtocolDMRTier1, ProtocolTETRADMO}
	for _, p := range camp {
		if !p.CampsWhenIdle() {
			t.Errorf("CampsWhenIdle(%s) = false, want true", p)
		}
	}
	// Trunked protocols (continuous control channel) must NOT camp — an
	// idle hunt round is a real failure for them.
	trunked := []Protocol{ProtocolP25, ProtocolP25Phase2, ProtocolDMR, ProtocolNXDN,
		ProtocolTETRA, ProtocolMotorola, ProtocolEDACS, ProtocolLTR, ProtocolMPT1327}
	for _, p := range trunked {
		if p.CampsWhenIdle() {
			t.Errorf("CampsWhenIdle(%s) = true, want false", p)
		}
	}
}

func TestSystemValidate(t *testing.T) {
	good := System{Name: "Test", Protocol: ProtocolP25, ControlChannels: []uint32{851_000_000}}
	if err := good.Validate(); err != nil {
		t.Errorf("good system: %v", err)
	}
	cases := []struct {
		name string
		sys  System
	}{
		{"missing name", System{Protocol: ProtocolP25, ControlChannels: []uint32{851_000_000}}},
		{"unknown protocol", System{Name: "X", ControlChannels: []uint32{851_000_000}}},
		{"no channels", System{Name: "X", Protocol: ProtocolP25}},
		{"freq too low", System{Name: "X", Protocol: ProtocolP25, ControlChannels: []uint32{100}}},
		{"freq too high", System{Name: "X", Protocol: ProtocolP25, ControlChannels: []uint32{2_000_000_000}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.sys.Validate(); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestHuntOrderPrefersLastKnown(t *testing.T) {
	s := System{
		Name:            "X",
		Protocol:        ProtocolP25,
		ControlChannels: []uint32{851_000_000, 852_000_000, 853_000_000},
	}
	got := s.HuntOrder(852_000_000)
	want := []uint32{852_000_000, 851_000_000, 853_000_000}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("HuntOrder = %v, want %v", got, want)
	}
}

func TestHuntOrderNoCacheReturnsConfigOrder(t *testing.T) {
	s := System{ControlChannels: []uint32{1, 2, 3}}
	got := s.HuntOrder(0)
	want := []uint32{1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("HuntOrder(0) = %v, want %v", got, want)
	}
}

func TestHuntOrderIgnoresUnknownLastKnown(t *testing.T) {
	s := System{ControlChannels: []uint32{1, 2, 3}}
	got := s.HuntOrder(99)
	want := []uint32{1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("HuntOrder(99) = %v, want %v", got, want)
	}
}
