package survey

import "testing"

func TestSnapChannelBandwidth(t *testing.T) {
	cases := []struct {
		name string
		in   uint32
		want uint32
	}{
		{"the 11.6 kHz artifact reads as a 12.5 kHz channel", 11_600, 12_500},
		{"13.5 kHz reads as 12.5 kHz", 13_500, 12_500},
		{"6.0 kHz reads as 6.25 kHz", 6_000, 6_250},
		{"23 kHz reads as 25 kHz", 23_000, 25_000},
		{"exactly 12.5 kHz is unchanged", 12_500, 12_500},
		{"a genuinely wideband signal is left as-measured", 150_000, 150_000},
		{"an in-between value with no nearby standard is unchanged", 18_000, 18_000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SnapChannelBandwidth(tc.in); got != tc.want {
				t.Errorf("SnapChannelBandwidth(%d) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}
