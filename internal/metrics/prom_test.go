package metrics

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestEventsCounterIncrements(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	m, err := New(bus, "v0.0.0-test")
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go m.Run(ctx)

	for i := 0; i < 3; i++ {
		bus.Publish(events.Event{Kind: events.KindCCLocked, Payload: nil})
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if testutil.ToFloat64(m.eventsTotal.WithLabelValues("cc.locked")) >= 3 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	got := testutil.ToFloat64(m.eventsTotal.WithLabelValues("cc.locked"))
	if got != 3 {
		t.Errorf("cc.locked counter = %v, want 3", got)
	}
}

func TestCallsActiveTracksStartAndEnd(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	m, _ := New(bus, "test")
	defer m.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go m.Run(ctx)

	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "Alpha", GroupID: 1, FrequencyHz: 1},
		DeviceSerial: "X",
		StartedAt:    time.Now(),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: trunking.CallStart{
		Grant: trunking.Grant{System: "Alpha", GroupID: 2, FrequencyHz: 2}, DeviceSerial: "Y",
		StartedAt: time.Now(),
	}})

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if testutil.ToFloat64(m.activeCalls) == 2 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := testutil.ToFloat64(m.activeCalls); got != 2 {
		t.Errorf("active = %v, want 2", got)
	}

	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant: cs.Grant, DeviceSerial: cs.DeviceSerial,
		StartedAt: cs.StartedAt, EndedAt: cs.StartedAt.Add(time.Second),
		Reason: trunking.EndReasonNormal,
	}})
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if testutil.ToFloat64(m.activeCalls) == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := testutil.ToFloat64(m.activeCalls); got != 1 {
		t.Errorf("active after end = %v, want 1", got)
	}
	if got := testutil.ToFloat64(m.callsTotal.WithLabelValues("normal")); got != 1 {
		t.Errorf("calls_total{normal} = %v, want 1", got)
	}
}

func TestDecodeErrorEventIncrementsCounter(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	m, _ := New(bus, "test")
	defer m.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go m.Run(ctx)

	for i := 0; i < 4; i++ {
		bus.Publish(events.Event{
			Kind:    events.KindDecodeError,
			Payload: events.DecodeError{Protocol: "p25", Stage: events.StageNIDBCH},
		})
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if testutil.ToFloat64(m.decodeErrors.WithLabelValues("p25", "nid-bch")) >= 4 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := testutil.ToFloat64(m.decodeErrors.WithLabelValues("p25", "nid-bch")); got != 4 {
		t.Errorf("decode_errors{p25,nid-bch} = %v, want 4", got)
	}
}

func TestRecordHooks(t *testing.T) {
	m, _ := New(nil, "test")
	defer m.Close()

	m.RecordIQUnderrun("rtlsdr", "AAA")
	m.RecordIQUnderrun("rtlsdr", "AAA")
	m.RecordUSBReconnect("rtlsdr", "AAA")
	m.RecordDecodeError("p25", "tsbk-crc")
	m.SetSDRAttached("rtlsdr", "AAA", "control", true)

	if got := testutil.ToFloat64(m.iqUnderruns.WithLabelValues("rtlsdr", "AAA")); got != 2 {
		t.Errorf("iq underruns = %v, want 2", got)
	}
	if got := testutil.ToFloat64(m.usbReconnects.WithLabelValues("rtlsdr", "AAA")); got != 1 {
		t.Errorf("usb reconnects = %v, want 1", got)
	}
	if got := testutil.ToFloat64(m.decodeErrors.WithLabelValues("p25", "tsbk-crc")); got != 1 {
		t.Errorf("decode errors = %v, want 1", got)
	}
	if got := testutil.ToFloat64(m.sdrAttached.WithLabelValues("rtlsdr", "AAA", "control")); got != 1 {
		t.Errorf("sdr attached = %v, want 1", got)
	}
}

func TestHandlerScrapeContainsExpectedSeries(t *testing.T) {
	m, _ := New(nil, "v1.2.3")
	defer m.Close()
	m.RecordDecodeError("p25", "tsbk-crc")

	srv := httptest.NewServer(m.Handler())
	defer srv.Close()
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	text := string(body)
	for _, want := range []string{
		`gophertrunk_build_info{version="v1.2.3"} 1`,
		`gophertrunk_decode_errors_total{protocol="p25",stage="tsbk-crc"} 1`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("scrape missing %q.\nbody:\n%s", want, text)
		}
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	bus := events.NewBus(2)
	defer bus.Close()
	m, _ := New(bus, "test")
	go m.Run(context.Background())
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if err := m.Close(); err != nil {
		t.Errorf("second Close = %v, want nil", err)
	}
}

func TestSDRAttachedGaugeFromBusEvents(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	m, err := New(bus, "v0.0.0-test")
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go m.Run(ctx)

	st := sdr.SDRStatus{Driver: "rtlsdr", Serial: "00000001", Role: "control"}
	bus.Publish(events.Event{Kind: events.KindSDRAttached, Payload: st})

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if testutil.ToFloat64(m.sdrAttached.WithLabelValues(st.Driver, st.Serial, st.Role)) == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := testutil.ToFloat64(m.sdrAttached.WithLabelValues(st.Driver, st.Serial, st.Role)); got != 1 {
		t.Errorf("after KindSDRAttached: gauge = %v, want 1", got)
	}

	bus.Publish(events.Event{Kind: events.KindSDRDetached, Payload: st})

	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if testutil.ToFloat64(m.sdrAttached.WithLabelValues(st.Driver, st.Serial, st.Role)) == 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := testutil.ToFloat64(m.sdrAttached.WithLabelValues(st.Driver, st.Serial, st.Role)); got != 0 {
		t.Errorf("after KindSDRDetached: gauge = %v, want 0", got)
	}
}
