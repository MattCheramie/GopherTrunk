package main

import (
	"context"
	"errors"
	"sync"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/tuner"
)

// pagingWidebandGuardFrac is the share of the SDR's IQ band a wideband
// paging group leaves unused at each edge (the DDC bank's alias guard).
const pagingWidebandGuardFrac = 0.05

// newWidebandPagingBank builds the DDC bank of a paging.wideband group on an
// SDR streaming at sampleRateHz. Construction (buildWidebandPagingGroup) and
// every stream session (decodeWidebandPagingGroup) build it the same way, so
// a channel accepted at start-up is always in band when its tap is added.
func newWidebandPagingBank(sampleRateHz uint32) *tuner.DDCBank {
	return tuner.NewDDCBank(float64(sampleRateHz), pagingWidebandRateHz, pagingWidebandGuardFrac)
}

// runWidebandPagingGroup drives one paging.wideband group for the daemon's
// lifetime: tune the group's SDR to the group centre, run its IQ stream, and
// split that stream through a tuner.DDCBank into one narrow-band tap per
// channel, each feeding its POCSAG / FLEX receiver.
//
// The group streams the SDR itself. It used to only Subscribe to the SDR's
// iqtap broker, and a broker fans IQ out only while a primary StreamIQ
// session runs: on a paging-only config (no trunking system on that dongle)
// nothing ever started one, so the group decoded pages only while a live
// view — the Spectrum panel — was open and streaming the dongle for its own
// sake, and went deaf again when the view closed. It now goes through
// runSingleChannelDecoder like every other decoder that pins an SDR: the
// first decoder on a serial drives StreamIQ (later ones Subscribe), and a
// failed tune, a stream that cannot open or a stream that ends are retried.
func (d *Daemon) runWidebandPagingGroup(ctx context.Context, g widebandPagingGroup) error {
	freqs := make([]uint32, len(g.channels))
	for i, ch := range g.channels {
		freqs[i] = ch.freq
	}
	d.log.Info("paging.wideband: group starting",
		"serial", g.serial, "center_hz", g.centerFreq,
		"sample_rate_hz", g.sampleRateHz, "channels_hz", freqs)
	return d.runSingleChannelDecoder(ctx, "paging.wideband", g.serial, g.centerFreq, nil,
		func(ctx context.Context, in <-chan []complex64) error {
			return d.decodeWidebandPagingGroup(ctx, g, in)
		})
}

// decodeWidebandPagingGroup runs the group's receivers over one IQ stream
// session: it pumps in through a fresh DDC bank until the stream or ctx
// ends, then closes the per-channel feeds so the receivers drain and exit.
func (d *Daemon) decodeWidebandPagingGroup(ctx context.Context, g widebandPagingGroup, in <-chan []complex64) error {
	bank := newWidebandPagingBank(g.sampleRateHz)

	// One buffered IQ channel per registered tap. The DDC sink copies each
	// narrow-band chunk into the feed (the bank reuses its output slice
	// across taps); a full feed drops the chunk rather than back-pressuring
	// the shared bank and stalling the sibling taps. Paging is low-rate, so
	// the 64-deep buffer is effectively never exhausted.
	var (
		wg    sync.WaitGroup
		feeds []chan []complex64
	)
	for i := range g.channels {
		ch := g.channels[i]
		feed := make(chan []complex64, 64)
		if err := bank.AddTap(ch.offsetHz, func(out []complex64) {
			if len(out) == 0 {
				return
			}
			cp := make([]complex64, len(out))
			copy(cp, out)
			select {
			case feed <- cp:
			default:
			}
		}); err != nil {
			// buildWidebandPagingGroup already dropped out-of-band channels
			// against the same bank, so this is not expected.
			d.log.Warn("paging.wideband: channel offset out of band, skipping",
				"serial", g.serial, "freq_hz", ch.freq,
				"offset_hz", ch.offsetHz, "err", err)
			continue
		}
		feeds = append(feeds, feed)
		wg.Add(1)
		go func() {
			defer wg.Done()
			var perr error
			switch {
			case ch.pocsag != nil:
				perr = ch.pocsag.Process(ctx, feed)
			case ch.flex != nil:
				perr = ch.flex.Process(ctx, feed)
			}
			if perr != nil && !errors.Is(perr, context.Canceled) {
				d.log.Warn("paging.wideband: receiver exited with error",
					"serial", g.serial, "freq_hz", ch.freq, "err", perr)
			}
		}()
	}
	defer func() {
		for _, f := range feeds {
			close(f)
		}
		wg.Wait()
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case chunk, ok := <-in:
			if !ok {
				return nil
			}
			bank.Process(chunk)
		}
	}
}
