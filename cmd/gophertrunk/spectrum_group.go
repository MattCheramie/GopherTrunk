package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/api"
	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/spectrum"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/iqtap"
)

// The stitched spectrum of an sdr.groups entry: one per-member producer on
// each member's broker, composed into a panorama by spectrum.Stitcher. The
// members are separate streams on separate clocks, so the panorama is a
// display composite (each bin from the member that owns it), not one
// coherent FFT.

// minGroupMemberFFT is the smallest per-member FFT a group stream runs.
const minGroupMemberFFT = 64

// groupRate is the rate the group's members stream at: the first member's
// broker rate when it has one (the actual hardware rate), else the plan's.
func (p *spectrumProvider) groupRate(plan config.SDRGroupPlan) uint32 {
	for _, m := range plan.Members {
		if br := p.brokers[m.Serial]; br != nil && br.SampleRateHz() != 0 {
			return br.SampleRateHz()
		}
	}
	return plan.SampleRateHz
}

func groupStitchMembers(plan config.SDRGroupPlan) []spectrum.StitchMember {
	out := make([]spectrum.StitchMember, len(plan.Members))
	for i, m := range plan.Members {
		out[i] = spectrum.StitchMember{OwnLowHz: m.OwnLowHz, OwnHighHz: m.OwnHighHz}
	}
	return out
}

// memberFFTSize splits a requested panorama bin count across the members:
// the largest power of two per member that keeps the panorama near the
// request, so a group stream costs about what one device's would.
func memberFFTSize(requested, members int) int {
	if members < 1 {
		members = 1
	}
	n := minGroupMemberFFT
	for n*2*members <= requested {
		n *= 2
	}
	return n
}

// Groups implements api.SpectrumGroupProvider.
func (p *spectrumProvider) Groups() []api.SpectrumDevice {
	if p == nil {
		return nil
	}
	out := make([]api.SpectrumDevice, 0, len(p.groups))
	for _, plan := range p.groups {
		if len(plan.Members) == 0 {
			continue
		}
		// The exact owned span; a stream's frames report it quantized to
		// that stream's bin width.
		lo, hi := uint64(plan.Members[0].OwnLowHz), uint64(plan.Members[len(plan.Members)-1].OwnHighHz)
		c, span := uint32((lo+hi+1)/2), uint32(hi-lo+1)
		dev := api.SpectrumDevice{
			Serial:       sdrGroupSpectrumPrefix + plan.Name,
			Driver:       "group",
			Product:      fmt.Sprintf("%d-SDR group", len(plan.Members)),
			Role:         "group",
			CenterHz:     c,
			SampleRateHz: span,
			Group:        true,
		}
		for i, m := range plan.Members {
			dev.Members = append(dev.Members, m.Serial)
			if i+1 < len(plan.Members) {
				dev.SeamsHz = append(dev.SeamsHz, m.OwnHighHz)
			}
		}
		out = append(out, dev)
	}
	return out
}

// groupPlanFor returns the plan a "group:<name>" id names.
func (p *spectrumProvider) groupPlanFor(serial string) (config.SDRGroupPlan, bool) {
	if !strings.HasPrefix(serial, sdrGroupSpectrumPrefix) {
		return config.SDRGroupPlan{}, false
	}
	name := strings.TrimPrefix(serial, sdrGroupSpectrumPrefix)
	for _, plan := range p.groups {
		if plan.Name == name {
			return plan, true
		}
	}
	return config.SDRGroupPlan{}, false
}

// openGroupStream runs one producer per member and emits a stitched frame
// once every member has delivered a fresh frame, or when the slowest has
// been silent for two frame intervals (a lost member then shows at the
// floor instead of freezing the panorama).
func (p *spectrumProvider) openGroupStream(ctx context.Context, plan config.SDRGroupPlan, fftSize int, fps float64) (<-chan api.SpectrumFrame, func(), error) {
	n := len(plan.Members)
	if n == 0 {
		return nil, nil, errors.New("spectrum: group has no members")
	}
	if fps <= 0 {
		fps = 10
	}
	memberFFT := memberFFTSize(fftSize, n)
	streamCtx, cancel := context.WithCancel(ctx)
	var (
		subs     []*iqtap.Subscriber
		releases []func()
	)
	cleanup := func() {
		cancel()
		for _, s := range subs {
			s.Close()
		}
		for _, r := range releases {
			r()
		}
	}
	type memberFrame struct {
		idx int
		f   spectrum.Frame
	}
	frames := make(chan memberFrame, 4*n)
	rate := p.groupRate(plan)
	for i, m := range plan.Members {
		br := p.brokers[m.Serial]
		if br == nil {
			cleanup()
			return nil, nil, fmt.Errorf("spectrum: group %s member %s is not in the pool: %w", plan.Name, m.Serial, api.ErrUnknownDevice)
		}
		prod, err := spectrum.New(spectrum.Options{
			FFTSize: memberFFT, FrameRate: fps,
			CenterFreqHz: br.CenterHz(), SampleRateHz: br.SampleRateHz(),
		})
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("spectrum: %w", err)
		}
		releases = append(releases, p.viewerIQ.acquire(m.Serial))
		sub := br.Subscribe()
		subs = append(subs, sub)
		out := make(chan spectrum.Frame, 2)
		go func() {
			defer close(out)
			_ = prod.Run(streamCtx, sub.C, out)
		}()
		go func(idx int, br *iqtap.Broker) {
			for f := range out {
				f.CenterHz, f.SampleRate = br.CenterHz(), br.SampleRateHz()
				select {
				case frames <- memberFrame{idx, f}:
				case <-streamCtx.Done():
					return
				}
			}
		}(i, br)
	}

	st := spectrum.NewStitcher(groupStitchMembers(plan), rate, memberFFT)
	center, span := st.Span()
	wireOut := make(chan api.SpectrumFrame, 8)
	go func() {
		defer close(wireOut)
		latest := make([]spectrum.Frame, n)
		fresh := make([]bool, n)
		interval := time.Duration(float64(time.Second) / fps)
		stale := time.NewTimer(2 * interval)
		defer stale.Stop()
		emit := func() bool {
			bins := st.Stitch(latest)
			for i := range fresh {
				fresh[i] = false
			}
			if !stale.Stop() {
				select {
				case <-stale.C:
				default:
				}
			}
			stale.Reset(2 * interval)
			if bins == nil {
				return true
			}
			select {
			case wireOut <- api.SpectrumFrame{
				TimestampNs: time.Now().UnixNano(), CenterHz: center, SampleRateHz: span, Bins: bins,
			}:
				return true
			case <-streamCtx.Done():
				return false
			}
		}
		for {
			select {
			case <-streamCtx.Done():
				return
			case mf := <-frames:
				latest[mf.idx], fresh[mf.idx] = mf.f, true
				all := true
				for _, f := range fresh {
					all = all && f
				}
				if all && !emit() {
					return
				}
			case <-stale.C:
				any := false
				for _, f := range latest {
					any = any || len(f.Bins) > 0
				}
				if any {
					if !emit() {
						return
					}
				} else {
					stale.Reset(2 * interval)
				}
			}
		}
	}()
	return wireOut, cleanup, nil
}

var _ api.SpectrumGroupProvider = (*spectrumProvider)(nil)
