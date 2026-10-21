package main

import (
	"sync"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/voice/composer"
)

// decodedPCMHz is the rate of every vocoder's decoded PCM (the Vocoder
// interface contract): the recorder's digital-voice tap always carries 8 kHz,
// whatever recordings.sample_rate says.
const decodedPCMHz = 8000

// liveRates are the PCM rates on the live paths (#1242 follow-up). Two rates
// arrive — decoded digital voice at decodedPCMHz, analog FM at
// recordings.sample_rate — and each consumer assumes one: the host player and
// the web/gRPC stream run at the player's rate (the web WAV header reads it),
// the tone-out detector at recordings.sample_rate. Nothing converted between
// them, so with any rate other than 8000 configured, digital voice played on
// the live stream at the wrong speed (6x fast at 48000: a whistle), analog was
// mislabelled on gRPC, and tone-out mis-tuned on digital calls.
type liveRates struct {
	analogHz uint32 // the composer's FM-chain PCM, and tone-out's rate
	liveHz   uint32 // host player, web stream header, gRPC label
}

func liveRatesFor(cfg config.Config) liveRates {
	analog := cfg.Recordings.SampleRate
	if analog == 0 {
		analog = 8000
	}
	live := cfg.Audio.SampleRate
	if live == 0 {
		live = analog
	}
	return liveRates{analogHz: analog, liveHz: live}
}

// wireLiveSinks builds the two live fan-outs. analog is the composer's own
// sink list (its PCM is analog FM at lr.analogHz): the decoder, tone-out as
// is, and the player and stream converted to the live rate. decoded is the
// recorder's decoded-PCM tap (always decodedPCMHz): tone-out converted to
// the rate it is tuned for and the stream to the live rate, passed through
// level (the optional live-loudness AGC, which runs at decodedPCMHz) when
// set. tone and player may be nil.
func wireLiveSinks(lr liveRates, decoder, tone, player, stream composer.PCMSink, level func(composer.PCMSink) composer.PCMSink) (analog, decoded []composer.PCMSink) {
	analog = []composer.PCMSink{decoder}
	if tone != nil {
		analog = append(analog, tone)
		decoded = append(decoded, withRate(tone, decodedPCMHz, lr.analogHz))
	}
	if player != nil {
		analog = append(analog, withRate(player, lr.analogHz, lr.liveHz))
	}
	analog = append(analog, withRate(stream, lr.analogHz, lr.liveHz))
	live := withRate(stream, decodedPCMHz, lr.liveHz)
	if level != nil {
		live = level(live)
	}
	decoded = append(decoded, live)
	return analog, decoded
}

// withRate returns next fed through a resampler from fromHz to toHz, or next
// itself when the rates already agree (the default 8000 everywhere stays
// byte-identical).
func withRate(next composer.PCMSink, fromHz, toHz uint32) composer.PCMSink {
	if next == nil || fromHz == 0 || toHz == 0 || fromHz == toHz {
		return next
	}
	g := gcdU32(fromHz, toHz)
	return &rateSink{next: next, l: int(toHz / g), m: int(fromHz / g), per: map[string]*rateState{}}
}

// rateSink resamples PCM per device serial. Each serial keeps its own
// resampler so a call's audio is continuous across chunks; a new CallID on
// the serial (a reused voice tap) starts a fresh one so the previous call's
// filter tail cannot bleed into the next.
type rateSink struct {
	next composer.PCMSink
	l, m int

	mu  sync.Mutex
	per map[string]*rateState
}

type rateState struct {
	callID uint64
	rs     *dsp.Resampler
	in     []complex64
	out    []complex64
}

func (s *rateSink) convert(serial string, callID uint64, samples []int16) []int16 {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.per[serial]
	if st == nil || st.callID != callID {
		taps := (12*max(s.l, s.m) + s.l - 1) / s.l
		st = &rateState{callID: callID, rs: dsp.NewResampler(s.l, s.m, max(taps, 8), 6)}
		s.per[serial] = st
	}
	st.in = st.in[:0]
	for _, v := range samples {
		st.in = append(st.in, complex(float32(v), 0))
	}
	st.out = st.rs.Process(st.out[:0], st.in)
	pcm := make([]int16, len(st.out))
	for i, c := range st.out {
		pcm[i] = int16(min(max(real(c), -32768), 32767))
	}
	return pcm
}

func (s *rateSink) WritePCM(serial string, samples []int16) error {
	pcm := s.convert(serial, 0, samples)
	if len(pcm) == 0 {
		return nil
	}
	return s.next.WritePCM(serial, pcm)
}

func (s *rateSink) WritePCMForCall(serial string, callID uint64, samples []int16) error {
	pcm := s.convert(serial, callID, samples)
	if len(pcm) == 0 {
		return nil
	}
	if cs, ok := s.next.(interface {
		WritePCMForCall(string, uint64, []int16) error
	}); ok {
		return cs.WritePCMForCall(serial, callID, pcm)
	}
	return s.next.WritePCM(serial, pcm)
}

func gcdU32(a, b uint32) uint32 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
