package hunt

import (
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

func TestIdentityNote(t *testing.T) {
	tests := []struct {
		name      string
		res       *siglab.Result
		wantEmpty bool
		wantHas   string
	}{
		{
			name:      "nil result",
			res:       nil,
			wantEmpty: true,
		},
		{
			name:      "not locked",
			res:       &siglab.Result{Locked: false, Topology: &siglab.TopologySnapshot{}},
			wantEmpty: true,
		},
		{
			name:      "no topology",
			res:       &siglab.Result{Locked: true},
			wantEmpty: true,
		},
		{
			name: "identity resolved",
			res: &siglab.Result{
				Locked:   true,
				Topology: &siglab.TopologySnapshot{WACN: 0xABCDE, SystemID: 0x123},
				Detail:   &siglab.P25P1Detail{CCStats: &siglab.CCStatsBreakdown{}},
			},
			wantEmpty: true,
		},
		{
			name: "locked but NSB never decoded",
			res: &siglab.Result{
				Locked:   true,
				Topology: &siglab.TopologySnapshot{}, // WACN/SystemID zero
				Detail: &siglab.P25P1Detail{CCStats: &siglab.CCStatsBreakdown{
					TSBKDecoded:    40,
					RFSSStatusSeen: 0,
					AdjacentSeen:   3,
					NetStatusSeen:  0,
				}},
			},
			wantHas: "Network Status Broadcast",
		},
		{
			name: "NSB decoded but no WACN parsed",
			res: &siglab.Result{
				Locked:   true,
				Topology: &siglab.TopologySnapshot{},
				Detail: &siglab.P25P1Detail{CCStats: &siglab.CCStatsBreakdown{
					NetStatusSeen: 2,
				}},
			},
			wantHas: "parsed as zero",
		},
		{
			name: "not P25 (no detail)",
			res: &siglab.Result{
				Locked:   true,
				Topology: &siglab.TopologySnapshot{},
			},
			wantEmpty: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := identityNote(tt.res)
			if tt.wantEmpty {
				if got != "" {
					t.Fatalf("identityNote = %q, want empty", got)
				}
				return
			}
			if !strings.Contains(got, tt.wantHas) {
				t.Fatalf("identityNote = %q, want substring %q", got, tt.wantHas)
			}
		})
	}
}
