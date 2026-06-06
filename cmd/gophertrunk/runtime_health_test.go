package main

import (
	"testing"
	"time"
)

func TestHeartbeatInterval(t *testing.T) {
	cases := []struct {
		name string
		sec  int
		want time.Duration
	}{
		{"negative disables", -1, 0},
		{"zero uses default", 0, 60 * time.Second},
		{"positive verbatim", 5, 5 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := heartbeatInterval(tc.sec); got != tc.want {
				t.Errorf("heartbeatInterval(%d) = %v, want %v", tc.sec, got, tc.want)
			}
		})
	}
}
