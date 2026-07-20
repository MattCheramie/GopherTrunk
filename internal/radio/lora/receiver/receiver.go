// Package receiver wires the LoRa PHY decoder onto a live wide-band IQ
// stream. One physical SDR capture is split by a tuner.Bank into N
// narrow-band LoRa sub-channels; each sub-channel runs a streaming
// dechirp/FFT demodulator (internal/radio/lora) and publishes one
// events.KindLoRaFrame per decoded frame.
//
// The pipeline is:
//
//	wide-band IQ chunks (InputRateHz, complex64)
//	  → tuner.Bank (ChannelizerBank when many taps, else DDCBank)
//	  → per-tap narrow-band IQ at osf*BW
//	  → lora.Demodulator(s) (one per candidate spreading factor)
//	  → lora.Frame → events.KindLoRaFrame on the bus
//
// Spreading factor is auto-detected by running a demodulator per candidate
// SF over each sub-channel's buffered samples; coding rate is read from the
// explicit header. This trades CPU for coverage — pinning a single SF per
// sub-channel in config avoids the parallel probes.
package receiver

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/tuner"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/lora"
	"github.com/MattCheramie/GopherTrunk/internal/radio/lora/lorawan"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// channelizerThreshold is the tap count at or above which the shared-filter
// ChannelizerBank beats per-tap DDCs (see internal/dsp/tuner docs).
const channelizerThreshold = 7

// ChannelConfig describes one LoRa sub-channel to decode within the
// dongle's IQ band.
type ChannelConfig struct {
	OffsetHz    float64       // offset from the dongle centre frequency
	FrequencyHz uint32        // absolute centre frequency (metadata only)
	SF          int           // spreading factor 7..12; 0 = auto-detect SF7..12
	SyncWord    uint8         // expected sync word (0x12 private, 0x34 LoRaWAN)
	LDRO        lora.LDROMode // Low Data Rate Optimize; LDROAuto (0) = the (SF,BW) rule
}

// Options configures a Receiver.
type Options struct {
	InputRateHz uint32          // wide-band SDR sample rate
	BW          lora.Bandwidth  // LoRa bandwidth for every sub-channel
	Oversample  int             // samples per chip; 0 → lora.DefaultOversample
	Channels    []ChannelConfig // sub-channels to decode
	Bus         *events.Bus     // frames publish on KindLoRaFrame
	Log         *slog.Logger
	// Keys, when set, enables LoRaWAN MAC decoding: the MIC is verified and
	// the FRMPayload decrypted for any frame whose DevAddr has session keys.
	// A nil store still parses the cleartext MAC header of LoRaWAN frames.
	Keys *lorawan.KeyStore
}

// Receiver runs the wide-band LoRa decode pipeline against a stream of IQ
// chunks.
type Receiver struct {
	bank tuner.Bank
	bus  *events.Bus
	log  *slog.Logger
	keys *lorawan.KeyStore
	chs  []*subChannel

	framesEmitted atomic.Uint64
}

// New constructs a Receiver. It builds the tuner bank and registers one tap
// per sub-channel. Returns an error if the bus is missing, no channels are
// configured, or a tap offset falls outside the usable IQ band.
func New(opts Options) (*Receiver, error) {
	if opts.Bus == nil {
		return nil, errors.New("lora/receiver: Bus is required")
	}
	if opts.InputRateHz == 0 {
		return nil, errors.New("lora/receiver: InputRateHz is required")
	}
	if len(opts.Channels) == 0 {
		return nil, errors.New("lora/receiver: at least one channel is required")
	}
	bw := opts.BW
	if bw == 0 {
		bw = lora.BW125
	}
	osf := opts.Oversample
	if osf <= 0 {
		osf = lora.DefaultOversample
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}

	inRate := float64(opts.InputRateHz)
	outRate := float64(bw) * float64(osf)
	if outRate >= inRate {
		return nil, fmt.Errorf("lora/receiver: output rate %.0f >= input rate %.0f", outRate, inRate)
	}

	r := &Receiver{bus: opts.Bus, log: log, keys: opts.Keys}
	r.bank = newBank(inRate, outRate, len(opts.Channels))

	for _, cc := range opts.Channels {
		sc := newSubChannel(cc, bw, osf, r.emit)
		if err := r.bank.AddTap(cc.OffsetHz, sc.sink); err != nil {
			return nil, fmt.Errorf("lora/receiver: tap at %.0f Hz: %w", cc.OffsetHz, err)
		}
		r.chs = append(r.chs, sc)
	}
	return r, nil
}

