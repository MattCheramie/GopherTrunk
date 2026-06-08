package main

import "testing"

func TestChooseHuntSDR(t *testing.T) {
	brokers := map[string]bool{"ctrl": true, "voice1": true, "voice2": true}
	has := func(s string) bool { return brokers[s] }

	cases := []struct {
		name       string
		requested  string
		control    string
		voice      []string
		wantSerial string
		wantBorrow bool
		wantErr    bool
	}{
		{
			name:      "explicit non-control dedicates",
			requested: "voice1", control: "ctrl", voice: []string{"voice1", "voice2"},
			wantSerial: "voice1", wantBorrow: false,
		},
		{
			name:      "explicit control borrows",
			requested: "ctrl", control: "ctrl", voice: []string{"voice1", "voice2"},
			wantSerial: "ctrl", wantBorrow: true,
		},
		{
			name:      "explicit unknown serial errors",
			requested: "nope", control: "ctrl", voice: []string{"voice1"},
			wantErr: true,
		},
		{
			name:      "auto prefers spare voice when >=2",
			requested: "", control: "ctrl", voice: []string{"voice1", "voice2"},
			wantSerial: "voice2", wantBorrow: false,
		},
		{
			name:      "auto borrows control with <2 voice",
			requested: "", control: "ctrl", voice: []string{"voice1"},
			wantSerial: "ctrl", wantBorrow: true,
		},
		{
			name:      "auto borrows control with no voice",
			requested: "", control: "ctrl", voice: nil,
			wantSerial: "ctrl", wantBorrow: true,
		},
		{
			name:      "no sdr at all errors",
			requested: "", control: "", voice: nil,
			wantErr: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			serial, borrow, err := chooseHuntSDR(c.requested, c.control, c.voice, has)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got serial=%q borrow=%v", serial, borrow)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if serial != c.wantSerial || borrow != c.wantBorrow {
				t.Errorf("got (%q, %v), want (%q, %v)", serial, borrow, c.wantSerial, c.wantBorrow)
			}
		})
	}
}
