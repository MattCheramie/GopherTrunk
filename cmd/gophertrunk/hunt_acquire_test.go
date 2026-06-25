package main

import "testing"

func TestChooseHuntSDR(t *testing.T) {
	// "" is keyed so the blank-USB-serial fallback case has a broker.
	brokers := map[string]bool{"ctrl": true, "voice1": true, "voice2": true, "": true}
	has := func(s string) bool { return brokers[s] }

	cases := []struct {
		name       string
		requested  string
		control    string
		voice      []string
		pool       []string
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
		{
			// Discovery use case: an SDR is pooled but no trunked system is
			// configured, so it has no control role and controlSerial is empty.
			name:      "auto falls back to any pooled broker (no control)",
			requested: "", control: "", voice: nil, pool: []string{"ctrl"},
			wantSerial: "ctrl", wantBorrow: false,
		},
		{
			// Blank USB serial: control SDR is keyed under "", so controlSerial
			// is "" but the device is still borrowable via the pool fallback.
			name:      "auto falls back to blank-serial control and borrows",
			requested: "", control: "", voice: nil, pool: []string{""},
			wantSerial: "", wantBorrow: true,
		},
		{
			name:      "auto errors when pooled SDR has no broker",
			requested: "", control: "", voice: nil, pool: []string{"nobroker"},
			wantErr: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			serial, borrow, err := chooseHuntSDR(c.requested, c.control, c.voice, c.pool, has)
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

func TestHuntBorrowBlocked(t *testing.T) {
	cases := []struct {
		name        string
		borrow      bool
		liveEngines int
		confirmed   bool
		want        bool
	}{
		{"dedicated spare never blocks", false, 1, false, false},
		{"borrow, no live engine, never blocks", true, 0, false, false},
		{"borrow with live engine, unconfirmed, blocks", true, 1, false, true},
		{"borrow with live engine, confirmed, proceeds", true, 1, true, false},
		{"borrow with multiple engines, unconfirmed, blocks", true, 2, false, true},
		{"confirm without borrow is irrelevant", false, 2, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := huntBorrowBlocked(c.borrow, c.liveEngines, c.confirmed); got != c.want {
				t.Errorf("huntBorrowBlocked(%v,%d,%v) = %v, want %v",
					c.borrow, c.liveEngines, c.confirmed, got, c.want)
			}
		})
	}
}