// newBank picks the cheaper tuner implementation for the tap count.
func newBank(inRate, outRate float64, taps int) tuner.Bank {
	const guardFrac = 0.05
	if taps >= channelizerThreshold {
		// Size the channelizer so taps land in distinct bins; a modest
		// branch length keeps the prototype filter sharp enough.
		channels := taps * 2
		return tuner.NewChannelizerBank(inRate, outRate, guardFrac, channels, 12, 8.0)
	}
	return tuner.NewDDCBank(inRate, outRate, guardFrac)
}

// Process pumps wide-band IQ chunks through the bank until ctx cancels or
// in closes. On a clean stream end it flushes each sub-channel's buffer.
func (r *Receiver) Process(ctx context.Context, in <-chan []complex64) error {
	if in == nil {
		return errors.New("lora/receiver: input channel is nil")
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case chunk, ok := <-in:
			if !ok {
				for _, sc := range r.chs {
					sc.flush()
				}
				return nil
			}
			r.bank.Process(chunk)
		}
	}
}

// loRaWANSyncWord is the public sync word that marks a frame as LoRaWAN.
const loRaWANSyncWord = 0x34

// emit publishes one decoded frame on the bus, attempting LoRaWAN MAC
// decoding (and, with keys, MIC verification + decryption) when the
// sub-channel carries the LoRaWAN public sync word.
func (r *Receiver) emit(cc ChannelConfig, f lora.Frame) {
	out := storage.LoRaFrame{
		FrequencyHz: cc.FrequencyHz,
		SF:          f.SF,
		CR:          f.CR,
		BandwidthHz: int(f.BW),
		SNR:         f.SNRdB,
		CFOHz:       f.CFOHz,
		CRCOK:       f.CRCOK,
		PayloadHex:  hex.EncodeToString(f.Payload),
		FPort:       -1,
	}
	if cc.SyncWord == loRaWANSyncWord {
		r.fillLoRaWAN(&out, f.Payload)
	}
	r.bus.Publish(events.Event{Kind: events.KindLoRaFrame, Payload: out})
	r.framesEmitted.Add(1)
}

// fillLoRaWAN parses payload as a LoRaWAN frame and populates the MAC fields
// of out. It tolerates a nil key store (header-only parse).
func (r *Receiver) fillLoRaWAN(out *storage.LoRaFrame, payload []byte) {
	ks := r.keys
	if ks == nil {
		ks = lorawan.NewKeyStore()
	}
	d, err := ks.Decode(payload)
	if err != nil {
		return // not a well-formed LoRaWAN frame
	}
	out.IsLoRaWAN = true
	out.MType = d.MType.String()
	if d.MType.IsData() {
		out.DevAddr = fmt.Sprintf("%08X", d.DevAddr)
		out.FCnt = d.FCnt32()
		out.FPort = d.FPort
		out.MICOK = d.MICOK
		out.Decrypted = d.Decrypted
		if d.Decrypted {
			out.DecodedJS = hex.EncodeToString(d.Plaintext)
		}
	}
}

// Stats reports cumulative counters for /metrics + debugging.
type Stats struct {
	FramesEmitted uint64
}

// Stats returns a snapshot of the receiver counters.
func (r *Receiver) Stats() Stats {
	return Stats{FramesEmitted: r.framesEmitted.Load()}
}

// --- per sub-channel streaming demod ------------------------------------

// maxBufferSamples bounds a sub-channel's working buffer. When exceeded
// without a decode, the oldest samples are dropped (keeping a tail so a
// frame straddling the boundary survives).
const maxBufferSamples = 1 << 22 // ~4M samples

type subChannel struct {
	cfg    ChannelConfig
	demods []*lora.Demodulator // one per candidate SF
	emit   func(ChannelConfig, lora.Frame)

	mu     sync.Mutex
	buf    []complex64
	cursor int
	// minDecode is the smallest buffered tail (past the cursor) that
	// guarantees the largest frame would be fully present, so a failed
	// decode after a preamble lock means a genuinely bad frame rather than
	// a truncated one.
	minDecode int
}

