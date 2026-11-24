package main

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/iqtap"
)

// TestSpectrumProviderGroupStream pins the stitched group view: the group is
// listed with its panorama span and seams, a stream on group:<name> yields
// frames spanning the whole group from every member's broker, and tuning the
// group or one of its members is refused.
func TestSpectrumProviderGroupStream(t *testing.T) {
	_, plans, err := groupDaemonConfig().WithSDRGroupsExpanded()
	if err != nil {
		t.Fatal(err)
	}
	plan := plans[0]
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	brokers := map[string]*iqtap.Broker{}
	for _, m := range plan.Members {
		br := iqtap.New(newStreamingFake(m.Serial), 0, nil)
		_ = br.SetCenterFreq(m.CenterFreqHz)
		_ = br.SetSampleRate(plan.SampleRateHz)
		prim, err := br.StreamIQ(ctx)
		if err != nil {
			t.Fatalf("broker StreamIQ: %v", err)
		}
		go func() {
			for range prim {
			}
		}()
		brokers[m.Serial] = br
	}
	p := &spectrumProvider{brokers: brokers, groups: plans, log: slog.New(slog.DiscardHandler)}

	groups := p.Groups()
	if len(groups) != 1 {
		t.Fatalf("Groups() = %+v", groups)
	}
	g := groups[0]
	if g.Serial != "group:south" || !g.Group || len(g.Members) != 3 || len(g.SeamsHz) != 2 {
		t.Errorf("group device = %+v", g)
	}
	lo, hi := float64(g.CenterHz)-float64(g.SampleRateHz)/2, float64(g.CenterHz)+float64(g.SampleRateHz)/2
	if lo > float64(plan.Members[0].LowHz)+10_000 || hi < float64(plan.Members[2].HighHz)-10_000 {
		t.Errorf("panorama %.0f-%.0f Hz does not span the members %d-%d", lo, hi, plan.Members[0].LowHz, plan.Members[2].HighHz)
	}

	frames, cleanup, err := p.OpenStream(ctx, "group:south", 1024, 50)
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	defer cleanup()
	select {
	case f, ok := <-frames:
		if !ok {
			t.Fatal("group stream closed")
		}
		bin := float64(plan.SampleRateHz) / float64(memberFFTSize(1024, 3))
		if absf(float64(f.CenterHz)-float64(g.CenterHz)) > bin || absf(float64(f.SampleRateHz)-float64(g.SampleRateHz)) > 2*bin {
			t.Errorf("frame span %d/%d, device says %d/%d", f.CenterHz, f.SampleRateHz, g.CenterHz, g.SampleRateHz)
		}
		if want := 3 * memberFFTSize(1024, 3); len(f.Bins) < want*8/10 || len(f.Bins) > want {
			t.Errorf("%d bins, want about %d", len(f.Bins), want)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no stitched frame within 3 s")
	}

	if err := p.Tune("group:south", 852_000_000); err == nil {
		t.Error("tuning a group was accepted")
	}
	if err := p.Tune(plan.Members[0].Serial, 852_000_000); err == nil || !strings.Contains(err.Error(), "member of SDR group") {
		t.Errorf("tuning a member: %v", err)
	}
}

func TestMemberFFTSize(t *testing.T) {
	for _, tc := range []struct{ req, n, want int }{
		{4096, 1, 4096}, {4096, 3, 1024}, {4096, 4, 1024}, {64, 3, 64}, {16384, 2, 8192},
	} {
		if got := memberFFTSize(tc.req, tc.n); got != tc.want {
			t.Errorf("memberFFTSize(%d,%d) = %d, want %d", tc.req, tc.n, got, tc.want)
		}
	}
}

func absf(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
