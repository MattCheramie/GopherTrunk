package trunking

import "testing"

func TestGrantStringTimeslot(t *testing.T) {
	tests := []struct {
		name string
		g    Grant
		want string
	}{
		{
			name: "no timeslot omits the ts token",
			g:    Grant{System: "Sys", Protocol: "p25", GroupID: 100, SourceID: 7, FrequencyHz: 851_000_000},
			want: "Sys/p25 tg=100 src=7 freq=851000000 ",
		},
		{
			name: "ts1 renders ts1",
			g:    Grant{System: "Sys", Protocol: "dmr-tier3", GroupID: 1, SourceID: 2, FrequencyHz: 460_000_000, Timeslot: 1},
			want: "Sys/dmr-tier3 tg=1 src=2 freq=460000000 ts1 ",
		},
		{
			name: "ts2 renders ts2 alongside flags",
			g:    Grant{System: "Sys", Protocol: "dmr-tier3", GroupID: 1, SourceID: 2, FrequencyHz: 460_000_000, Timeslot: 2, Emergency: true},
			want: "Sys/dmr-tier3 tg=1 src=2 freq=460000000 ts2 !",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.g.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}
