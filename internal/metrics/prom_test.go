package metrics

import (
	"context"
	"io"
	"math"
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
	m, err := New(bus, nil, "v0.0.0-test")
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
	m, _ := New(bus, nil, "test")
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

	active := m.activeCalls.WithLabelValues("Alpha", "unknown")
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if testutil.ToFloat64(active) == 2 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := testutil.ToFloat64(active); got != 2 {
		t.Errorf("active = %v, want 2", got)
	}

	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant: cs.Grant, DeviceSerial: cs.DeviceSerial,
		StartedAt: cs.StartedAt, EndedAt: cs.StartedAt.Add(time.Second),
		Reason: trunking.EndReasonNormal,
	}})
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if testutil.ToFloat64(active) == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := testutil.ToFloat64(active); got != 1 {
		t.Errorf("active after end = %v, want 1", got)
	}
	if got := testutil.ToFloat64(m.callsTotal.WithLabelValues("Alpha", "unknown", "false", "normal")); got != 1 {
		t.Errorf("calls_total{Alpha,unknown,false,normal} = %v, want 1", got)
	}
}

func TestDecodeErrorEventIncrementsCounter(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	m, _ := New(bus, nil, "test")
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
	m, _ := New(nil, nil, "test")
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
	m, _ := New(nil, nil, "v1.2.3")
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
	m, _ := New(bus, nil, "test")
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
	m, err := New(bus, nil, "v0.0.0-test")
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

func TestCallsTotalCarriesGrantDimensions(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	m, _ := New(bus, nil, "test")
	defer m.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go m.Run(ctx)

	grant := trunking.Grant{
		System:    "NET-7",
		Protocol:  "p25",
		GroupID:   42,
		Encrypted: true,
	}
	start := time.Now()
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: trunking.CallStart{
		Grant: grant, DeviceSerial: "Z", StartedAt: start,
	}})

	started := m.callsStarted.WithLabelValues("NET-7", "p25", "true")
	active := m.activeCalls.WithLabelValues("NET-7", "p25")
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if testutil.ToFloat64(started) == 1 && testutil.ToFloat64(active) == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := testutil.ToFloat64(started); got != 1 {
		t.Errorf("calls_started_total{NET-7,p25,true} = %v, want 1", got)
	}
	if got := testutil.ToFloat64(active); got != 1 {
		t.Errorf("calls_active{NET-7,p25} = %v, want 1", got)
	}

	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant: grant, DeviceSerial: "Z",
		StartedAt: start, EndedAt: start.Add(time.Second),
		Reason: trunking.EndReasonPreempted,
	}})

	total := m.callsTotal.WithLabelValues("NET-7", "p25", "true", "preempted")
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if testutil.ToFloat64(total) == 1 && testutil.ToFloat64(active) == 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := testutil.ToFloat64(total); got != 1 {
		t.Errorf("calls_total{NET-7,p25,true,preempted} = %v, want 1", got)
	}
	if got := testutil.ToFloat64(active); got != 0 {
		t.Errorf("calls_active{NET-7,p25} after end = %v, want 0", got)
	}
}

type fakeLockState struct {
	freq uint32
	sys  string
}

func (f fakeLockState) LockedFrequencyHz() uint32 { return f.freq }
func (f fakeLockState) LockedNAC() uint16         { return 0 }
func (f fakeLockState) SystemName() string        { return f.sys }

func TestControlChannelFrequencyAndTransitions(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	m, _ := New(bus, nil, "test")
	defer m.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go m.Run(ctx)

	const freqHz = uint32(851012500)
	bus.Publish(events.Event{Kind: events.KindCCLocked, Payload: fakeLockState{freq: freqHz, sys: "S"}})

	freqGauge := m.ccFrequencyHz.WithLabelValues("S")
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if testutil.ToFloat64(freqGauge) == float64(freqHz) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := testutil.ToFloat64(freqGauge); got != float64(freqHz) {
		t.Errorf("control_channel_frequency_hz{S} = %v, want %v", got, freqHz)
	}
	if got := testutil.ToFloat64(m.ccTransitions.WithLabelValues("S", "locked")); got != 1 {
		t.Errorf("transitions{S,locked} = %v, want 1", got)
	}

	bus.Publish(events.Event{Kind: events.KindCCLost, Payload: fakeLockState{freq: freqHz, sys: "S"}})

	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if testutil.ToFloat64(m.ccTransitions.WithLabelValues("S", "lost")) == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := testutil.ToFloat64(m.ccTransitions.WithLabelValues("S", "lost")); got != 1 {
		t.Errorf("transitions{S,lost} = %v, want 1", got)
	}

	srv := httptest.NewServer(m.Handler())
	defer srv.Close()
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(body), `gophertrunk_control_channel_frequency_hz{system="S"}`) {
		t.Errorf("control_channel_frequency_hz{S} should be deleted after CC lost; body:\n%s", string(body))
	}
}

type fakePool struct {
	statuses []sdr.SDRStatus
}

func (f *fakePool) Snapshot() []sdr.SDRStatus { return f.statuses }

func TestSDRSnapshotCollectorEmitsPerDevice(t *testing.T) {
	pool := &fakePool{statuses: []sdr.SDRStatus{
		{Driver: "rtlsdr", Serial: "A", Role: "control", Attached: true, GainTenthDB: 296, GainAuto: false, PPM: -2, BiasTee: true},
		{Driver: "rtlsdr", Serial: "B", Role: "voice", Attached: true, GainAuto: true, PPM: 0, BiasTee: false},
		{Driver: "rtlsdr", Serial: "C", Role: "voice", Attached: false, GainTenthDB: 100},
	}}
	m, err := New(nil, pool, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

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
		`gophertrunk_sdr_ppm{driver="rtlsdr",role="control",serial="A"} -2`,
		`gophertrunk_sdr_bias_tee{driver="rtlsdr",role="control",serial="A"} 1`,
		`gophertrunk_sdr_gain_auto{driver="rtlsdr",role="control",serial="A"} 0`,
		`gophertrunk_sdr_gain_db{driver="rtlsdr",role="control",serial="A"} 29.6`,
		`gophertrunk_sdr_gain_auto{driver="rtlsdr",role="voice",serial="B"} 1`,
		`gophertrunk_sdr_gain_db{driver="rtlsdr",role="voice",serial="B"} NaN`,
		`gophertrunk_sdr_bias_tee{driver="rtlsdr",role="voice",serial="B"} 0`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("scrape missing %q.\nbody:\n%s", want, text)
		}
	}
	if strings.Contains(text, `serial="C"`) {
		t.Errorf("detached device C should not appear in snapshot; body:\n%s", text)
	}

	// Spot-check the NaN path through the registry gather, since the
	// scrape text always renders NaN identically — Gather lets us check
	// the actual float bits.
	families, err := m.Registry().Gather()
	if err != nil {
		t.Fatal(err)
	}
	var foundNaN bool
	for _, fam := range families {
		if fam.GetName() != "gophertrunk_sdr_gain_db" {
			continue
		}
		for _, mf := range fam.GetMetric() {
			var serial string
			for _, lp := range mf.GetLabel() {
				if lp.GetName() == "serial" {
					serial = lp.GetValue()
				}
			}
			if serial == "B" && math.IsNaN(mf.GetGauge().GetValue()) {
				foundNaN = true
			}
		}
	}
	if !foundNaN {
		t.Errorf("expected NaN gain_db for serial B (AGC), did not find it")
	}
}