func newSubChannel(cfg ChannelConfig, bw lora.Bandwidth, osf int, emit func(ChannelConfig, lora.Frame)) *subChannel {
	sc := &subChannel{cfg: cfg, emit: emit}
	sfs := []int{cfg.SF}
	if cfg.SF == 0 {
		sfs = nil
		for sf := lora.MinSF; sf <= lora.MaxSF; sf++ {
			sfs = append(sfs, sf)
		}
	}
	maxSps := 0
	for _, sf := range sfs {
		d := lora.NewDemodulatorMode(sf, osf, bw, cfg.LDRO)
		sc.demods = append(sc.demods, d)
		if d.SymbolLen() > maxSps {
			maxSps = d.SymbolLen()
		}
	}
	// A 255-byte payload at CR 4/8 is the worst case; allow generous slack.
	sc.minDecode = maxSps * 700
	return sc
}

// sink is the tuner.SinkFunc: it copies the reused slice and decodes what
// it can.
func (sc *subChannel) sink(out []complex64) {
	if len(out) == 0 {
		return
	}
	sc.mu.Lock()
	sc.buf = append(sc.buf, out...)
	sc.decodeReady(false)
	sc.mu.Unlock()
}

// flush decodes any remaining buffered frames at stream end. Caller must
// not hold sc.mu.
func (sc *subChannel) flush() {
	sc.mu.Lock()
	sc.decodeReady(true)
	sc.buf = nil
	sc.cursor = 0
	sc.mu.Unlock()
}

// decodeReady attempts to decode frames from the cursor. When force is
// false it only decodes while enough trailing samples are buffered to hold
// a complete worst-case frame, so it never mistakes a truncated frame for a
// corrupt one. sc.mu must be held.
func (sc *subChannel) decodeReady(force bool) {
	for {
		avail := len(sc.buf) - sc.cursor
		if avail <= 0 {
			break
		}
		if !force && avail < sc.minDecode {
			break
		}
		prev := sc.cursor
		f, end, ok := sc.bestDecode(sc.cursor)
		if ok {
			sc.emit(sc.cfg, f)
		}
		// Always advance the cursor so the loop terminates, even if a
		// decode reported an end at or behind the cursor.
		if end <= prev {
			end = prev + 1
		}
		sc.cursor = end
		if !force {
			break
		}
	}
	sc.compact()
}

// bestDecode tries every candidate SF at start and returns the best decode
// (preferring a CRC-valid frame). When no demod decodes a frame, the
// returned end is the nearest safe skip a demod reported (past a failed
// preamble lock, or the buffer end when nothing locked) so the caller makes
// real forward progress instead of crawling one sample at a time.
func (sc *subChannel) bestDecode(start int) (lora.Frame, int, bool) {
	var best lora.Frame
	bestEnd := start
	found := false
	failNext := -1
	for _, d := range sc.demods {
		f, end, ok := d.DecodeFrom(sc.buf, start)
		if !ok {
			if failNext < 0 || end < failNext {
				failNext = end
			}
			continue
		}
		// Prefer a CRC-valid frame; otherwise keep the first valid header.
		if !found || (f.CRCOK && !best.CRCOK) {
			best, bestEnd, found = f, end, true
			if f.CRCOK {
				break
			}
		}
	}
	if found {
		return best, bestEnd, true
	}
	if failNext < 0 {
		failNext = len(sc.buf)
	}
	return best, failNext, false
}

// compact drops consumed samples once the cursor has advanced far, keeping
// memory bounded. sc.mu must be held.
func (sc *subChannel) compact() {
	if sc.cursor == 0 {
		// Bound an unproductive buffer that never locked.
		if len(sc.buf) > maxBufferSamples {
			keep := sc.minDecode
			drop := len(sc.buf) - keep
			sc.buf = append(sc.buf[:0], sc.buf[drop:]...)
		}
		return
	}
	if sc.cursor >= len(sc.buf) {
		sc.buf = sc.buf[:0]
		sc.cursor = 0
		return
	}
	sc.buf = append(sc.buf[:0], sc.buf[sc.cursor:]...)
	sc.cursor = 0
}
