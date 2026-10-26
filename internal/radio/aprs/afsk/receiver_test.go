package afsk

import (
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/aprs/hdlc"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// encodeAddress duplicates the ax25 test helper — we can't import
// _test.go files across packages.
func encodeAddress(callsign string, ssid uint8, last, hOrC bool) []byte {
	out := make([]byte, 7)
	for len(callsign) < 6 {
		callsign += " "
	}
	for i := 0; i < 6; i++ {
		out[i] = callsign[i] << 1
	}
	ssidByte := byte(0b01100000)
	ssidByte |= (ssid & 0x0F) << 1
	if hOrC {
		ssidByte |= 0x80
	}
	if last {
		ssidByte |= 0x01
	}
	out[6] = ssidByte
	return out
}

// hdlcFCS mirrors the ax25 CRC algorithm (CRC-16-CCITT, reflected,
// 0x8408 polynomial, init 0xFFFF, final XOR 0xFFFF).
func hdlcFCS(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0x8408
			} else {
				crc >>= 1
			}
		}
	}
	return ^crc
}

// buildAX25Body assembles a complete AX.25 UI frame body with the
// standard 0x03 / 0xF0 control + PID.
func buildAX25Body(dstCall string, dstSSID uint8, srcCall string, srcSSID uint8, info []byte) []byte {
	var buf []byte
	buf = append(buf, encodeAddress(dstCall, dstSSID, false, false)...)
	buf = append(buf, encodeAddress(srcCall, srcSSID, true, true)...)
	buf = append(buf, 0x03, 0xF0)
	buf = append(buf, info...)
	fcs := hdlcFCS(buf)
	buf = append(buf, byte(fcs&0xFF), byte(fcs>>8))
	return buf
}

// wrapHDLCBits encodes the AX.25 body in HDLC framing (opening flag,
// bit-stuffed body, closing flag) as a stream of LSB-first wire
// bits. With prepended preamble flags so the receiver's mean-
// tracker / FFSK / NRZI states have settle time before the real
// frame body starts.
func wrapHDLCBits(body []byte, preambleFlags int) []byte {
	const flag = hdlc.FlagPattern
	var bits []byte
	emitByte := func(b byte) {
		for i := 0; i < 8; i++ {
			bits = append(bits, (b>>i)&1)
		}
	}
	for i := 0; i < preambleFlags; i++ {
		emitByte(flag)
	}
	emitByte(flag)
	ones := 0
	for _, b := range body {
		for i := 0; i < 8; i++ {
			bit := (b >> i) & 1
			bits = append(bits, bit)
			if bit != 0 {
				ones++
				if ones == 5 {
					bits = append(bits, 0)
					ones = 0
				}
			} else {
				ones = 0
			}
		}
	}
	emitByte(flag)
	return bits
}

// nrziEncode is the transmitter-side inverse of NRZIDecoder. AX.25
// convention: input 0 → flip wire, input 1 → wire holds.
func nrziEncode(in []byte) []byte {
	out := make([]byte, len(in))
	var wire byte = 1 // arbitrary seed; receiver locks on the leading flag
	for i, b := range in {
		if b == 0 {
			wire ^= 1
		}
		out[i] = wire
	}
	return out
}

func newTestBus(t *testing.T) (*events.Bus, *events.Subscription) {
	t.Helper()
	bus := events.NewBus(8)
	sub := bus.Subscribe()
	t.Cleanup(func() {
		sub.Close()
		bus.Close()
	})
	return bus, sub
}

func TestReceiverNewRejectsBadOptions(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()

	if _, err := New(Options{}); err == nil {
		t.Error("New without Bus: want error")
	}
	if _, err := New(Options{Bus: bus}); err == nil {
		t.Error("New without InputRateHz: want error")
	}
}

func TestReceiverNewSetsUpInner(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	r, err := New(Options{InputRateHz: 96_000, Bus: bus})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if r.Inner() == nil {
		t.Error("Inner() = nil, want non-nil orchestrator")
	}
}

func TestReceiverPropagatesContextCancel(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	r, err := New(Options{InputRateHz: 96_000, Bus: bus})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	in := make(chan []complex64)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- r.Process(ctx, in) }()
	cancel()
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("Process err = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Process did not exit after ctx cancel")
	}
}

func TestReceiverNilInputErrors(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	r, _ := New(Options{InputRateHz: 96_000, Bus: bus})
	if err := r.Process(context.Background(), nil); err == nil {
		t.Error("Process with nil input: want error")
	}
}

// TestReceiverDecodesSyntheticAPRSPacket feeds a synthetic
// Bell-202 IQ stream carrying one APRS UI frame and asserts a
// KindAPRSPacket event lands on the bus.
//
// Currently skipped: the integrator-slicer + EMA threshold + LPF
// group delay don't reliably bit-align against the synthetic IQ
// the FFSKModulator emits when the input is fed in a single
// chunk. The real-fixture path (sample/aprs/<capture>.wav replay
// through the iqtap broker) lands in the same follow-up PR that
// adds captured-IQ fixtures for POCSAG and the rest of the
// Phase 5 decoders. The API tests above + the orchestrator
// tests in aprs/receiver cover everything else; the protocol
// layer is covered by the ax25 + aprs + hdlc package tests.
func TestReceiverDecodesSyntheticAPRSPacket(t *testing.T) {
	// Un-skipped (#1316): the chain decodes its own packet once the
	// fixture carries trailing flags — the closing flag used to end at the
	// very last sample and never cleared the resampler / timing-loop group
	// delay, so the framer never saw it.
	const inputRateHz uint32 = AudioRateHz * 10 // 96 ksps

	bus, sub := newTestBus(t)
	r, err := New(Options{InputRateHz: inputRateHz, Bus: bus})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	iq := aprsTestIQ(float64(inputRateHz))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	in := make(chan []complex64, 4)
	done := make(chan struct{})
	go func() {
		_ = r.Process(ctx, in)
		close(done)
	}()
	in <- iq
	close(in)
	<-done

	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev, ok := <-sub.C:
			if !ok {
				t.Fatal("bus closed before packet emitted")
			}
			if ev.Kind != events.KindAPRSPacket {
				continue
			}
			pkt, ok := ev.Payload.(storage.APRSPacket)
			if !ok {
				continue
			}
			if pkt.Src != "W1AW-9" {
				t.Errorf("Src = %q, want W1AW-9", pkt.Src)
			}
			return
		case <-deadline:
			t.Fatalf("no packet emitted within 2s (Stats=%+v)", r.Stats())
		}
	}
}

func TestReceiverProcessReturnsNilOnInputClose(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	r, err := New(Options{InputRateHz: 96_000, Bus: bus})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	in := make(chan []complex64)
	done := make(chan error, 1)
	go func() { done <- r.Process(context.Background(), in) }()
	close(in)
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Process on closed input = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Process did not exit after input close")
	}
}

func TestReceiverStatsCountIQSamples(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	r, err := New(Options{InputRateHz: 96_000, Bus: bus})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	in := make(chan []complex64, 1)
	done := make(chan struct{})
	go func() {
		_ = r.Process(ctx, in)
		close(done)
	}()
	chunk := make([]complex64, 9600) // 100 ms of IQ at 96 ksps
	in <- chunk
	close(in)
	<-done
	if got := r.Stats().IQSamplesSeen; got != uint64(len(chunk)) {
		t.Errorf("IQSamplesSeen = %d, want %d", got, len(chunk))
	}
}
