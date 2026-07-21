package main

import (
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
)

// TestWarnPagingNeedsStorage is the regression test for issue #565: a paging
// subsystem configured without storage.path decodes fine but persists
// nothing, so GET /api/v1/pager/messages returns 503 and the operator sees
// the Pagers panel "not working" with no hint that storage.path is the
// missing piece. The daemon must surface that at load time as a startup
// warning. Before the fix no such warning existed — paging + no storage was
// silent.
func TestWarnPagingNeedsStorage(t *testing.T) {
	pocsag := config.Config{
		Paging: config.PagingConfig{
			POCSAG: []config.PagingPOCSAGConfig{{Serial: "00000022", FrequencyHz: 153_350_000}},
		},
	}

	cases := []struct {
		name      string
		cfg       config.Config
		wantWarn  bool
		mustMatch []string
	}{
		{
			name:      "pocsag without storage warns",
			cfg:       pocsag,
			wantWarn:  true,
			mustMatch: []string{"storage.path", "paging", "565"},
		},
		{
			name: "flex without storage warns",
			cfg: config.Config{
				Paging: config.PagingConfig{
					FLEX: []config.PagingFLEXConfig{{Serial: "00000022", FrequencyHz: 148_000_000}},
				},
			},
			wantWarn: true,
		},
		{
			name: "wideband paging without storage warns",
			cfg: config.Config{
				Paging: config.PagingConfig{
					Wideband: []config.PagingWidebandConfig{{Serial: "pager-sdr"}},
				},
			},
			wantWarn: true,
		},
		{
			name: "paging with storage set does not warn",
			cfg: func() config.Config {
				c := pocsag
				c.Storage.Path = "/var/lib/gophertrunk/gt.db"
				return c
			}(),
			wantWarn: false,
		},
		{
			name:     "no paging configured does not warn",
			cfg:      config.Config{},
			wantWarn: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := &Daemon{}
			d.warnPagingNeedsStorage(tc.cfg)
			warns := d.StartupWarnings()

			if !tc.wantWarn {
				if len(warns) != 0 {
					t.Fatalf("expected no warning, got %v", warns)
				}
				return
			}

			if len(warns) != 1 {
				t.Fatalf("expected exactly one warning, got %d: %v", len(warns), warns)
			}
			for _, sub := range tc.mustMatch {
				if !strings.Contains(warns[0], sub) {
					t.Errorf("warning %q missing expected substring %q", warns[0], sub)
				}
			}
		})
	}
}
